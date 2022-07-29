/* istanbul ignore file */
import { log } from "./logger"

import { loadConfig } from "./config/config"
import { startApp } from "./app"
import * as path from "path"

if (process.argv.length !== 3) {
    log.error("Expected config file as argument.")
    process.exit(1)
}

let configFile = process.argv[2]
configFile = configFile.startsWith(".") ? path.join(__dirname, "..", configFile) : configFile
log.info(`Using config from file ${configFile}`)
loadConfig(configFile)

startApp().then()
