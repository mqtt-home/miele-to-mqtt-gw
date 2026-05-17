package metrics

import (
	"errors"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/mqtt-home/miele-to-mqtt-gw/miele/transform"
)

// resetState reinitializes the package-level singleton between tests. The
// real binary calls Init() exactly once at startup, but these tests need a
// clean slate per case.
func resetState() {
	s = &state{connection: "unknown", devices: map[string]transform.SmallMessage{}}
}

func TestSnapshot_DefaultShape(t *testing.T) {
	resetState()
	snap := snapshot().(map[string]any)

	wantKeys := []string{"connection", "devices", "sse", "polling", "token"}
	for _, k := range wantKeys {
		if _, ok := snap[k]; !ok {
			t.Errorf("missing key %q in snapshot: %+v", k, snap)
		}
	}

	if snap["connection"] != "unknown" {
		t.Errorf("connection = %v, want unknown", snap["connection"])
	}

	devs, ok := snap["devices"].(map[string]transform.SmallMessage)
	if !ok {
		t.Fatalf("devices wrong type: %T", snap["devices"])
	}
	if len(devs) != 0 {
		t.Errorf("expected empty devices, got %d entries", len(devs))
	}

	sseSection := snap["sse"].(map[string]any)
	if v, _ := sseSection["events_total"].(int64); v != 0 {
		t.Errorf("events_total = %v, want 0", sseSection["events_total"])
	}

	pollingSection := snap["polling"].(map[string]any)
	if v, _ := pollingSection["success_total"].(int64); v != 0 {
		t.Errorf("polling.success_total = %v, want 0", v)
	}
	if v, _ := pollingSection["error_total"].(int64); v != 0 {
		t.Errorf("polling.error_total = %v, want 0", v)
	}
	if v, _ := pollingSection["last_error"].(string); v != "" {
		t.Errorf("polling.last_error = %q, want empty", v)
	}

	tokenSection := snap["token"].(map[string]any)
	if v, _ := tokenSection["refresh_total"].(int64); v != 0 {
		t.Errorf("token.refresh_total = %v, want 0", v)
	}
}

func TestSetSSEConnection(t *testing.T) {
	resetState()
	SetSSEConnection("connected")
	if snapshot().(map[string]any)["connection"] != "connected" {
		t.Error("connection not updated")
	}
	SetSSEConnection("disconnected")
	if snapshot().(map[string]any)["connection"] != "disconnected" {
		t.Error("connection not updated to disconnected")
	}
}

func TestRecordSSEEvent(t *testing.T) {
	resetState()
	now := time.Date(2026, 1, 2, 12, 0, 0, 0, time.UTC)
	RecordSSEEvent(now)
	RecordSSEEvent(now.Add(time.Minute))

	sse := snapshot().(map[string]any)["sse"].(map[string]any)
	if sse["events_total"].(int64) != 2 {
		t.Errorf("events_total = %v, want 2", sse["events_total"])
	}
	if !sse["last_event"].(time.Time).Equal(now.Add(time.Minute)) {
		t.Errorf("last_event = %v, want %v", sse["last_event"], now.Add(time.Minute))
	}
}

func TestRecordDevice_AddAndUpdate(t *testing.T) {
	resetState()
	m1 := transform.SmallMessage{Phase: "PRE_WASH", PhaseID: 1794, State: "RUNNING"}
	m2 := transform.SmallMessage{Phase: "DRYING", PhaseID: 1799, State: "RUNNING"}

	RecordDevice("dev-a", m1)
	RecordDevice("dev-b", m2)
	devs := snapshot().(map[string]any)["devices"].(map[string]transform.SmallMessage)
	if len(devs) != 2 {
		t.Errorf("len = %d, want 2", len(devs))
	}
	if devs["dev-a"].Phase != "PRE_WASH" {
		t.Errorf("dev-a phase = %q, want PRE_WASH", devs["dev-a"].Phase)
	}

	// Update in place.
	RecordDevice("dev-a", m2)
	devs = snapshot().(map[string]any)["devices"].(map[string]transform.SmallMessage)
	if len(devs) != 2 {
		t.Errorf("after update len = %d, want 2", len(devs))
	}
	if devs["dev-a"].Phase != "DRYING" {
		t.Errorf("dev-a updated phase = %q, want DRYING", devs["dev-a"].Phase)
	}
}

