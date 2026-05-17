package sse

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/mqtt-home/miele-to-mqtt-gw/miele/api"
	"github.com/philipparndt/go-logger"
)

// URL is the Miele Server-Sent Events endpoint. Tests can override Client.URL
// directly.
const URL = "https://api.mcs3.miele.com/v1/devices/all/events"

// Options is the wiring passed to StartSSE.
type Options struct {
	// AccessToken supplies the bearer used on every connection attempt. It is
	// read each time a new SSE stream is established, so callers can swap in a
	// freshly-refreshed token by updating the underlying source and calling
	// Close to trigger a reconnect.
	AccessToken func() string
	// OnDevices is invoked with the device set parsed from each `devices`
	// event payload.
	OnDevices func(devices []api.Device)
	// OnEvent fires once per successfully-dispatched event with the time of
	// dispatch. Used by metrics; may be nil.
	OnEvent func(now time.Time)
	// OnStatus is invoked with `connected` after the body opens and an event
	// is dispatched (or `connected` is held), `disconnected` whenever a stream
	// ends below the backoff threshold, and `degraded` when the failure streak
	// reaches the threshold while polling is healthy. May be nil.
	OnStatus func(state string)
	// OnPollingHealthy reports whether the parallel polling loop has had at
	// least one successful poll since process start. When non-nil and the
	// failure streak has reached FailureThreshold, the status callback emits
	// `degraded` instead of `disconnected`. May be nil (treated as always
	// false — i.e. the bridge stays on `disconnected`).
	OnPollingHealthy func() bool
	// OnFailure fires once per registered failure with the post-increment
	// streak count and the absolute time of the next reconnect attempt. Used
	// by metrics; may be nil.
	OnFailure func(streak int, nextRetry time.Time)
	// OnSuccess fires once on the first successful event dispatch that resets
	// the streak. Used by metrics; may be nil.
	OnSuccess func()
	// WatchdogTimeout, if >0, closes the active stream when no event has been
	// received for that long. Mirrors hue-to-mqtt-gw's watchdog.
	WatchdogTimeout time.Duration
	// ReconnectDelay between attempts. Kept for back-compat; if set and
	// BaseReconnectDelay is unset, it is used as the base delay. Defaults to
	// 5s when both are unset.
	ReconnectDelay time.Duration
	// BaseReconnectDelay is the delay between attempts while the consecutive
	// failure streak is below FailureThreshold.
	BaseReconnectDelay time.Duration
	// MaxReconnectDelay caps the step-table delay used at and beyond the
	// failure threshold.
	MaxReconnectDelay time.Duration
	// FailureThreshold is the consecutive-failure count at which the backoff
	// step table engages. Defaults to 5 when zero.
	FailureThreshold int
	// HTTPClient lets tests swap the transport.
	HTTPClient *http.Client
	// URL lets tests point at a stub server.
	URL string
}

// Client reads the Miele SSE stream until stopped, dispatching device updates
// and surfacing connection state to the caller.
type Client struct {
	opts Options

	// backoffTable holds the precomputed step values used at and beyond
	// FailureThreshold. The last entry equals MaxReconnectDelay.
	backoffTable []time.Duration

	mu                       sync.Mutex
	resp                     *http.Response
	lastEvent                time.Time
	stopOnce                 sync.Once
	stopCh                   chan struct{}
	watchdog                 *time.Ticker
	consecutiveFailures      int
	dispatchedThisConnection bool
	nextRetryAt              time.Time
	lastStatus               string
}

// Start launches the SSE client in a goroutine and returns it. Call Stop to
// terminate.
func Start(opts Options) *Client {
	if opts.HTTPClient == nil {
		opts.HTTPClient = http.DefaultClient
	}
	if opts.URL == "" {
		opts.URL = URL
	}
	if opts.BaseReconnectDelay <= 0 {
		if opts.ReconnectDelay > 0 {
			opts.BaseReconnectDelay = opts.ReconnectDelay
		} else {
			opts.BaseReconnectDelay = 5 * time.Second
		}
	}
	if opts.MaxReconnectDelay <= 0 {
		opts.MaxReconnectDelay = 10 * time.Minute
	}
	if opts.MaxReconnectDelay < opts.BaseReconnectDelay {
		opts.MaxReconnectDelay = opts.BaseReconnectDelay
	}
	if opts.FailureThreshold <= 0 {
		opts.FailureThreshold = 5
	}
	c := &Client{opts: opts, stopCh: make(chan struct{})}
	c.backoffTable = buildBackoffTable(opts.BaseReconnectDelay, opts.MaxReconnectDelay)
	go c.loop()
	return c
}

