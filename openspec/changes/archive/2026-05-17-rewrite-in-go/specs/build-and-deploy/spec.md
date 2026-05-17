## ADDED Requirements

### Requirement: Makefile-driven workflow

The repository SHALL provide a top-level `Makefile` with at least the following targets, matching the convention used by `hue-to-mqtt-gw`:

- `build`: produce a static binary under `./bin/` for the host platform
- `test`: run `go test ./...` with coverage
- `lint`: run `go vet ./...` (and `golangci-lint run` when available)
- `run`: run the binary against a config file (accepts `CONFIG=<path>`)
- `image`: build the production Docker image
- `clean`: remove build artifacts

#### Scenario: Build target produces a binary

- **WHEN** a developer runs `make build`
- **THEN** a runnable binary is produced under `./bin/`
- **AND** the build uses `-ldflags="-s -w" -trimpath` for a stripped binary

#### Scenario: Test target runs all Go tests

- **WHEN** a developer runs `make test`
- **THEN** all Go unit tests across the module are executed
- **AND** the run exits non-zero if any test fails

### Requirement: Multi-stage distroless Docker image

The repository SHALL provide a `Dockerfile` using a multi-stage build: a Go builder stage produces the static binary, and a final stage based on a distroless image (`gcr.io/distroless/static:nonroot`) copies only the binary and runs as a non-root user.

#### Scenario: Image build succeeds

- **WHEN** `docker build` runs on the Dockerfile
- **THEN** the resulting image runs the binary as its entrypoint
- **AND** the image does NOT contain a Go toolchain, npm, Node.js, or shell utilities beyond what distroless provides

#### Scenario: Container starts with mounted config

- **WHEN** the container is started with `/config/config.json` mounted in
- **THEN** the binary is invoked with that path as its single argument
- **AND** the existing `production/docker-compose.yml` continues to work with the new image after only updating the image tag

### Requirement: Unit-test coverage of core logic

The Go module SHALL include unit tests for:

- config loading with environment-variable substitution and defaults
- Miele OAuth login flow, including code fetch and token exchange (stubbed HTTP)
- token-refresh decision logic (`needsRefresh`)
- SSE event parsing (data-line accumulation, JSON unmarshal, reconnect path)
- polling loop (skipped when no token, fetches when token present)
- small-message transformation, covering the running, missing-fields, and OFF cases from the message-transform spec
- MQTT publish path including retain flag and JSON null-omission
- dedup behavior (suppress identical, re-emit after restart)

#### Scenario: `go test ./...` runs the full suite

- **WHEN** `go test ./...` is run from the repository root
- **THEN** tests for each module above are executed
- **AND** the run exits zero when all tests pass

### Requirement: Removal of the TypeScript implementation

The change SHALL remove the legacy TypeScript tree (`app/` directory, including `package.json`, `tsconfig.json`, `jest.config.js`, `lib/**`, and `test/**`) and update or remove any tooling that depended on it (`build.sh`, npm-specific CI steps, `renovate.json` npm package rules) as part of the same change.

#### Scenario: No npm artifacts remain

- **WHEN** the change is applied
- **THEN** the repository contains no `package.json`, `package-lock.json`, `tsconfig.json`, or `node_modules`
- **AND** the `Dockerfile` does NOT reference Node.js
- **AND** `README.md` build/run instructions reference the Makefile and the Go binary, not `npm`
