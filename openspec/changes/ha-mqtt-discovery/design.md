## Context

The bridge already publishes one retained MQTT message per device update on `<topic>/<deviceId>` (parsed small JSON) and `<topic>/<deviceId>/full` (raw Miele payload). Bridge connection state lives on `<topic>/bridge/miele` (`unknown` / `connected` / `disconnected` / `degraded`). The Miele "full" payload includes the descriptive metadata HA's device registry needs:

- `ident.deviceIdentLabel.fabNumber` — the serial number (unique per appliance)
- `ident.type.value_localized` — appliance type as a string (e.g. "Dishwasher")
- `ident.xkmIdentLabel.techType` — model code (e.g. "G7560")
- `ident.xkmIdentLabel.releaseVersion` — firmware version

Issue #136 asks the bridge to publish HA's [MQTT-discovery](https://www.home-assistant.io/integrations/mqtt/#mqtt-discovery) config payloads so HA auto-creates a device tile and named entities per appliance, without each user writing template sensors by hand.

## Goals / Non-Goals

**Goals:**

- One-time opt-in (`mqtt.discovery.enabled = true`) auto-publishes HA discovery config for each appliance the bridge sees, surfacing the entities a typical user wants: state, phase, remaining duration (string and minutes), time completed.
- HA's device registry shows manufacturer "Miele", the appliance type ("Dishwasher", "Dryer", …) as the model line, the firmware version, and the appliance's serial number as `identifiers`.
- Availability is wired to the existing `bridge/miele` topic so HA marks the entire device "unavailable" when SSE+polling are both down.
- Clean removal: when the bridge stops, the retained config payloads it owns are cleared so HA removes the device from the registry.
- Zero change to the existing small/full topics. Discovery is layered *on top of* them via `value_template`.

**Non-Goals:**

- Per-entity customization at runtime. The set of published entities is fixed in the bridge; advanced users who want more can keep using the raw `<topic>/<deviceId>/full` topic and write their own templates.
- Actionable entities (switches/buttons). The Miele bridge today is read-only; a future change could add command topics, but that is out of scope here.
- Per-appliance-type entity sets. Issue #136's commenter pointed out that not all entities make sense for every type, but in practice all five small-message fields *are* meaningful for any running appliance (an oven still reports `state`, `phase`, etc., even if `remaining_duration` is uninteresting when off). We publish the same five for every appliance to keep the code simple; HA users can hide unwanted entities per-device in their dashboards.
- Discovery for the bridge itself (e.g. a connectivity sensor for `bridge/miele`). The availability topic already covers the meaningful surface for HA users.

## Decisions

### Discovery payload shape

Each discovery payload is a JSON object with the standard HA fields:

```jsonc
{
  "name": "State",
  "unique_id": "miele_<fabNumber>_state",
  "object_id":  "miele_<fabNumber>_state",
  "state_topic": "<topic>/<deviceId>",
  "value_template": "{{ value_json.state }}",
  "availability": [
    { "topic": "<topic>/bridge/miele",
      "payload_available": "connected",
      "payload_not_available": "disconnected" },
    { "topic": "<topic>/bridge/miele",
      "payload_available": "degraded",
      "payload_not_available": "disconnected" }
  ],
  "availability_mode": "any",
  "device": {
    "identifiers":   ["miele_<fabNumber>"],
    "manufacturer":  "Miele",
    "model":         "<type.value_localized> (<techType>)",
    "name":          "<configured-prefix> <type.value_localized> <fabNumber>",
    "sw_version":    "<releaseVersion>",
    "serial_number": "<fabNumber>"
  }
}
```

For the numeric `remaining_minutes` entity we additionally set `unit_of_measurement: "min"` and `device_class: "duration"`. The other entities are plain string sensors with no unit.

*Why two availability entries with `availability_mode: any`:* the bridge's `bridge/miele` topic publishes either `connected` *or* `degraded` while polling is keeping device updates flowing. HA's single-availability mode would mark the device unavailable on `degraded` — but `degraded` actually means "data is still arriving via polling, just slower." Listing both as `payload_available` with `mode: any` keeps the device "available" in either state, only flipping unavailable on a true `disconnected`.

### Entity set published per device

A fixed list of five entities, all keyed off fields in the small JSON message:

| Entity                  | `value_template`                                | Notes                              |
| ----------------------- | ----------------------------------------------- | ---------------------------------- |
| `state`                 | `{{ value_json.state }}`                        | Miele status enum string           |
| `phase`                 | `{{ value_json.phase }}`                        | Program phase string               |
| `remaining_duration`    | `{{ value_json.remainingDuration }}`            | `HH:MM` string                     |
| `remaining_minutes`     | `{{ value_json.remainingDurationMinutes }}`     | int, `unit: min`, `class: duration`|
| `time_completed`        | `{{ value_json.timeCompleted }}`                | Wall-clock `HH:MM`                 |

