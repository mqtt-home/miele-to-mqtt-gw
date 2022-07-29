import { connectMqtt } from "./mqtt/mqtt-client"
import { startSSE } from "./miele/SSEClient"
import { log } from "./logger"
import cron from "node-cron"

export const triggerFullUpdate = async () => {
}

export const startApp = async () => {
    const mqttCleanUp = await connectMqtt()
    await triggerFullUpdate()

    const sse = startSSE("")
    sse.addEventListener("message", event => {
        for (const data of JSON.parse(event.data)) {
            // takeEvent(data)
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
