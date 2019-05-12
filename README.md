# miele-to-mqtt-gw

Convert the miele@home data to mqtt messages

This application will post two MQTT messages for each connected device.
One short message and a full message.

## Miele REST API state

Please note that the Miele REST API (public API - https://www.miele.com/developer/swagger-ui/index.html) will currently not work. The token request will complete sucessfully but the REST call using bearer authentication fails with 401. Miele is informed about the issue.

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

# build

build the docker container using `build.sh`

# run

Obtain you API credentials from https://www.miele.com/developer/

copy the `config-example.json` to `/production/config/config.json`
```
cd ./production
docker-compose up -d
```

## openHAB configuration

see [openHAB example](openHAB.md)
