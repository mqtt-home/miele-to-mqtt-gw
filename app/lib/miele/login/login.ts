import axios from "axios"
import { getAppConfig } from "../../config/config"
import { fetchCode } from "./code"
import { fetchToken, TokenResult } from "./token"

export const login = async () => {
    const code = await fetchCode()
    return await fetchToken(code)
}

/* eslint-disable camelcase */
export const refreshToken = async (refresh_token: string) => {
    const config = getAppConfig().miele
    const response = await axios.post(
        "https://api.mcs3.miele.com/thirdparty/token",
        new URLSearchParams({
            client_id: config["client-id"],
            client_secret: config["client-secret"],
            refresh_token,
            grant_type: "refresh_token"
        }),
        {
            headers: {
                "Content-Type": "application/x-www-form-urlencoded"
            },
            maxRedirects: 0
        })

    return (await response.data) as TokenResult
}
