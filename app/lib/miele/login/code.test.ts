import { rest } from "msw"
import { setupServer } from "msw/node"
import { applyConfig } from "../../config/config"
import { testConfig } from "../miele-testutils"
import { codeUrl, fetchCode } from "./code"
const server = setupServer()

describe("code", () => {
    describe("error cases", () => {
        beforeAll(() => {
            applyConfig(testConfig())
            server.listen()
        })
        afterEach(() => server.resetHandlers())
        afterAll(() => server.close())

        test("Location header missing", async () => {
            server.use(rest.post(codeUrl, (req, res, ctx) => {
                return res(
                    ctx.status(302)
                )
            }))

            expect.assertions(1)
            try {
                await fetchCode()
            }
            catch (e: any) {
                expect(e.toString()).toBe("Error: Cannot fetch code. Location missing.")
            }
        })

        test("code missing", async () => {
            server.use(rest.post(codeUrl, (req, res, ctx) => {
                return res(
                    ctx.status(302),
                    ctx.set("Location", "https://example.org/no-code")
                )
            }))

            expect.assertions(1)
            try {
                await fetchCode()
            }
            catch (e: any) {
                expect(e.toString()).toBe("Error: Cannot fetch code. Code missing.")
            }
        })
    })
})
