# app-config Specification

## Purpose
TBD - created by archiving change rewrite-in-go. Update Purpose after archive.
## Requirements
### Requirement: JSON config file loaded by path

The system SHALL accept a path to a JSON config file as its single command-line argument and load it at startup; any other argument count SHALL cause the application to log an error and exit non-zero.

#### Scenario: Valid config path

- **WHEN** the binary is invoked with exactly one argument pointing to an existing readable JSON file
- **THEN** the system reads the file, applies env-var substitution and defaults, parses it into the typed config, and proceeds with startup

#### Scenario: Missing config argument

- **WHEN** the binary is invoked with zero or more than one argument
- **THEN** the system logs `Expected config file as argument.` and exits with a non-zero status

### Requirement: Environment-variable substitution

The system SHALL substitute occurrences of `${NAME}` in the raw config-file contents with the value of the environment variable `NAME` before JSON parsing. A missing environment variable SHALL be substituted with the empty string.

#### Scenario: Substitution succeeds

- **WHEN** the config file contains `"username": "${MIELE_USERNAME}"` and `MIELE_USERNAME=alice` is in the environment
- **THEN** the parsed config has `miele.username = "alice"`

#### Scenario: Missing env var substitutes empty

- **WHEN** the config file contains `"username": "${MIELE_USERNAME}"` and `MIELE_USERNAME` is not set
- **THEN** the parsed config has `miele.username = ""`

### Requirement: Defaults applied to optional fields

The system SHALL apply the same defaults as the existing TypeScript implementation:

- `mqtt.qos`: `1`
- `mqtt.retain`: `true`
- `mqtt.bridge-info`: `true`
- `miele.mode`: `"sse"`
- `miele.country-code`: `"de-DE"`
- `miele.connection-check-interval`: `10000` (ms)
- `miele.persistToken`: `true`
- `send-full-update`: `true`
- `loglevel`: `"info"`

#### Scenario: Unset fields receive defaults

- **WHEN** a config omits `mqtt.qos`
- **THEN** the loaded config exposes `mqtt.qos = 1`

#### Scenario: Explicit values override defaults

- **WHEN** a config sets `miele.mode` to `"polling"`
- **THEN** the loaded config exposes `miele.mode = "polling"` and SSE is not started

### Requirement: Config schema parity with TypeScript version

The system SHALL accept exactly the config keys understood by the existing TypeScript implementation, including:

- `mqtt`: `url`, `topic`, `username`, `password`, `client-id`, `retain`, `qos`, `bridge-info`, `bridge-info-topic`, `deduplicate`
- `miele`: `client-id`, `client-secret`, `country-code`, `username`, `password`, `mode`, `polling-interval`, `token` (with `access`, `refresh`, `validUntil`), `connection-check-interval`, `persistToken`
- top-level: `names`, `send-full-update`, `loglevel`

The system SHALL NOT introduce new top-level keys or rename existing ones in this change.

#### Scenario: Existing user config loads unchanged

- **WHEN** a user provides a `config.json` produced for the TypeScript 3.x release
- **THEN** the system loads it without requiring edits
- **AND** all documented runtime behaviors (mode, retain, dedup, persist) match what the TypeScript version did with the same file

### Requirement: Token recovery from config

The system SHALL, when `miele.token.access` and `miele.token.refresh` are present at load time, install them as the current token before any login attempt, deriving `expiresAt` from `miele.token.validUntil` if present and otherwise defaulting to one hour after load time.

#### Scenario: Stored token used at startup

- **WHEN** the loaded config has `miele.token` populated with valid `access`, `refresh`, and `validUntil`
- **THEN** the system uses that token for the first API call without performing a username/password login

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

