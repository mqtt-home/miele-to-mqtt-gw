# miele-to-mqtt-gw

[![mqtt-smarthome](https://img.shields.io/badge/mqtt-smarthome-blue.svg)](https://github.com/mqtt-smarthome/mqtt-smarthome)

Convert the miele@home data to MQTT messages

This application will post two MQTT messages for each connected device: one short message and a full message.

# Releases

## Production (4.x)
The current production version is 4.x and is implemented in Go. It produces a
single static binary distributed as a distroless container image.

## Legacy releases
- 3.x — TypeScript / Node.js (https://github.com/mqtt-home/miele-to-mqtt-gw/tree/3.x-node)
- 2.x — Java (https://github.com/mqtt-home/miele-to-mqtt-gw/tree/2.x-java)

## Example short message

The short message is already parsed/interpreted and contains only the most relevant information.

```json
{
  "phase": "DRYING",
  "remainingDurationMinutes": 4,
  "timeCompleted": "12:35",
  "remainingDuration": "0:04",
  "phaseId": 1799,
  "state": "RUNNING"
}
```

## Example full message

The full message is exactly the message provided by Miele without any changes.
See [fullmessage-example](fullmessage-example.md)

## Example configuration

### With username/password

```json
{
  "mqtt": {
    "url": "tcp://192.168.2.2:1883",
    "client-id": "miele-mqtt-gw",
    "username": "username",
    "password": "password",
    "retain": true,

    "topic": "home/miele",
    "deduplicate": true
  },

  "miele": {
    "client-id": "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
    "client-secret": "12345678901234567890123456789012",
    "polling-interval": 30,
    "username": "miele_at_home_user@example.com",
    "password": "miele_at_home_password",
    "country-code": "de-DE",
    "mode": "sse"
  }
}
```

### With access token
```json
{
  "mqtt": {
    "url": "tcp://192.168.2.2:1883",
    "client-id": "miele-mqtt-gw",
    "username": "username",
    "password": "password",
    "retain": true,

    "topic": "home/miele",
    "deduplicate": true
  },

  "miele": {
    "client-id": "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
    "client-secret": "12345678901234567890123456789012",
    "mode": "sse",
    "token": {
      "access": "access_token",
      "refresh": "refresh_token"
    }
  }
}
```

#### Country code
Two-letter language/country code. Examples:
- `en-US`
- `de-DE` (default)
- `fr-FR`
- etc.

Make sure you have write access to the configuration file, so that the token can be persisted.

# Use server-sent events

Miele provides a server-sent-events API. To enable this, set the `mode`
property in your configuration to `SSE`. With SSE enabled, you will get faster notifications when some device state
changes. This is an experimental setting and not enabled by default.

# Deduplicate messages

When `deduplicate` is set to `true`, no duplicate MQTT messages will be sent.

# Bridge status

The bridge maintains two status topics:

## Topic: `.../bridge/state`

| Value     | Description                          |
| --------- | ------------------------------------ |
| `online`  | The bridge is started                |
| `offline` | The bridge is currently not started. |

## Topic: `.../bridge/miele`

| Value          | Description                                                                       |
| -------------- | --------------------------------------------------------------------------------- |
| `unknown`      | Unknown connection status (initial value)                                         |
| `connected`    | Miele API is connected (SSE healthy)                                              |
| `disconnected` | Miele API is not connected (brief failure below backoff threshold)                |
| `degraded`     | SSE has been failing but polling is still delivering device updates               |

### SSE failure backoff

In `mode: "sse"` the bridge always runs the polling loop alongside SSE as a
fallback. After several consecutive SSE failures (e.g. the upstream returning
`504 Gateway Time-out`) the reconnect delay is increased exponentially and the
bridge reports `degraded` on `bridge/miele` to signal that polling is carrying
device updates. The defaults are tunable in the config:

```json
{
  "miele": {
    "sse-backoff": {
      "failure-threshold": 5,
      "base-delay": "5s",
      "max-delay": "10m"
    }
  }
}
```

The `sse-backoff` block is optional. With it omitted, the defaults shown above
apply.

# Home Assistant integration

The bridge can publish Home Assistant
[MQTT-discovery](https://www.home-assistant.io/integrations/mqtt/#mqtt-discovery)
config so each appliance shows up automatically as a device tile with named
entities — no template YAML required. Discovery is **opt-in**:

```json
{
  "mqtt": {
    "deduplicate": true,
    "discovery": {
      "enabled": true,
      "prefix": "homeassistant",
      "device-name-prefix": "Miele"
    }
  }
}
```

All three discovery fields are optional; the values above are the defaults
applied when `enabled` is set to `true`. `mqtt.deduplicate: true` is
recommended alongside discovery — it suppresses the retained re-publish of
each discovery config on every device update, which otherwise adds steady
broker traffic.

## Topics produced

For every appliance the bridge sees, it publishes one retained discovery
config per entity:

```
<prefix>/sensor/miele_<id>/state/config
<prefix>/sensor/miele_<id>/phase/config
<prefix>/sensor/miele_<id>/remaining_duration/config
<prefix>/sensor/miele_<id>/remaining_minutes/config
<prefix>/sensor/miele_<id>/time_completed/config
```

`<id>` is `ident.deviceIdentLabel.fabNumber` from the Miele full payload
(globally unique per appliance). If that field is missing, the bridge falls
back to the Miele API device id and logs a warning.

## Entities

| Entity                | Source field (small message)    | Notes                                      |
| --------------------- | ------------------------------- | ------------------------------------------ |
| `state`               | `state`                         | Miele status (`RUNNING`, `OFF`, …)         |
| `phase`               | `phase`                         | Program phase (`DRYING`, `MAIN_WASH`, …)   |
| `remaining_duration`  | `remainingDuration`             | `HH:MM` string                             |
| `remaining_minutes`   | `remainingDurationMinutes`      | `unit: min`, `device_class: duration`      |
| `time_completed`      | `timeCompleted`                 | Wall-clock `HH:MM`                         |

Each entity's `state_topic` points at the existing `<mqtt.topic>/<deviceId>`
small-message JSON and uses `value_template` to pick out the relevant field —
the data plane is unchanged.

## Device registry mapping

| HA `device` field | Source                                       |
| ----------------- | -------------------------------------------- |
| `manufacturer`    | `"Miele"`                                    |
| `model`           | `ident.type.value_localized`, else `techType`|
| `sw_version`      | `ident.xkmIdentLabel.releaseVersion`         |
| `serial_number`   | The resolved `<id>` (fabNumber or fallback)  |
| `name`            | `<device-name-prefix> <type> <id>`           |

## Availability

Each entity uses `availability_mode: any` with two entries against
`<mqtt.topic>/bridge/miele` — one matching `connected`, one matching
`degraded`. HA shows the device as available in either healthy or degraded
state and only marks it unavailable on a true `disconnected`. (HA 2023.8 or
newer respects `availability_mode: any`; older versions fall back to a single
availability check that flips on `degraded`.)

## Removal on shutdown

On `SIGTERM` / `SIGINT`, the bridge publishes an empty retained payload to
every discovery topic it announced during the run, so HA removes the device
from its registry. A hard crash (kill -9, OOM, panic) leaves the retained
payloads in the broker; the next bridge start re-asserts them.

# run

Obtain your API credentials from https://www.miele.com/developer/

copy the `config-example.json` to `/production/config/config.json`

```
cd ./production
docker-compose up -d
```

## Logging

Set te timezone in the docker-compose file to your local timezone.

Example:

```
environment:
  TZ: "Europe/Berlin"
```

Set the log-level in the configuration file:
```json
{
  "loglevel": "info"
}
```

Valid log levels are:
`fatal`, `error`, `warn`, `info`, `debug`, `trace`

Not all levels are currently used.

# Diagnostics

The bridge exposes a small diagnostic HTTP listener on `:6060`:

- `/debug/vars` — Go `expvar` snapshot. Includes a `miele` object alongside the
  shared `mqtt` object:

  ```json
  {
    "miele": {
      "connection": "connected",
      "devices": {
        "000123456789": {
          "phase": "DRYING", "phaseId": 1799, "state": "RUNNING",
          "remainingDuration": "0:04", "remainingDurationMinutes": 4,
          "timeCompleted": "12:35"
        }
      },
      "sse":     { "last_event": "...", "events_total": 1234, "consecutive_failures": 0, "next_retry_after": "" },
      "polling": { "last_attempt": "...", "last_success": "...", "last_error": "", "success_total": 23, "error_total": 0 },
      "token":   { "expires_at": "...", "last_refresh": "...", "refresh_total": 5 }
    }
  }
  ```

  Tail a single field with `curl -s http://localhost:6060/debug/vars | jq .miele.sse`.
  `sse.consecutive_failures` and `sse.next_retry_after` reflect the
  exponential-backoff state when the upstream is degraded — see
  [Bridge status](#bridge-status).

- `/debug/pprof/*` — standard Go pprof endpoints (goroutines, heap, CPU, …).

`:6060` is intended to be reachable from a trusted network only — do not
expose it to the public internet. The container does not bind a public port
by default; use SSH port-forwarding or a private overlay network when you
need to inspect it from another host.

# build

The bridge is a Go application. The repository ships a `Makefile` and a
multi-stage `Dockerfile` (distroless final stage).

## Local build

The Go module lives under `app/`. Run the Make targets from there:

```
cd app
make build              # produces ./app/build/miele2mqtt
make test               # go test ./...
make vet                # go vet ./...
make run                # build then run against production/config/config.json
```

The binary takes the config-file path as its single argument:

```
./app/build/miele2mqtt /path/to/config.json
```

## Docker image

```
cd app
make image              # builds pharndt/mielemqtt:latest
```

The image is based on `gcr.io/distroless/static:nonroot`. Mount the config at
`/var/lib/miele-to-mqtt-gw/config.json` (matches the existing
`production/docker-compose.yaml`).

## Environment-variable substitution in the config

`${NAME}` placeholders in `config.json` are replaced with the value of the
`NAME` environment variable before parsing; missing variables become empty
strings. Example:

```json
{
  "miele": {
    "username": "${MIELE_USERNAME}",
    "password": "${MIELE_PASSWORD}"
  }
}
```

## openHAB configuration

see [openHAB example](openHAB.md)