// buildBackoffTable returns the step delays applied at and beyond the failure
// threshold. The first step is meaningfully larger than the base (~6x) and
// the last entry equals max. Steps grow ~6x each so the default 5s→10m chain
// is 30s, 3m, 10m; a 5s base with 10m max yields [30s, 3m, 10m].
func buildBackoffTable(base, max time.Duration) []time.Duration {
	if max <= base {
		return []time.Duration{max}
	}
	table := []time.Duration{}
	d := base * 6
	for d < max {
		table = append(table, d)
		d *= 6
	}
	table = append(table, max)
	return table
}

// Stop ends the read loop and closes any active connection. Safe to call
// concurrently and more than once.
func (c *Client) Stop() {
	c.stopOnce.Do(func() {
		close(c.stopCh)
		c.closeConn()
		c.stopWatchdog()
	})
}

// Close forces the current stream to disconnect so the loop reconnects with a
// fresh token (used by the token-refresh path).
func (c *Client) Close() {
	c.closeConn()
}

func (c *Client) loop() {
	for {
		select {
		case <-c.stopCh:
			return
		default:
		}

		c.connect()

		// Connect returned; this attempt produced no further events. If no
		// event was dispatched on this connection, count it as a failure.
		c.mu.Lock()
		dispatched := c.dispatchedThisConnection
		c.mu.Unlock()
		if !dispatched {
			c.recordFailure()
		}

		delay := c.selectBackoffDelay()
		c.emitPostAttemptStatus()

		select {
		case <-c.stopCh:
			return
		case <-time.After(delay):
			logger.Info("[SSE] Reconnecting...")
		}
	}
}

// selectBackoffDelay picks the next reconnect delay and updates nextRetryAt.
// Returns the base delay while below threshold; otherwise the step-table
// value at index (streak - threshold), saturating at the last entry.
func (c *Client) selectBackoffDelay() time.Duration {
	c.mu.Lock()
	defer c.mu.Unlock()

	var delay time.Duration
	if c.consecutiveFailures < c.opts.FailureThreshold {
		delay = c.opts.BaseReconnectDelay
	} else {
		idx := c.consecutiveFailures - c.opts.FailureThreshold
		if idx >= len(c.backoffTable) {
			idx = len(c.backoffTable) - 1
		}
		delay = c.backoffTable[idx]
	}
	c.nextRetryAt = time.Now().Add(delay)
	return delay
}

// emitPostAttemptStatus signals `degraded` when the streak has crossed the
// threshold and polling is healthy; otherwise emits `disconnected`. Called
// once per connect attempt after the attempt has ended.
func (c *Client) emitPostAttemptStatus() {
	if c.opts.OnStatus == nil {
		return
	}
	c.mu.Lock()
	streak := c.consecutiveFailures
	threshold := c.opts.FailureThreshold
	c.mu.Unlock()

	status := "disconnected"
	if streak >= threshold && c.pollingHealthy() {
		status = "degraded"
	}
	c.emitStatus(status)
}

// emitStatus calls OnStatus only when the new state differs from the
// last-emitted state — this keeps MQTT publishes idempotent across retries.
func (c *Client) emitStatus(state string) {
	if c.opts.OnStatus == nil {
		return
	}
	c.mu.Lock()
	same := c.lastStatus == state
	c.lastStatus = state
	c.mu.Unlock()
	if !same {
		c.opts.OnStatus(state)
	}
}

func (c *Client) pollingHealthy() bool {
	if c.opts.OnPollingHealthy == nil {
		return false
	}
	return c.opts.OnPollingHealthy()
}

// recordFailure increments the streak and notifies metrics.
func (c *Client) recordFailure() {
	c.mu.Lock()
	c.consecutiveFailures++
	streak := c.consecutiveFailures
	c.mu.Unlock()

	if c.opts.OnFailure != nil {
		// Estimate the next-retry timestamp using the same logic as
		// selectBackoffDelay would compute; the loop will overwrite it
		// immediately, but this gives metrics readers a consistent value.
		c.opts.OnFailure(streak, c.estimateNextRetry(streak))
	}
}

func (c *Client) estimateNextRetry(streak int) time.Time {
	var delay time.Duration
	if streak < c.opts.FailureThreshold {
		delay = c.opts.BaseReconnectDelay
	} else {
		idx := streak - c.opts.FailureThreshold
		if idx >= len(c.backoffTable) {
			idx = len(c.backoffTable) - 1
		}
		delay = c.backoffTable[idx]
	}
	return time.Now().Add(delay)
}

// recordSuccess resets the streak on first event dispatch after a (re)connect.
func (c *Client) recordSuccess() {
	c.mu.Lock()
	hadStreak := c.consecutiveFailures > 0
	c.consecutiveFailures = 0
	c.dispatchedThisConnection = true
	c.nextRetryAt = time.Time{}
	c.mu.Unlock()

	if hadStreak && c.opts.OnSuccess != nil {
		c.opts.OnSuccess()
	}
}

