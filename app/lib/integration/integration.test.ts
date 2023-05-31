import * as Buffer from "buffer"
import path from "path"
import { GenericContainer, StartedTestContainer, Wait } from "testcontainers"
import { JEST_INTEGRATION_TIMEOUT, JEST_DEFAULT_TIMEOUT } from "../../test/test-utils"
import { startApp } from "../app"
import { applyConfig, ConfigMqtt, getAppConfig } from "../config/config"
import { log } from "../logger"
import { testConfig } from "../miele/miele-testutils"
import { createMqttInstance, MqttInstance, subscribe } from "../mqtt/mqtt-client"
jest.setTimeout(JEST_INTEGRATION_TIMEOUT)

type Message = {
    topic: string
    payload: any
}

type CleanUpTask = () => void

describe("Integration test", () => {
    let mqtt: StartedTestContainer
    const cleanUpTasks: CleanUpTask[] = []

    /* eslint-disable no-async-promise-executor */
    const waitForMessages = (instance: MqttInstance, config: ConfigMqtt) => {
        return new Promise<Message[]>(async (resolve) => {
            const client = instance.client

            const messages: Message[] = []
            const onMessage = (topic: string, payload: Buffer) => {
                messages.push({
                    topic,
                    payload: JSON.parse(payload.toString("utf-8"))
                })

                if (messages.length === 3) {
                    resolve(messages)
                }
            }

            client.on("message", onMessage)

            await subscribe(client, `${config.topic}/#`)
            const cleanUp = await startApp()
            cleanUpTasks.push(() => cleanUp())
        })
    }

    beforeAll(async () => {
        log.off()

        const buildRoot = path.resolve(__dirname, "../../test")
        const mqttContainer = await GenericContainer.fromDockerfile(path.resolve(buildRoot, "activemq"))
            .build()

        mqtt = await mqttContainer
            .withExposedPorts(1883, 8161)
            .withHealthCheck({
                test: ["CMD-SHELL", "curl -f http://localhost:8161 || exit 1"]
            })
            .withWaitStrategy(Wait.forHealthCheck())
            .start()

        applyConfig({
            ...testConfig(),
            mqtt: {
                url: `tcp://${mqtt.getHost()}:${mqtt.getMappedPort(1883)}`,
                topic: "miele"
            }
        })
    })

    afterAll(async () => {
        cleanUpTasks.forEach(task => {
            try {
                task()
            }
            catch (e) {
                // do nothing
            }
        })

        await mqtt?.stop()
        jest.setTimeout(JEST_DEFAULT_TIMEOUT)
        await process.nextTick(() => {})
    })

    test("Message is published", async () => {
        const config = getAppConfig().mqtt
        const instance = await createMqttInstance(config)

        const messages = await waitForMessages(instance, config)
        expect(messages.length).toBe(3)
        expect(messages[0]).toStrictEqual({
            topic: "miele/bridge/state",
            payload: "online"
        })

        instance.exit()
    })
})
