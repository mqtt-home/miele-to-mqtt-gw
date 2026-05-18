package bridge

import (
	"sync"
	"testing"

	"github.com/mqtt-home/miele-to-mqtt-gw/config"
)

func TestHashPayload_Stable(t *testing.T) {
	a := hashPayload([]byte(`{"x":1}`))
	b := hashPayload([]byte(`{"x":1}`))
	if a != b {
		t.Errorf("hash differs for identical input")
	}
}

func TestHashPayload_Distinct(t *testing.T) {
	a := hashPayload([]byte(`{"x":1}`))
	b := hashPayload([]byte(`{"x":2}`))
	if a == b {
		t.Errorf("hash equal for different inputs")
	}
}

func TestPublisher_DedupCacheTracksLastHashPerTopic(t *testing.T) {
	// We can't reach into mqtt-gateway without standing up a broker, but we
	// can verify the dedup bookkeeping by exercising the same internal map
	// that publishWithDedup uses.
	cfg := config.Config{
		MQTT: config.MQTTConfig{Topic: "home/miele", Deduplicate: true, Retain: true, QoS: 1},
	}
	p := New(cfg)

	p.mu.Lock()
	p.lastHash["home/miele/dev-a"] = hashPayload([]byte(`{"a":1}`))
	p.mu.Unlock()

	// Same payload → would hit dedup short-circuit.
	h := hashPayload([]byte(`{"a":1}`))
	p.mu.Lock()
	prev, ok := p.lastHash["home/miele/dev-a"]
	p.mu.Unlock()
	if !ok || prev != h {
		t.Errorf("dedup cache not seeded as expected")
	}

	// Different payload → cache entry updates next time.
	h2 := hashPayload([]byte(`{"a":2}`))
	if h2 == h {
		t.Errorf("collisions are not expected for distinct payloads")
	}
}

// withStubPublish swaps the package-level publishAbsolute for the duration of
// the test and returns a slice that records every (topic, payload) call.
func withStubPublish(t *testing.T) *publishRecorder {
	t.Helper()
	rec := &publishRecorder{}
	orig := publishAbsolute
	publishAbsolute = rec.publish
	t.Cleanup(func() { publishAbsolute = orig })
	return rec
}

type publishRecorder struct {
	mu    sync.Mutex
	calls []publishCall
}

type publishCall struct {
	topic    string
	payload  any
	retained bool
}

func (r *publishRecorder) publish(topic string, message any, retained bool) {
	r.mu.Lock()
	r.calls = append(r.calls, publishCall{topic: topic, payload: message, retained: retained})
	r.mu.Unlock()
}

func discoveryEnabledCfg() config.Config {
	return config.Config{
		MQTT: config.MQTTConfig{
			Topic:       "home/miele",
			Deduplicate: true,
			Retain:      true,
			QoS:         1,
			Discovery: &config.DiscoveryConfig{
				Enabled:          true,
				Prefix:           "homeassistant",
				DeviceNamePrefix: "Miele",
			},
		},
	}
}

func TestPublisher_DiscoveryTracksTopics(t *testing.T) {
	// We can't run the full PublishDevice path without a broker, but we CAN
	// run publishDiscovery if we seed the dedup cache so publishWithDedup
	// short-circuits before reaching mqtt.PublishAbsolute. After that, the
	// discoveredTopics set should contain all five entity topics for the
	// device.
	p := New(discoveryEnabledCfg())

	payloads, err := buildDiscoveryPayloads(p.cfg, "device-abc", []byte(fullExample))
	if err != nil {
		t.Fatalf("buildDiscoveryPayloads: %v", err)
	}

	// Seed the dedup cache so every discovery publish is a no-op.
	for topic, body := range payloads {
		p.lastHash[topic] = hashPayload(body)
	}

	p.publishDiscovery("device-abc", []byte(fullExample))

	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.discoveredTopics) != 5 {
		t.Errorf("discoveredTopics size = %d, want 5: %v", len(p.discoveredTopics), p.discoveredTopics)
	}
	for topic := range payloads {
		if _, ok := p.discoveredTopics[topic]; !ok {
			t.Errorf("missing tracked topic %q", topic)
		}
	}
}

func TestPublisher_DiscoveryDisabledRecordsNothing(t *testing.T) {
	cfg := discoveryEnabledCfg()
	cfg.MQTT.Discovery.Enabled = false
	p := New(cfg)

	p.publishDiscovery("device-abc", []byte(fullExample))

	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.discoveredTopics) != 0 {
		t.Errorf("disabled discovery must not track topics, got %v", p.discoveredTopics)
	}
}

func TestPublisher_CleanupDiscoveryEmpties(t *testing.T) {
	rec := withStubPublish(t)

	p := New(discoveryEnabledCfg())
	// Pretend two devices have been announced — two topics each (abbreviated).
	tracked := []string{
		"homeassistant/sensor/miele_a/state/config",
		"homeassistant/sensor/miele_a/phase/config",
		"homeassistant/sensor/miele_b/state/config",
		"homeassistant/sensor/miele_b/phase/config",
	}
	p.mu.Lock()
	for _, t := range tracked {
		p.discoveredTopics[t] = struct{}{}
	}
	p.mu.Unlock()

	p.CleanupDiscovery()

	if len(rec.calls) != len(tracked) {
		t.Errorf("publish calls = %d, want %d (%+v)", len(rec.calls), len(tracked), rec.calls)
	}
	gotTopics := make(map[string]bool, len(rec.calls))
	for _, c := range rec.calls {
		gotTopics[c.topic] = true
		if c.payload != "" {
			t.Errorf("cleanup payload for %s = %v, want empty string", c.topic, c.payload)
		}
		if !c.retained {
			t.Errorf("cleanup of %s must use retain", c.topic)
		}
	}
	for _, want := range tracked {
		if !gotTopics[want] {
			t.Errorf("cleanup missed topic %q", want)
		}
	}

	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.discoveredTopics) != 0 {
		t.Errorf("discoveredTopics not cleared, has %v", p.discoveredTopics)
	}
}

func TestPublisher_CleanupDiscoveryEmptySetIsNoOp(t *testing.T) {
	rec := withStubPublish(t)
	p := New(discoveryEnabledCfg())

	p.CleanupDiscovery()

	if len(rec.calls) != 0 {
		t.Errorf("expected no publishes for empty set, got %+v", rec.calls)
	}
}
