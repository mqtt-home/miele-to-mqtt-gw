## Why

The bridge already serves `expvar` (and `pprof`) on `:6060`, and the shared `mqtt-gateway` publishes a `mqtt` expvar with connection details. The Miele side is currently invisible: there's no way to inspect the current device state, SSE event flow, polling health, or token expiry without scraping MQTT or attaching a debugger. Adding a single `expvar.Publish("miele", ...)` gives a cheap, dependency-free runtime view that matches the convention already established by `mqtt-gateway`.

## What Changes

- Publish a new top-level `expvar` named `miele` exposing:
  - **devices**: per-device snapshot of the most recently produced small message (`phase`, `phaseId`, `state`, `remainingDuration`, `remainingDurationMinutes`, `timeCompleted`), keyed by device id.
  - **sse**: current connection state (`unknown` | `connected` | `disconnected`), last event timestamp (RFC3339), total event count.
  - **polling**: last attempt timestamp, last success timestamp, last error message, total success count, total error count.
  - **token**: current access-token expiry timestamp (RFC3339), last refresh timestamp, total refresh count.
- All values are read-only and updated in-place by the existing event paths (SSE event handler, polling handler, status callback, login/refresh callsites).
- Surfaced under `/debug/vars` alongside the existing `mqtt`, `cmdline`, `memstats`, etc.
- No new MQTT topics, no new config fields, no new external dependencies. The legacy `expvar` import stays.

## Capabilities

### New Capabilities

- `runtime-metrics`: a single expvar surface (`/debug/vars` → `"miele"`) exposing device snapshots and counters/timestamps for SSE, polling, and token-refresh activity.

### Modified Capabilities

<!-- None — the runtime metrics live alongside the existing app-runtime spec without changing its requirements. -->

## Impact

- **Affected code**: `app/main.go` (or a small new `app/metrics/` package) registers the expvar; `app/app.go`, `app/miele/sse/`, `app/miele/login/` get small `metrics.RecordX(...)` callsites at the event boundaries.
- **Memory**: small constant overhead. The devices map is bounded by the number of devices on the account (typically <10) and stores parsed small-message values, not raw payloads.
- **Concurrency**: metrics must be safe for concurrent updates from SSE, polling, and login goroutines — use `sync/atomic` for counters and a `sync.RWMutex`-guarded snapshot map.
- **Operations**: no change to MQTT contract or config. Existing dashboards/scripts are unaffected; new ones can scrape `http://<host>:6060/debug/vars`.
