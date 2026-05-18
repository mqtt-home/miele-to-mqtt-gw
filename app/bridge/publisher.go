package bridge

import (
	"crypto/sha256"
	"encoding/hex"
	"sync"

	"github.com/mqtt-home/miele-to-mqtt-gw/config"
	"github.com/mqtt-home/miele-to-mqtt-gw/miele/transform"
	"github.com/philipparndt/go-logger"
	"github.com/philipparndt/mqtt-gateway/mqtt"
)

// publishAbsolute is the indirection used by the discovery layer so unit
// tests can swap in a stub without standing up a real MQTT broker.
var publishAbsolute = mqtt.PublishAbsolute

// Publisher is the application-side MQTT wrapper. It owns the dedup cache and
// knows about Miele-specific topic conventions (`<id>` for small, `<id>/full`
// for the raw payload, `bridge/miele` for Miele connection status).
type Publisher struct {
	cfg config.Config

	mu               sync.Mutex
	lastHash         map[string]string
	mieleLast        string
	discoveredTopics map[string]struct{}
}

// New builds a Publisher for the given config. It does NOT establish the MQTT
// connection — call Start on mqtt-gateway separately.
func New(cfg config.Config) *Publisher {
	return &Publisher{
		cfg:              cfg,
		lastHash:         make(map[string]string),
		mieleLast:        "",
		discoveredTopics: make(map[string]struct{}),
	}
}

// PublishDevice serializes and publishes the small message and the raw "full"
// message for a single device. Honors the dedup flag.
func (p *Publisher) PublishDevice(deviceID string, small transform.SmallMessage, rawFull []byte) {
	smallTopic := p.cfg.MQTT.Topic + "/" + deviceID
	fullTopic := p.cfg.MQTT.Topic + "/" + deviceID + "/full"

	smallBytes, err := transform.MarshalNoNulls(small)
	if err != nil {
		logger.Error("marshal small message", "device", deviceID, "error", err)
		return
	}

	fullBytes, err := transform.StripNulls(rawFull)
	if err != nil {
		// If the raw payload isn't valid JSON for some reason, fall back to
		// publishing as-is rather than dropping it.
		logger.Warn("strip nulls from full payload", "device", deviceID, "error", err)
		fullBytes = rawFull
	}

	p.publishWithDedup(smallTopic, smallBytes)
	p.publishWithDedup(fullTopic, fullBytes)

	p.publishDiscovery(deviceID, rawFull)
}

// publishDiscovery sends one retained HA discovery config per entity for this
// device, and records the topics so they can be cleared on shutdown. No-op
// when discovery is disabled in config.
func (p *Publisher) publishDiscovery(deviceID string, rawFull []byte) {
	payloads, err := buildDiscoveryPayloads(p.cfg, deviceID, rawFull)
	if err != nil {
		logger.Warn("build discovery payloads", "device", deviceID, "error", err)
		return
	}
	if len(payloads) == 0 {
		return
	}
	for topic, body := range payloads {
		p.mu.Lock()
		p.discoveredTopics[topic] = struct{}{}
		p.mu.Unlock()
		p.publishWithDedup(topic, body)
	}
}

// CleanupDiscovery publishes an empty retained payload to every discovery
// topic the bridge has published during the run, so Home Assistant removes
// the device from its registry. Safe to call when discovery was disabled —
// the tracked set will be empty.
func (p *Publisher) CleanupDiscovery() {
	p.mu.Lock()
	topics := make([]string, 0, len(p.discoveredTopics))
	for t := range p.discoveredTopics {
		topics = append(topics, t)
	}
	p.discoveredTopics = make(map[string]struct{})
	p.mu.Unlock()

	for _, t := range topics {
		publishAbsolute(t, "", p.cfg.MQTT.Retain)
	}
}

func (p *Publisher) publishWithDedup(topic string, payload []byte) {
	if p.cfg.MQTT.Deduplicate {
		h := hashPayload(payload)
		p.mu.Lock()
		prev, ok := p.lastHash[topic]
		p.lastHash[topic] = h
		p.mu.Unlock()
		if ok && prev == h {
			logger.Trace("dedup: skipping identical payload", "topic", topic)
			return
		}
	}

	mqtt.PublishAbsolute(topic, string(payload), p.cfg.MQTT.Retain)
}

// PublishMieleState updates `<topic>/bridge/miele` to one of "unknown",
// "connected", "disconnected". Identical consecutive states are suppressed.
func (p *Publisher) PublishMieleState(state string) {
	p.mu.Lock()
	prev := p.mieleLast
	p.mieleLast = state
	p.mu.Unlock()
	if prev == state {
		return
	}
	logger.Info("Miele connection state", "state", state)
	mqtt.PublishAbsolute(p.cfg.MieleStateTopic(), state, p.cfg.MQTT.Retain)
}

func hashPayload(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
