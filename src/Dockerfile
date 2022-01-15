# ---- Build ----
FROM openjdk:18-jdk-alpine as builder

ARG MIELE_CLIENT_ID
ARG MIELE_CLIENT_SECRET
ARG MIELE_PASSWORD
ARG MIELE_USERNAME

ENV MIELE_CLIENT_ID $MIELE_CLIENT_ID
ENV MIELE_CLIENT_SECRET $MIELE_CLIENT_SECRET
ENV MIELE_PASSWORD $MIELE_PASSWORD
ENV MIELE_USERNAME $MIELE_USERNAME

LABEL maintainer="Philipp Arndt <2f.mail@gmx.de>"
LABEL version="1.0"
LABEL description="miele@home to mqtt gateway"


ENV LANG en_US.UTF-8
ENV TERM xterm

WORKDIR /opt/miele-to-mqtt-gw

RUN apk update --no-cache && apk add --no-cache maven

COPY . /opt/miele-to-mqtt-gw

RUN mvn -B package

# ---- Prod ----
FROM openjdk:18-jdk-alpine
RUN mkdir /opt/app
WORKDIR /opt/app
COPY --from=builder /opt/miele-to-mqtt-gw/de.rnd7.mieletomqtt/target/miele-to-mqtt-gw.jar .
COPY logback.xml .

ENV LOGBACK_XML ./miele-to-mqtt-gw.jar/logback.xml
CMD java -Dlogback.configurationFile=$LOGBACK_XML -jar ./miele-to-mqtt-gw.jar /var/lib/miele-to-mqtt-gw/config.json
