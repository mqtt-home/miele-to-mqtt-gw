import Duration from "@icholy/duration";
import { add, formatHours, formatTime, parseDuration } from "./duration";
import { DeviceStatus, MieleDevice, Phase } from "./miele-types"

export const fetchDevices = async (token: string) => {
    const response = await fetch("https://api.mcs3.miele.com/v1/devices/", {
        headers: {
            "Authorization": `Bearer ${token}`,
            "Content-Type": "application/json",
        },
    })

    const devices = await response.json()
    const result: MieleDevice[] = []
    for (let key of Object.keys(devices)) {
        result.push({
            id: key,
            data: devices[key]
        })
    }
    return result
}

export const smallMessage = (device: MieleDevice) => {
    const state = device.data.state
    const message = {
        phase: Phase[state.programPhase.value_raw] ?? Phase[-1],
        phaseId: state.programPhase.value_raw,
        state: DeviceStatus[state.status.value_raw] ?? DeviceStatus[-1],
    }

    const remainingDuration = parseDuration(state.remainingTime)

    if (state.status.value_raw === DeviceStatus.OFF) {
        return {
            ...message,
            remainingDurationMinutes: 0,
            remainingDuration: formatHours(Duration.hours(0)),
            timeCompleted: formatTime(new Date()),
        }
    }
    else {
        return {
            ...message,
            remainingDurationMinutes: remainingDuration.minutes(),
            remainingDuration: formatHours(remainingDuration),
            timeCompleted: formatTime(add(new Date(), remainingDuration)),
        }
    }
}
