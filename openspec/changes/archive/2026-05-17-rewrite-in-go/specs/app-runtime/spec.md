## ADDED Requirements

### Requirement: Application bootstrap and shutdown

The system SHALL, on startup, in this order: (1) load and validate the config, (2) connect to MQTT and publish `online` to the bridge state topic, (3) acquire a Miele access token (login or recover-from-config), (4) start SSE (if `mode: sse`), (5) start the polling loop, (6) start the periodic token-refresh check, (7) start the pprof listener if enabled. On shutdown the system SHALL stop these components in reverse order and exit cleanly.

#### Scenario: Clean startup

- **WHEN** all dependencies (config valid, MQTT reachable, Miele reachable) are available
- **THEN** the application reaches a "ready" state and logs `Application is now ready.`

#### Scenario: Startup failure exits non-zero

- **WHEN** initial login fails, MQTT connect fails, or another startup step throws
- **THEN** the system logs the failure and exits with a non-zero status

#### Scenario: Graceful shutdown on SIGINT/SIGTERM

- **WHEN** the process receives `SIGINT` or `SIGTERM`
- **THEN** the system stops the polling and token-refresh tickers, closes the SSE connection, disconnects from MQTT (which delivers the retained `offline` state via last-will or an explicit publish), and exits

### Requirement: Periodic token-refresh check

The system SHALL run a periodic task (every minute, matching the existing TypeScript `* * * * *` cron) that checks whether a token refresh is needed and, if so, restarts the Miele connection with a refreshed token.

#### Scenario: Periodic check triggers refresh

- **WHEN** the periodic check finds the current token expires within 24 hours
- **THEN** the system performs a token refresh and restarts the SSE connection

#### Scenario: No refresh needed

- **WHEN** the periodic check finds the token still valid for more than 24 hours
- **THEN** the system does nothing and waits for the next tick

### Requirement: Structured logging

The system SHALL use the shared `go-logger` for all log output and respect the `loglevel` from config (`fatal`, `error`, `warn`, `info`, `debug`, `trace`).

#### Scenario: Log level filters output

- **WHEN** `loglevel` is `info`
- **THEN** `debug` and `trace` messages are suppressed
- **AND** `info`, `warn`, `error`, and `fatal` messages are emitted

### Requirement: Optional pprof diagnostic endpoint

The system SHALL, when pprof is enabled in config, expose `net/http/pprof` on the configured port (bound to localhost by default) for live diagnostics, matching the posture used by `hue-to-mqtt-gw`.

#### Scenario: Pprof enabled

- **WHEN** the config enables pprof with a port
- **THEN** the system serves `/debug/pprof/*` on that port for the lifetime of the process

#### Scenario: Pprof disabled

- **WHEN** the config does not enable pprof
- **THEN** the system does not bind any HTTP listener
