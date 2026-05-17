package transform

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mqtt-home/miele-to-mqtt-gw/miele/api"
)

// SmallMessage is the JSON shape published on `<topic>/<deviceId>`. Field
// order and tag names match the TS `smallMessage` output byte-for-byte (modulo
// timeCompleted's wall-clock value).
type SmallMessage struct {
	Phase                    string `json:"phase"`
	PhaseID                  int    `json:"phaseId"`
	State                    string `json:"state"`
	RemainingDurationMinutes int    `json:"remainingDurationMinutes"`
	RemainingDuration        string `json:"remainingDuration"`
	TimeCompleted            string `json:"timeCompleted"`
}

// Build constructs the small message for a single device update. now lets
// tests pin the wall-clock used by timeCompleted.
func Build(d api.Device, now time.Time) SmallMessage {
	phaseRaw := readInt(d.Data, []string{"state", "programPhase", "value_raw"}, -1)
	statusRaw := readInt(d.Data, []string{"state", "status", "value_raw"}, -1)
	remaining := readIntArray(d.Data, []string{"state", "remainingTime"})

	dur := parseDuration(remaining)
	if DeviceStatus(statusRaw) == DeviceStatusOff {
		dur = 0
	}

	completed := now.Add(dur)

	return SmallMessage{
		Phase:                    PhaseName(Phase(phaseRaw)),
		PhaseID:                  phaseRaw,
		State:                    DeviceStatusName(DeviceStatus(statusRaw)),
		RemainingDurationMinutes: int(dur / time.Minute),
		RemainingDuration:        formatHours(dur),
		TimeCompleted:            fmt.Sprintf("%d:%02d", completed.Hour(), completed.Minute()),
	}
}

// parseDuration mirrors lib/miele/duration.ts: [hours, minutes] is the only
// shape the API emits in practice. We also handle [seconds, minutes, hours]
// defensively but anything else collapses to zero.
func parseDuration(a []int) time.Duration {
	switch len(a) {
	case 2:
		return time.Duration(a[0]*60+a[1]) * time.Minute
	case 3:
		return time.Duration(a[0]*3600+a[1]*60+a[2]) * time.Second
	default:
		return 0
	}
}

func formatHours(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	totalMin := int(d / time.Minute)
	return fmt.Sprintf("%d:%02d", totalMin/60, totalMin%60)
}

// readInt walks a JSON object path and returns the value as int, or
// fallback if any segment is missing or the leaf isn't a number.
func readInt(raw json.RawMessage, path []string, fallback int) int {
	if len(raw) == 0 {
		return fallback
	}
	var generic any
	if err := json.Unmarshal(raw, &generic); err != nil {
		return fallback
	}
	v := walk(generic, path)
	switch n := v.(type) {
	case float64:
		return int(n)
	case json.Number:
		i, err := n.Int64()
		if err != nil {
			return fallback
		}
		return int(i)
	default:
		return fallback
	}
}

func readIntArray(raw json.RawMessage, path []string) []int {
	if len(raw) == 0 {
		return nil
	}
	var generic any
	if err := json.Unmarshal(raw, &generic); err != nil {
		return nil
	}
	v := walk(generic, path)
	arr, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]int, 0, len(arr))
	for _, e := range arr {
		switch n := e.(type) {
		case float64:
			out = append(out, int(n))
		default:
			return nil
		}
	}
	return out
}

func walk(v any, path []string) any {
	for _, seg := range path {
		m, ok := v.(map[string]any)
		if !ok {
			return nil
		}
		v = m[seg]
	}
	return v
}

// MarshalNoNulls serializes v to JSON and removes any keys whose value is
// the literal null. This matches the TS `convertBody`'s `JSON.stringify`
// replacer that drops nulls.
func MarshalNoNulls(v any) ([]byte, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return stripNulls(raw)
}

// StripNulls removes keys whose value is null from arbitrarily nested JSON.
func StripNulls(raw []byte) ([]byte, error) {
	return stripNulls(raw)
}

func stripNulls(raw []byte) ([]byte, error) {
	var v any
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	if err := dec.Decode(&v); err != nil {
		return nil, err
	}
	cleaned := dropNulls(v)
	return json.Marshal(cleaned)
}

func dropNulls(v any) any {
	switch x := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(x))
		for k, vv := range x {
			if vv == nil {
				continue
			}
			out[k] = dropNulls(vv)
		}
		return out
	case []any:
		out := make([]any, 0, len(x))
		for _, e := range x {
			if e == nil {
				continue
			}
			out = append(out, dropNulls(e))
		}
		return out
	default:
		return v
	}
}
