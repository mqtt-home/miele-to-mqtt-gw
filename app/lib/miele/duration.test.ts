import Duration from "@icholy/duration"
import { add, formatHours, formatTime, parseDuration } from "./duration"

describe("duration", () => {
    it.each([
        [[1, 45], 105],
        [[1, 45, 13], 105],
        [[1], 0],
        [undefined, 0]
    ])("parses the duration %p expecting %p", (numbers: number[] | undefined, result: any) => {
        expect(parseDuration(numbers).minutes()).toBe(result)
    })

    it.each([
        [[1, 65], "2:05"],
        [[1, 45], "1:45"],
        [[1, 45, 13], "1:45"],
        [[1, 1], "1:01"],
        [[1], "0:00"],
        [undefined, "0:00"]
    ])("format the duration %p expecting %p", (numbers: number[] | undefined, result: string) => {
        expect(formatHours(parseDuration(numbers))).toBe(result)
    })

    it.each([
        [0, "5:15"],
        [1, "5:16"],
        [59, "6:14"],
        [200, "8:35"],
        [800, "18:35"]
    ])("add minutes to date %p expecting %p", (minutes: number, result: string) => {
        const date = new Date(2022, 0, 1, 5, 15)
        expect(formatTime(add(date, Duration.minutes(minutes)))).toBe(result)
    })
})
