import { ConfigMiele, getAppConfig } from "../config/config"
import { fetchCode } from "./code";
import { fetchToken, TokenResult } from "./token";

export const login = async (config: ConfigMiele = getAppConfig().miele) => {
    const code = await fetchCode(config)
    return await fetchToken(code, config)
}

export const refreshToken = async (refresh_token: string, config: ConfigMiele = getAppConfig().miele) => {
    const response = await fetch("https://api.mcs3.miele.com/thirdparty/token", {
        body: new URLSearchParams({
            "client_id": config["client-id"],
            client_secret: config["client-secret"],
            refresh_token,
            grant_type: "refresh_token"
        }),
        headers: {
            "Content-Type": "application/x-www-form-urlencoded"
        },
        method: "POST",
        redirect: "manual"
    })

    return (await response.json()) as TokenResult
}
