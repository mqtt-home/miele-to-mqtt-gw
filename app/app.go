package main

import (
	"context"
	"sync"
	"time"

	"github.com/mqtt-home/miele-to-mqtt-gw/bridge"
	"github.com/mqtt-home/miele-to-mqtt-gw/config"
	"github.com/mqtt-home/miele-to-mqtt-gw/metrics"
	"github.com/mqtt-home/miele-to-mqtt-gw/miele/api"
	"github.com/mqtt-home/miele-to-mqtt-gw/miele/login"
	"github.com/mqtt-home/miele-to-mqtt-gw/miele/sse"
	"github.com/mqtt-home/miele-to-mqtt-gw/miele/transform"
	"github.com/philipparndt/go-logger"
	"github.com/philipparndt/mqtt-gateway/mqtt"
)

// app wires together login + SSE + polling + MQTT publishing. It is created
// once per run by main, after config has been loaded and MQTT has connected.
type app struct {
	cfg     config.Config
	mgr     *login.Manager
	api     *api.Client
	pub     *bridge.Publisher
	sse     *sse.Client

	mu            sync.Mutex
	stopRefresh   chan struct{}
	stopPolling   chan struct{}
	refreshDone   chan struct{}
	pollingDone   chan struct{}
}

func newApp(cfg config.Config, mgr *login.Manager) *app {
	return &app{
		cfg:    cfg,
		mgr:    mgr,
		api:    api.NewClient(),
		pub:    bridge.New(cfg),
	}
}

// start launches SSE (when configured), the polling loop, and the periodic
// token-refresh check. It does NOT block.
func (a *app) start(ctx context.Context) {
	metrics.SetSSEConnection("unknown")
	a.pub.PublishMieleState("unknown")
	a.mgr.SetOnRefresh(metrics.RecordTokenRefresh)

	if a.cfg.Miele.Mode == "sse" {
		backoff := a.cfg.Miele.SSEBackoff
		a.sse = sse.Start(sse.Options{
			AccessToken:      a.mgr.CurrentAccessToken,
			OnDevices:        a.onDevices,
			OnEvent:          metrics.RecordSSEEvent,
			OnPollingHealthy: pollingHealthy,
			OnFailure:        metrics.RecordSSEFailure,
			OnSuccess:        metrics.RecordSSESuccess,
			OnStatus: func(s string) {
				switch s {
				case "connected":
					metrics.SetSSEConnection("connected")
					a.pub.PublishMieleState("connected")
				case "disconnected":
					metrics.SetSSEConnection("disconnected")
					a.pub.PublishMieleState("disconnected")
				case "degraded":
					metrics.SetSSEConnection("degraded")
					a.pub.PublishMieleState("degraded")
				}
			},
			BaseReconnectDelay: backoff.BaseDelayDuration(),
			MaxReconnectDelay:  backoff.MaxDelayDuration(),
			FailureThreshold:   backoff.FailureThreshold,
		})
	} else {
		logger.Info("SSE disabled; running in polling-only mode")
	}

	a.startPolling(ctx)
	a.startRefreshCheck(ctx)
}

// onDevices is the single device-update path used by both SSE and polling.
func (a *app) onDevices(devs []api.Device) {
	now := time.Now()
	for _, d := range devs {
		small := transform.Build(d, now)
		metrics.RecordDevice(d.ID, small)
		a.pub.PublishDevice(d.ID, small, []byte(d.Data))
	}
}

// startPolling runs the polling loop. In SSE mode it still runs as a fallback
// (per commit 7620b8e); the dedup layer suppresses redundant publishes.
func (a *app) startPolling(ctx context.Context) {
	interval := time.Duration(a.cfg.Miele.PollingInterval) * time.Second
	if interval <= 0 {
		interval = 60 * time.Second
	}
	a.stopPolling = make(chan struct{})
	a.pollingDone = make(chan struct{})

	logger.Info("Polling started", "interval", interval)

	// Run an initial fetch right away so retained MQTT state appears quickly.
	go func() {
		defer close(a.pollingDone)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		a.pollOnce(ctx)
		for {
			select {
			case <-a.stopPolling:
				return
			case <-ctx.Done():
				return
			case <-ticker.C:
				a.pollOnce(ctx)
			}
		}
	}()
}

func (a *app) pollOnce(ctx context.Context) {
	tok := a.mgr.CurrentAccessToken()
	if tok == "" {
		logger.Warn("Polling skipped: no access token")
		return
	}
	logger.Debug("Polling devices")
	now := time.Now()
	metrics.RecordPollAttempt(now)
	devs, err := a.api.FetchDevices(ctx, tok)
	if err != nil {
		metrics.RecordPollError(time.Now(), err)
		logger.Error("Polling failed", "error", err)
		return
	}
	first := metrics.PollSuccessTotal() == 0
	metrics.RecordPollSuccess(time.Now())
	if first {
		ids := make([]string, 0, len(devs))
		for _, d := range devs {
			ids = append(ids, d.ID)
		}
		logger.Info("Polled devices", "devices", ids)
	}
	a.onDevices(devs)
}

// startRefreshCheck runs the once-per-minute token-refresh decision loop. On
// a positive decision it forces an SSE close (so the next reconnect uses the
// new bearer) and re-runs Login.
func (a *app) startRefreshCheck(ctx context.Context) {
	a.stopRefresh = make(chan struct{})
	a.refreshDone = make(chan struct{})

	go func() {
		defer close(a.refreshDone)
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-a.stopRefresh:
				return
			case <-ctx.Done():
				return
			case <-ticker.C:
				a.maybeRefresh(ctx)
			}
		}
	}()
}

func (a *app) maybeRefresh(ctx context.Context) {
	tok := a.mgr.Current()
	if !login.NeedsRefresh(tok, time.Now()) {
		return
	}
	logger.Info("Token refresh required. Reconnecting.")
	if a.sse != nil {
		a.sse.Close()
	}
	if _, err := a.mgr.Login(ctx); err != nil {
		logger.Error("Re-login failed", "error", err)
	}
}

// pollingHealthy reports whether the polling loop has had at least one
// successful poll since process start. The SSE client uses this to decide
// whether a sustained failure streak should surface as `degraded` (polling
// is filling in) or stay on `disconnected` (nothing is working).
func pollingHealthy() bool {
	return metrics.PollSuccessTotal() > 0
}

// stop tears down all background loops.
func (a *app) stop() {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.sse != nil {
		a.sse.Stop()
	}
	if a.stopPolling != nil {
		close(a.stopPolling)
		<-a.pollingDone
		a.stopPolling = nil
	}
	if a.stopRefresh != nil {
		close(a.stopRefresh)
		<-a.refreshDone
		a.stopRefresh = nil
	}
	// Publish a final disconnected state so subscribers see a clean transition.
	mqtt.PublishAbsolute(a.cfg.MieleStateTopic(), "disconnected", a.cfg.MQTT.Retain)
}
