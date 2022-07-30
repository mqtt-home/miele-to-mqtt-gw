import cron from "node-cron"
import { log } from "./logger"
import { login } from "./miele/login/login"
import { smallMessage } from "./miele/miele"
import { startSSE } from "./miele/sse-client"
import { connectMqtt, publish } from "./mqtt/mqtt-client"

export const triggerFullUpdate = async () => {
}

export const startApp = async () => {
    const mqttCleanUp = await connectMqtt()
    await triggerFullUpdate()

    const token = await login()

    const { sse, registerDevicesListener } = startSSE(token.access_token)

    registerDevicesListener((devices) => {
        for (const device of devices) {
            log.info(JSON.stringify(smallMessage(device)))
            publish(smallMessage(device), device.id)
            publish(device.data, `${device.id}/full`)
        }
    })
    log.info("Application is now ready.")

    log.info("Scheduling hourly-full-update.")
    const task = cron.schedule("0 * * * *", triggerFullUpdate)
    task.start()

    return () => {
        mqttCleanUp()
        sse.close()
        task.stop()
    }
}
