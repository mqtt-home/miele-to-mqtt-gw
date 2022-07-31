import { applyConfig } from "../config/config"
import { getToken } from "./login/login"
import { testConfig } from "./miele-testutils"
import { MieleDevice } from "./miele-types"
import { startSSE } from "./sse-client"

describe("sse-client", () => {
    beforeAll(() => {
        applyConfig(testConfig())
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
