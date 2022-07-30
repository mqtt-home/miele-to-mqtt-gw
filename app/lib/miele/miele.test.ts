import { login } from "./login/login"
import { fetchDevices, smallMessage } from "./miele"
import { testConfig } from "./miele-testutils"

describe("miele", () => {
    test("fetch devices", async () => {
        const token = await login(testConfig())
        const devices = await fetchDevices(token.access_token)
        expect(devices.length).toBe(1)
        expect(devices[0].data.ident.type.value_localized).toBe("Dishwasher")
        for (const device of devices) {
            console.log(smallMessage(device))
        }
    })
})
