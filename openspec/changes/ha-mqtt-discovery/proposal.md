## Why

Home Assistant ships a built-in [MQTT discovery](https://www.home-assistant.io/integrations/mqtt/#mqtt-discovery) layer: when an external bridge publishes retained config payloads on `<discovery_prefix>/<component>/<unique_id>/config`, HA auto-creates the matching device + entities without any user-side YAML. The bridge today already publishes Miele state as JSON on `<topic>/<deviceId>` (issue #136), but every user has to hand-roll an HA `mqtt.sensor` template per appliance. Adding discovery once in the bridge means HA users see each Miele appliance as a proper device tile with named entities (state, phase, remaining time, completion time) the moment the bridge connects.

The Miele full payload already carries the strings we need for the HA device registry (manufacturer "Miele", model from `ident.type.value_localized` or `ident.xkmIdentLabel.techType`, software version from `ident.xkmIdentLabel.releaseVersion`, serial from `ident.deviceIdentLabel.fabNumber`), so this is a publish-only change — no new upstream calls.

## What Changes

- New optional config block `mqtt.discovery` with fields `enabled` (bool, default `false` so existing users see no change), `prefix` (string, default `"homeassistant"` per the HA convention), and `device-name-prefix` (string, default `"Miele"` — prepended to entity display names).
- On every device update, publish (retained) HA discovery config payloads on `<prefix>/sensor/miele_<deviceId>/<entity>/config` for each entity the bridge knows how to populate:
  - `state` — Miele status (`RUNNING`, `OFF`, …) from the small message
  - `phase` — Miele program phase (`DRYING`, `MAIN_WASH`, …)
  - `remaining_duration` — `HH:MM` string
  - `remaining_minutes` — integer minutes (numeric sensor, `unit_of_measurement: min`, `device_class: duration`)
  - `time_completed` — wall-clock string
- Each discovery payload's `state_topic` points at the existing `<topic>/<deviceId>` JSON message and uses `value_template` to extract the relevant field, so the data plane stays unchanged.
- An `availability_topic` of `<topic>/bridge/miele` is included with `payload_available: connected` and `payload_available: degraded` (HA accepts multiple `availability` entries; falling back to `payload_not_available: disconnected`). Discovery republishes on every device update — same retained-payload semantics already used by the existing small/full topics, so HA sees idempotent re-announcements.
- A bridge-wide retained "going away" mechanism: on graceful shutdown publish empty payloads to every previously-announced discovery topic so HA removes the entities. (HA's documented removal protocol.) This is cheap and prevents stale entities lingering after a bridge uninstall.
- README updates documenting the new config block, the produced topics, the unique-id scheme (`miele_<fabNumber>_<entity>`), and how to disable discovery.

## Capabilities

### New Capabilities

- `ha-discovery`: publishing Home Assistant MQTT-discovery config payloads for each Miele device the bridge sees, including the device-registry metadata, per-entity state topics with value templates, and clean removal on shutdown.

### Modified Capabilities

- `app-config`: adds the optional `mqtt.discovery` block with defaults documented in `app-config/spec.md`.
- `mqtt-bridge`: defines the discovery-publish path that runs alongside the existing small/full publishes and the new shutdown cleanup step.

## Impact

- **Affected code**: new `app/bridge/discovery.go` (payload assembly + topic naming), `app/bridge/publisher.go` (call into the discovery layer from `PublishDevice` and `stop`), `app/config/config.go` (new optional config struct + defaults), `app/app.go` (call into a cleanup hook on `stop`).
- **Behavior on existing configs**: zero change. `discovery.enabled` defaults to `false`; users have to opt in.
- **MQTT contract**: the existing `<topic>/<deviceId>` (small) and `<topic>/<deviceId>/full` topics are unchanged. New retained topics appear only when discovery is enabled, all under the configurable `mqtt.discovery.prefix`.
- **Memory/CPU**: discovery payload is built once per device update and dropped through the existing dedup layer; payload size is on the order of 1–2 KB per entity. Negligible.
- **No new external dependencies.**
