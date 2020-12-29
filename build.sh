#!/bin/bash
set MIELE_CLIENT_ID [lindex $argv 0];
set MIELE_CLIENT_SECRET [lindex $argv 1];
set MIELE_PASSWORD [lindex $argv 2];
set MIELE_USERNAME [lindex $argv 2];

docker build -t pharndt/mielemqtt \
--build-arg MIELE_CLIENT_ID=$MIELE_CLIENT_ID \
--build-arg MIELE_CLIENT_SECRET=$MIELE_CLIENT_SECRET \
--build-arg MIELE_PASSWORD=$MIELE_PASSWORD \
--build-arg MIELE_USERNAME=$MIELE_USERNAME \
.
