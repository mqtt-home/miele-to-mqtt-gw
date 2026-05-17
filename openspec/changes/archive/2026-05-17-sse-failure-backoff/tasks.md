## 1. Config schema

- [x] 1.1 Add `SSEBackoffConfig` struct to `app/config/config.go` with fields `FailureThreshold int`, `BaseDelay string`, `MaxDelay string` (raw strings parsed later) and the matching `json:"failure-threshold"`, `json:"base-delay"`, `json:"max-delay"` tags. Add it under `MieleConfig` as `SSEBackoff *SSEBackoffConfig json:"sse-backoff,omitempty"`.
- [x] 1.2 Extend `ApplyDefaults` to populate the backoff defaults when `Miele.SSEBackoff` is nil or has zero-valued fields: threshold=5, base-delay=`"5s"`, max-delay=`"10m"`.
- [x] 1.3 Add a load-time validation step inside `LoadConfig` (or a helper called from it) that parses `BaseDelay` / `MaxDelay` with `time.ParseDuration` and returns a wrapped error naming the offending field. Store the parsed `time.Duration` values on a sibling private field (`baseDelayParsed`, `maxDelayParsed`) or expose accessor methods so the rest of the app does not re-parse.
- [x] 1.4 Update `config_test.go`: add cases for (a) absent block uses defaults, (b) partial override keeps defaults for omitted fields, (c) custom durations parse correctly, (d) invalid duration returns an error.

## 2. SSE backoff core

- [x] 2.1 Extend `sse.Options` with backoff inputs: `FailureThreshold int`, `BaseReconnectDelay time.Duration`, `MaxReconnectDelay time.Duration`. Keep `ReconnectDelay` for back-compat (treated as the base if `BaseReconnectDelay` is unset). Default `FailureThreshold` to 5 when zero.
- [x] 2.2 In `Client`, add fields `consecutiveFailures int`, `dispatchedThisConnection bool`, `nextRetryAt time.Time` guarded by the existing mutex.
- [x] 2.3 Replace the fixed `time.After(c.opts.ReconnectDelay)` in `loop()` with a `selectBackoffDelay()` helper that returns the next delay: base while `consecutiveFailures < threshold`, otherwise the step-table value clamped to `MaxReconnectDelay`. Compute the step table once at `Start()` from base/max (multiply by ~6 per step, last step == max).
- [x] 2.4 Increment `consecutiveFailures` on each failure path: failed `http.NewRequestWithContext`, transport `Do` error, non-200 status, `readLoop` returning without having dispatched any event on that connection (track via `dispatchedThisConnection`), and watchdog-triggered close with no dispatch. Set `dispatchedThisConnection = false` at the start of `connect()`.
- [x] 2.5 Reset `consecutiveFailures` to 0 inside the `dispatch` closure in `readLoop` after a successful `OnDevices` call, and set `dispatchedThisConnection = true`.
- [x] 2.6 Update `nextRetryAt` whenever `selectBackoffDelay` is computed, and clear it when entering `connect()`.

## 3. Status reporting (connected / disconnected / degraded)

- [x] 3.1 Add an `OnPollingHealthy func() bool` callback to `sse.Options` so the SSE client can ask the app whether the parallel polling loop has had a successful poll since process start.
- [x] 3.2 In `loop()`, after computing the post-failure status, emit `degraded` (instead of `disconnected`) when `consecutiveFailures >= threshold` AND `OnPollingHealthy()` returns true. Below threshold, keep emitting `disconnected` exactly as today.
- [x] 3.3 On successful event dispatch in `readLoop`, if the previous reported status was `degraded` or `disconnected`, emit `connected` (the existing `connect()` already emits `connected` once on body-open; ensure the dispatch path can flip from `degraded`→`connected` without requiring a body close/re-open).
- [x] 3.4 In `app/app.go`, expose a `pollingHealthy()` helper that returns true once `metrics.RecordPollSuccess` has been called at least once. Wire it into `sse.Options.OnPollingHealthy` at startup.
- [x] 3.5 Extend the SSE `OnStatus` switch in `app/app.go` to handle `degraded`: call `metrics.SetSSEConnection("degraded")` and `a.pub.PublishMieleState("degraded")`.

