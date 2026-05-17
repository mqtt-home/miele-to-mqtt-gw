## Context

`miele-to-mqtt-gw` bridges the Miele @home cloud API to a local MQTT broker. The current TypeScript/Node.js implementation (3.x) works but carries the Node runtime in every container and consumes more memory than peer bridges in the same smart-home stack. Sibling project `hue-to-mqtt-gw` and others have settled on a common Go-based pattern using:

- shared `go-logger` for structured logging,
- shared `mqtt-gateway` module wrapping `paho.mqtt.golang` (connection lifecycle, retained publishes, status topic helpers),
- JSON config with environment-variable substitution (the same syntax already shipped here in commit `8e41000`),
- a pprof HTTP listener for live diagnostics,
- a multi-stage Dockerfile producing a `distroless` image,
- a Makefile driving build, test, run, and image build.

The existing MQTT contract is part of the public interface of this bridge — published topics, retained values, and bridge status semantics are documented in `README.md` and consumed by user automations (e.g. openHAB items). The rewrite is purely an implementation swap; the *contract* with brokers, automations, and the config file MUST not change.

**Constraints:**

- 100% compatibility with the current MQTT topic layout, retain behavior, and message JSON.
- 100% compatibility with the current `config.json` schema, including both username/password and access-token forms, env-var substitution, polling and SSE modes, `deduplicate`, `retain`, and the disable-token-persistence flag added in commit `b955568`.
- Must follow the conventions established in `hue-to-mqtt-gw` (folder layout, shared logger, shared mqtt-gateway, pprof, Makefile, Dockerfile).
- This is a single-maintainer hobby project; complexity should match that scope.

## Goals / Non-Goals

**Goals:**

- Replace `app/` (TypeScript) with a Go module that compiles to a single static binary.
- Keep the runtime container small (distroless base), and the binary self-contained.
- Preserve the documented MQTT contract bit-for-bit (same topics, same JSON fields, same retain flag, same `bridge/state` / `bridge/miele` semantics).
- Preserve the documented `config.json` schema and env-var substitution.
- Provide unit tests for: config loading + env-var substitution, Miele OAuth login, token refresh decision logic, SSE event parsing, polling loop, small-message transformation, dedup behavior, MQTT publish path, bridge-status reporting.
- Provide a Makefile and a Dockerfile that match the style of `hue-to-mqtt-gw`.

**Non-Goals:**

- Adding new MQTT topics, payload fields, or config options. (This is a port, not a feature change.)
- Supporting both implementations in parallel. The TypeScript tree is removed; there is no Node fallback in the same image.
- Re-implementing the test-stub generator (`test-stub-generator.js`) unless it's needed for the Go test fixtures; if test fixtures from the TS tree can be reused as static JSON, prefer that.
- Integration tests against the live Miele API in CI by default. The current setup makes these optional and gated on env vars; that posture stays the same.
- Changing the openHAB documentation/usage examples.

## Decisions

### Language & runtime: Go (latest stable)

Go gives a single static binary, ~10× smaller resident memory than Node for an idle long-running bridge, and aligns with the other smart-home bridges in the same author's stack (`hue-to-mqtt-gw`, `mqtt-lamarzocco`). No need for a VM, no need for `npm` at build time.

