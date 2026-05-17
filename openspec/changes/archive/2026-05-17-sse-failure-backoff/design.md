## Context

The Go SSE client in `app/miele/sse/sse.go` runs a tight loop: connect → read → on any failure, log, wait `ReconnectDelay` (default 5s), repeat. There is no failure-streak tracking and no awareness that the upstream might be sustained-down. During the outage reported in issue #45, the Miele API's openresty layer returned `504 Gateway Time-out` after ~60s for every connect attempt. With a 5s base delay, the bridge made ~720 doomed connections per hour while polling (which runs in parallel since commit `7620b8e`) silently kept delivering device updates.

The MQTT bridge status published on `bridge/miele` only has two non-initial values today: `connected` and `disconnected`. From an operator perspective `disconnected` flickers in and out during the outage, even though the bridge is actually fine — polling is filling in. There is no way to express "SSE is degraded, polling is carrying us".

The polling loop and the device-update path (`onDevices`) are already mode-agnostic. So this change is bounded to the SSE side: smarter retry pacing and a clearer status surface.

## Goals / Non-Goals

**Goals:**

- Stop hammering a known-failing SSE endpoint while preserving fast recovery on a single transient blip.
- Give operators a clear `degraded` bridge state that distinguishes "SSE is broken but device updates still flow" from "everything is broken".
- Keep the existing config schema valid (defaults match current behavior on the first ~25 seconds of any outage).
- Surface enough diagnostic state in the existing `miele` expvar that an operator can see *why* the bridge is in degraded mode without reading logs.

**Non-Goals:**

- Boosting the polling interval when SSE is degraded. Polling already runs at its configured interval; if operators want faster updates during an SSE outage they can tune `miele.polling-interval`. A dynamic polling interval would complicate the dedup story and the recovery path with little real-world benefit.
- Switching the configured `miele.mode` value at runtime. The mode stays `sse`; what changes is the SSE client's reconnect pacing and the reported status. `mode=polling` users are unaffected by this change.
- Circuit-breaker semantics that stop SSE entirely. The SSE attempt always continues, just spaced out — so as soon as Miele recovers we notice within the current max-delay window (10 minutes by default).
- Notifications/alerting integrations. The `degraded` state on the existing MQTT topic and the expvar fields are the surface; operators can build alerts on top.

## Decisions

### Backoff scheme: exponential with a step table

Reconnect delay is selected from a fixed step table, not computed from a multiplier. Steps: `5s, 30s, 2m, 10m`. The index advances by one each failure starting at the configured threshold (default 5) and saturates at the last step. On a successful event dispatch, the index resets to 0 (so the next reconnect delay is the base 5s).

*Alternatives considered:*

- **Multiplicative backoff (`delay *= 2`, cap at 10m).** Cleaner code but the early-outage curve is too aggressive: 5s, 10s, 20s, 40s, 80s … crossing the minute mark only at the 5th failure. The step table jumps to 30s on the first triggered step which is what we actually want.
- **Jitter on each delay.** Worth doing if many bridges hit the same Miele tenant at once, but in practice this is a personal/self-hosted bridge with one connection — adding jitter would obscure debug output without measurable benefit.

### Failure counted: anything that ends a connect attempt without successfully dispatching an event

The streak counter increments for any of: failed request build, HTTP transport error, non-200 response (including 504), EOF on the body before any event has been dispatched on that connection, watchdog timeout. It resets when a `devices` event successfully dispatches.

*Rationale:* A 504 from openresty is a non-200; an upstream timeout often manifests as a slow EOF. Treating "connected once but never received a real event" as a failure correctly classifies the issue-#45 mode where the response opens and then dies. The success criterion is *event dispatched*, not *body opened*, because Miele's behavior during the outage was to open the body and silently never deliver an event.

*Alternatives considered:* counting only HTTP errors. Rejected — misses the silent-stream failure mode which is exactly what issue #45 hit. (Their case happened to surface as 504 after openresty's own timeout, but other outages have presented as a connected body that never emits.)

### Status reporting: add `degraded` between `connected` and `disconnected`

The SSE client gains an `OnStatus("degraded")` call when the failure streak crosses the threshold while polling is healthy. `connected` is emitted on first successful event after a degraded period. `disconnected` keeps its current meaning (per-attempt connection state during the brief window before the threshold).

In `app.go` the status callback maps to MQTT `bridge/miele` and to `metrics.SetSSEConnection`. The metrics package already accepts an arbitrary string; only the MQTT side and README need a documented enum.

*Why a new state instead of overloading `disconnected`:* operators tune Home Assistant / openHAB alerts on `bridge/miele`. Today a flapping `disconnected` is a true alert ("SSE is broken"). After this change `disconnected` keeps that meaning, and `degraded` becomes a separate, longer-lived signal that operators can choose to alert on or ignore.

### Config shape: nested optional block

```jsonc
{
  "miele": {
    "sse-backoff": {
      "failure-threshold": 5,    // streak count at which backoff engages
      "base-delay": "5s",        // delay before threshold is hit
      "max-delay": "10m"         // ceiling for the step table
    }
  }
}
```

All three fields are optional. Delays are parsed with `time.ParseDuration` so operators can write `"5s"`, `"500ms"`, etc. Missing values get the defaults stated above. The step table between `base-delay` and `max-delay` is computed (multiply by ~6 each step) rather than configurable per-step — keeping the config simple.

*Alternatives considered:* flat top-level keys (`miele.sse-backoff-threshold`, …). Rejected — three related keys read better grouped, and the existing config style already groups related fields under nested objects (`miele.token`, `mqtt`).

### Expvar additions: `sse.consecutive_failures` and `sse.next_retry_after`

`consecutive_failures` is the current streak (`int`). `next_retry_after` is the RFC3339 timestamp at which the next reconnect will fire (empty string when not in a wait). Both update at the same callsites as the existing `sse.last_event` and `sse.events_total`.

## Risks / Trade-offs

- **Operators may not notice the new `degraded` value.** Mitigation: documented in README under the bridge-status table, and the expvar exposes the streak so a curious operator can see it.
- **A real Miele incident lasting >10m means a 10-minute recovery window after the upstream comes back.** Mitigation: the max delay is configurable; default 10m is a deliberate trade-off against connection-attempt cost.
- **Resetting on first dispatched event misses partial-recovery cases** where Miele dispatches one event then stalls. The watchdog (already in place) will catch the stall and re-engage backoff. Net result is one extra "burned" base-delay step, which is acceptable.
- **The status callback is now called more often per state** (each transition rather than each connect attempt). The MQTT publisher deduplicates on the bridge topic via the existing retain semantics, so this is a no-op on the wire.

## Migration Plan

1. Ship the change with defaults that match current behavior on short outages (<25s). Existing configs require no edits.
2. README updates: bridge-status enum gains `degraded`; document the new `miele.sse-backoff` block as optional with its defaults.
3. Rollback: removing the change reverts to fixed 5s reconnects. Persisted tokens and MQTT contract are unaffected, so rollback is a clean binary swap.

## Open Questions

- Should the bridge publish `degraded` immediately at startup when the first SSE attempt fails but polling has not yet had a chance to succeed? **Decision:** No — keep `disconnected` for that case so the existing meaning holds. `degraded` requires the streak threshold to be crossed *and* at least one successful poll on the parallel polling loop. If polling has not yet succeeded, the bridge is genuinely disconnected from the user's perspective.
