## ADDED Requirements

### Requirement: Single `miele` expvar surface

The system SHALL publish exactly one top-level `expvar` named `miele` reachable at `GET /debug/vars` on the pprof listener. The value SHALL be a JSON object containing the keys `connection`, `devices`, `sse`, `polling`, and `token` as defined by the requirements below.

#### Scenario: expvar visible alongside mqtt

- **WHEN** the application is running and an HTTP client requests `/debug/vars`
- **THEN** the JSON response contains a top-level key `"miele"` whose value is a JSON object
- **AND** the response also contains the existing `"mqtt"` key from `mqtt-gateway` (the new surface MUST NOT replace or shadow other expvars)

### Requirement: Miele connection state

The `miele.connection` field SHALL hold the most recently observed Miele connection state, mirroring the value published on the `bridge/miele` MQTT topic: `"unknown"` before any state transition, then `"connected"` or `"disconnected"`.

#### Scenario: Initial value

- **WHEN** the application has started but has not yet observed an SSE or polling result
- **THEN** `miele.connection` is `"unknown"`

#### Scenario: SSE connects

- **WHEN** the SSE client emits `connected` via its status callback
- **THEN** `miele.connection` becomes `"connected"`

#### Scenario: SSE disconnects

- **WHEN** the SSE client emits `disconnected`
- **THEN** `miele.connection` becomes `"disconnected"`

### Requirement: Per-device snapshot

The `miele.devices` field SHALL be a JSON object keyed by device id, whose values are the most recent small-message JSON shape for that device (`phase`, `phaseId`, `state`, `remainingDuration`, `remainingDurationMinutes`, `timeCompleted`).

#### Scenario: Device update updates snapshot

- **WHEN** a device update is processed (from SSE or polling) and the small message is computed for device `<id>`
- **THEN** `miele.devices["<id>"]` equals that small-message JSON
- **AND** subsequent updates for the same `<id>` replace the entry in place

#### Scenario: Devices accumulate across the run

- **WHEN** updates for multiple device ids have been processed
- **THEN** `miele.devices` contains one entry per id seen since process start
- **AND** entries are never removed during the lifetime of the process

### Requirement: SSE metrics

The `miele.sse` field SHALL be a JSON object with at least the keys `last_event` (RFC3339 timestamp string, zero-valued when no event has been received) and `events_total` (integer count of dispatched events since process start).

#### Scenario: Counter increments per dispatch

- **WHEN** the SSE client dispatches a `devices` event to the application
- **THEN** `miele.sse.events_total` increases by 1
- **AND** `miele.sse.last_event` is updated to the dispatch time

#### Scenario: No events yet

- **WHEN** the process has started but no SSE event has been dispatched
- **THEN** `miele.sse.events_total` is `0`
- **AND** `miele.sse.last_event` is the zero-valued RFC3339 timestamp

### Requirement: Polling metrics

The `miele.polling` field SHALL be a JSON object with the keys `last_attempt`, `last_success`, `last_error` (string, empty when no error), `success_total`, and `error_total`.

#### Scenario: Successful poll

- **WHEN** a polling iteration calls `FetchDevices` and the call returns without error
- **THEN** `miele.polling.last_attempt` and `miele.polling.last_success` are updated
- **AND** `miele.polling.success_total` increases by 1
- **AND** `miele.polling.last_error` is unchanged

#### Scenario: Failed poll

- **WHEN** a polling iteration calls `FetchDevices` and the call returns an error
- **THEN** `miele.polling.last_attempt` is updated
- **AND** `miele.polling.last_error` is set to the error string
- **AND** `miele.polling.error_total` increases by 1
- **AND** `miele.polling.last_success` is unchanged

#### Scenario: Polling skipped (no token)

- **WHEN** the polling iteration runs but no access token is available
- **THEN** none of the polling counters are incremented (the skip is logged but not counted)

### Requirement: Token metrics

The `miele.token` field SHALL be a JSON object with the keys `expires_at`, `last_refresh` (both RFC3339), and `refresh_total` (integer).

#### Scenario: Successful login/refresh

- **WHEN** `Login` completes successfully and returns a token
- **THEN** `miele.token.expires_at` equals the returned token's `expiresAt`
- **AND** `miele.token.last_refresh` is set to the current time
- **AND** `miele.token.refresh_total` increases by 1

### Requirement: Concurrency safety

The metrics surface SHALL be safe for concurrent updates from the SSE, polling, and token-refresh goroutines, and concurrent reads from the `expvar.Func` invocation triggered by HTTP requests.

#### Scenario: Race-free under load

- **WHEN** SSE events, polling iterations, and token refreshes update metrics concurrently and a separate goroutine repeatedly reads `/debug/vars`
- **THEN** the program runs without data-race detector failures (`go test -race`)

### Requirement: No new external surface or dependency

The change SHALL NOT introduce new MQTT topics, new config-file keys, or new Go module dependencies. It SHALL reuse the existing `expvar` package and the existing `:6060` listener already exposed for pprof.

#### Scenario: Config remains unchanged

- **WHEN** a user provides a config from before this change
- **THEN** the application starts unchanged
- **AND** the `miele` expvar is published with default zero values until the first event arrives
