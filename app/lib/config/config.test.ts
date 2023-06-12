import Duration from "@icholy/duration"
import * as fs from "fs"
import * as os from "os"
import path from "path"
import { log } from "../logger"
import { add } from "../miele/duration"
// eslint-disable-next-line camelcase
import { __TEST_getToken, setToken } from "../miele/login/login"
import { applyConfig, applyDefaults, getAppConfig, loadConfig, persistToken, recoverToken } from "./config"

describe("Config", () => {
    test("default values", async () => {
        const config = {
            mqtt: {
                url: "tcp://192.168.1.1:1883",
                topic: "miele"
            }
        }

        expect(applyDefaults(config)).toStrictEqual({
            loglevel: "info",
            miele: {
                "connection-check-interval": 10000,
                "country-code": "de-DE",
                mode: "sse"
            },
            mqtt: {
                "bridge-info": true,
                qos: 1,
                retain: true,
                topic: "miele",
                url: "tcp://192.168.1.1:1883"
            },
            "send-full-update": true
        })

        expect(applyDefaults(config)["send-full-update"]).toBeTruthy()
    })

    test("disable send-full-update", async () => {
        const config = {
            mqtt: {
                url: "tcp://192.168.1.1:1883",
                topic: "hue"
            },
            hue: {
                host: "192.168.1.1",
                "api-key": "some-api-key"
            },
            "send-full-update": false
        }

        expect(applyDefaults(config)["send-full-update"]).toBeFalsy()
    })

    test("load from file", () => {
        loadConfig(path.join(__dirname, "../../../production/config/config-example.json"))
        log.off()
        expect(getAppConfig().miele.mode).toBe("sse")
    })

    describe("Token recover", () => {
        beforeEach(() => {
            log.off()
            setToken(undefined)
        })

        test("recover token", () => {
            applyConfig({
                miele: {
                    token: {
                        access: "access_token",
                        refresh: "refresh_token",
                        validUntil: add(new Date(), Duration.days(7)).toISOString()
                    }
                }
            })
            log.off()
            recoverToken()
            expect(__TEST_getToken()!.access_token).toBe("access_token")
        })

        test("cannot recover token", () => {
            applyConfig({
            })
            log.off()
            recoverToken()
            expect(__TEST_getToken()).toBeFalsy()
        })
    })

    describe("Persist token", () => {
        beforeEach(() => {
            log.off()
            setToken(undefined)
        })

        const token = {
            access: "access-token",
            refresh: "refresh-token",
            validUntil: "valid-until"
        }

        test("Persist token", () => {
            const tmp = fs.mkdtempSync(path.join(os.tmpdir(), "miele-mqtt-test"))
            const config = path.join(tmp, "config.json")
            fs.writeFileSync(config, JSON.stringify({
                mqtt: {
                    url: "tcp://192.168.1.1:1883",
                    topic: "miele"
                },
                miele: {

                }
            }))

            loadConfig(config)
            log.off()

            persistToken(token)

            const data = fs.readFileSync(config)
            const configuration = JSON.parse(data.toString("utf-8"))
            expect(configuration).toStrictEqual(
                {
                    mqtt: { url: "tcp://192.168.1.1:1883", topic: "miele" },
                    miele: {
                        token
                    }
                }
            )
        })

        test("No token change", () => {
            const tmp = fs.mkdtempSync(path.join(os.tmpdir(), "miele-mqtt-test"))
            const config = path.join(tmp, "config.json")
            fs.writeFileSync(config, JSON.stringify({
                mqtt: {
                    url: "tcp://192.168.1.1:1883",
                    topic: "miele"
                },
                miele: {
                    token
                },
                loglevel: "error"
            }))

            loadConfig(config)
            log.off()
            fs.writeFileSync(config, JSON.stringify({
                miele: {
                    token
                }
            }))
            persistToken({
                access: "access-token",
                refresh: "refresh-token",
                validUntil: "valid-until"
            })

            const data = fs.readFileSync(config)
            const configuration = JSON.parse(data.toString("utf-8"))
            expect(configuration).toStrictEqual(
                {
                    miele: {
                        token
                    }
                }
            )
        })
    })
})
