## 1. Metrics package

- [x] 1.1 Create `app/metrics/metrics.go` with a package-level state struct holding: `connection string`, `devices map[string]transform.SmallMessage`, `sseLastEvent time.Time`, `sseEventsTotal int64`, `pollLastAttempt`, `pollLastSuccess`, `pollLastError`, `pollErrorMessage`, `pollSuccessTotal`, `pollErrorTotal`, `tokenExpiresAt`, `tokenLastRefresh`, `tokenRefreshTotal`.
- [x] 1.2 Add `Init()` that registers `expvar.Publish("miele", expvar.Func(snapshot))` exactly once (use `sync.Once`).
- [x] 1.3 Add `snapshot()` returning a `map[string]any` matching the JSON shape in `specs/runtime-metrics/spec.md` (`connection`, `devices`, `sse`, `polling`, `token`).
- [x] 1.4 Add public update funcs: `SetSSEConnection(string)`, `RecordSSEEvent(time.Time)`, `RecordDevice(id string, msg transform.SmallMessage)`, `RecordPollAttempt(time.Time)`, `RecordPollSuccess(time.Time)`, `RecordPollError(time.Time, error)`, `RecordTokenRefresh(now, expiresAt time.Time)`.
- [x] 1.5 Guard mutable state with a `sync.RWMutex`; counters use `atomic.Int64` for hot paths if it simplifies the writes.

## 2. Callsite wiring

- [x] 2.1 Call `metrics.Init()` from `app/main.go` after `logger.Init` and before the listener starts.
- [x] 2.2 In `app/app.go:onDevices`, call `metrics.RecordDevice(d.ID, small)` for each device in the loop.
- [x] 2.3 In `app/app.go:pollOnce`, call `metrics.RecordPollAttempt(now)` at the start; on success call `metrics.RecordPollSuccess(now)` and on error call `metrics.RecordPollError(now, err)`. (The "no token" skip branch records nothing.)
- [x] 2.4 In the SSE `OnStatus` callback in `app/app.go`, call `metrics.SetSSEConnection(state)` for each transition. The first call is `unknown` from `Init()`.
- [x] 2.5 In `app/miele/sse/sse.go` dispatch path, call `metrics.RecordSSEEvent(now)` once per dispatched `devices` event (after JSON parse, before invoking `OnDevices`). Avoid an import cycle by accepting a callback on `sse.Options` rather than importing the metrics package from `sse`.
- [x] 2.6 In `app/miele/login/login.go:Login`, after the final successful token install, call `metrics.RecordTokenRefresh(now, final.ExpiresAt)`. Use the same callback-injection pattern if needed to keep `login` free of any `metrics` dependency, or accept a one-way import since `metrics` does not depend on `login`.
- [x] 2.7 Initialize `metrics.SetSSEConnection("unknown")` from `app.go:start` so the expvar reports `unknown` even before SSE has run.

## 3. Tests

- [x] 3.1 `app/metrics/metrics_test.go`: unit-test the public update functions and `snapshot()` shape (`connection`, `devices`, `sse`, `polling`, `token` keys present; correct types for counters and timestamps).
- [x] 3.2 Test: `RecordPollSuccess` increments `success_total` and updates both `last_attempt` and `last_success` but leaves `last_error` empty.
- [x] 3.3 Test: `RecordPollError` increments `error_total` and sets `last_error` to `err.Error()` but leaves `last_success` unchanged.
- [x] 3.4 Test: `RecordDevice` adds and then updates an entry under the same id.
- [x] 3.5 Test: `RecordTokenRefresh` writes `expires_at`, `last_refresh`, and increments `refresh_total`.
- [x] 3.6 `go test -race` against the metrics package with concurrent writers + a goroutine reading `snapshot()` to ensure no data races.

## 4. Documentation

- [x] 4.1 Add a short section to `README.md` (`# Diagnostics` or extend `# Logging`) noting that `:6060/debug/vars` exposes the `miele` runtime view alongside `mqtt`. Include an example trimmed JSON response.
- [x] 4.2 Note in `README.md` that `:6060` (pprof + expvar) is intended to be reachable from a trusted network only.

## 5. Validation

- [x] 5.1 Run `go test ./...` and `go vet ./...` — clean.
- [x] 5.2 Run `go test -race ./app/metrics/...` — clean.
- [ ] 5.3 Build the binary locally, run against a real `config.json`, then `curl http://localhost:6060/debug/vars | jq .miele` and verify all five sub-keys appear and update as devices update.

  *Pending: needs a live Miele account; flagged for manual validation.*
