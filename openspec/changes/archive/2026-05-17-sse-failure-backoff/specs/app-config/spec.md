## ADDED Requirements

### Requirement: SSE backoff configuration block

The system SHALL accept an optional `miele.sse-backoff` object in the JSON config with three optional fields: `failure-threshold` (integer, consecutive failures before backoff engages, default `5`), `base-delay` (Go duration string, delay while below threshold, default `"5s"`), and `max-delay` (Go duration string, ceiling for the step-table delay, default `"10m"`). Duration fields SHALL be parsed via `time.ParseDuration`. Unset or zero-valued fields SHALL receive their documented defaults.

#### Scenario: Defaults applied when block is absent

- **WHEN** a config omits the `miele.sse-backoff` block entirely
- **THEN** the loaded config behaves as if `failure-threshold=5`, `base-delay=5s`, `max-delay=10m`

#### Scenario: Partial overrides

- **WHEN** the config sets `miele.sse-backoff.failure-threshold` to `10` and leaves the other two fields unset
- **THEN** the loaded config exposes `failure-threshold=10`, `base-delay=5s`, `max-delay=10m`

#### Scenario: Custom durations parsed

- **WHEN** the config sets `miele.sse-backoff.base-delay` to `"2s"` and `miele.sse-backoff.max-delay` to `"5m"`
- **THEN** the parsed config exposes those durations
- **AND** the step table is recomputed so the last step equals the configured max-delay

#### Scenario: Invalid duration rejected at load time

- **WHEN** the config sets `miele.sse-backoff.base-delay` to a value that `time.ParseDuration` cannot parse (e.g. `"five seconds"`)
- **THEN** `LoadConfig` returns an error describing the offending field
- **AND** the application exits non-zero (per the existing config-load failure path)

### Requirement: Backwards compatibility for configs without the backoff block

The system SHALL load configs written before this change without requiring edits. The new `miele.sse-backoff` block SHALL be optional in the JSON schema sense — its absence SHALL NOT cause a validation error and SHALL produce identical behavior to a block whose fields all use the documented defaults.

#### Scenario: Pre-existing config loads unchanged

- **WHEN** a user provides a `config.json` produced before this change
- **THEN** the system loads it without requiring edits
- **AND** SSE reconnects use the default backoff parameters