All five are `sensor` components (HA's most permissive type for free-form values). We deliberately do not use `binary_sensor` for "is running" because the bridge does not expose a single boolean today; downstream users can derive one from the `state` sensor via a template if they want it.

### unique_id scheme

`miele_<fabNumber>_<entity>`. The fabNumber comes from `ident.deviceIdentLabel.fabNumber` which is the Miele serial number, globally unique per appliance. Using it (rather than the Miele API device ID, which is a hashed value) gives HA a stable identifier that survives a bridge reinstall and tracks the physical device. If `fabNumber` is missing for some reason, we fall back to the Miele API device ID — at the cost of HA treating a fresh install as a new device — and emit a warning log so the operator notices.

### Topic naming

`<discovery_prefix>/sensor/miele_<fabNumber>/<entity>/config`.

*Alternatives considered:*

- `<prefix>/sensor/miele/<fabNumber>_<entity>/config` (one HA "node" per bridge): rejected because some HA UI views still treat the second path segment as a grouping key, and grouping by appliance reads better in HA's MQTT device list.
- `<prefix>/sensor/<deviceId>/<entity>/config` (no `miele_` prefix): rejected because the bare Miele device ID is opaque and could collide with other integrations using the same discovery prefix.

### Discovery republish cadence

Every device update republishes the discovery config (retained). This is wasteful in steady state but keeps the code dead-simple and matches what other bridges in the same ecosystem do (Zigbee2MQTT, etc.). The dedup layer already in `publisher.publishWithDedup` will suppress identical retained re-publishes when `mqtt.deduplicate` is on, so users who care about MQTT broker noise can enable it.

*Alternatives considered:*

- Republish only once per session, on first sight of each device. Rejected: if HA starts after the bridge, it would never see the config payload — even though retained MQTT solves that case for *new* devices, this would still miss the scenario where a user clears retained state on the broker.
- Republish only when the device-registry fields (`model`, `sw_version`, etc.) change. Workable but adds bookkeeping; the dedup layer gives us the same effect for free.

### Cleanup on shutdown

The bridge tracks the set of discovery topics it has ever published in a `map[string]struct{}` on the Publisher. On `stop`, it iterates that set and publishes an empty (retained) payload to each, which is HA's documented removal protocol. We do NOT clear discovery on each device disappearing from the Miele account — devices only "disappear" when the operator removes them from miele@home, and we don't want a transient API hiccup to nuke the HA registry entry. Cleanup is an explicit shutdown step.

*Alternatives considered:*

- LWT-based removal: would require setting an extra MQTT will per discovery topic, which the underlying mqtt-gateway does not support out of the box. Skipped.

### Config defaults

```jsonc
{
  "mqtt": {
    "discovery": {
      "enabled": false,
      "prefix": "homeassistant",
      "device-name-prefix": "Miele"
    }
  }
}
```

`enabled: false` is the safe default — turning the feature on adds new retained topics to the user's broker, and we want that to be an explicit opt-in. Once enabled, the feature is fully automatic: no per-device config needed.

## Risks / Trade-offs

- **Stale retained config payloads survive a bridge crash.** Mitigation: the cleanup runs only on graceful shutdown. A hard crash leaves the entities until either (a) the bridge restarts and re-publishes (current state remains correct) or (b) the user clears them manually. This is the same trade-off every other MQTT-discovery bridge makes.
- **`fabNumber` missing in some payloads.** Mitigation: fall back to the Miele device ID and log a warning; HA still gets a stable per-process identifier, just not one that survives a Miele-side ID change. Confirmed by the example payload in `fullmessage-example.md` that fabNumber is present on the expected appliance types.
- **HA users on older HA versions (pre 2023.8) won't honor `availability_mode: any`.** Mitigation: the simpler single-availability form (`connected` only) still works there — HA will just mark the device unavailable on `degraded`, which is the pre-issue-#45 behavior anyway. Documented in README.
- **Discovery republish per update bloats broker traffic.** Mitigation: documented use of `mqtt.deduplicate: true` to suppress identical retained re-publishes; the payloads don't change unless the appliance's model/firmware does.

## Migration Plan

1. Ship the change with `discovery.enabled: false`. Existing users see no change.
2. README adds a "Home Assistant integration" section with the four-line opt-in config snippet and a screenshot of the resulting HA device tile (optional, follow-up).
3. Rollback: setting `discovery.enabled: false` and restarting cleans up via the shutdown step; users can also retain-clear the discovery topics manually if needed.

## Open Questions

- Should we expose the `state_class` field on `remaining_minutes` (e.g. `measurement`) so HA's history graphs render it as a line chart? **Decision:** Yes — `state_class: "measurement"` is appropriate for a value that varies smoothly over time and matches HA conventions for duration sensors. Adds one key, no downside.
- Should we publish a binary "is running" sensor derived from `state == RUNNING`? **Decision:** No, keep it as a non-goal for this change. Users who want it can build a template binary_sensor on top of the `state` sensor; baking it in would commit us to a particular interpretation of which Miele statuses count as "running" (does `PAUSE` count? `PROGRAMMED_WAITING_TO_START`?), and that question is better deferred until users ask.
