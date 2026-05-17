// Package metrics exposes runtime values from the Miele bridge via expvar.
//
// One top-level expvar named "miele" is published once on Init() and surfaced
// at /debug/vars alongside the "mqtt" expvar from mqtt-gateway. Update entry
// points are package-level functions that the app/SSE/login layers call at
// existing event boundaries.
package metrics

import (
	"expvar"
	"sync"
	"sync/atomic"
	"time"

	"github.com/mqtt-home/miele-to-mqtt-gw/miele/transform"
)

// State holds the runtime values exposed under the "miele" expvar.
//
// Mutable fields are guarded by `mu`. Counters that get written on every
// SSE/poll dispatch use atomic.Int64 to avoid taking the mutex on the hot
// path; everything else is read/written under the lock.
type state struct {
	mu sync.RWMutex

	connection string

	devices map[string]transform.SmallMessage

	sseLastEvent            time.Time
	sseEventsTotal          atomic.Int64
	sseConsecutiveFailures  int
	sseNextRetryAt          time.Time

	pollLastAttempt  time.Time
	pollLastSuccess  time.Time
	pollLastError    time.Time
	pollErrorMessage string
	pollSuccessTotal atomic.Int64
	pollErrorTotal   atomic.Int64

	tokenExpiresAt    time.Time
	tokenLastRefresh  time.Time
	tokenRefreshTotal atomic.Int64
}

var (
	s        = &state{connection: "unknown", devices: map[string]transform.SmallMessage{}}
	initOnce sync.Once
)

// Init registers the "miele" expvar exactly once. Subsequent calls are no-ops.
// Safe to call from main() before any other goroutines start.
func Init() {
	initOnce.Do(func() {
		expvar.Publish("miele", expvar.Func(snapshot))
	})
}

// snapshot returns a JSON-encodable view of the current metrics. Called by
// expvar.Func on every /debug/vars request.
func snapshot() any {
	s.mu.RLock()
	defer s.mu.RUnlock()

	devicesCopy := make(map[string]transform.SmallMessage, len(s.devices))
	for k, v := range s.devices {
		devicesCopy[k] = v
	}

	nextRetry := ""
	if !s.sseNextRetryAt.IsZero() {
		nextRetry = s.sseNextRetryAt.Format(time.RFC3339)
	}

	return map[string]any{
		"connection": s.connection,
		"devices":    devicesCopy,
		"sse": map[string]any{
			"last_event":           s.sseLastEvent,
			"events_total":         s.sseEventsTotal.Load(),
			"consecutive_failures": s.sseConsecutiveFailures,
			"next_retry_after":     nextRetry,
		},
		"polling": map[string]any{
			"last_attempt":  s.pollLastAttempt,
			"last_success":  s.pollLastSuccess,
			"last_error":    s.pollErrorMessage,
			"success_total": s.pollSuccessTotal.Load(),
			"error_total":   s.pollErrorTotal.Load(),
		},
		"token": map[string]any{
			"expires_at":    s.tokenExpiresAt,
			"last_refresh":  s.tokenLastRefresh,
			"refresh_total": s.tokenRefreshTotal.Load(),
		},
	}
}

// SetSSEConnection records the most recent Miele connection state. Values
// match `bridge/miele`: "unknown", "connected", "disconnected".
func SetSSEConnection(state_ string) {
	s.mu.Lock()
	s.connection = state_
	s.mu.Unlock()
}

// RecordSSEEvent updates the SSE event timestamp and increments the counter.
func RecordSSEEvent(now time.Time) {
	s.mu.Lock()
	s.sseLastEvent = now
	s.mu.Unlock()
	s.sseEventsTotal.Add(1)
}

// RecordSSEFailure records the current consecutive-failure streak and the
// time of the next scheduled reconnect attempt.
func RecordSSEFailure(streak int, nextRetry time.Time) {
	s.mu.Lock()
	s.sseConsecutiveFailures = streak
	s.sseNextRetryAt = nextRetry
	s.mu.Unlock()
}

// RecordSSESuccess resets the SSE failure-streak metrics after a successful
// event dispatch.
func RecordSSESuccess() {
	s.mu.Lock()
	s.sseConsecutiveFailures = 0
	s.sseNextRetryAt = time.Time{}
	s.mu.Unlock()
}

// PollSuccessTotal returns the cumulative number of successful poll cycles.
// Used by the app to decide whether polling is "healthy" for status reporting.
func PollSuccessTotal() int64 {
	return s.pollSuccessTotal.Load()
}

// RecordDevice replaces the snapshot for a single device id.
func RecordDevice(id string, msg transform.SmallMessage) {
	s.mu.Lock()
	s.devices[id] = msg
	s.mu.Unlock()
}

// RecordPollAttempt updates the last attempted poll timestamp.
func RecordPollAttempt(now time.Time) {
	s.mu.Lock()
	s.pollLastAttempt = now
	s.mu.Unlock()
}

// RecordPollSuccess updates both the attempt and success timestamps and
// increments the success counter. Leaves the last_error message untouched.
func RecordPollSuccess(now time.Time) {
	s.mu.Lock()
	s.pollLastAttempt = now
	s.pollLastSuccess = now
	s.mu.Unlock()
	s.pollSuccessTotal.Add(1)
}

// RecordPollError updates the attempt timestamp, sets the last error message
// and timestamp, and increments the error counter. Leaves last_success
// untouched.
func RecordPollError(now time.Time, err error) {
	msg := ""
	if err != nil {
		msg = err.Error()
	}
	s.mu.Lock()
	s.pollLastAttempt = now
	s.pollLastError = now
	s.pollErrorMessage = msg
	s.mu.Unlock()
	s.pollErrorTotal.Add(1)
}

// RecordTokenRefresh updates the token expiry and refresh timestamps and
// increments the refresh counter.
func RecordTokenRefresh(now, expiresAt time.Time) {
	s.mu.Lock()
	s.tokenLastRefresh = now
	s.tokenExpiresAt = expiresAt
	s.mu.Unlock()
	s.tokenRefreshTotal.Add(1)
}
