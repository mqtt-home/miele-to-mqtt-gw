# miele-polling Specification

## Purpose
TBD - created by archiving change rewrite-in-go. Update Purpose after archive.
## Requirements
### Requirement: Periodic polling of devices

The system SHALL periodically fetch the device list from the Miele REST API and emit each device through the same device-update path used by SSE, so polling and SSE consumers are interchangeable.

#### Scenario: Polling tick fetches and publishes

- **WHEN** the polling timer fires and the current access token is available
- **THEN** the system calls the devices endpoint
- **AND** for each returned device invokes the device-update handler with `id` and `data` set the same way as an SSE-delivered event

#### Scenario: Polling without a token is skipped

- **WHEN** the polling timer fires but no access token is available yet
- **THEN** the system logs a warning that polling was skipped and does not call the devices endpoint

### Requirement: Polling as fallback while SSE is active

The system SHALL run the polling loop even when `miele.mode` is `sse`, so that polling can act as a fallback if SSE misses an event, matching the existing behavior introduced in commit `7620b8e`.

#### Scenario: SSE mode still runs the polling loop

- **WHEN** `miele.mode` is `sse`
- **THEN** the polling loop runs at its configured interval and publishes any state diffs alongside SSE-driven updates
- **AND** the deduplication layer ensures redundant identical payloads are not sent twice

#### Scenario: Polling-only mode

- **WHEN** `miele.mode` is `polling`
- **THEN** SSE is not started and the polling loop is the sole source of device updates

### Requirement: Configurable polling interval

The system SHALL respect `miele.polling-interval` (in seconds) for the polling tick frequency, falling back to the documented default when unset.

#### Scenario: Custom polling interval

- **WHEN** `miele.polling-interval` is set to `30`
- **THEN** the polling timer fires approximately every 30 seconds

