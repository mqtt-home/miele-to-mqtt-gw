## MODIFIED Requirements

### Requirement: Miele connection state

The `miele.connection` field SHALL hold the most recently observed Miele connection state, mirroring the value published on the `bridge/miele` MQTT topic: `"unknown"` before any state transition, then one of `"connected"`, `"disconnected"`, or `"degraded"`.

#### Scenario: Initial value

- **WHEN** the application has started but has not yet observed an SSE or polling result
- **THEN** `miele.connection` is `"unknown"`

#### Scenario: SSE connects

- **WHEN** the SSE client emits `connected` via its status callback
- **THEN** `miele.connection` becomes `"connected"`

#### Scenario: SSE disconnects

- **WHEN** the SSE client emits `disconnected`
- **THEN** `miele.connection` becomes `"disconnected"`

#### Scenario: SSE enters degraded mode

- **WHEN** the SSE client emits `degraded` via its status callback (failure streak crossed the backoff threshold while polling is healthy)
- **THEN** `miele.connection` becomes `"degraded"`

### Requirement: SSE metrics

The `miele.sse` field SHALL be a JSON object with at least the keys `last_event` (RFC3339 timestamp string, zero-valued when no event has been received), `events_total` (integer count of dispatched events since process start), `consecutive_failures` (integer count of the current connect-failure streak; reset to `0` on a dispatched event), and `next_retry_after` (RFC3339 timestamp string indicating when the next reconnect attempt is scheduled; empty string when not currently in a wait).

#### Scenario: Counter increments per dispatch

- **WHEN** the SSE client dispatches a `devices` event to the application
- **THEN** `miele.sse.events_total` increases by 1
- **AND** `miele.sse.last_event` is updated to the dispatch time
- **AND** `miele.sse.consecutive_failures` is reset to `0`

#### Scenario: No events yet

- **WHEN** the process has started but no SSE event has been dispatched
- **THEN** `miele.sse.events_total` is `0`
- **AND** `miele.sse.last_event` is the zero-valued RFC3339 timestamp
- **AND** `miele.sse.consecutive_failures` is `0`
- **AND** `miele.sse.next_retry_after` is the empty string

#### Scenario: Failure streak reflected in expvar

- **WHEN** three SSE connect attempts fail in a row without an event dispatch in between
- **THEN** `miele.sse.consecutive_failures` is `3`
- **AND** `miele.sse.next_retry_after` is the RFC3339 timestamp at which the next reconnect will fire

#### Scenario: Streak resets on success

- **WHEN** a `devices` event is dispatched after a streak of failures
- **THEN** `miele.sse.consecutive_failures` is `0`
- **AND** `miele.sse.next_retry_after` is the empty string
