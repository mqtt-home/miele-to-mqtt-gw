import { applyConfig, ConfigMiele } from "./config/config"
// eslint-disable-next-line camelcase
import { __TEST_setCheck, registerConnectionCheck, unregisterConnectionCheck } from "./connection"
import { log } from "./logger"
import { testConfig } from "./miele/miele-testutils"

const config: ConfigMiele = {
    "connection-check-interval": 10,
    "client-id": "",
    "client-secret": "",
    "country-code": "",
    "polling-interval": 0,
    mode: "sse",
    password: "",
    token: undefined,
    username: ""
}

describe("connection", () => {
    beforeEach(() => {
        applyConfig(testConfig())
        log.off()
    })

    afterEach(() => {
        unregisterConnectionCheck()
    })

    const sleep = (ms: number) => new Promise(resolve => setTimeout(resolve, ms))

    test("success", async () => {
        __TEST_setCheck(() => Promise.resolve(true))

        const fn = jest.fn()

        registerConnectionCheck(fn, config)

        await sleep(100)

        expect(fn).not.toHaveBeenCalled()

        unregisterConnectionCheck()
    })

    test("failed", async () => {
        __TEST_setCheck(() => Promise.resolve(false))
        const fn = jest.fn()

        registerConnectionCheck(fn, config)

        await sleep(100)
        expect(fn).not.toHaveBeenCalled()

        __TEST_setCheck(() => Promise.resolve(true))

        await sleep(100)
        expect(fn).toHaveBeenCalled()

        unregisterConnectionCheck()
    })

    test("check disabled", async () => {
        __TEST_setCheck(() => Promise.resolve(false))
        const fn = jest.fn()

        registerConnectionCheck(fn, { ...config, "connection-check-interval": 0 })

        await sleep(100)
        expect(fn).not.toHaveBeenCalled()

        __TEST_setCheck(() => Promise.resolve(true))

        await sleep(100)
        expect(fn).not.toHaveBeenCalled()

        unregisterConnectionCheck()
    })

    test("already registered", async () => {
        __TEST_setCheck(() => Promise.resolve(false))
        const fn = jest.fn()

        let check1 = registerConnectionCheck(fn, config)
        let check2 = registerConnectionCheck(fn, config)

        expect(check1).toBe(check2)

        unregisterConnectionCheck()
    })
})
