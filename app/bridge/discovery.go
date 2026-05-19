package bridge

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mqtt-home/miele-to-mqtt-gw/config"
	"github.com/philipparndt/go-logger"
)

// DiscoveryAvailability is one entry in the HA `availability` array.
type DiscoveryAvailability struct {
	Topic               string `json:"topic"`
	PayloadAvailable    string `json:"payload_available"`
	PayloadNotAvailable string `json:"payload_not_available"`
}

// DiscoveryDevice is the HA device-registry block embedded in every entity's
// discovery payload — all entities for one appliance share these fields so
// HA groups them under a single device tile.
type DiscoveryDevice struct {
	Identifiers  []string `json:"identifiers"`
	Manufacturer string   `json:"manufacturer"`
	Model        string   `json:"model,omitempty"`
	Name         string   `json:"name"`
	SWVersion    string   `json:"sw_version,omitempty"`
	SerialNumber string   `json:"serial_number"`
}

// DiscoveryPayload is the JSON shape HA expects on
// `<prefix>/sensor/<node>/<entity>/config`.
type DiscoveryPayload struct {
	Name              string                  `json:"name"`
	UniqueID          string                  `json:"unique_id"`
	ObjectID          string                  `json:"object_id"`
	StateTopic        string                  `json:"state_topic"`
	ValueTemplate     string                  `json:"value_template"`
	UnitOfMeasurement string                  `json:"unit_of_measurement,omitempty"`
	DeviceClass       string                  `json:"device_class,omitempty"`
	StateClass        string                  `json:"state_class,omitempty"`
	Availability      []DiscoveryAvailability `json:"availability"`
	AvailabilityMode  string                  `json:"availability_mode"`
	Device            DiscoveryDevice         `json:"device"`
}

// discoveryEntity captures the fixed per-entity differences in the discovery
// payload set.
type discoveryEntity struct {
	key               string // url-safe entity slug, e.g. "remaining_minutes"
	displayName       string // shown in HA, e.g. "Remaining minutes"
	valueTemplate     string
	unitOfMeasurement string
	deviceClass       string
	stateClass        string
}

// discoveryEntities is the fixed set published per device. Order is stable for
// deterministic tests.
var discoveryEntities = []discoveryEntity{
	{
		key:           "state",
		displayName:   "State",
		valueTemplate: "{{ value_json.state }}",
	},
	{
		key:           "phase",
		displayName:   "Phase",
		valueTemplate: "{{ value_json.phase }}",
	},
	{
		key:           "remaining_duration",
		displayName:   "Remaining duration",
		valueTemplate: "{{ value_json.remainingDuration }}",
	},
	{
		key:               "remaining_minutes",
		displayName:       "Remaining minutes",
		valueTemplate:     "{{ value_json.remainingDurationMinutes }}",
		unitOfMeasurement: "min",
		deviceClass:       "duration",
		stateClass:        "measurement",
	},
	{
		key:           "time_completed",
		displayName:   "Time completed",
		valueTemplate: "{{ value_json.timeCompleted }}",
	},
}

// identity captures the descriptive metadata pulled from the Miele full
// payload's `ident.*` block.
type identity struct {
	id            string // fabNumber when present, else the Miele device id
	usedDeviceID  bool   // true when we fell back to the API device id
	typeLocalized string
	techType      string
	swVersion     string
}

// extractIdentity walks the raw full payload to extract the four
// device-registry-relevant strings. Returns sane zero values when fields are
// missing.
func extractIdentity(deviceID string, rawFull []byte) identity {
	out := identity{}
	if len(rawFull) == 0 {
		out.id = deviceID
		out.usedDeviceID = true
		return out
	}
	var generic any
	if err := json.Unmarshal(rawFull, &generic); err != nil {
		out.id = deviceID
		out.usedDeviceID = true
		return out
	}

	out.typeLocalized = walkString(generic, []string{"ident", "type", "value_localized"})
	out.techType = walkString(generic, []string{"ident", "xkmIdentLabel", "techType"})
	out.swVersion = walkString(generic, []string{"ident", "xkmIdentLabel", "releaseVersion"})

	fab := walkString(generic, []string{"ident", "deviceIdentLabel", "fabNumber"})
	if fab == "" {
		out.id = deviceID
		out.usedDeviceID = true
	} else {
		out.id = fab
	}
	return out
}

// walkString descends a JSON object map by path and returns the leaf as a
// string. Returns "" when any segment is missing or the leaf isn't a string.
func walkString(v any, path []string) string {
	for _, seg := range path {
		m, ok := v.(map[string]any)
		if !ok {
			return ""
		}
		v = m[seg]
	}
	s, _ := v.(string)
	return s
}

// buildDiscoveryPayloads assembles the per-entity discovery topic + JSON map
// for one device. Returns nil, nil when discovery is disabled in cfg.
//
// The caller is expected to publish each entry with the retain flag — that's
// HA's discovery contract. We fall back to the Miele API device id when the
// full payload is missing fabNumber, logging a warning.
func buildDiscoveryPayloads(cfg config.Config, deviceID string, rawFull []byte) (map[string][]byte, error) {
	disc := cfg.MQTT.Discovery
	if disc == nil || !disc.Enabled {
		return nil, nil
	}

	id := extractIdentity(deviceID, rawFull)
	if id.usedDeviceID {
		logger.Warn("[discovery] fabNumber missing in full payload, falling back to Miele device id",
			"device", deviceID)
	}

	stateTopic := cfg.MQTT.Topic + "/" + deviceID
	availabilityTopic := cfg.MieleStateTopic()
	availability := []DiscoveryAvailability{
		{Topic: availabilityTopic, PayloadAvailable: "connected", PayloadNotAvailable: "disconnected"},
		{Topic: availabilityTopic, PayloadAvailable: "degraded", PayloadNotAvailable: "disconnected"},
	}

	model := id.typeLocalized
	if model == "" {
		model = id.techType
	}

	device := DiscoveryDevice{
		Identifiers:  []string{"miele_" + id.id},
		Manufacturer: "Miele",
		Model:        model,
		Name:         joinNonEmpty(" ", disc.DeviceNamePrefix, id.typeLocalized, id.id),
		SWVersion:    id.swVersion,
		SerialNumber: id.id,
	}

	out := make(map[string][]byte, len(discoveryEntities))
	for _, e := range discoveryEntities {
		uid := "miele_" + id.id + "_" + e.key
		topic := strings.TrimRight(disc.Prefix, "/") + "/sensor/miele_" + id.id + "/" + e.key + "/config"
		payload := DiscoveryPayload{
			Name:              e.displayName,
			UniqueID:          uid,
			ObjectID:          uid,
			StateTopic:        stateTopic,
			ValueTemplate:     e.valueTemplate,
			UnitOfMeasurement: e.unitOfMeasurement,
			DeviceClass:       e.deviceClass,
			StateClass:        e.stateClass,
			Availability:      availability,
			AvailabilityMode:  "any",
			Device:            device,
		}
		b, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("marshal discovery payload for %s/%s: %w", id.id, e.key, err)
		}
		out[topic] = b
	}
	return out, nil
}

// joinNonEmpty joins the given parts with sep, skipping any empty strings.
func joinNonEmpty(sep string, parts ...string) string {
	kept := make([]string, 0, len(parts))
	for _, p := range parts {
		if p != "" {
			kept = append(kept, p)
		}
	}
	return strings.Join(kept, sep)
}
