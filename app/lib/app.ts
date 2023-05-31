import EventSource from "eventsource"
import cron from "node-cron"
import { log } from "./logger"
import { login, needsRefresh } from "./miele/login/login"
import { smallMessage } from "./miele/miele"
import { startSSE } from "./miele/sse-client"
import { connectMqtt, publish } from "./mqtt/mqtt-client"

export const triggerFullUpdate = async () => {
    if (needsRefresh()) {
        log.info("Token refresh required. Reconnecting now.")
        eventSource?.close()
        await start()
    }
}

let eventSource: EventSource

const start = async () => {
    const token = await (login())

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
    try {
        const mqttCleanUp = await connectMqtt()
        await start()
        await triggerFullUpdate()
        log.info("Application is now ready.")

        log.info("Scheduling token-update.")
        const task = cron.schedule("* * * * *", triggerFullUpdate)
        task.start()

        return () => {
            mqttCleanUp()
            eventSource?.close()
            task.stop()
        }
    }
    catch (e) {
        log.error("Application failed to start", e)
        process.exit(1)
    }
}
