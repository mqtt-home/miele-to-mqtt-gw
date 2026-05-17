## ADDED Requirements

### Requirement: Exponential reconnect backoff after consecutive failures

The system SHALL track consecutive SSE connect failures and increase the delay between reconnect attempts according to a step table once a configurable failure threshold is reached. The streak counter SHALL increment on any of: HTTP request build failure, transport error, non-200 response status (including 504 Gateway Time-out), EOF on the response body before any `devices` event has been dispatched on that connection, and watchdog timeout. The streak counter SHALL reset to zero on the first `devices` event successfully dispatched after a (re)connect.

The default step table SHALL be `5s, 30s, 2m, 10m` with the index advancing by one each failure starting at the threshold and saturating at the last step. The base delay (used while the streak is below the threshold) SHALL default to `5s` and the maximum delay (cap of the step table) SHALL default to `10m`. The default failure threshold SHALL be `5`.

#### Scenario: First few failures use the base delay

- **WHEN** the SSE connect attempt fails for the first time and the consecutive-failure streak (post-increment) is less than the configured threshold
- **THEN** the system waits the base delay (default `5s`) before reconnecting
- **AND** does NOT signal the `degraded` status

#### Scenario: Backoff engages at the threshold

- **WHEN** the consecutive-failure streak reaches the configured threshold (default `5`)
- **THEN** the next reconnect delay is the first step beyond the base (default `30s`)
- **AND** the system signals `degraded` to the status callback if the parallel polling loop has had at least one successful poll since process start

#### Scenario: Backoff saturates at the maximum delay

- **WHEN** the consecutive-failure streak exceeds the number of steps in the table
- **THEN** the reconnect delay stays at the configured maximum (default `10m`) for all subsequent failures until a success resets the streak

#### Scenario: Success resets the streak

- **WHEN** a `devices` event is successfully dispatched after one or more failures
- **THEN** the consecutive-failure counter resets to `0`
- **AND** the next reconnect delay returns to the base delay
- **AND** the system signals `connected` to the status callback

#### Scenario: 504 Gateway Time-out counted as a failure

- **WHEN** the SSE endpoint returns HTTP 504 (or any non-200 response)
- **THEN** the streak counter increments
- **AND** the response body is closed before the next attempt

#### Scenario: Silent stream counted as a failure

- **WHEN** an SSE connection opens successfully but the response body ends (EOF or watchdog timeout) before any `devices` event has been dispatched
- **THEN** the streak counter increments
- **AND** the standard backoff selection applies for the next attempt

## MODIFIED Requirements

### Requirement: SSE bridge status reporting

The system SHALL report the SSE connection state to the MQTT layer via the values `connected`, `disconnected`, `degraded`, and `unknown` so it can be published on `bridge/miele`. `unknown` is the initial value before the first connection attempt. `connected` is reported after the body opens and is held until the connection ends. `disconnected` is reported when a connection ends or fails to establish while the failure streak is still below the configured backoff threshold. `degraded` is reported when the failure streak reaches the threshold and the parallel polling loop has had at least one successful poll since process start; it is held until SSE successfully dispatches an event again.

#### Scenario: Connection successfully established

- **WHEN** the SSE connection is established and the response body opens with status 200
- **THEN** the system signals `connected` to the status callback

#### Scenario: Brief disconnect below threshold

- **WHEN** an SSE connection drops or fails to establish and the consecutive-failure streak (post-increment) is still below the configured threshold
- **THEN** the system signals `disconnected` to the status callback

#### Scenario: Sustained failures with polling healthy

- **WHEN** the consecutive-failure streak reaches the configured threshold AND the polling loop has had at least one successful poll since process start
- **THEN** the system signals `degraded` to the status callback
- **AND** holds the `degraded` state across subsequent failed attempts until SSE recovers

#### Scenario: Sustained failures with polling also failing

- **WHEN** the consecutive-failure streak reaches the threshold AND polling has not yet had a successful poll since process start
- **THEN** the system continues to signal `disconnected` (NOT `degraded`)

#### Scenario: Recovery from degraded

- **WHEN** SSE successfully dispatches a `devices` event after a `degraded` period
- **THEN** the system signals `connected` to the status callback

### Requirement: Automatic reconnect on disconnect

The system SHALL, on any SSE read error, EOF, non-200 status, or watchdog timeout, close the current response body and re-establish the SSE connection until the application is stopped. The delay between attempts SHALL be selected per the exponential reconnect backoff requirement: base delay below the configured threshold, and the appropriate step from the backoff table at or beyond the threshold.

#### Scenario: Stream disconnects unexpectedly

- **WHEN** the underlying HTTP body returns an error or EOF
- **THEN** the system closes the response, waits the backoff-selected delay, and reconnects with a fresh request using the current access token
- **AND** continues this loop until a stop signal is received

#### Scenario: Watchdog timeout closes the stream

- **WHEN** no event has been received within the configured watchdog interval (if enabled)
- **THEN** the system closes the SSE response so the reconnect loop establishes a new one
- **AND** the watchdog-triggered closure counts toward the consecutive-failure streak when no event was dispatched on that connection
