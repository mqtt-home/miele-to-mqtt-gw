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

        const check = registerConnectionCheck(fn, config)

        await sleep(100)

        expect(fn).not.toHaveBeenCalled()

        check?.unref()
    })

    test("failed", async () => {
        __TEST_setCheck(() => Promise.resolve(false))
        const fn = jest.fn()

        const check = registerConnectionCheck(fn, config)

        await sleep(100)
        expect(fn).not.toHaveBeenCalled()

        __TEST_setCheck(() => Promise.resolve(true))

        await sleep(100)
        expect(fn).toHaveBeenCalled()

        check?.unref()
    })

    test("check disabled", async () => {
        __TEST_setCheck(() => Promise.resolve(false))
        const fn = jest.fn()

        const check = registerConnectionCheck(fn, { ...config, "connection-check-interval": 0 })

        await sleep(100)
        expect(fn).not.toHaveBeenCalled()

        __TEST_setCheck(() => Promise.resolve(true))

        await sleep(100)
        expect(fn).not.toHaveBeenCalled()

        check?.unref()
    })
})
