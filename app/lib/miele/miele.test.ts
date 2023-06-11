import { JEST_INTEGRATION_TIMEOUT } from "../../test/test-utils"
import { applyConfig } from "../config/config"
import { log } from "../logger"
import { getToken } from "./login/login"
import { fetchDevices, smallMessage } from "./miele"
import { testConfig } from "./miele-testutils"

jest.setTimeout(JEST_INTEGRATION_TIMEOUT)

describe("miele", () => {
    beforeAll(() => {
        applyConfig(testConfig())
        log.off()
    })

    afterAll(() => {
        log.on()
    })

    test("fetch devices", async () => {
        const token = await getToken()
        const devices = await fetchDevices(token.access_token)
        expect(devices.length).toBe(1)
        expect(devices[0].data.ident.type.value_localized).toBe("Dishwasher")

        const small = smallMessage(devices[0])
        expect(small).toBeDefined()
    })

    test("convert to empty small message", () => {
        const small: any = smallMessage({} as any)
        delete small.timeCompleted
        expect(small).toStrictEqual({
            phase: "UNKNOWN",
            phaseId: -1,
            remainingDuration: "0:00",
            remainingDurationMinutes: 0,
            state: "UNKNOWN"
        })
    })
})
