import { getAppConfig } from "./config/config"
import { log } from "./logger"
import { ping } from "./miele/miele"

let checkConnection: ReturnType<typeof setTimeout> | undefined
let connectionLost = false

export const unregisterConnectionCheck = () => {
    checkConnection?.unref()
    checkConnection = undefined
}

let check = ping

// eslint-disable-next-line camelcase
export const __TEST_setCheck = (newCheck: () => Promise<boolean>) => {
    check = newCheck
}

export const registerConnectionCheck = (restartHook: () => Promise<void>, config = getAppConfig().miele) => {
    const interval = config["connection-check-interval"]
    if (checkConnection) {
        log.debug("Connection check already registered")
        return checkConnection
    }
    if (interval === 0) {
        log.debug("Internet connection check disabled")
        return
    }
    log.info("Internet connection will be checked every", { ms: interval })
    connectionLost = false
    checkConnection = setInterval(async () => {
        log.debug("Checking connection")

        if (!await check()) {
            log.debug("Connection check failed")
            if (!connectionLost) {
                connectionLost = true
                log.error("Connection lost. Waiting for connection to come back.")
            }
        }
        else if (connectionLost) {
            log.debug("Connection check success after connection was lost")
            await restartHook()
        }
        else {
            log.debug("Connection check success")
        }
    }, interval)

    return checkConnection
}
