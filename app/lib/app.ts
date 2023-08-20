import EventSource from "eventsource"
import cron from "node-cron"
import { getAppConfig } from "./config/config"
import { registerConnectionCheck, unregisterConnectionCheck } from "./connection"
import { log } from "./logger"
import { getCurrentToken, login, needsRefresh } from "./miele/login/login"
import { fetchDevices, smallMessage } from "./miele/miele"
import { startSSE } from "./miele/sse-client"
import { connectMqtt, publish } from "./mqtt/mqtt-client"

export const triggerFullUpdate = async () => {
    if (needsRefresh()) {
        log.info("Token refresh required. Reconnecting now.")
        await restart()
    }
}

export const polling = async () => {
    if (getAppConfig().miele.mode !== "polling") {
        return
    }

    const token = getCurrentToken()
    if (token) {
        log.debug("Polling started")
        const devices = await fetchDevices(token)
        log.debug("Polling done")
        for (const device of devices) {
            publish(smallMessage(device), device.id)
            publish(device.data, `${device.id}/full`)
        }
    }
    else {
        log.warn("Polling skipped (no token)")
    }
}

const restart = async () => {
    eventSource?.close()
    unregisterConnectionCheck()
    await start()
}

let eventSource: EventSource

const start = async () => {
    const token = await (login())

    const { sse, registerDevicesListener } = startSSE(token.access_token, restart)

    registerDevicesListener((devices) => {
        for (const device of devices) {
            publish(smallMessage(device), device.id)
            publish(device.data, `${device.id}/full`)
        }
    })

    registerConnectionCheck(restart)

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

        const pollingTask = cron.schedule("* * * * *", polling)
        pollingTask.start()

        return () => {
            mqttCleanUp()
            eventSource?.close()
            unregisterConnectionCheck()
            task.stop()
            pollingTask.stop()
        }
    }
    catch (e) {
        log.error("Application failed to start", e)
        process.exit(1)
    }
}
