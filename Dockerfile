FROM openjdk:8-jdk-alpine

LABEL maintainer="Philipp Arndt <2f.mail@gmx.de>"
LABEL version="1.0"
LABEL description="miele@home to mqtt gateway"


ENV LANG en_US.UTF-8
ENV TERM xterm

WORKDIR /opt/miele-to-mqtt-gw

RUN apk update --no-cache && apk add --no-cache maven

COPY src /opt/miele-to-mqtt-gw

RUN mvn install assembly:single
RUN cp ./de.rnd7.mieletomqtt/target/miele-to-mqtt-gw.jar ./miele-to-mqtt-gw.jar

CMD java -jar miele-to-mqtt-gw.jar /var/lib/miele-to-mqtt-gw/config.json
