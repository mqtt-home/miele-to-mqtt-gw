import Duration from "@icholy/duration"

export const parseDuration = (array: number[] | undefined) => {
    if (!array) {
        return Duration.seconds(0)
    }
    else if (array.length === 2) {
        return Duration.minutes(array[0] * 60 + array[1])
    }
    else if (array.length === 3) {
        return Duration.seconds(array[0] * 3600 + array[1] * 60 + array[0])
    }
    return Duration.seconds(0)
}

const padWithZero = (num: number) => {
    return String(num).padStart(2, "0")
}

export const formatHours = (duration: Duration) => {
    const hours = duration.hours()
    const minutes = duration.minutes() - duration.hours() * 60
    return `${hours}:${padWithZero(minutes)}`
}

export const formatTime = (date: Date) => {
    return `${date.getHours()}:${padWithZero(date.getMinutes())}`
}

export const add = (date: Date, duration: Duration) => {
    const result = new Date(date)
    result.setMinutes(date.getMinutes() + duration.minutes())
    return result
}
