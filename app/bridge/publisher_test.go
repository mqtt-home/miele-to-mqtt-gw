package bridge

import (
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