func (c *Client) connect() {
	token := c.opts.AccessToken()
	if token == "" {
		logger.Warn("[SSE] No access token, skipping connect")
		return
	}

	c.mu.Lock()
	c.dispatchedThisConnection = false
	c.nextRetryAt = time.Time{}
	c.mu.Unlock()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.opts.URL, nil)
	if err != nil {
		logger.Error("[SSE] build request", "error", err)
		return
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Accept-Language", "en-GB")

	logger.Info("[SSE] Connecting")
	resp, err := c.opts.HTTPClient.Do(req)
	if err != nil {
		if !errors.Is(err, context.Canceled) {
			logger.Error("[SSE] connect failed", "error", err)
		}
		return
	}
	if resp.StatusCode != http.StatusOK {
		logger.Error("[SSE] non-200 status", "status", resp.StatusCode)
		resp.Body.Close()
		return
	}

	c.mu.Lock()
	c.resp = resp
	c.lastEvent = time.Now()
	c.mu.Unlock()

	c.startWatchdog()

	// Body opened — emit connected. If a dispatch follows, it stays connected;
	// if the body ends without a dispatch, the loop will register a failure
	// and may flip to disconnected/degraded on the next attempt.
	c.emitStatus("connected")
	logger.Info("[SSE] Connected, reading events")

	c.readLoop(resp)

	resp.Body.Close()
	c.stopWatchdog()
}

// readLoop consumes the SSE wire format: lines beginning with `event:` set the
// current event name; `data:` lines accumulate; an empty line dispatches the
// accumulated payload.
func (c *Client) readLoop(resp *http.Response) {
	scanner := bufio.NewScanner(resp.Body)
	// Miele payloads can include the full device dump, which is sizable.
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)

	var (
		eventName string
		dataLines []string
	)

	dispatch := func() {
		defer func() {
			eventName = ""
			dataLines = nil
		}()
		if len(dataLines) == 0 {
			return
		}
		full := strings.Join(dataLines, "\n")

		c.mu.Lock()
		c.lastEvent = time.Now()
		c.mu.Unlock()

		// Miele emits a `devices` event whose payload is an object keyed by
		// device id. Other event names (ping, etc.) are ignored.
		if eventName != "devices" {
			logger.Trace("[SSE] Ignoring event", "type", eventName, "bytes", len(full))
			return
		}

		raw := map[string]json.RawMessage{}
		if err := json.Unmarshal([]byte(full), &raw); err != nil {
			logger.Error("[SSE] parse devices", "error", err)
			return
		}
		devs := make([]api.Device, 0, len(raw))
		for id, payload := range raw {
			devs = append(devs, api.Device{ID: id, Data: payload})
		}
		if c.opts.OnEvent != nil {
			c.opts.OnEvent(time.Now())
		}
		if c.opts.OnDevices != nil {
			c.opts.OnDevices(devs)
		}
		// A successful dispatch resets the streak and re-asserts `connected`
		// (no-op if already connected).
		c.recordSuccess()
		c.emitStatus("connected")
	}

	for scanner.Scan() {
		select {
		case <-c.stopCh:
			return
		default:
		}
		line := scanner.Text()
		switch {
		case strings.HasPrefix(line, "data:"):
			dataLines = append(dataLines, strings.TrimPrefix(strings.TrimPrefix(line, "data:"), " "))
		case strings.HasPrefix(line, "event:"):
			eventName = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
		case line == "":
			dispatch()
		case strings.HasPrefix(line, ":") || strings.HasPrefix(line, "id:"):
			// SSE comment or id, ignore.
		}
	}
	if err := scanner.Err(); err != nil {
		logger.Debug("[SSE] scanner error", "error", err)
	}
}

func (c *Client) closeConn() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.resp != nil {
		c.resp.Body.Close()
		c.resp = nil
	}
}

func (c *Client) startWatchdog() {
	if c.opts.WatchdogTimeout <= 0 {
		logger.Debug("[SSE] watchdog disabled")
		return
	}
	timeout := c.opts.WatchdogTimeout
	logger.Info("[SSE] watchdog enabled", "timeout", timeout)
	c.watchdog = time.NewTicker(timeout / 2)
	go func(ticker *time.Ticker) {
		for {
			select {
			case <-c.stopCh:
				return
			case _, ok := <-ticker.C:
				if !ok {
					return
				}
				c.mu.Lock()
				elapsed := time.Since(c.lastEvent)
				c.mu.Unlock()
				if elapsed > timeout {
					logger.Warn("[SSE] watchdog triggered, closing stream", "elapsed", elapsed)
					c.closeConn()
					return
				}
			}
		}
	}(c.watchdog)
}

func (c *Client) stopWatchdog() {
	c.mu.Lock()
	w := c.watchdog
	c.watchdog = nil
	c.mu.Unlock()
	if w != nil {
		w.Stop()
	}
}
