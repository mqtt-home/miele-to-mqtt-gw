# miele-to-mqtt-gw

[![mqtt-smarthome](https://img.shields.io/badge/mqtt-smarthome-blue.svg)](https://github.com/mqtt-smarthome/mqtt-smarthome)

Convert the miele@home data to MQTT messages

This application will post two MQTT messages for each connected device: one short message and a full message.

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

```json
{
  "mqtt": {
    "url": "tcp://192.168.2.2:1883",
    "client-id": "miele-mqtt-gw",
    "username": "username",
    "password": "password",
    "retain": true,

    "message-interval": 30,
    "full-message-topic": "home/miele"
  },

  "miele": {
    "client-id": "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
    "client-secret": "12345678901234567890123456789012",
    "username": "miele_at_home_user@example.com",
    "password": "miele_at_home_password",
    "mode": "polling"
  },
  
  "deduplicate": true,
  "timezone": "GMT+1"
}
```

# Use server-sent events

Miele provides a server-sent-events API. To enable this, set the `mode`
property in your configuration to `SSE`. With SSE enabled, you will get faster notifications when some device state
changes. This is an experimental setting and not enabled by default.

# Deduplicate messages

When `deduplicate` is set to `true`, no duplicate MQTT messages will be sent.

# Bridge state

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

# build

## Precondition

To execute test cases against the real Miele API, you need to set some environment variables.

| Name                | Value               |
| ------------------- | ------------------- |
| MIELE_CLIENT_ID     | your client id      |
| MIELE_CLIENT_SECRET | your client secret  |
| MIELE_USERNAME      | your Miele username |
| MIELE_PASSWORD      | your Miele password |

This is necessary to verify the login method is still working, and the API has not been changed incompatible.

Then, build the docker container using `build.sh`.

# run

Obtain your API credentials from https://www.miele.com/developer/

copy the `config-example.json` to `/production/config/config.json`

```
cd ./production
docker-compose up -d
```

## Logging

When you like to use a custom logging configuration, you can set the environment variable `LOGBACK_XML` in your compose
file and put a `logback.xml` to the config folder.

Example:

```
environment:
  TZ: "Europe/Berlin"
  LOGBACK_XML: /var/lib/miele-to-mqtt-gw/logback.xml
```

## openHAB configuration

see [openHAB example](openHAB.md)
