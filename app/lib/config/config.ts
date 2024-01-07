import Duration from "@icholy/duration"
import * as fs from "fs"
import { log } from "../logger"
import { add } from "../miele/duration"
import { setToken } from "../miele/login/login"

export type ConfigMqtt = {
    url: string,
    topic: string
    username?: string
    password?: string
    retain: boolean
    qos: (0 | 1 | 2)
    "bridge-info"?: boolean
    "bridge-info-topic"?: string
}

export type ConfigToken = {
    access: string,
    refresh: string
    validUntil?: string
}

export type ConfigMiele = {
    "client-id": string
    "client-secret": string
    "country-code": string
    username: string
    password: string

    mode: "sse" | "polling"
    "polling-interval"?: number

    token?: ConfigToken

    "connection-check-interval": number
}

export type Config = {
    mqtt: ConfigMqtt
    miele: ConfigMiele
    names: any,
    "send-full-update": boolean
    loglevel: string
}

let appConfig: Config

const mqttDefaults = {
    qos: 1,
    retain: true,
    "bridge-info": true
}

const mieleDefaults = {
    mode: "sse",
    "country-code": "de-DE",
    "connection-check-interval": 10_000
}

const configDefaults = {
    "send-full-update": true,
    loglevel: "info"
}

export const applyDefaults = (config: any) => {
    return {
        ...configDefaults,
        ...config,
        miele: { ...mieleDefaults, ...config.miele },
        mqtt: { ...mqttDefaults, ...config.mqtt }
    } as Config
}

let configFile: string

export const replaceEnvVariables = (input: string) => {
    const envVariableRegex = /\${([^}]+)}/g

    return input.replace(envVariableRegex, (_, envVarName) => {
        return process.env[envVarName] || ""
    })
}

export const loadConfig = (file: string) => {
    configFile = file
    const buffer = fs.readFileSync(file)
    const effectiveConfig = replaceEnvVariables(buffer.toString())
    log.trace("Using config", effectiveConfig)
    log.trace("parsing config")
    applyConfig(JSON.parse(effectiveConfig))
    recoverToken()
    return appConfig
}

const equals = (obj1: any, obj2: any) => {
    return JSON.stringify(obj1) === JSON.stringify(obj2)
}

export const persistToken = (token: ConfigToken) => {
    if (!configFile) {
        log.warn("No config file set. Not persisting token.")
        return
    }

    try {
        const buffer = fs.readFileSync(configFile)
        const config: Config = JSON.parse(buffer.toString())
        if (!equals(config.miele.token, token)) {
            log.info("Persisting token to config file", configFile)
            config.miele.token = token
            fs.writeFileSync(configFile, JSON.stringify(config, null, 2))
        }
        else {
            log.debug("Token did not change. Not persisting.")
        }
    }
    catch (e) {
        log.error("Failed to persist token to config file", configFile, e)
    }
}

export const recoverToken = () => {
    const token = appConfig.miele.token
    if (token) {
        log.info("Recovering token")
        let validUntil: Date | undefined
        if (token.validUntil) {
            validUntil = new Date(token.validUntil)
        }

        if (!validUntil) {
            validUntil = add(new Date(), Duration.hours(1))
        }

        setToken({
            access_token: token.access,
            refresh_token: token.refresh,
            token_type: "Bearer",
            expiresAt: validUntil
        })
    }
}

export const applyConfig = (config: any) => {
    appConfig = applyDefaults(config)
    log.configure(appConfig.loglevel.toUpperCase())
}

export const getAppConfig = () => {
    return appConfig
}
