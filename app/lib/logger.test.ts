import { createLogger, log, logLevelColor, setLogger, unpackError } from "./logger"
import { Writable } from "stream"
import * as winston from "winston"
import chalk from "chalk"

describe("Log format", () => {
    let output = ""
    let logger: winston.Logger

    beforeEach(() => {
        log.off()

        process.env.FORCE_COLOR = "0"

        const stream = new Writable()
        stream._write = (chunk, _, next) => {
            output = output += chunk.toString()
            next()
        }
        logger = createLogger(new winston.transports.Stream({ stream }))
        logger.level = "TRACE"
        setLogger(logger)
    })

    afterEach(() => {
        output = ""
    })

    describe("severities", () => {
        test("info log", () => {
            log.info("some info")

            expect(output).toMatch(/\d+-\d+-\d+T\d+:\d+:\d+.\d+Z \[.*INFO.*] some info.*/)
        })

        test("warn log", () => {
            log.warn("some warning")

            expect(output).toMatch(/\d+-\d+-\d+T\d+:\d+:\d+.\d+Z \[.*WARN.*] some warning.*/)
        })

        test("error log", () => {
            log.error("some error")

            expect(output).toMatch(/\d+-\d+-\d+T\d+:\d+:\d+.\d+Z \[.*ERROR.*] some error.*/)
        })

        test("fatal log", () => {
            log.fatal("some fatal error")

            expect(output).toMatch(/\d+-\d+-\d+T\d+:\d+:\d+.\d+Z \[.*FATAL.*] some fatal error.*/)
        })

        test("debug log", () => {
            log.debug("some debug message")

            expect(output).toMatch(/\d+-\d+-\d+T\d+:\d+:\d+.\d+Z \[.*DEBUG.*] some debug message.*/)
        })

        test("trace log", () => {
            log.trace("some trace message")

            expect(output).toMatch(/\d+-\d+-\d+T\d+:\d+:\d+.\d+Z \[.*TRACE.*] some trace message.*/)
        })
    })

    test("Configure log levels", () => {
        log.configure("WARN")
        expect(logger.level).toBe("WARN")
        log.configure("TRACE")
        expect(logger.level).toBe("TRACE")
    })

    test("error log with Error object", () => {
        log.error("some error", new Error("uups"))

        expect(output).toMatch(/\d+-\d+-\d+T\d+:\d+:\d+.\d+Z \[.*ERROR.*] some error.*/)
        expect(output).toContain("uups")
    })

    test("Log level colors", () => {
        expect(logLevelColor("fatal")).toBe(chalk.red("FATAL"))
        expect(logLevelColor("error")).toBe(chalk.red("ERROR"))
        expect(logLevelColor("warn")).toBe(chalk.yellow("WARN"))
        expect(logLevelColor("info")).toBe(chalk.blue("INFO"))
        expect(logLevelColor("debug")).toBe(chalk.magenta("DEBUG"))
        expect(logLevelColor("trace")).toBe(chalk.magenta("TRACE"))
        expect(logLevelColor("other")).toBe("OTHER")
    })

    describe("unpack errors", () => {
        test("no error", () => {
            const data = unpackError({ a: 1 })

            expect(data).toEqual({ a: 1 })
        })

        test("error", () => {
            const data = unpackError(new Error("uups"))
            expect(data.error.stack).toBeDefined()
            delete data.error.stack
            expect(data).toEqual({ error: { message: "uups", name: "Error" } })
        })

        test("array of errors", () => {
            const data = unpackError([new Error("uups"), new Error("uups2")])

            expect(data[0].error.stack).toBeDefined()
            delete data[0].error.stack
            expect(data[1].error.stack).toBeDefined()
            delete data[1].error.stack

            expect(data).toEqual([
                { error: { message: "uups", name: "Error" } },
                { error: { message: "uups2", name: "Error" } }
            ])
        })

        test("array of objects", () => {
            const data = unpackError([{ a: 1 }, { a: 2 }])

            expect(data).toEqual([{ a: 1 }, { a: 2 }])
        })

        test("array of mixed", () => {
            const data = unpackError([{ a: 1 }, new Error("uups")])
            expect(data[1].error.stack).toBeDefined()
            delete data[1].error.stack

            expect(data).toEqual([
                { a: 1 },
                { error: { message: "uups", name: "Error" } }
            ])
        })

        test("array of mixed 2", () => {
            const data = unpackError([new Error("uups"), { a: 1 }])
            expect(data[0].error.stack).toBeDefined()
            delete data[0].error.stack

            expect(data).toEqual([
                { error: { message: "uups", name: "Error" } },
                { a: 1 }
            ])
        })

        test("empty array", () => {
            const data = unpackError([])

            expect(data).toBeNull()
        })

        test("null", () => {
            const data = unpackError(null)

            expect(data).toBeNull()
        })

        test("undefined", () => {
            const data = unpackError(undefined)

            expect(data).toBeUndefined()
        })

        test("single string in array", () => {
            const data = unpackError(["uups"])

            expect(data).toBe("uups")
        })

        test("single object in array", () => {
            const data = unpackError([{ a: 1 }])

            expect(data).toEqual({ a: 1 })
        })
    })
})
