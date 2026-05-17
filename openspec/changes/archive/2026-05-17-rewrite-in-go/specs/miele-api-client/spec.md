## ADDED Requirements

### Requirement: OAuth2 password-grant login

The system SHALL be able to obtain a Miele access token using the configured `client-id`, `client-secret`, `username`, `password`, and `country-code`, by performing the same authorization-code/login flow used by the existing TypeScript implementation against `https://api.mcs3.miele.com`.

#### Scenario: Successful login with username and password

- **WHEN** the application starts with valid `username`, `password`, `client-id`, `client-secret`, and `country-code` and no stored token
- **THEN** the system performs the Miele login flow and obtains an `access_token`, `refresh_token`, and `expires_in`
- **AND** stores them in memory as the current token with an absolute `expiresAt` computed as `now + expires_in`

#### Scenario: Login fails on invalid credentials

- **WHEN** the Miele login endpoint rejects the provided credentials
- **THEN** the system logs an error including the failing step (code fetch vs. token exchange) and the application start-up fails

### Requirement: Access-token bootstrap from configuration

The system SHALL accept a pre-supplied `miele.token` object in the config (`access`, `refresh`, optional `validUntil`) and use it as the current token without performing a fresh username/password login.

#### Scenario: Token recovered from config on startup

- **WHEN** `miele.token.access` and `miele.token.refresh` are present in the config
- **THEN** the system loads them as the current token at startup
- **AND** uses `miele.token.validUntil` as `expiresAt` when present, otherwise defaults `expiresAt` to one hour after startup

### Requirement: Token refresh

The system SHALL refresh the access token by POSTing `grant_type=refresh_token` (with `client_id`, `client_secret`, `refresh_token`) to `https://api.mcs3.miele.com/thirdparty/token` and replace the current token with the response. Refresh SHALL be attempted when the current token's `expiresAt` is within one day of now or when an authenticated request returns an authentication failure.

#### Scenario: Refresh when token is close to expiry

- **WHEN** the periodic refresh check runs and `expiresAt <= now + 24h`
- **THEN** the system calls the token endpoint with the current `refresh_token`
- **AND** replaces the in-memory token with the new `access_token`, `refresh_token`, and computed `expiresAt`

#### Scenario: Refresh failure falls back to full login

- **WHEN** the refresh request fails
- **THEN** the system logs the failure and re-runs the username/password login flow to obtain a new token

### Requirement: Authenticated device listing

The system SHALL call `GET https://api.mcs3.miele.com/v1/devices/` with `Authorization: Bearer <access_token>` and `Content-Type: application/json` and return the response as a list of devices, each carrying an `id` (the JSON object key) and a `data` field containing the raw device payload.

#### Scenario: Devices fetched successfully

- **WHEN** `fetchDevices` is called with a valid access token
- **THEN** the system returns one entry per top-level key in the response body, with `id` set to that key and `data` set to the corresponding value
- **AND** preserves the raw `data` payload unchanged for downstream consumers (so it can be republished as the "full" message)

### Requirement: Optional token persistence

The system SHALL, when `miele.persistToken` is true (the default), write the current `access`, `refresh`, and `validUntil` back into the on-disk config file after a successful login or refresh; when `miele.persistToken` is false, the system MUST NOT write the token to disk.

#### Scenario: Token persisted on refresh

- **WHEN** the token is refreshed and `miele.persistToken` is true
- **THEN** the system reads the current config file, replaces `miele.token` with the new values, and writes the file back
- **AND** skips writing if the token object is byte-equal to the one already on disk

#### Scenario: Token persistence disabled

- **WHEN** `miele.persistToken` is false
- **THEN** the system MUST NOT modify the config file after refresh or login
- **AND** the token lives only in memory for the lifetime of the process
