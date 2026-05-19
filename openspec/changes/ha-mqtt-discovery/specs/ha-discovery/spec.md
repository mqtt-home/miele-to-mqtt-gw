## ADDED Requirements

### Requirement: Discovery is opt-in

The system SHALL publish Home Assistant MQTT-discovery payloads only when `mqtt.discovery.enabled` is `true`. When disabled (the default), no discovery topics SHALL be touched, and the bridge SHALL NOT publish or clear any retained payloads under the discovery prefix.

#### Scenario: Discovery disabled by default

- **WHEN** the config does not set `mqtt.discovery.enabled` (or sets it to `false`)
- **THEN** the bridge processes device updates as before
- **AND** publishes nothing under `<discovery_prefix>/sensor/.../config`

#### Scenario: Discovery enabled

- **WHEN** `mqtt.discovery.enabled` is `true`
- **THEN** every device update triggers a discovery publish for that device's entity set
- **AND** the discovery payloads are sent with the same retain flag and QoS as the device updates (`mqtt.retain`, `mqtt.qos`)

### Requirement: Per-entity discovery payloads

For each device update, the system SHALL publish exactly the following entity set (all as HA `sensor` components) under `<discovery_prefix>/sensor/miele_<id>/<entity>/config`:

| Entity                | `value_template`                            | Extra fields                                                |
| --------------------- | ------------------------------------------- | ----------------------------------------------------------- |
| `state`               | `{{ value_json.state }}`                    | —                                                           |
| `phase`               | `{{ value_json.phase }}`                    | —                                                           |
| `remaining_duration`  | `{{ value_json.remainingDuration }}`        | —                                                           |
| `remaining_minutes`   | `{{ value_json.remainingDurationMinutes }}` | `unit_of_measurement: "min"`, `device_class: "duration"`, `state_class: "measurement"` |
| `time_completed`      | `{{ value_json.timeCompleted }}`            | —                                                           |

The identifier `<id>` SHALL be the appliance's `ident.deviceIdentLabel.fabNumber` when present in the full payload. If `fabNumber` is missing or empty, the system SHALL fall back to the Miele API device id and log a warning naming the device.

#### Scenario: Five entities per device

- **WHEN** a device update produces a small message and the full payload contains `fabNumber = "000101234567"`
- **THEN** the bridge publishes (retained) to `<prefix>/sensor/miele_000101234567/state/config`, `.../phase/config`, `.../remaining_duration/config`, `.../remaining_minutes/config`, `.../time_completed/config`
- **AND** each payload's `state_topic` is `<mqtt.topic>/<deviceId>` (the existing small-message topic)
- **AND** each payload's `value_template` is the one listed in the entity table

#### Scenario: Numeric remaining_minutes carries unit + class

- **WHEN** the discovery payload for the `remaining_minutes` entity is constructed
- **THEN** the payload includes `"unit_of_measurement": "min"`, `"device_class": "duration"`, and `"state_class": "measurement"`

#### Scenario: Fallback when fabNumber is missing

- **WHEN** a device's full payload has no `ident.deviceIdentLabel.fabNumber` (or the value is the empty string)
- **THEN** the bridge uses the Miele API device id as `<id>` in the discovery topic and `unique_id`
- **AND** logs a warning message naming the affected device

### Requirement: Device-registry metadata

Each discovery payload SHALL include a `device` object so HA's device registry shows a single Miele appliance grouping all five entities. The fields SHALL be derived from the appliance's full payload:

- `identifiers`: `["miele_<id>"]`
- `manufacturer`: `"Miele"`
- `model`: `<ident.type.value_localized>` if present, else `<ident.xkmIdentLabel.techType>`, else the empty string
- `name`: `<mqtt.discovery.device-name-prefix> <ident.type.value_localized> <id>` (joined by single spaces, blanks elided)
- `sw_version`: `<ident.xkmIdentLabel.releaseVersion>` if present
- `serial_number`: `<id>`

#### Scenario: Device tile shows appliance metadata

- **WHEN** the full payload contains `ident.type.value_localized = "Dishwasher"`, `ident.xkmIdentLabel.techType = "G7560"`, `ident.xkmIdentLabel.releaseVersion = "03.59"`, `fabNumber = "000101234567"`, and `mqtt.discovery.device-name-prefix = "Miele"`
- **THEN** the discovery payload's `device.name` is `"Miele Dishwasher 000101234567"`
- **AND** `device.manufacturer` is `"Miele"`
- **AND** `device.model` is `"Dishwasher"`
- **AND** `device.sw_version` is `"03.59"`
- **AND** `device.identifiers` is `["miele_000101234567"]`
- **AND** `device.serial_number` is `"000101234567"`

### Requirement: Availability tied to bridge/miele state

Each discovery payload SHALL include an `availability` array referencing the existing Miele connection-state topic so HA marks the device unavailable only when the bridge is fully disconnected. Both `connected` and `degraded` SHALL count as "available":

```jsonc
"availability": [
  { "topic": "<mqtt.topic>/bridge/miele",
    "payload_available": "connected",
    "payload_not_available": "disconnected" },
  { "topic": "<mqtt.topic>/bridge/miele",
    "payload_available": "degraded",
    "payload_not_available": "disconnected" }
],
"availability_mode": "any"
```

#### Scenario: Degraded state keeps the device available

- **WHEN** the bridge publishes `degraded` to `<mqtt.topic>/bridge/miele`
- **THEN** HA's per-device availability stays `available` because at least one `availability` entry's `payload_available` matches

#### Scenario: Disconnected marks unavailable

- **WHEN** the bridge publishes `disconnected` to `<mqtt.topic>/bridge/miele`
- **THEN** every `availability` entry's `payload_not_available` matches and HA marks the device unavailable

### Requirement: Cleanup on graceful shutdown

The system SHALL keep an in-memory set of every discovery topic it has published during the current process lifetime, and on graceful shutdown SHALL publish an empty retained payload to each of those topics so HA removes the entities from its registry.

#### Scenario: Shutdown publishes empty payloads

- **WHEN** the bridge receives `SIGINT` or `SIGTERM` and runs its `stop` hook
- **THEN** for every `<topic>` it has previously published a discovery config to, the bridge publishes the empty byte string to `<topic>` with the retain flag set
- **AND** clears its in-memory set after the publishes complete

#### Scenario: Hard crash leaves retained payloads

- **WHEN** the bridge process exits without running its `stop` hook (kill -9, OOM, panic)
- **THEN** the discovery payloads remain in the broker
- **AND** the next bridge start either re-asserts them with up-to-date device metadata or leaves them intact, so HA never sees a hard "device removed then re-added" transition due to a crash

### Requirement: Unique-id and topic naming

The discovery topic SHALL be `<discovery_prefix>/sensor/miele_<id>/<entity>/config` and the `unique_id` SHALL be `miele_<id>_<entity>`, both using the same `<id>` rules as the device-registry section (fabNumber preferred, Miele API device id as fallback).

#### Scenario: Topic and unique_id derived consistently

- **WHEN** `<id>` resolves to `000101234567` and `<entity>` is `phase`
- **THEN** the discovery topic is `<prefix>/sensor/miele_000101234567/phase/config`
- **AND** the payload's `unique_id` is `miele_000101234567_phase`
- **AND** the payload's `object_id` is `miele_000101234567_phase`
