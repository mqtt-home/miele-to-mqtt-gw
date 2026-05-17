package sse

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/mqtt-home/miele-to-mqtt-gw/miele/api"
)

// flushWriter writes to the underlying ResponseWriter and immediately flushes
// so the SSE scanner on the client sees data as soon as it's available.
type flushWriter struct {
	w http.ResponseWriter
	f http.Flusher
}

func newFlushWriter(w http.ResponseWriter) *flushWriter {
	f, _ := w.(http.Flusher)
	return &flushWriter{w: w, f: f}
}

func (fw *flushWriter) Write(s string) {
	fw.w.Write([]byte(s))
	if fw.f != nil {
		fw.f.Flush()
	}
}

func TestSSE_DispatchEventThenClose(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer T" {
			t.Errorf("Authorization = %q", got)
		}
		if got := r.Header.Get("Accept"); got != "text/event-stream" {
			t.Errorf("Accept = %q", got)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		fw := newFlushWriter(w)
		fw.Write("event: devices\n")
		fw.Write("data: {\"abc\":{\"x\":1}}\n")
		fw.Write("\n")
		// Hold the connection open briefly to let the client process the event.
		time.Sleep(150 * time.Millisecond)
	}))
	t.Cleanup(srv.Close)

	var (
		mu      sync.Mutex
		devices []api.Device
		states  []string
	)
	done := make(chan struct{}, 1)
	c := Start(Options{
		AccessToken: func() string { return "T" },
		URL:         srv.URL,
		HTTPClient:  srv.Client(),
		OnDevices: func(d []api.Device) {
			mu.Lock()
			devices = append(devices, d...)
			mu.Unlock()
			select {
			case done <- struct{}{}:
			default:
			}
		},
		OnStatus: func(s string) {
			mu.Lock()
			states = append(states, s)
			mu.Unlock()
		},
		ReconnectDelay: 10 * time.Millisecond,
	})

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for device event")
	}
	c.Stop()

	mu.Lock()
	defer mu.Unlock()
	if len(devices) != 1 || devices[0].ID != "abc" {
		t.Errorf("devices = %+v", devices)
	}
	if string(devices[0].Data) != `{"x":1}` {
		t.Errorf("data = %s", string(devices[0].Data))
	}
	foundConn := false
	for _, s := range states {
		if s == "connected" {
			foundConn = true
			break
		}
	}
	if !foundConn {
		t.Errorf("never saw connected state: %+v", states)
	}
}

func TestSSE_AccumulatesMultiLineData(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		fw := newFlushWriter(w)
		fw.Write("event: devices\n")
		fw.Write("data: {\"a\":1,\n")
		fw.Write("data: \"b\":2}\n")
		fw.Write("\n")
		time.Sleep(150 * time.Millisecond)
	}))
	t.Cleanup(srv.Close)

	done := make(chan []api.Device, 1)
	c := Start(Options{
		AccessToken: func() string { return "T" },
		URL:         srv.URL,
		HTTPClient:  srv.Client(),
		OnDevices: func(d []api.Device) {
			select {
			case done <- d:
			default:
			}
		},
	})
	defer c.Stop()

	select {
	case devs := <-done:
		if len(devs) != 2 {
			t.Fatalf("len = %d, want 2 (two top-level keys after join)", len(devs))
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out")
	}
}

func TestSSE_IgnoresNonDeviceEvents(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		fw := newFlushWriter(w)
		fw.Write("event: ping\n")
		fw.Write("data: hello\n")
		fw.Write("\n")
		fw.Write("event: devices\n")
		fw.Write("data: {\"only\":{\"k\":\"v\"}}\n")
		fw.Write("\n")
		time.Sleep(200 * time.Millisecond)
	}))
	t.Cleanup(srv.Close)

	got := make(chan []api.Device, 4)
	c := Start(Options{
		AccessToken: func() string { return "T" },
		URL:         srv.URL,
		HTTPClient:  srv.Client(),
		OnDevices: func(d []api.Device) {
			got <- d
		},
	})
	defer c.Stop()

	select {
	case devs := <-got:
		if len(devs) != 1 || devs[0].ID != "only" {
			t.Errorf("got %+v", devs)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out")
	}
}

