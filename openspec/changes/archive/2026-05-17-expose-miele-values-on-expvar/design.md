## Context

The Go rewrite already exposes `net/http/pprof` and the standard library's `expvar` at `:6060` (imported for side-effect in `app/main.go`). The shared `philipparndt/mqtt-gateway` module already registers an `expvar.Publish("mqtt", ...)` exposing connection details (`connected_at`, `reconnects`, `subscriptions`, `last_reconnect`). The Miele side currently has no equivalent — operators have to subscribe to MQTT or read logs to see what's happening.

## Goals / Non-Goals

**Goals:**

- One additional top-level `expvar.Var` named `miele` rendered at `/debug/vars`.
- Cover the four runtime surfaces the operator most often wants to see: per-device snapshot, SSE flow, polling flow, token state.
- Update happens at the same callsites that already drive logging — no new background goroutines, no new ticker, no separate sample loop.
- Concurrency-safe under SSE + polling + refresh updating in parallel.
- Zero new external dependencies; zero new config fields.

**Non-Goals:**

- A Prometheus/OpenMetrics exporter. (`expvar` is enough for the personal/hobby scope; a future change can add a Prometheus surface if needed.)
- Per-event histograms or latency tracking.
- Exporting the *full* Miele device payload — the small message has the interesting fields and the raw payload is large and noisy.
- Authentication for `/debug/vars`. The pprof endpoint is already unauthenticated on `:6060` and expected to be reached only from inside a trusted network or via SSH port-forward; the same posture applies here.

## Decisions

### Single top-level expvar `miele` returning a nested map

The `mqtt-gateway` module already publishes one top-level expvar with a nested map via `expvar.Func`. We mirror this exactly so operators see `/debug/vars` with consistent shape:

```json
{
  "mqtt": { ... },
  "miele": {
    "connection": "connected",
    "devices": { "<id>": {"phase":"DRYING", ... }, ... },
    "sse":     { "last_event": "...", "events_total": 1234 },
    "polling": { "last_attempt": "...", "last_success": "...", "last_error": "", "success_total": 23, "error_total": 0 },
    "token":   { "expires_at": "...", "last_refresh": "...", "refresh_total": 5 }
  }
}
```

*Alternatives considered:* one top-level expvar per sub-surface (`miele_sse`, `miele_polling`, …). Rejected — pollutes `/debug/vars` and diverges from the `mqtt-gateway` convention.

### Package layout: `app/metrics/`

A small `app/metrics/` package owns the in-memory state and the `expvar.Publish` registration. Callsites import this package and call e.g. `metrics.RecordSSEEvent(time.Now())`, `metrics.RecordPollSuccess(time.Now())`, `metrics.RecordDevice(small)`, etc. The package's exported state is package-level (singleton) — matching how `mqtt-gateway` does it, since the application has exactly one instance of each.

*Alternatives considered:* embedding the metrics state inside the `app` struct and passing it through to SSE/login. Rejected — adds plumbing for what is naturally a process-global concern, and the singleton matches the `expvar` model.

### Concurrency primitives: `atomic.Int64` + `sync.RWMutex`

Counters (event totals, refresh totals, etc.) use `atomic.Int64`. Timestamps and the device snapshot map use a single `sync.RWMutex`. Reads happen only when `expvar.Func` is invoked (rare), so an `RWMutex` is fine. We deliberately avoid `sync.Map` — the read-heavy expvar invocation is rare enough that the simpler mutex wins on clarity.

### Timestamps as RFC3339 strings

`expvar` serializes via `json.Marshal`. `time.Time` already serializes to RFC3339, but expvar dashboards/scripts on personal-sized deployments usually expect strings; we store `time.Time` and let `json.Marshal` handle the wire shape. Zero values render as `"0001-01-01T00:00:00Z"` which is acceptable for "never happened yet" — callers can distinguish by counter == 0.

### Device snapshot shape

Each device entry is a copy of the most recent `transform.SmallMessage` (a value type, not a pointer). Snapshots are point-in-time: we never delete entries (a device that disappears from the account is shown with its last-seen state until the process restarts). This matches the MQTT retain semantics most operators are used to.

### Callsite list

The plumbing is intentionally tiny:

| Callsite | Call |
| --- | --- |
| `app.go:onDevices` (both SSE and polling) | `metrics.RecordDevice(d.ID, small)` per device |
| `app.go:pollOnce` start | `metrics.RecordPollAttempt(now)` |
| `app.go:pollOnce` success | `metrics.RecordPollSuccess(now)` |
| `app.go:pollOnce` error | `metrics.RecordPollError(now, err)` |
| `sse.Options.OnStatus` | `metrics.SetSSEConnection(state)` |
| `sse` per-event dispatch | `metrics.RecordSSEEvent(now)` |
| `login.Manager.Login` after success | `metrics.RecordTokenRefresh(now, tok.ExpiresAt)` |
| `main.go` init | `metrics.Init()` (calls `expvar.Publish("miele", ...)`) |

### Wiring into SSE

The current `sse.Options` exposes `OnEvent` only at the *batch* level (one call per dispatch, carrying the full `[]Device`). That's fine — we record one SSE event per dispatch. If we ever want a per-device counter, the existing `OnDevices` already iterates devices in `app.go`.

## Risks / Trade-offs

- **Risk:** `expvar` exposes the device snapshot publicly on `:6060`. Same posture as pprof — operators are expected to firewall this. → **Mitigation:** document in README that `:6060` is local-only; no behaviour change here.
- **Risk:** Snapshot map grows unbounded if device IDs churn. Practically: Miele device IDs are stable serials and one account has <10 devices. → **Mitigation:** none needed at this scope; flag if memory ever shows growth.
- **Risk:** Concurrent map updates under high SSE rate could become a contention point. → **Mitigation:** `RWMutex` with write-only locking on update and read-only locking inside `expvar.Func` is more than enough for the actual event rate (single-digit events/min on a real account).
- **Trade-off:** `last_error` is a free-form string set by polling failures. It's not redacted. Since the only sender is our own error formatting (no Miele server payload), this is acceptable.

## Migration Plan

No migration. The change is additive: `/debug/vars` gains one new key. No config, no MQTT topic, no docker change. Rollback is "ship the previous image."

## Open Questions

- Should we also expose `bridge_state` (online/offline) here? mqtt-gateway already publishes `mqtt.connected_at` which covers the same information; duplicating would be noise. **Decision:** skip for now — the only Miele-specific state we add under `miele.connection` is the bridge/miele value (`unknown`/`connected`/`disconnected`).
- Should the device entry include the full raw payload? **Decision:** no — too large; the small message is what dashboards care about.