func TestRecordPollSuccess(t *testing.T) {
	resetState()
	now := time.Date(2026, 1, 2, 12, 0, 0, 0, time.UTC)
	RecordPollAttempt(now)
	RecordPollSuccess(now.Add(time.Second))

	polling := snapshot().(map[string]any)["polling"].(map[string]any)
	if polling["success_total"].(int64) != 1 {
		t.Errorf("success_total = %v, want 1", polling["success_total"])
	}
	if polling["error_total"].(int64) != 0 {
		t.Errorf("error_total = %v, want 0", polling["error_total"])
	}
	if polling["last_error"].(string) != "" {
		t.Errorf("last_error = %q, want empty", polling["last_error"])
	}
	if !polling["last_attempt"].(time.Time).Equal(now.Add(time.Second)) {
		t.Errorf("last_attempt not updated: %v", polling["last_attempt"])
	}
	if !polling["last_success"].(time.Time).Equal(now.Add(time.Second)) {
		t.Errorf("last_success not updated: %v", polling["last_success"])
	}
}

func TestRecordPollError(t *testing.T) {
	resetState()
	now := time.Date(2026, 1, 2, 12, 0, 0, 0, time.UTC)
	// First a success so we can verify last_success is preserved.
	RecordPollSuccess(now)
	RecordPollError(now.Add(time.Minute), errors.New("boom"))

	polling := snapshot().(map[string]any)["polling"].(map[string]any)
	if polling["success_total"].(int64) != 1 {
		t.Errorf("success_total = %v, want 1", polling["success_total"])
	}
	if polling["error_total"].(int64) != 1 {
		t.Errorf("error_total = %v, want 1", polling["error_total"])
	}
	if polling["last_error"].(string) != "boom" {
		t.Errorf("last_error = %q, want boom", polling["last_error"])
	}
	if !polling["last_success"].(time.Time).Equal(now) {
		t.Errorf("last_success should remain at %v, got %v", now, polling["last_success"])
	}
	if !polling["last_attempt"].(time.Time).Equal(now.Add(time.Minute)) {
		t.Errorf("last_attempt = %v, want %v", polling["last_attempt"], now.Add(time.Minute))
	}
}

func TestRecordTokenRefresh(t *testing.T) {
	resetState()
	now := time.Date(2026, 1, 2, 12, 0, 0, 0, time.UTC)
	expires := now.Add(24 * time.Hour)
	RecordTokenRefresh(now, expires)
	RecordTokenRefresh(now.Add(time.Hour), expires.Add(time.Hour))

	tok := snapshot().(map[string]any)["token"].(map[string]any)
	if tok["refresh_total"].(int64) != 2 {
		t.Errorf("refresh_total = %v, want 2", tok["refresh_total"])
	}
	if !tok["last_refresh"].(time.Time).Equal(now.Add(time.Hour)) {
		t.Errorf("last_refresh = %v, want %v", tok["last_refresh"], now.Add(time.Hour))
	}
	if !tok["expires_at"].(time.Time).Equal(expires.Add(time.Hour)) {
		t.Errorf("expires_at = %v, want %v", tok["expires_at"], expires.Add(time.Hour))
	}
}

func TestInit_RegistersOnce(t *testing.T) {
	// Init uses sync.Once so calling multiple times is safe and the second
	// Publish would panic if it actually re-registered.
	Init()
	Init()
}

// TestConcurrentUpdatesRaceFree exercises every public update path from many
// goroutines while a parallel goroutine reads snapshot(). Run with -race.
func TestConcurrentUpdatesRaceFree(t *testing.T) {
	resetState()
	const goroutines = 8
	const iters = 200

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		for j := 0; j < iters*goroutines; j++ {
			_ = snapshot()
			runtime.Gosched()
		}
	}()

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iters; j++ {
				now := time.Now()
				switch j % 6 {
				case 0:
					SetSSEConnection("connected")
				case 1:
					RecordSSEEvent(now)
				case 2:
					RecordDevice("d", transform.SmallMessage{Phase: "X", PhaseID: id, State: "RUNNING"})
				case 3:
					RecordPollSuccess(now)
				case 4:
					RecordPollError(now, errors.New("err"))
				case 5:
					RecordTokenRefresh(now, now.Add(time.Hour))
				}
			}
		}(i)
	}

	wg.Wait()
}
