import { applyConfig } from "../../config/config"
import { testConfig } from "../miele-testutils"
import { login, refreshToken } from "./login"

describe("login", () => {
    beforeAll(() => {
        applyConfig(testConfig())
    })

    test("login", async () => {
        const token = await login()
        expect(token.access_token).toBeDefined()
        expect(token.refresh_token).toBeDefined()
        expect(token.token_type).toBeDefined()
        expect(token.expires_in).toBeDefined()

        const refreshed = await refreshToken(token.refresh_token)
        expect(refreshed.access_token).toBeDefined()
        expect(refreshed.refresh_token).toBeDefined()
        expect(refreshed.token_type).toBeDefined()
        expect(refreshed.expires_in).toBeDefined()
    })
})
