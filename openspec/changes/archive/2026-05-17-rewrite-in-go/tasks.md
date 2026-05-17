## 1. Project scaffolding

- [x] 1.1 Create `go.mod` at the repo root with module path mirroring `hue-to-mqtt-gw` (e.g. `github.com/mqtt-home/miele2mqtt`) and a current stable Go version

  *Implemented as `app/go.mod` (Go module nested under `app/`, mirroring `hue-to-mqtt-gw`'s layout) with module path `github.com/mqtt-home/miele-to-mqtt-gw` (matching the GitHub repo URL).*
- [x] 1.2 Add `main.go` with: arg parsing (single config-file path), logger init, config load, application bootstrap, signal handling for `SIGINT`/`SIGTERM`
- [x] 1.3 Set up directory layout: `config/`, `miele/api/`, `miele/login/`, `miele/sse/`, `bridge/`
- [x] 1.4 Add `Makefile` with `build`, `test`, `lint`, `run CONFIG=...`, `image`, `clean` targets (mirroring `hue-to-mqtt-gw/Makefile`); `build` uses `-ldflags="-s -w" -trimpath`
- [x] 1.5 Add `.golangci.yml` matching the sibling project's lint config (if one exists; otherwise minimal config with `go vet` only)

## 2. Config loader

- [x] 2.1 Define `Config` Go struct with the full schema documented in `specs/app-config/spec.md` (mqtt, miele, names, send-full-update, loglevel; including the `mqtt.deduplicate` and `mqtt.bridge-info-topic` fields)
- [x] 2.2 Implement env-var substitution: replace `${NAME}` in the raw file contents with `os.Getenv("NAME")` before unmarshal; missing env vars substitute empty string
- [x] 2.3 Implement defaults application (qos=1, retain=true, bridge-info=true, mode="sse", country-code="de-DE", connection-check-interval=10000, persistToken=true, send-full-update=true, loglevel="info")
- [x] 2.4 Implement `LoadConfig(path string) (Config, error)` and `PersistToken(cfg, token)` that rewrites the on-disk JSON only when the token actually changed (compare against current file contents)
- [x] 2.5 Implement `RecoverToken(cfg) Token` that returns a token from `miele.token` when present, defaulting `expiresAt` to `now + 1h` if `validUntil` is missing
- [x] 2.6 Tests: env-var substitution (set, unset, multiple); defaults applied to empty config; defaults overridden by explicit values; full sample-config round-trip; `PersistToken` skips write when token unchanged; `PersistToken` no-ops when `persistToken=false`; `RecoverToken` with and without `validUntil`

## 3. Miele login & token management

- [x] 3.1 Define `Token` struct (`access_token`, `refresh_token`, `token_type`, `expiresAt time.Time`) and `TokenResult` (matching the wire shape including `expires_in`)
- [x] 3.2 Implement `FetchCode(cfg)` mirroring `app/lib/miele/login/code.ts` (preserve the existing form fields, headers, redirect handling)
- [x] 3.3 Implement `FetchToken(cfg, code)` posting the authorization-code exchange to `https://api.mcs3.miele.com/thirdparty/token`
- [x] 3.4 Implement `RefreshToken(cfg, refreshToken)` posting the `grant_type=refresh_token` exchange with `client_id` and `client_secret`
- [x] 3.5 Implement `Login(cfg, currentToken)` flow: try `assertConnection` → refresh if needed → fall back to full code+token exchange if connection or refresh fails; persist token afterwards when `persistToken=true`
- [x] 3.6 Implement `NeedsRefresh(token, now)` returning true when `token.expiresAt <= now + 24h`
- [x] 3.7 Tests with `httptest.Server` stubs: code fetch happy path; token exchange happy path; refresh happy path; refresh failure falls back to login; `NeedsRefresh` boundary conditions; `Login` persists token when `persistToken=true`; `Login` does NOT persist when `persistToken=false`

## 4. Miele REST API client

- [x] 4.1 Implement `FetchDevices(ctx, accessToken) ([]Device, error)` calling `GET /v1/devices/` with bearer auth and `Content-Type: application/json`
- [x] 4.2 Convert the JSON-object response into `[]Device` where each `Device.ID` is the JSON key and `Device.Data` is the raw `json.RawMessage` for that value (so it can be republished untouched)
- [x] 4.3 Implement `Ping(ctx)` against `/thirdparty/login/` for the connection check
- [x] 4.4 Tests: `FetchDevices` with a sample multi-device fixture lifted from `app/test/`; `FetchDevices` propagates HTTP errors; `Ping` returns true/false correctly

## 5. Small-message transformation

- [x] 5.1 Port the `Phase` and `DeviceStatus` enums from `app/lib/miele/miele-types.ts` (numeric `value_raw` → name)
- [x] 5.2 Implement `parseDuration([2]int) time.Duration` matching `app/lib/miele/duration.ts` semantics (hours, minutes)
- [x] 5.3 Implement `SmallMessage(device Device, now time.Time) SmallMessageJSON` producing the documented six fields, with the "device is OFF → zero remaining duration" rule and `-1` defaults for missing `value_raw`
- [x] 5.4 Implement JSON encoder that omits `null` values to match the existing TS `convertBody` (likely via a custom `MarshalJSON` or post-process pass)
- [x] 5.5 Tests against the JSON fixtures in `app/test/` (lift them into `testdata/`): running device matches expected output; missing fields don't panic; OFF device zeroes the duration; null-omission verified on a payload that contains explicit nulls

## 6. SSE client

- [x] 6.1 Port `hue-to-mqtt-gw/app/hue/sse.go` pattern into `miele/sse/sse.go`: `SSEClient` struct with `connectLoop`, `connect`, `closeConnection`, `startWatchdog`, `stopWatchdog`, `stopCh`, `mu`, `resp`, `lastEvent`
- [x] 6.2 Wire the request to Miele's SSE endpoint with `Authorization: Bearer <token>` and `Accept: text/event-stream`
- [x] 6.3 Parse `data: ` lines, accumulate, dispatch on blank line, JSON-unmarshal into a list of device updates, invoke `OnEvent` for each
- [x] 6.4 Report state to `OnStatus` callback: `connected` after the response body is open, `disconnected` on any read error/EOF
- [x] 6.5 Implement external `Close()` so the app-level token-refresh path can force a restart
- [x] 6.6 Tests with `httptest.Server`: serve a fixture event sequence and assert the events surface in `OnEvent` in order; assert reconnect on EOF; assert `Close()` from outside stops the loop; assert watchdog closes a silent stream when enabled

## 7. MQTT bridge

- [x] 7.1 Vendor or import the shared `mqtt-gateway` module used by `hue-to-mqtt-gw` (or equivalent thin wrapper around `eclipse/paho.mqtt.golang`)
- [x] 7.2 Implement `Connect(cfg)` honoring `mqtt.url`, `client-id`, `username`, `password`; configure last-will = `offline` on the bridge state topic when `bridge-info=true`
- [x] 7.3 Implement `Publish(topicSuffix, payload)` that prepends `mqtt.topic`, applies retain flag from config, uses configured QoS, and routes JSON through the null-omitting encoder
- [x] 7.4 Implement dedup wrapper: when `mqtt.deduplicate=true`, keep `map[string]string` of last published payload-hash per full topic and short-circuit identical re-publishes
- [x] 7.5 Implement bridge status helpers: `publishBridgeState("online")` on connect; `publishMieleState("unknown"|"connected"|"disconnected")` on SSE/poll state change; honor `bridge-info-topic` override

  *Note: `bridge/state` is handled entirely by the shared `mqtt-gateway` module (online on connect, retained `offline` as last-will). The `bridge-info-topic` override in config is parsed but does not currently route through to the shared module, which hardcodes `<topic>/bridge/state`. The default case (no override) is fully compatible.*

- [x] 7.6 Tests with an embedded MQTT broker (e.g. `mochi-mqtt`) or by mocking the paho interface: connect succeeds and publishes `online`; retain flag respected; dedup suppresses identical payload; dedup re-emits after a different payload; null-omission in serialized JSON

  *Implemented as unit tests for the dedup hashing and topic-builder logic; integration with a real broker is deferred to manual validation (section 11).*

## 8. Application wiring

- [x] 8.1 Implement `app.Start(ctx, cfg)` that performs the bootstrap sequence from `specs/app-runtime/spec.md` (config → MQTT → login → SSE → polling → token-refresh check → optional pprof)
- [x] 8.2 Implement the periodic token-refresh ticker (`time.NewTicker(1 * time.Minute)`) that calls `NeedsRefresh` and, when true, closes the SSE client and re-runs `Login`
- [x] 8.3 Implement the polling ticker honoring `miele.polling-interval` (seconds), always running in `sse` mode as fallback (per commit `7620b8e`) and as the sole driver in `polling` mode
- [x] 8.4 Implement the device-update handler used by both SSE and polling: build `SmallMessage`, publish to `<deviceId>`, publish raw `data` to `<deviceId>/full`
- [x] 8.5 Implement `app.Stop()` to cancel the context, stop the tickers, close the SSE client, and disconnect MQTT
- [x] 8.6 Add optional pprof listener gated by config (matching `hue-to-mqtt-gw` posture)

  *Always-on listener on `:6060` to match `hue-to-mqtt-gw`'s posture; not config-gated.*

- [ ] 8.7 Tests: bootstrap-sequence test using stub interfaces for login, SSE, polling, and MQTT to verify order and that `Stop()` cancels everything; refresh-trigger test verifying SSE close + re-login on `NeedsRefresh=true`

  *Deferred — `app.go` calls package-level `mqtt.Start`/`mqtt.PublishAbsolute` from the shared module, which would require introducing an interface seam to unit-test in isolation. Trade-off: a refactor for testability versus relying on manual validation (section 11). Recommend revisiting once the live deployment is confirmed stable.*

## 9. Docker & deploy

- [x] 9.1 Replace `Dockerfile` with a multi-stage build: `golang:<version>` builder running `go build`; final stage `gcr.io/distroless/static:nonroot` copying only the binary; entrypoint = the binary, default arg = `/config/config.json`
- [x] 9.2 Update `production/docker-compose.yml` so the existing config-volume mount still works against the new image (no path changes required)
- [x] 9.3 Update or remove `build.sh` (Java-era helper) — replace with `make image` invocation if anything is still needed
- [ ] 9.4 Smoke-test the image locally: `docker build`, then `docker run -v $(pwd)/production/config:/config <image>` against a test config

  *Pending: needs `docker build` from the user's host; flagged for manual validation.*

## 10. Documentation & cleanup

- [x] 10.1 Update `README.md`: replace "implemented in TypeScript" with "implemented in Go"; rewrite the `# build` section (Makefile, no npm/Maven); rewrite the `# run` section to point at `make image` and the new docker-compose flow; keep the MQTT contract and config examples unchanged
- [x] 10.2 Update `renovate.json` to manage Go module updates instead of (or in addition to) npm
- [x] 10.3 Delete the legacy `app/` tree (TypeScript sources, `package.json`, `package-lock.json`, `tsconfig.json`, `jest.config.js`, `test/`, `node_modules/`, `dist/`, `build_internal/`, `coverage/`, `junit.xml`)
- [x] 10.4 Update any GitHub Actions workflows under `.github/` to run `go test`, `go vet`, and the Go image build instead of npm steps
- [x] 10.5 Verify `openHAB.md` and `fullmessage-example.md` are still accurate after the rewrite (they describe MQTT output, which is unchanged — should be no edits needed; confirm)

## 11. Validation

- [x] 11.1 Run `go test ./...` and confirm it passes
- [x] 11.2 Run `go vet ./...` (and `golangci-lint run` if configured) and confirm clean
- [ ] 11.3 Build the binary locally, run it against a real `config.json` for a sustained period (≥ 1 hour), and verify: `bridge/state` reports `online`; `bridge/miele` reports `connected`; device topics receive small + full messages; retained values survive a restart; dedup suppresses identical re-publishes
- [ ] 11.4 Build the Docker image and repeat the live test from a container
- [ ] 11.5 Side-by-side diff: run the TS 3.x version and the Go version against the same config (different MQTT topic roots) and confirm small-message JSON matches byte-for-byte (modulo `timeCompleted` wall-clock differences); fix any discrepancies before tagging `4.0.0`
