FROM node:18.17-alpine
COPY app/dist /opt/app/
WORKDIR /opt/app/

CMD ["node", "index.js", "/var/lib/miele-to-mqtt-gw/config.json"]
