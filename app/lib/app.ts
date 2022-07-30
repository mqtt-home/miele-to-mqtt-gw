import EventSource from "eventsource"
import cron from "node-cron"
import { log } from "./logger"
import { login } from "./miele/login/login"
import { smallMessage } from "./miele/miele"
import { startSSE } from "./miele/sse-client"
import { connectMqtt, publish } from "./mqtt/mqtt-client"

export const triggerFullUpdate = async () => {
    eventSource?.close()
    await start()
}

let eventSource: EventSource

const start = async () => {
    const token = await login()

    const { sse, registerDevicesListener } = startSSE(token.access_token)

    registerDevicesListener((devices) => {
        for (const device of devices) {
            publish(smallMessage(device), device.id)
            publish(device.data, `${device.id}/full`)
        }
    })

    eventSource = sse
}

export const startApp = async () => {
    const mqttCleanUp = await connectMqtt()
    await triggerFullUpdate()
    await start()

    log.info("Application is now ready.")

    log.info("Scheduling token-update.")
    const task = cron.schedule("0 0 1,15 * *", triggerFullUpdate)
    task.start()

    return () => {
        mqttCleanUp()
        eventSource?.close()
        task.stop()
    }
}