func TestSSE_ReconnectsAfterDisconnect(t *testing.T) {
	var connCount atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		n := connCount.Add(1)
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		fw := newFlushWriter(w)
		fw.Write("event: devices\n")
		fw.Write(fmt.Sprintf("data: {\"d%d\":{\"n\":%d}}\n", n, n))
		fw.Write("\n")
		// Then close the connection by returning.
	}))
	t.Cleanup(srv.Close)

	events := make(chan struct{}, 4)
	c := Start(Options{
		AccessToken:    func() string { return "T" },
		URL:            srv.URL,
		HTTPClient:     srv.Client(),
		ReconnectDelay: 30 * time.Millisecond,
		OnDevices:      func(_ []api.Device) { events <- struct{}{} },
	})
	defer c.Stop()

	for i := 0; i < 2; i++ {
		select {
		case <-events:
		case <-time.After(2 * time.Second):
			t.Fatalf("timed out waiting for event %d", i)
		}
	}
	if connCount.Load() < 2 {
		t.Errorf("expected at least 2 reconnects, got %d", connCount.Load())
	}
}

func TestSSE_CloseFromOutsideTerminatesStream(t *testing.T) {
	// Server holds the connection open and writes a heartbeat comment every
	// 20ms so it notices a client-side close quickly. The test calls
	// c.Close() externally and verifies the read loop returns and reports
	// disconnected.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		fw := newFlushWriter(w)
		for {
			select {
			case <-r.Context().Done():
				return
			case <-time.After(20 * time.Millisecond):
				fw.Write(": heartbeat\n\n")
			}
		}
	}))
	t.Cleanup(srv.Close)

	gotConnected := make(chan struct{}, 1)
	gotDisconnected := make(chan struct{}, 1)
	c := Start(Options{
		AccessToken:    func() string { return "T" },
		URL:            srv.URL,
		HTTPClient:     srv.Client(),
		ReconnectDelay: 5 * time.Second,
		OnStatus: func(s string) {
			switch s {
			case "connected":
				select {
				case gotConnected <- struct{}{}:
				default:
				}
			case "disconnected":
				select {
				case gotDisconnected <- struct{}{}:
				default:
				}
			}
		},
	})

	select {
	case <-gotConnected:
	case <-time.After(2 * time.Second):
		t.Fatal("never connected")
	}

	c.Close()

	select {
	case <-gotDisconnected:
	case <-time.After(2 * time.Second):
		t.Fatal("never disconnected after Close()")
	}
	c.Stop()
}

func TestSSE_NoTokenSkipsConnect(t *testing.T) {
	calls := atomic.Int32{}
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
	}))
	t.Cleanup(srv.Close)

	c := Start(Options{
		AccessToken:    func() string { return "" },
		URL:            srv.URL,
		HTTPClient:     srv.Client(),
		ReconnectDelay: 20 * time.Millisecond,
	})
	defer c.Stop()
	time.Sleep(100 * time.Millisecond)
	if calls.Load() != 0 {
		t.Errorf("connect called %d times, want 0", calls.Load())
	}
}

func TestSSE_DataLineWithoutSpaceAfterColon(t *testing.T) {
	// SSE allows `data:foo` (no space after colon). The hue-style reader in
	// the original code uses `strings.HasPrefix(line, "data: ")` and would
	// silently drop the value; we trim the optional single space instead.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		fw := newFlushWriter(w)
		fw.Write("event: devices\n")
		fw.Write("data:{\"id1\":{\"x\":42}}\n")
		fw.Write("\n")
		time.Sleep(150 * time.Millisecond)
	}))
	t.Cleanup(srv.Close)

	got := make(chan []api.Device, 1)
	c := Start(Options{
		AccessToken: func() string { return "T" },
		URL:         srv.URL,
		HTTPClient:  srv.Client(),
		OnDevices:   func(d []api.Device) { got <- d },
	})
	defer c.Stop()
	select {
	case devs := <-got:
		if len(devs) != 1 || devs[0].ID != "id1" {
			t.Errorf("got %+v", devs)
		}
		if !strings.Contains(string(devs[0].Data), "42") {
			t.Errorf("data = %s", devs[0].Data)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out")
	}
}
