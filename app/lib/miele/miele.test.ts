import { login, refreshToken } from "./login"
import { fetchDevices, smallMessage } from "./miele";
import { testConfig } from "./miele-testutils"

describe("miele", () => {
    test("fetch devices", async () => {
        const token = await login(testConfig())
        const devices = await fetchDevices(token.access_token)
        for (let device of devices) {
            console.log(smallMessage(device))
        }
    })
})
