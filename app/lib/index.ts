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
    log.error("Polling mode is not supported for version >= 3.x. Please use version 2.x when you like to use the polling mode.")
    process.exit(1)
}

startApp().then()