## 4. Metrics surface

- [x] 4.1 Add `sseConsecutiveFailures atomic.Int64` (or unguarded int read inside the existing mutex; pick whichever matches the surrounding pattern in `metrics.go`) and `sseNextRetryAt time.Time` to the metrics state.
- [x] 4.2 Add `RecordSSEFailure(streak int, nextRetry time.Time)` and `RecordSSESuccess()` update funcs. Wire them from the SSE client at the same points the counter changes (after increment / after reset).
- [x] 4.3 Update `snapshot()` to emit `consecutive_failures` and `next_retry_after` under the `sse` sub-object. `next_retry_after` is `""` when the stored time is zero, else RFC3339.
- [x] 4.4 Update `metrics_test.go`: a streak of N failures increments `consecutive_failures` to N and sets `next_retry_after`; a success call resets both fields.

## 5. App wiring

- [x] 5.1 In `app/app.go:start`, pass the new backoff options into `sse.Start(...)` reading from `a.cfg.Miele.SSEBackoff`. Use the parsed `time.Duration` values; do not re-parse the strings here.
- [x] 5.2 Keep the existing `ReconnectDelay: 5 * time.Second` line removed or adjusted so the new `BaseReconnectDelay` is authoritative.
- [x] 5.3 Verify the polling loop's `metrics.RecordPollSuccess` callsite is the source of truth for "polling healthy" — no additional plumbing needed beyond reading the metrics counter from `pollingHealthy()`.

## 6. Tests

- [x] 6.1 `app/miele/sse/sse_test.go`: add a stub server that returns 504 N times then 200 with one event. Assert: `OnStatus` sees `disconnected` for the first attempts below threshold, `degraded` once at-and-beyond threshold (with `OnPollingHealthy` returning true), then `connected` after the successful dispatch. Verify the streak reset is observable via injected callback.
- [x] 6.2 Test: with `OnPollingHealthy` returning false, the streak crossing the threshold still produces `disconnected` (NOT `degraded`).
- [x] 6.3 Test: a connection that opens 200 but the body returns EOF before any event counts as a failure (streak increments).
- [x] 6.4 Test: watchdog timeout with no dispatched event counts as a failure; watchdog timeout after at least one dispatched event does NOT count (the dispatch had already reset the streak).
- [x] 6.5 Test the backoff step-table calculation directly: with base=5s, max=10m, the table is `[5s, 30s, 2m, 10m]` (or similar — assert the boundary conditions: first step > base, last step == max, monotonic).
- [x] 6.6 Run `go test -race ./...` and ensure no new data-race findings.

## 7. Documentation

- [x] 7.1 Update `README.md` bridge-status documentation to list all four values (`unknown`, `connected`, `disconnected`, `degraded`) with a one-line meaning each. Note that `degraded` means "SSE has been failing but polling is still delivering device updates".
- [x] 7.2 Add a `miele.sse-backoff` block to `config-example.json` (commented as optional) showing the default values.
- [x] 7.3 Update the README diagnostics section to mention the two new expvar fields under `miele.sse`.

## 8. Validation

- [x] 8.1 Run `go test ./...` and `go vet ./...` — clean.
- [x] 8.2 Run `openspec validate sse-failure-backoff --strict` — clean.
- [ ] 8.3 Manual: with a config pointing at a stub server that returns 504, observe (a) reconnect delay growth over time, (b) `bridge/miele` transitions through `disconnected` → `degraded` once polling succeeds at least once, (c) recovery to `connected` when the stub returns events. Flag as pending if a live Miele account is required.
