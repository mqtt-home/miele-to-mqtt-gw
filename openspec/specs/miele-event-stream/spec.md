# miele-event-stream Specification

## Purpose
TBD - created by archiving change rewrite-in-go. Update Purpose after archive.
## Requirements
### Requirement: SSE subscription to device updates

The system SHALL, when `miele.mode` is `sse`, open a long-lived HTTP request to the Miele Server-Sent Events endpoint with `Authorization: Bearer <access_token>` and `Accept: text/event-stream`, parse the `data:` lines into device events, and dispatch them to a registered device-update handler.

#### Scenario: Receive and dispatch an SSE event

- **WHEN** the SSE stream emits one or more `data:` lines terminated by a blank line
- **THEN** the system accumulates the `data:` lines, joins them with newlines, parses the result as JSON, and invokes the device-update handler once per event
- **AND** the handler receives one device entry per device, each with an `id` and a `data` payload of the same shape as the polling response

### Requirement: Automatic reconnect on disconnect

The system SHALL, on any SSE read error, EOF, or watchdog timeout, close the current response body and re-establish the SSE connection until the application is stopped.

#### Scenario: Stream disconnects unexpectedly

- **WHEN** the underlying HTTP body returns an error or EOF
- **THEN** the system closes the response, waits a short backoff (5 seconds), and reconnects with a fresh request using the current access token
- **AND** continues this loop until a stop signal is received

#### Scenario: Watchdog timeout closes the stream

- **WHEN** no event has been received within the configured watchdog interval (if enabled)
- **THEN** the system closes the SSE response so the reconnect loop establishes a new one

### Requirement: Restart on token-refresh boundary

The system SHALL close the SSE stream and re-open it with a fresh token whenever the periodic token-refresh decision concludes that a refresh is required.

#### Scenario: Token refresh forces SSE restart

- **WHEN** the token-refresh check determines a refresh is needed and the application is in `sse` mode
- **THEN** the system closes the current SSE connection
- **AND** logs that a token refresh is required and is reconnecting
- **AND** establishes a new SSE connection using the new access token

### Requirement: SSE bridge status reporting

The system SHALL report the SSE connection state to the MQTT layer via the values `connected` and `disconnected` (and `unknown` before the first attempt) so it can be published on `bridge/miele`.

#### Scenario: Connection state changes

- **WHEN** the SSE connection is successfully established
- **THEN** the system signals `connected` to the status callback
- **WHEN** the SSE connection drops or fails to establish
- **THEN** the system signals `disconnected` to the status callback

### Requirement: Graceful shutdown

The system SHALL stop the SSE reader, watchdog goroutine, and reconnect loop cleanly when the application receives a shutdown signal.

#### Scenario: Application stop

- **WHEN** the application's shutdown hook is invoked
- **THEN** the SSE client closes its current connection, stops its watchdog ticker, and exits its reconnect loop without leaking goroutines

