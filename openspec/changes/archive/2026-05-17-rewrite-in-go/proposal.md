## Why

The miele-to-mqtt-gw is currently written in TypeScript/Node.js, which carries significant memory overhead for a long-running IoT bridge service. Rewriting in Go dramatically reduces memory usage, produces a single static binary, eliminates the Node.js runtime dependency, and aligns this project with the rest of the smart home stack (e.g., `hue-to-mqtt-gw`, `mqtt-lamarzocco`) so a single set of patterns, libraries, and operational tooling applies across all bridges.

## What Changes

- **BREAKING**: Complete rewrite of the application from TypeScript/Node.js to Go. The Node.js runtime is no longer required at runtime; the previous `app/` (TypeScript) sources are removed.
- **BREAKING**: Build/run interface changes. `npm run build` / `node dist/lib/index.js` are replaced with `go build` / a single static binary; container entrypoint changes accordingly.
- Replace Node.js dependencies with Go equivalents:
  - `mqtt` (npm) → `eclipse/paho.mqtt.golang` via the shared `mqtt-gateway` module
  - `eventsource` (npm) → Go SSE client (`r3labs/sse` or equivalent)
  - `axios` (npm) → `net/http` with a typed Miele REST client
  - `node-cron` → `time.Ticker` / scheduled goroutines
  - `winston` → shared `go-logger` module
  - `async-lock` → `sync.Mutex` / channels
- Adopt the same project patterns as `hue-to-mqtt-gw`: shared `go-logger`, shared `mqtt-gateway`, `net/http/pprof` endpoint for diagnostics, JSON config with environment-variable substitution (same syntax as the existing TypeScript build per commit `8e41000`).
- Add a Makefile with targets for building, testing, running, and producing a Docker image.
- Add a multi-stage Dockerfile producing a distroless container image.
- Implement unit tests covering Miele login/token refresh, SSE parsing, polling, small-message transformation, deduplication, config loading with env-var substitution, and MQTT publish paths.
- Maintain **100% MQTT topic and message compatibility** with the existing TypeScript version: same `<topic>/<deviceId>` short message, `<topic>/<deviceId>/full` full message, same `bridge/state` and `bridge/miele` status topics with the same `online`/`offline` and `unknown`/`connected`/`disconnected` values.
- Maintain the existing `config.json` schema, including both username/password and access-token forms, `mode: sse` with polling fallback, `deduplicate`, `retain`, and the disable-token-persistence option introduced in commit `b955568`.

## Capabilities

### New Capabilities

- `miele-api-client`: OAuth2 login (username/password and country code), token refresh, optional token persistence to disk, and authenticated REST calls against the Miele cloud (`/v1/devices`).
- `miele-event-stream`: Server-Sent Events subscription to Miele device updates, with reconnect on disconnect or token expiry.
- `miele-polling`: Periodic REST polling of devices as a fallback while SSE is active and as the primary mode when `mode: polling` is configured.
- `miele-message-transform`: Transformation of raw Miele device payloads into the documented "small message" shape (`phase`, `remainingDurationMinutes`, `timeCompleted`, `remainingDuration`, `phaseId`, `state`).
- `mqtt-bridge`: MQTT client lifecycle, retained publishes, deduplication of identical payloads per topic, and bridge status reporting on `bridge/state` and `bridge/miele`.
- `app-config`: JSON configuration loading with environment-variable substitution and validation of required fields.
- `app-runtime`: Application entrypoint, scheduled tasks (token refresh check, polling), pprof diagnostic endpoint, structured logging, and graceful shutdown.
- `build-and-deploy`: Makefile-driven build/test/run workflow and a multi-stage distroless Docker image.

### Modified Capabilities

<!-- None — no existing OpenSpec specs in openspec/specs/. -->

## Impact

- **Affected code**: Entire `app/` TypeScript tree (`app/lib/**`, `app/test/**`, `app/package.json`, `app/tsconfig.json`, `app/jest.config.js`) is removed. New Go module under the repository root (or under a Go-conventional layout) replaces it.
- **Build/CI**: `build.sh` and any GitHub Actions / `renovate.json` rules tied to npm/Node need to be updated for Go (`go build`, `go test`, Go module updates).
- **Docker**: `Dockerfile` and `production/docker-compose.yml` need to be updated to build/run the Go binary from a distroless base image. Container entrypoint changes from `node` to the compiled binary.
- **Dependencies**: All npm dependencies removed. New Go module dependencies: `paho.mqtt.golang`, an SSE client, and the shared internal modules `go-logger` and `mqtt-gateway`.
- **Runtime**: No Node.js needed in production. Memory footprint expected to drop substantially.
- **Operations**: Config file format and MQTT contract are unchanged, so existing deployments only need to swap the image; `config.json` does not need to be edited.
- **Documentation**: `README.md` build/run sections need updates (npm scripts → make targets; Node version notes removed).
