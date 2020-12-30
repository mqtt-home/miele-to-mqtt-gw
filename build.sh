#!/bin/bash
cd "$(dirname "$0")"
cd src

docker build -t pharndt/mielemqtt \
--build-arg MIELE_CLIENT_ID=$MIELE_CLIENT_ID \
--build-arg MIELE_CLIENT_SECRET=$MIELE_CLIENT_SECRET \
--build-arg MIELE_PASSWORD=$MIELE_PASSWORD \
--build-arg MIELE_USERNAME=$MIELE_USERNAME \
.
