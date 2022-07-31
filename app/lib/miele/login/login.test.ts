import { applyConfig } from "../../config/config"
import { testConfig } from "../miele-testutils"
import { convertToken, getToken, needsRefresh, refreshToken } from "./login"

describe("login", () => {
    beforeAll(() => {
        applyConfig(testConfig())
    })

    test("login", async () => {
        const token = await getToken()
        expect(token.access_token).toBeDefined()
        expect(token.refresh_token).toBeDefined()
        expect(token.token_type).toBeDefined()
        expect(token.expiresAt).toBeDefined()

        const refreshed = await refreshToken(token.refresh_token)
        expect(refreshed.access_token).toBeDefined()
        expect(refreshed.refresh_token).toBeDefined()
        expect(refreshed.token_type).toBeDefined()
        expect(refreshed.expires_in).toBeDefined()

        expect(token.access_token).not.toBe(refreshed.access_token)
    })

    test("convert expiry", () => {
        const token = convertToken({
            access_token: "DE_123456789abcdef12345678912345678",
            refresh_token: "DE_98765432109876543210987654321098",
            token_type: "Bearer",
            expires_in: 2592000
        }, new Date("2022-07-31"))

        expect(token.expiresAt.toISOString().split("T")[0]).toBe("2022-08-30")
    })

    it.each([
        ["2022-07-31", "2021-07-30", true],
        ["2022-07-31", "2022-07-30", true],
        ["2022-07-31", "2022-07-31", true],
        ["2022-07-31", "2022-08-01", true],
        ["2022-07-31", "2022-08-15", false],
        ["2022-07-31", "2023-07-31", false]
    ])("needs refresh now: %s expiresAt: %s", (now: string, expiresAt: string, expected: boolean) => {
        const result = needsRefresh({
            access_token: "",
            refresh_token: "",
            token_type: "Bearer",
            expiresAt: new Date(expiresAt)
        }, new Date(now))
        expect(result).toBe(expected)
    })
})
