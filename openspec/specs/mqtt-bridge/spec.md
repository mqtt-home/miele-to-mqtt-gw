# mqtt-bridge Specification

## Purpose
TBD - created by archiving change rewrite-in-go. Update Purpose after archive.
## Requirements
### Requirement: MQTT connection lifecycle

The system SHALL connect to the configured MQTT broker on startup, register a last-will message marking the bridge as offline, and disconnect cleanly on shutdown.

#### Scenario: Successful connect

- **WHEN** the application starts with a valid `mqtt.url`, optional `mqtt.username`/`password`, and unique `mqtt.client-id`
- **THEN** the MQTT client connects to the broker
- **AND** subscribes to `<mqtt.topic>/#` so it can observe its own topic tree
- **AND** publishes `online` to the bridge state topic (see "Bridge status reporting")

#### Scenario: Connection lost and recovered

- **WHEN** the broker connection is lost
- **THEN** the underlying MQTT client attempts to reconnect on its own
- **AND** the broker delivers the retained last-will `offline` until the bridge reconnects and republishes `online`

### Requirement: Topic layout

The system SHALL publish device messages under the configured `mqtt.topic` prefix using exactly these topics:

- `<mqtt.topic>/<deviceId>` — the small message
- `<mqtt.topic>/<deviceId>/full` — the unmodified raw device payload
- `<mqtt.topic>/bridge/state` (or `mqtt.bridge-info-topic` when configured) — the bridge online/offline state
- `<mqtt.topic>/bridge/miele` — the Miele connection state

#### Scenario: Device update produces both topics

- **WHEN** a device update is received from either SSE or polling
- **THEN** the system publishes the small-message JSON to `<mqtt.topic>/<deviceId>`
- **AND** publishes the raw `data` JSON to `<mqtt.topic>/<deviceId>/full`

### Requirement: Retained publishes

The system SHALL publish all device and bridge-status messages with the MQTT retain flag set to the value of `mqtt.retain` (default `true`), and with the configured `mqtt.qos` (default `1`).

#### Scenario: Retain flag honored

- **WHEN** `mqtt.retain` is `true`
- **THEN** every device publish and every bridge-status publish sets the retain flag on the broker
- **WHEN** `mqtt.retain` is `false`
- **THEN** no publish sets the retain flag

### Requirement: Bridge status reporting

The system SHALL publish, when `mqtt.bridge-info` is `true` (the default):

- `online` to the bridge state topic immediately after the MQTT connection is established
- `offline` as the last-will, delivered automatically by the broker if the bridge disconnects without a clean stop
- `unknown` | `connected` | `disconnected` to `<mqtt.topic>/bridge/miele` reflecting the Miele connection state

#### Scenario: Bridge starts up

- **WHEN** the bridge has just connected to MQTT
- **THEN** it publishes `online` (retained) to the bridge state topic

#### Scenario: Miele state transitions

- **WHEN** the Miele connection becomes available (SSE connected, or first successful poll)
- **THEN** the bridge publishes `connected` to `<mqtt.topic>/bridge/miele`
- **WHEN** the Miele connection is lost (SSE disconnect, repeated poll failures, or auth failure)
- **THEN** the bridge publishes `disconnected` to `<mqtt.topic>/bridge/miele`

#### Scenario: Bridge info disabled

- **WHEN** `mqtt.bridge-info` is `false`
- **THEN** no bridge-state or last-will message is configured

### Requirement: Deduplication of identical payloads

The system SHALL, when `mqtt.deduplicate` is `true`, suppress publishing a device message whose serialized payload is identical to the most recently published payload on the same topic during the current process lifetime.

#### Scenario: Identical consecutive payloads suppressed

- **WHEN** `mqtt.deduplicate` is `true` and a device update would produce the same JSON bytes as the previously published value for that topic
- **THEN** the system skips the publish

#### Scenario: Dedup resets on restart

- **WHEN** the process restarts
- **THEN** the in-memory dedup cache is empty, so the first publish for each topic after restart is always sent

