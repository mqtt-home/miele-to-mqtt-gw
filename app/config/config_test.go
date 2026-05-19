package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestReplaceEnvVariables(t *testing.T) {
	t.Setenv("FOO", "bar")
	t.Setenv("EMPTY", "")

	cases := []struct {
		in   string
		want string
	}{
		{`{"x": "${FOO}"}`, `{"x": "bar"}`},
		{`{"x": "${MISSING}"}`, `{"x": ""}`},
		{`{"a": "${FOO}", "b": "${FOO}"}`, `{"a": "bar", "b": "bar"}`},
		{`{"x": "${EMPTY}"}`, `{"x": ""}`},
		{`no vars here`, `no vars here`},
	}
	for _, c := range cases {
		got := string(ReplaceEnvVariables([]byte(c.in)))
		if got != c.want {
			t.Errorf("ReplaceEnvVariables(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestApplyDefaults_EmptyConfig(t *testing.T) {
	c := Config{}
	ApplyDefaults(&c)

	if c.MQTT.QoS != 1 {
		t.Errorf("MQTT.QoS = %d, want 1", c.MQTT.QoS)
	}
	if c.Miele.Mode != "sse" {
		t.Errorf("Miele.Mode = %q, want %q", c.Miele.Mode, "sse")
	}
	if c.Miele.CountryCode != "de-DE" {
		t.Errorf("Miele.CountryCode = %q, want %q", c.Miele.CountryCode, "de-DE")
	}
	if c.Miele.ConnectionCheckInterval != 10000 {
		t.Errorf("Miele.ConnectionCheckInterval = %d, want 10000", c.Miele.ConnectionCheckInterval)
	}
	if c.LogLevel != "info" {
		t.Errorf("LogLevel = %q, want %q", c.LogLevel, "info")
	}
}

func TestApplyDefaults_DoesNotOverrideExplicitValues(t *testing.T) {
	c := Config{
		MQTT:     MQTTConfig{QoS: 2},
		Miele:    MieleConfig{Mode: "polling", CountryCode: "en-US"},
		LogLevel: "debug",
	}
	ApplyDefaults(&c)
	if c.MQTT.QoS != 2 {
		t.Errorf("QoS = %d, want 2", c.MQTT.QoS)
	}
	if c.Miele.Mode != "polling" {
		t.Errorf("Mode = %q, want polling", c.Miele.Mode)
	}
	if c.Miele.CountryCode != "en-US" {
		t.Errorf("CountryCode = %q, want en-US", c.Miele.CountryCode)
	}
	if c.LogLevel != "debug" {
		t.Errorf("LogLevel = %q, want debug", c.LogLevel)
	}
}

func TestLoadConfig_AppliesDefaults(t *testing.T) {
	tmp := t.TempDir()
	p := filepath.Join(tmp, "config.json")
	body := `{
        "mqtt": {"url": "tcp://localhost:1883", "topic": "miele"}
    }`
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}

	c, err := LoadConfig(p)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if !c.MQTT.Retain {
		t.Error("Retain default should be true")
	}
	if !c.MQTT.BridgeInfo {
		t.Error("BridgeInfo default should be true")
	}
	if !c.Miele.PersistToken {
		t.Error("PersistToken default should be true")
	}
	if !c.SendFullUpdate {
		t.Error("SendFullUpdate default should be true")
	}
	if c.MQTT.QoS != 1 {
		t.Errorf("QoS default = %d, want 1", c.MQTT.QoS)
	}
	if c.Miele.Mode != "sse" {
		t.Errorf("Mode default = %q, want sse", c.Miele.Mode)
	}
	if c.LogLevel != "info" {
		t.Errorf("LogLevel default = %q, want info", c.LogLevel)
	}
}

func TestLoadConfig_ExplicitOverrideDefaults(t *testing.T) {
	tmp := t.TempDir()
	p := filepath.Join(tmp, "config.json")
	body := `{
        "mqtt": {"url": "tcp://localhost:1883", "topic": "miele", "retain": false, "bridge-info": false},
        "miele": {"persistToken": false, "mode": "polling"},
        "send-full-update": false
    }`
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	c, err := LoadConfig(p)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if c.MQTT.Retain {
		t.Error("Retain should be false")
	}
	if c.MQTT.BridgeInfo {
		t.Error("BridgeInfo should be false")
	}
	if c.Miele.PersistToken {
		t.Error("PersistToken should be false")
	}
	if c.SendFullUpdate {
		t.Error("SendFullUpdate should be false")
	}
	if c.Miele.Mode != "polling" {
		t.Errorf("Mode = %q, want polling", c.Miele.Mode)
	}
}

func TestLoadConfig_EnvVarSubstitution(t *testing.T) {
	t.Setenv("MIELE_USERNAME", "alice")
	t.Setenv("MIELE_PASSWORD", "secret")

	tmp := t.TempDir()
	p := filepath.Join(tmp, "config.json")
	body := `{
        "mqtt": {"url": "tcp://localhost:1883", "topic": "miele"},
        "miele": {
            "client-id": "cid", "client-secret": "csec",
            "username": "${MIELE_USERNAME}",
            "password": "${MIELE_PASSWORD}"
        }
    }`
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	c, err := LoadConfig(p)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if c.Miele.Username != "alice" {
		t.Errorf("Username = %q, want alice", c.Miele.Username)
	}
	if c.Miele.Password != "secret" {
		t.Errorf("Password = %q, want secret", c.Miele.Password)
	}
}

func TestLoadConfig_MissingEnvVarBecomesEmpty(t *testing.T) {
	os.Unsetenv("MIELE_NOPE")
	tmp := t.TempDir()
	p := filepath.Join(tmp, "config.json")
	body := `{
        "mqtt": {"url": "tcp://localhost:1883", "topic": "miele"},
        "miele": {"username": "${MIELE_NOPE}"}
    }`
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	c, err := LoadConfig(p)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if c.Miele.Username != "" {
		t.Errorf("Username = %q, want empty", c.Miele.Username)
	}
}

func TestLoadConfig_MissingFile(t *testing.T) {
	_, err := LoadConfig(filepath.Join(t.TempDir(), "nope.json"))
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestBridgeInfoTopic_Default(t *testing.T) {
	c := Config{MQTT: MQTTConfig{Topic: "home/miele"}}
	if got := c.BridgeInfoTopic(); got != "home/miele/bridge/state" {
		t.Errorf("BridgeInfoTopic = %q, want home/miele/bridge/state", got)
	}
}

func TestBridgeInfoTopic_Override(t *testing.T) {
	c := Config{MQTT: MQTTConfig{Topic: "home/miele", BridgeInfoTopic: "custom/topic"}}
	if got := c.BridgeInfoTopic(); got != "custom/topic" {
		t.Errorf("BridgeInfoTopic = %q, want custom/topic", got)
	}
}

func TestLoadConfig_DiscoveryDefaults(t *testing.T) {
	tmp := t.TempDir()
	p := filepath.Join(tmp, "config.json")
	body := `{"mqtt": {"url": "tcp://localhost:1883", "topic": "miele"}}`
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	c, err := LoadConfig(p)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	d := c.MQTT.Discovery
	if d == nil {
		t.Fatal("Discovery should be populated with defaults")
	}
	if d.Enabled {
		t.Errorf("Enabled = true, want default false")
	}
	if d.Prefix != "homeassistant" {
		t.Errorf("Prefix = %q, want homeassistant", d.Prefix)
	}
	if d.DeviceNamePrefix != "Miele" {
		t.Errorf("DeviceNamePrefix = %q, want Miele", d.DeviceNamePrefix)
	}
}

func TestLoadConfig_DiscoveryEnabledKeepsOtherDefaults(t *testing.T) {
	tmp := t.TempDir()
	p := filepath.Join(tmp, "config.json")
	body := `{
        "mqtt": {"url": "tcp://localhost:1883", "topic": "miele",
                 "discovery": {"enabled": true}}
    }`
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	c, err := LoadConfig(p)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	d := c.MQTT.Discovery
	if !d.Enabled {
		t.Error("Enabled should be true")
	}
	if d.Prefix != "homeassistant" {
		t.Errorf("Prefix = %q, want default homeassistant", d.Prefix)
	}
	if d.DeviceNamePrefix != "Miele" {
		t.Errorf("DeviceNamePrefix = %q, want default Miele", d.DeviceNamePrefix)
	}
}

func TestLoadConfig_DiscoveryCustomFields(t *testing.T) {
	tmp := t.TempDir()
	p := filepath.Join(tmp, "config.json")
	body := `{
        "mqtt": {"url": "tcp://localhost:1883", "topic": "miele",
                 "discovery": {"enabled": true, "prefix": "ha", "device-name-prefix": "Kitchen"}}
    }`
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	c, err := LoadConfig(p)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	d := c.MQTT.Discovery
	if d.Prefix != "ha" {
		t.Errorf("Prefix = %q, want ha", d.Prefix)
	}
	if d.DeviceNamePrefix != "Kitchen" {
		t.Errorf("DeviceNamePrefix = %q, want Kitchen", d.DeviceNamePrefix)
	}
}

func TestLoadConfig_SSEBackoffDefaults(t *testing.T) {
	tmp := t.TempDir()
	p := filepath.Join(tmp, "config.json")
	body := `{"mqtt": {"url": "tcp://localhost:1883", "topic": "miele"}}`
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	c, err := LoadConfig(p)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	b := c.Miele.SSEBackoff
	if b == nil {
		t.Fatal("SSEBackoff should be populated with defaults")
	}
	if b.FailureThreshold != 5 {
		t.Errorf("FailureThreshold = %d, want 5", b.FailureThreshold)
	}
	if b.BaseDelayDuration() != 5*time.Second {
		t.Errorf("BaseDelay = %v, want 5s", b.BaseDelayDuration())
	}
	if b.MaxDelayDuration() != 10*time.Minute {
		t.Errorf("MaxDelay = %v, want 10m", b.MaxDelayDuration())
	}
}

func TestLoadConfig_SSEBackoffPartialOverride(t *testing.T) {
	tmp := t.TempDir()
	p := filepath.Join(tmp, "config.json")
	body := `{
        "mqtt": {"url": "tcp://localhost:1883", "topic": "miele"},
        "miele": {"sse-backoff": {"failure-threshold": 10}}
    }`
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	c, err := LoadConfig(p)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	b := c.Miele.SSEBackoff
	if b.FailureThreshold != 10 {
		t.Errorf("FailureThreshold = %d, want 10", b.FailureThreshold)
	}
	if b.BaseDelayDuration() != 5*time.Second {
		t.Errorf("BaseDelay = %v, want default 5s", b.BaseDelayDuration())
	}
	if b.MaxDelayDuration() != 10*time.Minute {
		t.Errorf("MaxDelay = %v, want default 10m", b.MaxDelayDuration())
	}
}

func TestLoadConfig_SSEBackoffCustomDurations(t *testing.T) {
	tmp := t.TempDir()
	p := filepath.Join(tmp, "config.json")
	body := `{
        "mqtt": {"url": "tcp://localhost:1883", "topic": "miele"},
        "miele": {"sse-backoff": {"base-delay": "2s", "max-delay": "5m"}}
    }`
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	c, err := LoadConfig(p)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	b := c.Miele.SSEBackoff
	if b.BaseDelayDuration() != 2*time.Second {
		t.Errorf("BaseDelay = %v, want 2s", b.BaseDelayDuration())
	}
	if b.MaxDelayDuration() != 5*time.Minute {
		t.Errorf("MaxDelay = %v, want 5m", b.MaxDelayDuration())
	}
}

func TestLoadConfig_SSEBackoffInvalidDuration(t *testing.T) {
	tmp := t.TempDir()
	p := filepath.Join(tmp, "config.json")
	body := `{
        "mqtt": {"url": "tcp://localhost:1883", "topic": "miele"},
        "miele": {"sse-backoff": {"base-delay": "five seconds"}}
    }`
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := LoadConfig(p)
	if err == nil {
		t.Fatal("expected error for invalid duration")
	}
	if !strings.Contains(err.Error(), "base-delay") {
		t.Errorf("error should name the offending field, got: %v", err)
	}
}

func TestMieleStateTopic(t *testing.T) {
	c := Config{MQTT: MQTTConfig{Topic: "home/miele"}}
	if got := c.MieleStateTopic(); got != "home/miele/bridge/miele" {
		t.Errorf("MieleStateTopic = %q, want home/miele/bridge/miele", got)
	}
}

func TestPersistToken_WritesWhenChanged(t *testing.T) {
	tmp := t.TempDir()
	p := filepath.Join(tmp, "config.json")
	body := `{
        "mqtt": {"url": "tcp://localhost:1883", "topic": "miele"},
        "miele": {"client-id": "cid", "client-secret": "csec"}
    }`
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadConfig(p); err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	PersistToken(TokenConfig{Access: "a", Refresh: "r", ValidUntil: "2030-01-01T00:00:00Z"})

	out, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatalf("re-parse: %v", err)
	}
	miele, ok := m["miele"].(map[string]any)
	if !ok {
		t.Fatal("miele section missing")
	}
	tok, ok := miele["token"].(map[string]any)
	if !ok {
		t.Fatal("token missing")
	}
	if tok["access"] != "a" || tok["refresh"] != "r" || tok["validUntil"] != "2030-01-01T00:00:00Z" {
		t.Errorf("token = %#v", tok)
	}
}

func TestPersistToken_SkipsWhenUnchanged(t *testing.T) {
	tmp := t.TempDir()
	p := filepath.Join(tmp, "config.json")
	body := `{
        "mqtt": {"url": "tcp://localhost:1883", "topic": "miele"},
        "miele": {
            "client-id": "cid",
            "token": {"access":"a","refresh":"r","validUntil":"v"}
        }
    }`
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadConfig(p); err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	before, _ := os.Stat(p)
	PersistToken(TokenConfig{Access: "a", Refresh: "r", ValidUntil: "v"})
	after, _ := os.Stat(p)
	if before.ModTime() != after.ModTime() {
		// Allow filesystems with coarse mtime resolution: also accept identical bytes
		out, _ := os.ReadFile(p)
		if !strings.Contains(string(out), `"access":"a"`) && !strings.Contains(string(out), `"access": "a"`) {
			t.Errorf("file changed unexpectedly: %s", string(out))
		}
	}
}

func TestPersistToken_NoOpWhenPersistDisabled(t *testing.T) {
	tmp := t.TempDir()
	p := filepath.Join(tmp, "config.json")
	body := `{
        "mqtt": {"url": "tcp://localhost:1883", "topic": "miele"},
        "miele": {"client-id": "cid", "persistToken": false}
    }`
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadConfig(p); err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	PersistToken(TokenConfig{Access: "a", Refresh: "r"})

	out, _ := os.ReadFile(p)
	if strings.Contains(string(out), `"token"`) {
		t.Errorf("token should not have been persisted, file = %s", string(out))
	}
}
