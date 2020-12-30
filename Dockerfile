# ---- Prod ----
FROM openjdk:8-jdk-alpine
RUN mkdir /opt/app
WORKDIR /opt/app
COPY src/de.rnd7.mieletomqtt/target/miele-to-mqtt-gw.jar .
COPY logback.xml .

ENV LOGBACK_XML ./miele-to-mqtt-gw.jar/logback.xml
CMD java -Dlogback.configurationFile=$LOGBACK_XML -jar ./miele-to-mqtt-gw.jar /var/lib/miele-to-mqtt-gw/config.json