*Alternatives considered:* Rust (more ceremony, no library parity with the existing Go bridges); staying on TypeScript with bun/deno (doesn't solve the runtime-dependency or memory goal); Java (the 2.x branch already exists and was deliberately moved away from).

### MQTT: shared `mqtt-gateway` module (wraps `eclipse/paho.mqtt.golang`)

Reuse the shared module already used by `hue-to-mqtt-gw`. It encapsulates connection lifecycle, automatic reconnect, retained publishes, last-will, and `bridge/state` semantics, so the rewrite doesn't have to re-derive any of that. Dedup is layered on top inside this project (compare last-published JSON per topic).

*Alternatives considered:* using `paho.mqtt.golang` directly. Rejected because the shared module already exists, is battle-tested in sibling bridges, and gives consistent behavior across the stack.

### Logging: shared `go-logger`

Same reasoning — already used by sibling projects. Log levels in config (`fatal`, `error`, `warn`, `info`, `debug`, `trace`) map directly to the levels exposed by `go-logger`.

### HTTP client: standard library `net/http`

The Miele REST surface is small (login, token refresh, list devices). `net/http` plus a thin typed client is enough; no need for a heavyweight HTTP framework. Use `context.Context` for cancellation and timeouts.

*Alternatives considered:* `resty` / `go-resty`. Rejected — adds a dependency for marginal benefit at this scope.

### SSE: hand-rolled client over `net/http` + `bufio.Scanner` (pattern from `hue-to-mqtt-gw/app/hue/sse.go`)

Follow the existing pattern from `hue-to-mqtt-gw/app/hue/sse.go`: an `SSEClient` struct with `connectLoop()` / `connect()`, a `bufio.Scanner` reading the response body, manual accumulation of `data: ` lines until a blank line dispatches an event, a `stopCh` for cancellation, a `sync.Mutex` protecting the live response, and a watchdog goroutine that closes the stream if no event arrives within a configurable timeout. Reconnect is a 5-second wait then re-`connect()`; a token-refresh boundary closes the stream from the outside and the loop re-establishes with the new token. Bridge status (`connected` / `disconnected`) is reported via an `onStatus` callback so the MQTT layer can publish `bridge/miele`.

*Alternatives considered:* `r3labs/sse/v2` or another SSE library. Rejected — the sibling project demonstrates that the SSE handling needed here (data-line accumulation, watchdog, manual close-on-token-refresh) is short and well-understood, and using the same hand-rolled pattern keeps the bridges consistent and dependency-light.

### Scheduling: `time.Ticker` in goroutines

Two periodic tasks today: token-refresh check and polling. Both run "every minute" via `node-cron`. In Go these are trivially `time.NewTicker(time.Minute)` inside goroutines, with cancellation via `context.Context`. No need for a cron library.

### Config: JSON with `${VAR}` env-var substitution

Match what's already in the repo (commit `8e41000`). Loader reads file → substitutes `${VAR}` and `${VAR:-default}` → unmarshals into a typed struct → validates. Same fields, same shapes, same semantics as today.

### Token persistence

Match commit `b955568`: persistence is the default; when explicitly disabled by config, tokens live in memory only and are lost on restart. Path is the same config-file directory by default.

### Dedup

Implemented in-process: keep an in-memory `map[topic]lastPayloadHash`. When `deduplicate` is true and the new payload hashes to the same value, skip publish. Cleared on restart. Same semantics as the current TS implementation.

### Project layout

Mirror `hue-to-mqtt-gw`: the Go module lives under `app/`, with non-code
assets (production compose, openspec, README, license) at the repo root.

```
/                       # repo root
├── README.md
├── docker-compose.yml  # dev/build compose, points at ./app
├── renovate.json
├── production/         # production docker-compose + config
├── openspec/           # change proposals + specs
└── app/                # Go module
    ├── Makefile
    ├── Dockerfile      # multi-stage, distroless final
    ├── go.mod / go.sum
    ├── main.go         # entry: parse args, load config, wire deps, start
    ├── app.go          # bootstrap/teardown wiring (login, SSE, polling, refresh)
    ├── config/         # config struct + JSON + env-var substitution
    ├── miele/
    │   ├── login/      # OAuth, token refresh, persistence
    │   ├── api/        # devices fetch, REST types
    │   ├── sse/        # SSE client wrapper
    │   └── transform/  # enums + small-message + null-stripping JSON
    ├── bridge/         # MQTT publish + dedup + miele-state
    └── version/        # build metadata (ldflags-injected)
```

The legacy TypeScript `app/` tree (which lived in the same `app/` directory)
is deleted in the same change.

### Diagnostics

Expose `net/http/pprof` on a configurable port (default disabled / opt-in via config, matching `hue-to-mqtt-gw` posture). Useful for diagnosing leaks or goroutine pile-ups during long runs.

### Docker image

Multi-stage:

1. `golang:<version>` builder — `go build -ldflags="-s -w" -trimpath` produces a static binary.
2. `gcr.io/distroless/static:nonroot` runtime — copies only the binary and runs as non-root.

Config is mounted at `/config/config.json` (compatible with the existing `production/docker-compose.yml` layout).

### Makefile

Targets (mirrors `hue-to-mqtt-gw`):

- `make build` — `go build` into `./bin/miele-to-mqtt-gw`
- `make test` — `go test ./...`
- `make lint` — `go vet ./...` (and `golangci-lint run` if available)
- `make run CONFIG=...` — run locally against a config file
- `make image` — `docker build` the multi-stage image, tagged from the current version
- `make clean`

## Risks / Trade-offs

- **Risk:** Subtle JSON-shape divergence breaks user automations (openHAB items keyed on field names).
  → **Mitigation:** Reuse the existing fixture JSON from `app/test/` as Go test fixtures and assert byte-equivalent (or field-equivalent) small-message output. Document the small-message JSON in `README.md` and freeze it via unit tests.

- **Risk:** Miele SSE quirks (heartbeats, partial lines, reconnect-after-401) handled implicitly by `eventsource` in Node may differ in the Go SSE client.
  → **Mitigation:** Lift the reconnect-on-token-expiry behavior from `app.ts` into the Go side explicitly: on any SSE error, close the stream, refresh the token if needed, restart SSE. Cover this path with a unit test using a stub SSE server.

- **Risk:** OAuth2 login payload/headers may have evolved since the TS implementation last passed integration tests.
  → **Mitigation:** Port the existing integration tests (`MIELE_*` env-gated) so the same login flow is exercised against the real API on demand, just as today.

- **Risk:** Token-persistence file format differs between TS and Go, breaking upgrades.
  → **Mitigation:** Adopt the same on-disk JSON shape (`access`, `refresh`, expiry timestamp) so an upgrade from 3.x reads the existing token file without forcing a re-login. If the existing format is not portable cleanly, document that a one-time re-login is required.

- **Risk:** Config env-var substitution semantics drift from the TS implementation (e.g., default-value syntax).
  → **Mitigation:** Lift the substitution unit tests from the TS side directly into Go tests; same inputs, same outputs.

- **Trade-off:** Removing the TS tree in the same change makes rollback "ship the old image tag" rather than "flip a flag." Acceptable for a personal/hobby project with a single deployment per user; the container tag itself is the rollback mechanism.

- **Trade-off:** No parallel-run / canary mode. Users upgrade by pulling a new image; if it misbehaves they pin back to the last 3.x tag. Documented in the release notes.

## Migration Plan

1. Implement the Go module under the repo root alongside the existing `app/` (on a feature branch). CI runs both Node and Go test suites until cutover.
2. Validate the MQTT contract by running both versions in shadow against a test broker on the same config, and diffing published payloads per topic. Treat any diff (other than timestamps) as a bug.
3. Cut a `4.0.0-rc` Docker tag from the Go branch and run it against the real Miele account for a sustained period (≥ 1 week) to surface SSE / token-refresh edge cases.
4. Once stable, delete `app/`, update `README.md`, update `Dockerfile` and `production/docker-compose.yml`, and release `4.0.0`.
5. Rollback strategy: re-tag/pull the last published 3.x image. No data migration is required because MQTT state is retained on the broker and the token file format is compatible.

## Open Questions

- Should the env-var substitution support `${VAR:-default}` fallbacks in the Go port, or only bare `${VAR}`? Match whatever the TS implementation in commit `8e41000` does — confirm exact syntax during implementation.
- Pprof: on by default with a fixed port, or opt-in via config? Default to **opt-in via config** to match `hue-to-mqtt-gw`; revisit if needed.
- SSE watchdog timeout: `hue-to-mqtt-gw` exposes this as `sse-watchdog-ms`. Decide whether to surface the same knob in `miele.sse-watchdog-ms` or hard-code a sensible default (Miele heartbeats are less frequent than Hue).
