import { JEST_INTEGRATION_TIMEOUT } from "../../test/test-utils"
import { applyConfig } from "../config/config"
import { log } from "../logger"
import { getToken } from "./login/login"
import { testConfig } from "./miele-testutils"
import { MieleDevice } from "./miele-types"
import { startSSE } from "./sse-client"

jest.setTimeout(JEST_INTEGRATION_TIMEOUT)

describe("sse-client", () => {
    beforeAll(() => {
        applyConfig(testConfig())
        log.off()
    })

    test("integration", async () => {
        const token = await getToken()
        const { sse, registerDevicesListener } = startSSE(token.access_token)

        const devices = await new Promise<MieleDevice[]>((resolve) => {
            registerDevicesListener(devices => {
                resolve(devices)
            })
        })

        expect(devices.length).toBe(1)

        sse.close()
    })
})
