import cron from "node-cron"
import { log } from "./logger"
import { login } from "./miele/login/login"
import { convertDevices, smallMessage } from "./miele/miele"
import { startSSE } from "./miele/SSEClient"
import { connectMqtt } from "./mqtt/mqtt-client"

export const triggerFullUpdate = async () => {
}

export const startApp = async () => {
    const mqttCleanUp = await connectMqtt()
    await triggerFullUpdate()

    const token = await login()

    const sse = startSSE(token.access_token)
    sse.addEventListener("devices", (event) => {
        for (const device of convertDevices(JSON.parse(event.data))) {
            log.info(JSON.stringify(smallMessage(device)))
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
