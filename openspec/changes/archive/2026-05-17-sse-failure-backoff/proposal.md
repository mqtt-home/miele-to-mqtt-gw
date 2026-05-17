## Why

When Miele's SSE gateway is degraded (e.g. returns `504 Gateway Time-out` from the upstream openresty proxy, as reported in issue #45), the current SSE client retries every 5 seconds indefinitely. That generates log spam, hammers an already-struggling endpoint, and gives operators no clear signal that the bridge has fallen back to polling. The polling loop already runs in parallel as a fallback (commit `7620b8e`), so devices keep updating — but the SSE side stays loud and the bridge state on MQTT is unclear. This change adds an exponential backoff to SSE reconnects after sustained failures and reports a `degraded` bridge state so users can tell the difference between "SSE is healthy" and "we're surviving on polling".

## What Changes

- Track consecutive SSE connection failures (any non-success outcome: build-request error, transport error, non-200 status including 504, EOF before any event, watchdog timeout before any event).
- Apply exponential backoff to the SSE reconnect delay after a configurable failure threshold: 5s → 30s → 2m → 10m, capped at 10m. Reset to the base 5s delay as soon as a connection succeeds and dispatches at least one event.
- Report a new bridge status value `degraded` on `bridge/miele` when the failure streak crosses the backoff threshold and polling is the de-facto source of updates. Transition back to `connected` when SSE recovers, or `disconnected` when polling is also failing.
- Expose the failure streak and current backoff via the existing `miele` expvar surface (`sse.consecutive_failures`, `sse.next_retry_after`) for diagnostics.
- New config field `miele.sse-backoff` (optional object) lets operators tune the threshold and ceiling without touching code. Defaults match the values above so existing configs keep working unchanged.

## Capabilities

### New Capabilities

<!-- None — this change adjusts existing capabilities only. -->

### Modified Capabilities

- `miele-event-stream`: adds requirements for exponential reconnect backoff after consecutive failures and a `degraded` status value alongside `connected`/`disconnected`/`unknown`.
- `app-config`: adds the optional `miele.sse-backoff` block (threshold, base delay, max delay) with documented defaults.
- `runtime-metrics`: extends the `sse` expvar sub-object with `consecutive_failures` and `next_retry_after` fields.

## Impact

- **Affected code**: `app/miele/sse/sse.go` (failure tracking, backoff scheduling, success-reset), `app/app.go` (map the new `degraded` status to MQTT and metrics), `app/config/config.go` (new optional config block + defaults), `app/metrics/metrics.go` (two new fields in the `sse` snapshot).
- **Behavior on existing configs**: unchanged for the first ~30 seconds of any outage — the threshold defaults to 5 consecutive failures (~25 seconds at the 5s base delay) before backoff engages, so brief blips look the same as today.
- **MQTT contract**: a new `degraded` value can appear on `bridge/miele`. Existing subscribers that only expect `connected`/`disconnected` will see it as an unknown string; documented in README.
- **No new external dependencies.** No change to the device-update path — devices flow through `onDevices` exactly as today during both healthy and degraded states.
