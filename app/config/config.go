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
	URL             string           `json:"url"`
	Topic           string           `json:"topic"`
	ClientID        string           `json:"client-id,omitempty"`
	Username        string           `json:"username,omitempty"`
	Password        string           `json:"password,omitempty"`
	Retain          bool             `json:"retain"`
	QoS             byte             `json:"qos"`
	BridgeInfo      bool             `json:"bridge-info"`
	BridgeInfoTopic string           `json:"bridge-info-topic,omitempty"`
	Deduplicate     bool             `json:"deduplicate,omitempty"`
	Discovery       *DiscoveryConfig `json:"discovery,omitempty"`
}

// DiscoveryConfig controls Home Assistant MQTT-discovery output. All fields
// optional; defaults applied by ApplyDefaults.
type DiscoveryConfig struct {
	Enabled          bool   `json:"enabled,omitempty"`
	Prefix           string `json:"prefix,omitempty"`
	DeviceNamePrefix string `json:"device-name-prefix,omitempty"`
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
	// RefreshValidUntil is when the refresh token expires, i.e. when the next
	// login is required. Informational; empty when the provider did not report it.
	RefreshValidUntil string `json:"refreshValidUntil,omitempty"`
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

	// AuthType selects the full-login flow: AuthTypeLegacy (username/password
	// against api.mcs3.miele.com) or AuthTypeDevice (browser login against the
	// OAuth2 provider with device-specific permissions). Empty means derived,
	// see ResolvedAuthType.
	AuthType string `json:"auth-type,omitempty"`
	// RedirectURI overrides the redirect_uri of the browser login. Empty means
	// it is derived from the request to the login page.
	RedirectURI string `json:"redirect-uri,omitempty"`
	// AuthListen is the bind address of the temporary login page.
	AuthListen string `json:"auth-listen,omitempty"`
	// Scope is the OAuth2 scope requested by the browser login.
	Scope string `json:"scope,omitempty"`
	// EnableAuthServer opens the login page at startup although a working
	// token exists. It is removed from the config file after a successful
	// browser login.
	EnableAuthServer bool `json:"enable-auth-server,omitempty"`
	// ReloginWarningDays is how many days before the refresh token expires
	// the login page is opened and a relogin warning is published.
	ReloginWarningDays int `json:"relogin-warning-days,omitempty"`
}

const (
	AuthTypeLegacy = "legacy"
	AuthTypeDevice = "per-device"
)

// ResolvedAuthType returns the configured auth type, or derives it when unset:
// legacy when username and password are present, per-device otherwise.
func (m MieleConfig) ResolvedAuthType() string {
	if m.AuthType != "" {
		return m.AuthType
	}
	if m.Username != "" && m.Password != "" {
		return AuthTypeLegacy
	}
	return AuthTypeDevice
}

func validateAuth(m MieleConfig) error {
	switch m.AuthType {
	case "", AuthTypeDevice:
		return nil
	case AuthTypeLegacy:
		if m.Username == "" || m.Password == "" {
			return errors.New("miele.auth-type legacy requires miele.username and miele.password")
		}
		return nil
	default:
		return fmt.Errorf("miele.auth-type: unknown value %q (expected %q or %q)", m.AuthType, AuthTypeLegacy, AuthTypeDevice)
	}
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

// MieleNextLoginTopic carries when the refresh token expires, i.e. when the
// next browser login is required.
func (c Config) MieleNextLoginTopic() string {
	return c.MieleStateTopic() + "/next-login-required"
}

// MieleWarningTopic carries a warning for the user, e.g. a pending relogin.
func (c Config) MieleWarningTopic() string {
	return c.MieleStateTopic() + "/warning"
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
	if c.Miele.AuthListen == "" {
		c.Miele.AuthListen = ":6061"
	}
	if c.Miele.Scope == "" {
		c.Miele.Scope = "openid mcs_thirdparty_read"
	}
	if c.Miele.ReloginWarningDays == 0 {
		c.Miele.ReloginWarningDays = 14
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
	if c.MQTT.Discovery == nil {
		c.MQTT.Discovery = &DiscoveryConfig{}
	}
	if c.MQTT.Discovery.Prefix == "" {
		c.MQTT.Discovery.Prefix = "homeassistant"
	}
	if c.MQTT.Discovery.DeviceNamePrefix == "" {
		c.MQTT.Discovery.DeviceNamePrefix = "Miele"
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

	if err := validateAuth(c.Miele); err != nil {
		return Config{}, fmt.Errorf("parse config %s: %w", path, err)
	}

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

	newTok := map[string]any{
		"access":  token.Access,
		"refresh": token.Refresh,
	}
	if token.ValidUntil != "" {
		newTok["validUntil"] = token.ValidUntil
	}
	if token.RefreshValidUntil != "" {
		newTok["refreshValidUntil"] = token.RefreshValidUntil
	}

	changed, err := editMieleSection(path, func(mieleAny map[string]any) bool {
		if existing, ok := mieleAny["token"].(map[string]any); ok && tokensEqual(existing, newTok) {
			return false
		}
		mieleAny["token"] = newTok
		return true
	})
	if err != nil {
		logger.Error("Failed to persist token to config file", "file", path, "error", err)
		return
	}
	if !changed {
		logger.Debug("Token did not change. Not persisting.")
		return
	}

	// Keep the in-memory config in sync so subsequent reads see the new token.
	mu.Lock()
	cfg.Miele.Token = &TokenConfig{
		Access:            token.Access,
		Refresh:           token.Refresh,
		ValidUntil:        token.ValidUntil,
		RefreshValidUntil: token.RefreshValidUntil,
	}
	mu.Unlock()

	logger.Info("Persisted token to config file", "file", path)
}

// ClearEnableAuthServer removes miele.enable-auth-server from the on-disk
// config file once the browser login it requested has succeeded. It is a
// no-op when the flag is not set.
func ClearEnableAuthServer() {
	mu.Lock()
	enabled := cfg.Miele.EnableAuthServer
	cfg.Miele.EnableAuthServer = false
	path := cfgPath
	mu.Unlock()

	if !enabled || path == "" {
		return
	}
	_, err := editMieleSection(path, func(mieleAny map[string]any) bool {
		if _, ok := mieleAny["enable-auth-server"]; !ok {
			return false
		}
		delete(mieleAny, "enable-auth-server")
		return true
	})
	if err != nil {
		logger.Error("Failed to remove miele.enable-auth-server from the config file. Please remove it manually.", "file", path, "error", err)
		return
	}
	logger.Info("Removed miele.enable-auth-server from config file", "file", path)
}

// editMieleSection applies edit to the miele section of the config file at
// path and writes the file back when edit reports a change.
func editMieleSection(path string, edit func(mieleAny map[string]any) bool) (bool, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}

	// Round-trip through generic map to preserve unknown fields and ordering as
	// much as Go's encoder allows. This avoids re-emitting defaults that the
	// user did not write themselves.
	var root map[string]any
	if err := json.Unmarshal(raw, &root); err != nil {
		return false, err
	}

	mieleAny, _ := root["miele"].(map[string]any)
	if mieleAny == nil {
		mieleAny = make(map[string]any)
		root["miele"] = mieleAny
	}
	if !edit(mieleAny) {
		return false, nil
	}

	encoded, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return false, err
	}
	if err := os.WriteFile(path, encoded, 0o600); err != nil {
		return false, err
	}
	return true, nil
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
