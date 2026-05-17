package transform

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/mqtt-home/miele-to-mqtt-gw/miele/api"
)

func TestBuild_EmptyDevice(t *testing.T) {
	dev := api.Device{ID: "x", Data: json.RawMessage(`{}`)}
	now := time.Date(2026, 1, 2, 12, 30, 0, 0, time.UTC)
	got := Build(dev, now)
	want := SmallMessage{
		Phase:                    "UNKNOWN",
		PhaseID:                  -1,
		State:                    "UNKNOWN",
		RemainingDurationMinutes: 0,
		RemainingDuration:        "0:00",
		TimeCompleted:            "12:30",
	}
	if got != want {
		t.Errorf("Build empty = %+v, want %+v", got, want)
	}
}

func TestBuild_RunningDevice(t *testing.T) {
	dev := api.Device{
		ID: "d",
		Data: json.RawMessage(`{
            "state": {
                "programPhase": {"value_raw": 1799},
                "status": {"value_raw": 5},
                "remainingTime": [0, 4]
            }
        }`),
	}
	now := time.Date(2026, 1, 2, 12, 31, 0, 0, time.UTC)
	got := Build(dev, now)
	want := SmallMessage{
		Phase:                    "DRYING",
		PhaseID:                  1799,
		State:                    "RUNNING",
		RemainingDurationMinutes: 4,
		RemainingDuration:        "0:04",
		TimeCompleted:            "12:35",
	}
	if got != want {
		t.Errorf("Build = %+v, want %+v", got, want)
	}
}

func TestBuild_OffDevice_ZerosDuration(t *testing.T) {
	dev := api.Device{
		ID: "d",
		Data: json.RawMessage(`{
            "state": {
                "programPhase": {"value_raw": 0},
                "status": {"value_raw": 1},
                "remainingTime": [1, 30]
            }
        }`),
	}
	now := time.Date(2026, 1, 2, 8, 0, 0, 0, time.UTC)
	got := Build(dev, now)
	if got.State != "OFF" {
		t.Errorf("State = %q, want OFF", got.State)
	}
	if got.RemainingDurationMinutes != 0 {
		t.Errorf("RemainingDurationMinutes = %d, want 0", got.RemainingDurationMinutes)
	}
	if got.RemainingDuration != "0:00" {
		t.Errorf("RemainingDuration = %q, want 0:00", got.RemainingDuration)
	}
}

func TestBuild_LongDuration(t *testing.T) {
	dev := api.Device{
		ID: "d",
		Data: json.RawMessage(`{
            "state": {
                "programPhase": {"value_raw": 1795},
                "status": {"value_raw": 5},
                "remainingTime": [2, 15]
            }
        }`),
	}
	now := time.Date(2026, 1, 2, 9, 30, 0, 0, time.UTC)
	got := Build(dev, now)
	if got.RemainingDurationMinutes != 135 {
		t.Errorf("RemainingDurationMinutes = %d, want 135", got.RemainingDurationMinutes)
	}
	if got.RemainingDuration != "2:15" {
		t.Errorf("RemainingDuration = %q, want 2:15", got.RemainingDuration)
	}
	if got.TimeCompleted != "11:45" {
		t.Errorf("TimeCompleted = %q, want 11:45", got.TimeCompleted)
	}
}

func TestBuild_MissingFieldsDoNotPanic(t *testing.T) {
	dev := api.Device{ID: "d", Data: json.RawMessage(`{"state": {"foo": "bar"}}`)}
	got := Build(dev, time.Now())
	if got.PhaseID != -1 || got.Phase != "UNKNOWN" {
		t.Errorf("missing phase = %+v", got)
	}
}

func TestStripNulls_FlatObject(t *testing.T) {
	raw := []byte(`{"a":1,"b":null,"c":"x"}`)
	out, err := StripNulls(raw)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(out), `"b"`) {
		t.Errorf("null key not stripped: %s", string(out))
	}
	if !strings.Contains(string(out), `"a":1`) {
		t.Errorf("non-null key dropped: %s", string(out))
	}
}

func TestStripNulls_NestedAndArray(t *testing.T) {
	raw := []byte(`{"a":{"b":null,"c":2},"d":[1,null,3]}`)
	out, err := StripNulls(raw)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(out), `null`) {
		t.Errorf("nulls remain: %s", string(out))
	}
}

func TestMarshalNoNulls_RoundTrip(t *testing.T) {
	v := map[string]any{"a": 1, "b": nil, "c": "x"}
	out, err := MarshalNoNulls(v)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(out), `"b"`) {
		t.Errorf("null kept: %s", string(out))
	}
}

func TestParseDuration(t *testing.T) {
	if parseDuration([]int{0, 4}) != 4*time.Minute {
		t.Error("[0,4] should be 4 min")
	}
	if parseDuration([]int{1, 30}) != 90*time.Minute {
		t.Error("[1,30] should be 90 min")
	}
	if parseDuration([]int{}) != 0 {
		t.Error("empty should be zero")
	}
	if parseDuration(nil) != 0 {
		t.Error("nil should be zero")
	}
}

func TestFormatHours(t *testing.T) {
	cases := []struct {
		d    time.Duration
		want string
	}{
		{0, "0:00"},
		{4 * time.Minute, "0:04"},
		{59 * time.Minute, "0:59"},
		{60 * time.Minute, "1:00"},
		{135 * time.Minute, "2:15"},
	}
	for _, c := range cases {
		if got := formatHours(c.d); got != c.want {
			t.Errorf("formatHours(%v) = %q, want %q", c.d, got, c.want)
		}
	}
}
