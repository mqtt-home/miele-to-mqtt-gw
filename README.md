# miele-to-mqtt-gw

Convert the miele@home data to mqtt messages

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
