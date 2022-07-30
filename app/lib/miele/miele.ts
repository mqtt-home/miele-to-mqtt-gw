import Duration from "@icholy/duration"
import axios from "axios"
import { add, formatHours, formatTime, parseDuration } from "./duration"
import { DeviceStatus, MieleDevice, Phase } from "./miele-types"

export const fetchDevices = async (token: string) => {
    const response = await axios.get(
        "https://api.mcs3.miele.com/v1/devices/",
        {
            headers: {
                Authorization: `Bearer ${token}`,
                "Content-Type": "application/json"
            }
        })

    return convertDevices(await response.data)
}

export const convertDevices = (devices: any) => {
    const result: MieleDevice[] = []
    for (const key of Object.keys(devices)) {
        result.push({
            id: key,
            data: devices[key]
        })
    }
    return result
}

export const smallMessage = (device: MieleDevice) => {
    const phase = device.data?.state?.programPhase?.value_raw ?? -1
    const status = device.data?.state?.status?.value_raw ?? -1
    const remainingTime = device.data?.state?.remainingTime ?? []

    let remainingDuration = parseDuration(remainingTime)
    if (status === DeviceStatus.OFF) {
        remainingDuration = Duration.hours(0)
    }

    return {
        phase: Phase[phase],
        phaseId: phase,
        state: DeviceStatus[status],
        remainingDurationMinutes: remainingDuration.minutes(),
        remainingDuration: formatHours(remainingDuration),
        timeCompleted: formatTime(add(new Date(), remainingDuration))
    }
}
