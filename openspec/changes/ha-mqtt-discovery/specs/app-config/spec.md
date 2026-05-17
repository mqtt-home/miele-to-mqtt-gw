## ADDED Requirements

### Requirement: MQTT discovery configuration block

The system SHALL accept an optional `mqtt.discovery` object in the JSON config with three optional fields: `enabled` (bool, default `false`), `prefix` (string, default `"homeassistant"`), and `device-name-prefix` (string, default `"Miele"`). Unset fields SHALL receive their documented defaults.

#### Scenario: Defaults applied when block absent

- **WHEN** a config omits the `mqtt.discovery` block entirely
- **THEN** the loaded config behaves as if `enabled = false`, `prefix = "homeassistant"`, `device-name-prefix = "Miele"`
- **AND** no Home Assistant discovery topics are published

#### Scenario: Enable with defaults

- **WHEN** the config sets `mqtt.discovery.enabled = true` and leaves the other two fields unset
- **THEN** the loaded config exposes `prefix = "homeassistant"` and `device-name-prefix = "Miele"`

#### Scenario: Custom prefix and device-name prefix

- **WHEN** the config sets `mqtt.discovery.prefix = "ha"` and `mqtt.discovery.device-name-prefix = "Kitchen"`
- **THEN** the loaded config exposes those values
- **AND** all discovery topics are published under `ha/sensor/.../config`
- **AND** every device's HA `name` field starts with `"Kitchen "`

### Requirement: Backwards compatibility for configs without the discovery block

The system SHALL load configs written before this change without requiring edits. The new `mqtt.discovery` block SHALL be optional in the JSON schema sense — its absence SHALL NOT cause a validation error and SHALL produce identical behavior to a block whose fields all use the documented defaults.

#### Scenario: Pre-existing config loads unchanged

- **WHEN** a user provides a `config.json` produced before this change
- **THEN** the system loads it without requiring edits
- **AND** discovery is disabled (matching the documented default)
