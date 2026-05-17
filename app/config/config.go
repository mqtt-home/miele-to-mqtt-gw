package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/philipparndt/go-logger"
	gwconfig "github.com/philipparndt/mqtt-gateway/config"
)

type MQTTConfig struct {
	URL             string `json:"url"`
	Topic           string `json:"topic"`
	ClientID        string `json:"client-id,omitempty"`
	Username        string `json:"username,omitempty"`
	Password        string `json:"password,omitempty"`
	Retain          bool   `json:"retain"`
	QoS             byte   `json:"qos"`
	BridgeInfo      bool   `json:"bridge-info"`
	BridgeInfoTopic string `json:"bridge-info-topic,omitempty"`
	Deduplicate     bool   `json:"deduplicate,omitempty"`
}

func (m MQTTConfig) ToGatewayConfig() gwconfig.MQTTConfig {
	return gwconfig.MQTTConfig{
		URL:      m.URL,
		Retain:   m.Retain,
		Topic:    m.Topic,
		QoS:      m.QoS,
		Username: m.Username,
		Password: m.Password,
	}
}

type TokenConfig struct {
	Access     string `json:"access"`
	Refresh    string `json:"refresh"`
	ValidUntil string `json:"validUntil,omitempty"`
}

type MieleConfig struct {
	ClientID                string             `json:"client-id"`
	ClientSecret            string             `json:"client-secret"`
	CountryCode             string             `json:"country-code,omitempty"`
	Username                string             `json:"username,omitempty"`
	Password                string             `json:"password,omitempty"`
	Mode                    string             `json:"mode,omitempty"`
	PollingInterval         int                `json:"polling-interval,omitempty"`
	Token                   *TokenConfig       `json:"token,omitempty"`
	ConnectionCheckInterval int                `json:"connection-check-interval,omitempty"`
	PersistToken            bool               `json:"persistToken"`
	SSEBackoff              *SSEBackoffConfig  `json:"sse-backoff,omitempty"`
}

// SSEBackoffConfig tunes the exponential reconnect behavior of the SSE client.
// All fields are optional; defaults are applied by ApplyDefaults.
type SSEBackoffConfig struct {
	FailureThreshold int    `json:"failure-threshold,omitempty"`
	BaseDelay        string `json:"base-delay,omitempty"`
	MaxDelay         string `json:"max-delay,omitempty"`

	// Parsed durations. Populated by LoadConfig after JSON unmarshalling.
	// Not part of the JSON schema.
	baseDelayParsed time.Duration
	maxDelayParsed  time.Duration
}

// BaseDelayDuration returns the parsed base-delay. Falls back to a re-parse if
// the value was set programmatically without going through LoadConfig.
func (c *SSEBackoffConfig) BaseDelayDuration() time.Duration {
	if c == nil {
		return 0
	}
	if c.baseDelayParsed > 0 {
		return c.baseDelayParsed
	}
	d, _ := time.ParseDuration(c.BaseDelay)
	return d
}

// MaxDelayDuration returns the parsed max-delay. See BaseDelayDuration.
func (c *SSEBackoffConfig) MaxDelayDuration() time.Duration {
	if c == nil {
		return 0
	}
	if c.maxDelayParsed > 0 {
		return c.maxDelayParsed
	}
	d, _ := time.ParseDuration(c.MaxDelay)
	return d
}

type Config struct {
	MQTT           MQTTConfig        `json:"mqtt"`
	Miele          MieleConfig       `json:"miele"`
	Names          map[string]string `json:"names,omitempty"`
	SendFullUpdate bool              `json:"send-full-update"`
	LogLevel       string            `json:"loglevel,omitempty"`
}

func (c Config) BridgeInfoTopic() string {
	if c.MQTT.BridgeInfoTopic != "" {
		return c.MQTT.BridgeInfoTopic
	}
	return c.MQTT.Topic + "/bridge/state"
}

func (c Config) MieleStateTopic() string {
	return c.MQTT.Topic + "/bridge/miele"
}

var (
	mu         sync.RWMutex
	cfg        Config
	cfgPath    string
	cfgLoaded  bool
)

// ApplyDefaults fills in unset fields with the defaults documented in the
// app-config spec. Booleans default to true via explicit JSON-unmarshal
// preseeding in LoadConfig — this function handles the rest.
func ApplyDefaults(c *Config) {
	if c.MQTT.QoS == 0 {
		c.MQTT.QoS = 1
	}
	if c.Miele.Mode == "" {
		c.Miele.Mode = "sse"
	}
	if c.Miele.CountryCode == "" {
		c.Miele.CountryCode = "de-DE"
	}
	if c.Miele.ConnectionCheckInterval == 0 {
		c.Miele.ConnectionCheckInterval = 10000
	}
	if c.LogLevel == "" {
		c.LogLevel = "info"
	}
	if c.Names == nil {
		c.Names = make(map[string]string)
	}
	if c.Miele.SSEBackoff == nil {
		c.Miele.SSEBackoff = &SSEBackoffConfig{}
	}
	if c.Miele.SSEBackoff.FailureThreshold <= 0 {
		c.Miele.SSEBackoff.FailureThreshold = 5
	}
	if c.Miele.SSEBackoff.BaseDelay == "" {
		c.Miele.SSEBackoff.BaseDelay = "5s"
	}
	if c.Miele.SSEBackoff.MaxDelay == "" {
		c.Miele.SSEBackoff.MaxDelay = "10m"
	}
}

