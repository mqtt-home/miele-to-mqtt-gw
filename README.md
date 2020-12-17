# miele-to-mqtt-gw

Convert the miele@home data to mqtt messages

This application will post two MQTT messages for each connected device.
One short message and a full message.

## Example short message

The short message is already parsed/interpreted and contatins only the most relevant 
information.

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
    "password": "miele_at_home_password"
  },

  "timezone": "GMT+1"
}
```

# build

build the docker container using `build.sh`

# run

Obtain you API credentials from https://www.miele.com/developer/

copy the `config-example.json` to `/production/config/config.json`
```
cd ./production
docker-compose up -d
```

## Logging

When you like to use a custom logging configuration, you can set the environment
variable `LOGBACK_XML` in your compose file and put a `logback.xml`to the config folder.

Example:
```
environment:
  TZ: "Europe/Berlin"
  LOGBACK_XML: /var/lib/miele-to-mqtt-gw/logback.xml
```

## openHAB configuration

see [openHAB example](openHAB.md)
