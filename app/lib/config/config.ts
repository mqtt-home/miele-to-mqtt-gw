import * as fs from "fs"

export type ConfigMqtt = {
    url: string,
    topic: string
    username?: string
    password?: string
    retain: boolean
    qos: (0|1|2)
    "bridge-info"?: boolean
    "bridge-info-topic"?: string
}

export type ConfigMiele = {
    "client-id": string
    "client-secret": string
    username: string
    password: string

    mode: "sse"|"polling"
    "polling-interval": number

    token?: {
        access: string,
        refresh: string
        validUntil: string
    }
}

export type Config = {
    mqtt: ConfigMqtt
    miele: ConfigMiele
    names: any,
    "send-full-update": boolean
}

let appConfig: Config

const mqttDefaults = {
    qos: 1,
    retain: true,
    "bridge-info": true
}

const mieleDefaults = {
    mode: "polling",
    "polling-interval": 60
}

const configDefaults = {
    "send-full-update": true
}

export const applyDefaults = (config: any) => {
    return {
        ...configDefaults,
        ...config,
        miele: { ...mieleDefaults, ...config.hue },
        mqtt: { ...mqttDefaults, ...config.mqtt }
    } as Config
}

export const loadConfig = (file: string) => {
    const buffer = fs.readFileSync(file)
    applyConfig(JSON.parse(buffer.toString()))
}

export const applyConfig = (config: any) => {
    appConfig = applyDefaults(config)
}

export const getAppConfig = () => {
    return appConfig
}

export const setTestConfig = (config: Config) => {
    appConfig = config
}