// parseSSEBackoff parses BaseDelay / MaxDelay into time.Duration values and
// stores them on the SSEBackoffConfig. Returns a wrapped error naming the
// offending field if either string fails to parse.
func parseSSEBackoff(b *SSEBackoffConfig) error {
	if b == nil {
		return nil
	}
	base, err := time.ParseDuration(b.BaseDelay)
	if err != nil {
		return fmt.Errorf("miele.sse-backoff.base-delay: %w", err)
	}
	max, err := time.ParseDuration(b.MaxDelay)
	if err != nil {
		return fmt.Errorf("miele.sse-backoff.max-delay: %w", err)
	}
	b.baseDelayParsed = base
	b.maxDelayParsed = max
	return nil
}

// ReplaceEnvVariables substitutes ${NAME} in raw bytes from environment
// variables; missing vars become empty strings. Mirrors the TS behavior.
func ReplaceEnvVariables(input []byte) []byte {
	return gwconfig.ReplaceEnvVariables(input)
}

// LoadConfig reads a JSON config from path, substitutes environment variables,
// applies defaults, and stores it as the process-wide config.
func LoadConfig(path string) (Config, error) {
	if path == "" {
		return Config{}, errors.New("config path is empty")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config %s: %w", path, err)
	}

	data = ReplaceEnvVariables(data)

	c := Config{
		MQTT: MQTTConfig{
			Retain:     true,
			BridgeInfo: true,
		},
		Miele: MieleConfig{
			PersistToken: true,
		},
		SendFullUpdate: true,
	}

	if err := json.Unmarshal(data, &c); err != nil {
		return Config{}, fmt.Errorf("parse config %s: %w", path, err)
	}

	ApplyDefaults(&c)

	if err := parseSSEBackoff(c.Miele.SSEBackoff); err != nil {
		return Config{}, fmt.Errorf("parse config %s: %w", path, err)
	}

	mu.Lock()
	cfg = c
	cfgPath = path
	cfgLoaded = true
	mu.Unlock()

	logger.Debug("Config loaded", "file", path, "mode", c.Miele.Mode, "loglevel", c.LogLevel)
	return c, nil
}

// Get returns the currently loaded config. Returns the zero value if no config
// has been loaded.
func Get() Config {
	mu.RLock()
	defer mu.RUnlock()
	return cfg
}

// Path returns the path the active config was loaded from.
func Path() string {
	mu.RLock()
	defer mu.RUnlock()
	return cfgPath
}

// PersistToken writes the given token into the on-disk config file's
// miele.token field. It is a no-op when no config has been loaded, when
// persistToken is false, or when the on-disk token already matches.
func PersistToken(token TokenConfig) {
	mu.RLock()
	loaded := cfgLoaded
	path := cfgPath
	persist := cfg.Miele.PersistToken
	mu.RUnlock()

	if !loaded || path == "" {
		logger.Warn("No config file set. Not persisting token.")
		return
	}
	if !persist {
		logger.Debug("Token persistence disabled. Skipping persist.")
		return
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		logger.Error("Failed to read config file for token persistence", "file", path, "error", err)
		return
	}

	// Round-trip through generic map to preserve unknown fields and ordering as
	// much as Go's encoder allows. This avoids re-emitting defaults that the
	// user did not write themselves.
	var raw3 map[string]any
	if err := json.Unmarshal(raw, &raw3); err != nil {
		logger.Error("Failed to parse config file for token persistence", "file", path, "error", err)
		return
	}

	mieleAny, _ := raw3["miele"].(map[string]any)
	if mieleAny == nil {
		mieleAny = make(map[string]any)
		raw3["miele"] = mieleAny
	}

	newTok := map[string]any{
		"access":  token.Access,
		"refresh": token.Refresh,
	}
	if token.ValidUntil != "" {
		newTok["validUntil"] = token.ValidUntil
	}

	if existing, ok := mieleAny["token"].(map[string]any); ok {
		if tokensEqual(existing, newTok) {
			logger.Debug("Token did not change. Not persisting.")
			return
		}
	}

	mieleAny["token"] = newTok

	encoded, err := json.MarshalIndent(raw3, "", "  ")
	if err != nil {
		logger.Error("Failed to encode config for token persistence", "error", err)
		return
	}

	if err := os.WriteFile(path, encoded, 0o600); err != nil {
		logger.Error("Failed to write config file for token persistence", "file", path, "error", err)
		return
	}

	// Keep the in-memory config in sync so subsequent reads see the new token.
	mu.Lock()
	cfg.Miele.Token = &TokenConfig{
		Access:     token.Access,
		Refresh:    token.Refresh,
		ValidUntil: token.ValidUntil,
	}
	mu.Unlock()

	logger.Info("Persisted token to config file", "file", path)
}

func tokensEqual(a, b map[string]any) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if w, ok := b[k]; !ok || v != w {
			return false
		}
	}
	return true
}
