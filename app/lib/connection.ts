import dns from "dns"
import { getAppConfig } from "./config/config"
import { log } from "./logger"

let checkConnection: ReturnType<typeof setTimeout>
let connectionLost = false

export const unregisterConnectionCheck = () => {
    checkConnection?.unref()
}

export const registerConnectionCheck = (restartHook: () => Promise<void>, config = getAppConfig().miele) => {
    const interval = config["connection-check-interval"]
    if (interval === 0) {
        log.debug("Internet connection check disabled")
        return
    }
    log.info("Internet connection will be checked every", { ms: interval })
    connectionLost = false
    checkConnection = setInterval(() => {
        log.debug("Checking connection")
        dns.resolve("api.mcs3.miele.com", (err) => {
            if (err) {
                log.debug("Connection check failed", err)
                if (!connectionLost) {
                    connectionLost = true
                    log.error("Connection lost. Waiting for connection to come back.", err)
                }
            }
            else if (connectionLost) {
                log.debug("Connection check success after connection was lost")
                restartHook().then()
            }
            else {
                log.debug("Connection check success")
            }
        })
    }, interval)
}
