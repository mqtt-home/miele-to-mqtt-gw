import path from "path"
import { applyDefaults, getAppConfig, loadConfig } from "./config"

describe("Config", () => {
    test("default values", async () => {
        const config = {
            mqtt: {
                url: "tcp://192.168.1.1:1883",
                topic: "hue"
            }
        }

        expect(applyDefaults(config)).toStrictEqual({
            miele: {
                mode: "sse"
            },
            mqtt: {
                "bridge-info": true,
                qos: 1,
                retain: true,
                topic: "hue",
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
        expect(getAppConfig().miele.mode).toBe("sse")
    })
})
