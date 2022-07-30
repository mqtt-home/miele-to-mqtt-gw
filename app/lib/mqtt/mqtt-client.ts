import mqtt from "mqtt"

import { ConfigMqtt, getAppConfig } from "../config/config"
import { log } from "../logger"

export type MqttInstance = {
    client: mqtt.MqttClient
    exit: () => void
}

export const makeId = (length: number) => {
    let result = ""
    const characters = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
    const charactersLength = characters.length
    for (let i = 0; i < length; i++) {
        result += characters.charAt(Math.floor(Math.random() *
            charactersLength))
    }
    return result
}

let client: mqtt.MqttClient

export const publish = (message: any, topic: string) => {
    const config = getAppConfig()
    const fullTopic = `${config.mqtt.topic}/${topic}`
    publishAbsolute(message, fullTopic)
}

export const publishAbsolute = (message: any, fullTopic: string) => {
    const config = getAppConfig()
    if (!client) {
        log.error(`MQTT not available, cannot publish to ${fullTopic}`)
        return
    }

    const body = JSON.stringify(message, (key, value) => {
        if (value !== null) {
            return value
        }
    })
    client.publish(fullTopic, body, { retain: config.mqtt.retain })
}

const brideTopic = () => {
    const config = getAppConfig()
    return config.mqtt["bridge-info-topic"] ?? `${config.mqtt.topic}/bridge/state`
}

const online = () => {
    const config = getAppConfig()
    if (config.mqtt["bridge-info"]) {
        publishAbsolute("online", brideTopic())
    }
}

const willMessage = () => {
    const config = getAppConfig()
    if (config.mqtt["bridge-info"]) {
        return { topic: brideTopic(), payload: "offline", qos: config.mqtt.qos, retain: config.mqtt.retain }
    }
    else {
        return undefined
    }
}

export const subscribe = (client: mqtt.MqttClient, topic: string) => {
    return new Promise((resolve, reject) => {
        client.subscribe(topic, (err) => {
            if (!err) {
                resolve(undefined)
            }
            else {
                reject(err)
            }
        })
    })
}

export const connectMqtt: (() => Promise<() => void>) = async (config = getAppConfig().mqtt) => {
    const instance = await createMqttInstance(config)
    client = instance.client

    await subscribe(client, `${config.topic}/#`)
    online()
    log.info("MQTT subscription active")
    return client.end
}

export const createMqttInstance = (config: ConfigMqtt) => {
    const options = {
        clean: true,
        connectTimeout: 4000,
        clientId: makeId(9),
        username: config.username,
        password: config.password,
        will: willMessage()
    }

    return new Promise<MqttInstance>((resolve) => {
        const instance = mqtt.connect(config.url, options)
        instance.on("connect", () => {
            log.info("MQTT Connected")
            resolve({
                client: instance,
                exit: () => instance.end()
            })
        })
    })
}
