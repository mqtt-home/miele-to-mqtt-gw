## ADDED Requirements

### Requirement: Small-message transformation

The system SHALL transform a raw Miele device payload into a "small message" with exactly the fields documented in `README.md`: `phase`, `phaseId`, `state`, `remainingDurationMinutes`, `remainingDuration`, `timeCompleted`. The output JSON SHALL be byte-equivalent to the output of the existing TypeScript `smallMessage` function for the same input payload (modulo the wall-clock value embedded in `timeCompleted`).

#### Scenario: Standard running device

- **WHEN** the raw payload contains `state.programPhase.value_raw`, `state.status.value_raw`, and `state.remainingTime` (an `[hours, minutes]` pair)
- **THEN** the small message contains:
  - `phase` set to the human-readable name of the phase enum for `value_raw`
  - `phaseId` set to that numeric `value_raw`
  - `state` set to the human-readable name of the device-status enum for `state.status.value_raw`
  - `remainingDurationMinutes` set to the total remaining minutes (hours × 60 + minutes)
  - `remainingDuration` formatted as `H:MM`
  - `timeCompleted` formatted as `HH:mm` of `now + remainingDuration`

#### Scenario: Missing fields default safely

- **WHEN** `state.programPhase.value_raw` or `state.status.value_raw` is absent from the payload
- **THEN** the system substitutes `-1` for the missing numeric and resolves the corresponding enum name accordingly
- **AND** the message is still emitted (it MUST NOT throw or drop the device)

#### Scenario: Device is OFF zeroes the remaining duration

- **WHEN** `state.status.value_raw` equals the `OFF` enum value
- **THEN** `remainingDurationMinutes` is `0` and `remainingDuration` is `0:00` regardless of `remainingTime`

### Requirement: Full-message passthrough

The system SHALL publish the device's raw `data` payload (as received from the Miele REST or SSE response) without modification on the `<deviceId>/full` topic.

#### Scenario: Full message preserves raw payload

- **WHEN** a device update is processed for device `<deviceId>`
- **THEN** the system publishes the unmodified `data` JSON on `<topic-prefix>/<deviceId>/full`
- **AND** publishes the small message on `<topic-prefix>/<deviceId>`

### Requirement: Null-field omission in JSON output

The system SHALL omit JSON object fields whose value is explicitly null when serializing messages, matching the behavior of the existing TypeScript `convertBody` (which uses a `JSON.stringify` replacer dropping nulls).

#### Scenario: Null field is omitted

- **WHEN** a message object contains a field whose value is `null`
- **THEN** the serialized JSON MUST NOT contain that field
- **AND** non-null fields are preserved unchanged
