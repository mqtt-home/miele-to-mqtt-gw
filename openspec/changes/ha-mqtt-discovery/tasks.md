## 1. Config schema

- [x] 1.1 Add `DiscoveryConfig` struct to `app/config/config.go` with `Enabled bool`, `Prefix string`, `DeviceNamePrefix string` and JSON tags `enabled`, `prefix`, `device-name-prefix`. Add it under `MQTTConfig` as `Discovery *DiscoveryConfig json:"discovery,omitempty"`.
- [x] 1.2 Extend `ApplyDefaults` so `Discovery` is non-nil after load: zero-valued `Enabled` stays `false`, `Prefix` defaults to `"homeassistant"`, `DeviceNamePrefix` defaults to `"Miele"`.
- [x] 1.3 Update `config_test.go`: (a) absent block uses defaults and `Enabled=false`, (b) `Enabled=true` with other fields unset keeps defaults, (c) custom `Prefix` and `DeviceNamePrefix` round-trip correctly.

## 2. Discovery payload builder

- [x] 2.1 Create `app/bridge/discovery.go` with a small `DiscoveryPayload` struct that matches the HA-discovery JSON shape (use `json:",omitempty"` on optional fields so the on-the-wire bytes don't include zero strings).
- [x] 2.2 Add a `extractIdentity` helper that walks the raw full payload to pull out `fabNumber`, `type.value_localized`, `xkmIdentLabel.techType`, and `xkmIdentLabel.releaseVersion`. Use the existing `walk` pattern from `transform/transform.go` (consider exporting it or duplicating — duplicating is fine since this is a small reach into the payload).
- [x] 2.3 Define a fixed table of the five entities (`state`, `phase`, `remaining_duration`, `remaining_minutes`, `time_completed`) with their `value_template` strings and per-entity overrides (the unit/class/state_class for `remaining_minutes`).
- [x] 2.4 Add `buildDiscoveryPayloads(cfg config.Config, deviceID string, rawFull []byte) (topicToPayload map[string][]byte, err error)` that uses the identity helper to produce the topic-keyed map of marshalled JSON for the five entities. Falls back to `deviceID` if `fabNumber` is missing and logs a warning naming the device.
- [x] 2.5 Unit-test the builder: feed it the `fullmessage-example.md` payload bytes and assert the five expected topics and that each payload contains the expected `device`, `unique_id`, `state_topic`, `value_template`, and `availability` keys.
- [x] 2.6 Test the fabNumber fallback: payload without `ident.deviceIdentLabel.fabNumber` produces topics keyed by the Miele device id and a warning log (capture via the logger or just assert the topic shape).
- [x] 2.7 Test that `remaining_minutes` includes `unit_of_measurement: "min"`, `device_class: "duration"`, `state_class: "measurement"` and the other four entities do NOT include those keys.

## 3. Publisher integration

- [x] 3.1 Extend `Publisher` with a `discoveredTopics map[string]struct{}` (guarded by the existing mutex) tracking every discovery topic published during the run.
- [x] 3.2 In `PublishDevice`, after the small/full publishes, when `cfg.MQTT.Discovery.Enabled` is true: call `buildDiscoveryPayloads`, iterate the map, send each through `publishWithDedup` (so dedup applies), and add each topic to `discoveredTopics`.
- [x] 3.3 Add a `CleanupDiscovery()` method that, for each entry in `discoveredTopics`, publishes the empty byte string with the retain flag set, then clears the set. No-op when discovery is disabled or the set is empty.
- [x] 3.4 Wire `CleanupDiscovery()` into `app/app.go:stop` so it runs after the SSE/polling stop but before the final `disconnected` publish on `bridge/miele`. Match the order documented in the mqtt-bridge spec delta.

## 4. App-level tests

- [x] 4.1 `app/bridge/publisher_test.go`: add a case that constructs a Publisher with `Discovery.Enabled=true`, calls `PublishDevice` twice with the same full payload, and asserts (a) discovery topics show up in `discoveredTopics`, (b) with `mqtt.deduplicate=true` the second call does not re-publish identical bytes.
- [x] 4.2 Test that with `Discovery.Enabled=false` no discovery topics are published or tracked.
- [x] 4.3 Test `CleanupDiscovery()`: after publishing for two devices, calling it produces one empty-payload publish per previously-tracked topic and clears the set.

## 5. Documentation

- [x] 5.1 Update `README.md` with a new "Home Assistant integration" section that documents the four-line opt-in (`mqtt.discovery.enabled: true`), the topics produced, the entities exposed, and the device-registry metadata mapping.
- [x] 5.2 Add a `mqtt.discovery` block (commented as optional) to `config-example.json` showing all three fields and the defaults.
- [x] 5.3 Note in the README that `mqtt.deduplicate: true` is recommended when discovery is enabled, to keep retained-republish traffic in check.

## 6. Validation

- [x] 6.1 Run `go test ./...` and `go vet ./...` — clean.
- [x] 6.2 Run `go test -race ./bridge/...` — clean (the discoveredTopics set is touched from the publish path and the cleanup path).
- [x] 6.3 Run `openspec validate ha-mqtt-discovery --strict` — clean.
- [ ] 6.4 Manual: with `discovery.enabled: true` against a local HA instance + Mosquitto, verify (a) the device tile appears with manufacturer/model/firmware fields populated, (b) the five entities update as the appliance state changes, (c) bringing the bridge down via SIGTERM removes the device from HA. Flag as pending if a live HA instance is not available.
