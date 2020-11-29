# ---- Build ----
FROM gradle:jdk14 as build

LABEL maintainer="Philipp Arndt <2f.mail@gmx.de>"
LABEL version="1.0"
LABEL description="miele@home to mqtt gateway"


ENV LANG en_US.UTF-8
ENV TERM xterm

WORKDIR /opt/miele-to-mqtt-gw

RUN apt-get update && apt-get install -y \
  maven \
  && rm -rf /var/lib/apt/lists/*

COPY src /opt/miele-to-mqtt-gw

RUN mvn install assembly:single

# ---- Prod ----
FROM gradle:jre14
RUN mkdir /opt/app
WORKDIR /opt/app
COPY --from=build /opt/miele-to-mqtt-gw/de.rnd7.mieletomqtt/target/miele-to-mqtt-gw.jar .
COPY logback.xml .

ENV LOGBACK_XML ./miele-to-mqtt-gw.jar/logback.xml
CMD java -Dlogback.configurationFile=$LOGBACK_XML -jar ./miele-to-mqtt-gw.jar /var/lib/miele-to-mqtt-gw/config.json
