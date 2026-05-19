package bridge

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/mqtt-home/miele-to-mqtt-gw/config"
)

// fullExample is the same shape as fullmessage-example.md, trimmed to the
// fields the discovery layer reads.
const fullExample = `{
  "ident": {
    "xkmIdentLabel": { "releaseVersion": "03.59", "techType": "EK037" },
    "deviceIdentLabel": { "fabNumber": "000101234567", "techType": "G7560" },
    "type": { "value_raw": 7, "value_localized": "Dishwasher" }
  },
  "state": { "status": { "value_raw": 5 } }
}`

func discoveryCfg() config.Config {
	c := config.Config{
		MQTT: config.MQTTConfig{
			Topic: "home/miele",
			Discovery: &config.DiscoveryConfig{
				Enabled:          true,
				Prefix:           "homeassistant",
				DeviceNamePrefix: "Miele",
			},
		},
	}
	return c
}

func TestBuildDiscoveryPayloads_FiveEntities(t *testing.T) {
	cfg := discoveryCfg()
	got, err := buildDiscoveryPayloads(cfg, "device-abc", []byte(fullExample))
	if err != nil {
		t.Fatalf("buildDiscoveryPayloads: %v", err)
	}
	if len(got) != 5 {
		t.Fatalf("got %d topics, want 5: %v", len(got), keys(got))
	}
	for _, suffix := range []string{
		"/sensor/miele_000101234567/state/config",
		"/sensor/miele_000101234567/phase/config",
		"/sensor/miele_000101234567/remaining_duration/config",
		"/sensor/miele_000101234567/remaining_minutes/config",
		"/sensor/miele_000101234567/time_completed/config",
	} {
		expected := "homeassistant" + suffix
		if _, ok := got[expected]; !ok {
			t.Errorf("missing topic %q; got %v", expected, keys(got))
		}
	}
}

func TestBuildDiscoveryPayloads_DeviceRegistryFields(t *testing.T) {
	cfg := discoveryCfg()
	got, err := buildDiscoveryPayloads(cfg, "device-abc", []byte(fullExample))
	if err != nil {
		t.Fatalf("buildDiscoveryPayloads: %v", err)
	}
	any := pickAny(got)
	var p DiscoveryPayload
	if err := json.Unmarshal(any, &p); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if p.Device.Manufacturer != "Miele" {
		t.Errorf("manufacturer = %q", p.Device.Manufacturer)
	}
	if p.Device.Model != "Dishwasher" {
		t.Errorf("model = %q, want Dishwasher", p.Device.Model)
	}
	if p.Device.SWVersion != "03.59" {
		t.Errorf("sw_version = %q, want 03.59", p.Device.SWVersion)
	}
	if p.Device.Name != "Miele Dishwasher 000101234567" {
		t.Errorf("device name = %q", p.Device.Name)
	}
	if p.Device.SerialNumber != "000101234567" {
		t.Errorf("serial_number = %q", p.Device.SerialNumber)
	}
	if len(p.Device.Identifiers) != 1 || p.Device.Identifiers[0] != "miele_000101234567" {
		t.Errorf("identifiers = %v", p.Device.Identifiers)
	}
	if p.StateTopic != "home/miele/device-abc" {
		t.Errorf("state_topic = %q", p.StateTopic)
	}
	if p.AvailabilityMode != "any" {
		t.Errorf("availability_mode = %q, want any", p.AvailabilityMode)
	}
	if len(p.Availability) != 2 {
		t.Fatalf("availability len = %d, want 2", len(p.Availability))
	}
	if p.Availability[0].PayloadAvailable != "connected" || p.Availability[1].PayloadAvailable != "degraded" {
		t.Errorf("availability payload_available values = %+v", p.Availability)
	}
}

func TestBuildDiscoveryPayloads_FabNumberFallback(t *testing.T) {
	// Payload without fabNumber: identity falls back to the Miele device id.
	rawFull := `{
        "ident": {
            "xkmIdentLabel": { "techType": "G7560", "releaseVersion": "03.59" },
            "type": { "value_localized": "Dishwasher" }
        }
    }`
	cfg := discoveryCfg()
	got, err := buildDiscoveryPayloads(cfg, "device-no-fab", []byte(rawFull))
	if err != nil {
		t.Fatalf("buildDiscoveryPayloads: %v", err)
	}
	wantPrefix := "homeassistant/sensor/miele_device-no-fab/"
	for topic := range got {
		if !strings.HasPrefix(topic, wantPrefix) {
			t.Errorf("topic %q does not use device id fallback", topic)
		}
	}
}

func TestBuildDiscoveryPayloads_RemainingMinutesHasUnitClass(t *testing.T) {
	cfg := discoveryCfg()
	got, err := buildDiscoveryPayloads(cfg, "device-abc", []byte(fullExample))
	if err != nil {
		t.Fatalf("buildDiscoveryPayloads: %v", err)
	}

	for topic, raw := range got {
		var p DiscoveryPayload
		if err := json.Unmarshal(raw, &p); err != nil {
			t.Fatalf("unmarshal %s: %v", topic, err)
		}
		isMinutes := strings.Contains(topic, "/remaining_minutes/")
		if isMinutes {
			if p.UnitOfMeasurement != "min" {
				t.Errorf("remaining_minutes unit_of_measurement = %q, want min", p.UnitOfMeasurement)
			}
			if p.DeviceClass != "duration" {
				t.Errorf("remaining_minutes device_class = %q, want duration", p.DeviceClass)
			}
			if p.StateClass != "measurement" {
				t.Errorf("remaining_minutes state_class = %q, want measurement", p.StateClass)
			}
		} else {
			if p.UnitOfMeasurement != "" || p.DeviceClass != "" || p.StateClass != "" {
				t.Errorf("entity %s should not have unit/class fields: %+v", topic, p)
			}
		}
	}
}

func TestBuildDiscoveryPayloads_DisabledReturnsNil(t *testing.T) {
	cfg := discoveryCfg()
	cfg.MQTT.Discovery.Enabled = false
	got, err := buildDiscoveryPayloads(cfg, "device-abc", []byte(fullExample))
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil when disabled, got %v", got)
	}
}

func TestBuildDiscoveryPayloads_ModelFallsBackToTechType(t *testing.T) {
	// When ident.type.value_localized is missing, model uses xkmIdentLabel.techType.
	rawFull := `{
        "ident": {
            "deviceIdentLabel": { "fabNumber": "000101234567" },
            "xkmIdentLabel": { "techType": "G7560", "releaseVersion": "03.59" }
        }
    }`
	cfg := discoveryCfg()
	got, _ := buildDiscoveryPayloads(cfg, "device-abc", []byte(rawFull))
	any := pickAny(got)
	var p DiscoveryPayload
	_ = json.Unmarshal(any, &p)
	if p.Device.Model != "EK037" && p.Device.Model != "G7560" {
		t.Errorf("model fallback should be techType, got %q", p.Device.Model)
	}
	if p.Device.Name != "Miele 000101234567" {
		// no type_localized → name has only prefix + id (joinNonEmpty skips blanks)
		t.Errorf("device name = %q, want %q", p.Device.Name, "Miele 000101234567")
	}
}

// Helpers.

func keys(m map[string][]byte) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func pickAny(m map[string][]byte) []byte {
	for _, v := range m {
		return v
	}
	return nil
}
