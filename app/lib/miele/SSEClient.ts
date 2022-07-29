import EventSource from "eventsource"
import { log } from "../logger"

export const startSSE = (token: string) => {
    log.info("Starting Server-Sent events")

    const eventSourceInitDict = {
        headers: {
            "Accept-Language": "en-GB",
            Authorization: "Bearer " + token,
            Accept: "text/event-stream"
        }
    }

    const sse = new EventSource("https://api.mcs3.miele.com/v1/devices/all/events", eventSourceInitDict)
    sse.onerror = (err: any) => {
        if (err) {
            log.error(err)
        }
    }
    return sse
}
