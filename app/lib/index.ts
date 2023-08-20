/* istanbul ignore file */
import * as path from "path"
import { startApp } from "./app"

import { loadConfig } from "./config/config"
import { log } from "./logger"

if (process.argv.length !== 3) {
    log.error("Expected config file as argument.")
    process.exit(1)
}

let configFile = process.argv[2]
configFile = configFile.startsWith(".") ? path.join(__dirname, "..", configFile) : configFile
log.info("Using config from file", configFile)
const config = loadConfig(configFile)

if (config.miele.mode === "polling") {
    log.info("Polling mode enabled. SSE is still active, using polling as fallback.")
}

startApp().then()
