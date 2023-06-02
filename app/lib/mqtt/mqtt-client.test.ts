import { convertBody } from "./mqtt-client"

describe("mqtt-client", () => {
    it("should not stringify string as JSON", () => {
        expect(convertBody("test")).toBe("test")
    })

    it("should stringify object as JSON", () => {
        expect(convertBody({ test: "test" })).toBe("{\"test\":\"test\"}")
    })
})
