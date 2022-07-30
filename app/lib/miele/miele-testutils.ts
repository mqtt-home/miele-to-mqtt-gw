import { Config } from "../config/config"

const forceEnv = (propName: string) => {
    // macOS note: sudo vi /etc/launchd.conf
    const value = process.env[propName]
    if (!value) {
        throw Error(`"ENV property ${propName} is required to run this test case."`)
    }
    return value
}

export const testConfig = () => {
    return {
        miele: {
            "client-id": forceEnv("MIELE_CLIENT_ID"),
            "client-secret": forceEnv("MIELE_CLIENT_SECRET"),
            username: forceEnv("MIELE_USERNAME"),
            password: forceEnv("MIELE_PASSWORD"),
            mode: "sse",
            "polling-interval": 30
        }
    } as Config
}
