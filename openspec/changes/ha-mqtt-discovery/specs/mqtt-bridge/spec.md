## ADDED Requirements

### Requirement: Discovery publishes alongside device updates

When `mqtt.discovery.enabled` is `true`, the system SHALL publish Home Assistant MQTT-discovery config payloads at the same call site that publishes the small and full device messages, so HA observes the discovery config and the first data payload within the same broker round-trip.

#### Scenario: Discovery + small + full all publish for one update

- **WHEN** a device update arrives (from SSE or polling) and `mqtt.discovery.enabled` is `true`
- **THEN** the bridge publishes the small message to `<mqtt.topic>/<deviceId>`
- **AND** the full payload to `<mqtt.topic>/<deviceId>/full`
- **AND** one retained discovery config payload per entity to `<discovery_prefix>/sensor/miele_<id>/<entity>/config`

#### Scenario: Dedup applies to discovery topics

- **WHEN** `mqtt.deduplicate` is `true` and the discovery payload for a given topic is byte-identical to the previous publish on that topic during the current process lifetime
- **THEN** the discovery publish is suppressed
- **AND** the device's small and full publishes still apply their own dedup checks independently

### Requirement: Discovery cleanup runs in the bridge shutdown sequence

The system SHALL extend its existing graceful-shutdown step (the one that publishes a final `disconnected` to `<mqtt.topic>/bridge/miele`) to also publish empty retained payloads to every discovery topic the bridge has published during the run, before the MQTT client disconnects.

#### Scenario: Shutdown order

- **WHEN** the bridge runs its `stop` hook
- **THEN** the SSE/polling loops are stopped first (current behavior)
- **AND** the bridge publishes empty payloads to all tracked discovery topics
- **AND** the final `disconnected` is published to `<mqtt.topic>/bridge/miele`
- **AND** the MQTT client disconnects cleanly
