import { applyConfig } from "../config/config"
import { login } from "./login/login"
import { fetchDevices, smallMessage } from "./miele"
import { testConfig } from "./miele-testutils"

describe("miele", () => {
    beforeAll(() => {
        applyConfig(testConfig())
    })

    test("fetch devices", async () => {
        const token = await login()
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
