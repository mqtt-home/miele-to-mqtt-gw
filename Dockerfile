# ---- Prod ----
FROM openjdk:17-jdk-alpine
LABEL maintainer="Philipp Arndt <2f.mail@gmx.de>"
LABEL version="1.0"
LABEL description="Miele to mqtt gateway"

RUN mkdir /opt/app
WORKDIR /opt/app
COPY src/de.rnd7.mieletomqtt/target/miele-to-mqtt-gw.jar .
COPY src/logback.xml .

ENV LOGBACK_XML ./miele-to-mqtt-gw.jar/logback.xml
CMD java -Dlogback.configurationFile=$LOGBACK_XML -jar ./miele-to-mqtt-gw.jar /var/lib/miele-to-mqtt-gw/config.json
