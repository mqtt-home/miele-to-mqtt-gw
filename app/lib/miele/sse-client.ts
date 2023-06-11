import EventSource from "eventsource"
import { log } from "../logger"
import { convertDevices } from "./miele"
import { MieleDevice } from "./miele-types"

type DevicesListener = (devices: MieleDevice[]) => void

export const startSSE = (token: string, restart: () => Promise<void>) => {
    log.info("Starting Server-Sent events")

    const eventSourceInitDict = {
        headers: {
            "Accept-Language": "en-GB",
            Authorization: "Bearer " + token,
            Accept: "text/event-stream"
        }
    }

    const sse = new EventSource("https://api.mcs3.miele.com/v1/devices/all/events", eventSourceInitDict)
    // setInterval(() => log.debug("SSE", sse), 1000)
    sse.onerror = (err: any) => {
        if (err) {
            log.error("SSE error", err)
            log.info("Restarting SSE")
            restart().then()
        }
    }
    sse.addEventListener("open", () => log.info("SSE connection opened"))
    sse.addEventListener("error", (err) => log.error("SSE error", err))
    sse.addEventListener("close", () => log.info("SSE connection closed"))

    const registerDevicesListener = (listener: DevicesListener) => {
        sse.addEventListener("devices", (event) => listener(convertDevices(JSON.parse(event.data))))
    }

    return {
        sse,
        registerDevicesListener
    }
}
