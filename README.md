# miele-to-mqtt-gw

[![mqtt-smarthome](https://img.shields.io/badge/mqtt-smarthome-blue.svg)](https://github.com/mqtt-smarthome/mqtt-smarthome)

Convert the miele@home data to MQTT messages

This application will post two MQTT messages for each connected device: one short message and a full message.

# Releases

## Production (2.x)
The current production version is 2.x and is implemented in Java.
See https://github.com/mqtt-home/miele-to-mqtt-gw/tree/2.x-java

## Prerelease (3.x)
The prerelease is version 3.x and implemented in TypeScript.

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

| Value          | Description                |
| -------------- | -------------------------- |
| `unknown`      | Unknown connection status  |
| `connected`    | Miele API is connected     |
| `disconnected` | Miele API is not connected |

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

# build

## GitHub access token

Make sure you have a GitHub access token in your `~/.m2/settings.xml`
```xml
<servers>
  <server>
    <id>github</id>
    <username>your username</username>
    <password>your access token</password>
  </server>
</servers>
```

See https://docs.github.com/en/packages/guides/configuring-apache-maven-for-use-with-github-packages

## Test cases against real Miele API

To execute test cases against the real Miele API, you need to set some environment variables.

| Name                | Value               |
| ------------------- | ------------------- |
| MIELE_CLIENT_ID     | your client id      |
| MIELE_CLIENT_SECRET | your client secret  |
| MIELE_USERNAME      | your Miele username |
| MIELE_PASSWORD      | your Miele password |

This is necessary to verify the login method is still working, and the API has not been changed incompatible.

## Build container

Build the docker container using `build.sh`.

## openHAB configuration

see [openHAB example](openHAB.md)
