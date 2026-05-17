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
	// OnStatus is invoked with `connected` after the body opens and
	// `disconnected` whenever a stream ends. May be nil.
	OnStatus func(state string)
	// WatchdogTimeout, if >0, closes the active stream when no event has been
	// received for that long. Mirrors hue-to-mqtt-gw's watchdog.
	WatchdogTimeout time.Duration
	// ReconnectDelay between attempts. Defaults to 5s.
	ReconnectDelay time.Duration
	// HTTPClient lets tests swap the transport.
	HTTPClient *http.Client
	// URL lets tests point at a stub server.
	URL string
}

// Client reads the Miele SSE stream until stopped, dispatching device updates
// and surfacing connection state to the caller.
type Client struct {
	opts Options

	mu        sync.Mutex
	resp      *http.Response
	lastEvent time.Time
	stopOnce  sync.Once
	stopCh    chan struct{}
	watchdog  *time.Ticker
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
	if opts.ReconnectDelay <= 0 {
		opts.ReconnectDelay = 5 * time.Second
	}
	c := &Client{opts: opts, stopCh: make(chan struct{})}
	go c.loop()
	return c
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

		if c.opts.OnStatus != nil {
			c.opts.OnStatus("disconnected")
		}

		select {
		case <-c.stopCh:
			return
		case <-time.After(c.opts.ReconnectDelay):
			logger.Info("[SSE] Reconnecting...")
		}
	}
}

func (c *Client) connect() {
	token := c.opts.AccessToken()
	if token == "" {
		logger.Warn("[SSE] No access token, skipping connect")
		return
	}

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

	if c.opts.OnStatus != nil {
		c.opts.OnStatus("connected")
	}
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
		if c.opts.OnDevices != nil {
			c.opts.OnDevices(devs)
		}
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
