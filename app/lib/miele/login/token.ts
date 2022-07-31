import axios from "axios"
import { ConfigMiele, getAppConfig } from "../../config/config"

export type TokenResult = {
    access_token: string,
    refresh_token: string,
    token_type: string,
    expires_in: number
}

export type Token = {
    access_token: string,
    refresh_token: string,
    token_type: string,
    expiresAt: Date
}

export const fetchToken = async (code: string) => {
    const config: ConfigMiele = getAppConfig().miele
    const response = await axios.post(
        "https://api.mcs3.miele.com/thirdparty/token",
        new URLSearchParams({
            client_id: config["client-id"],
            client_secret: config["client-secret"],
            code,
            redirect_uri: "/v1/devices",
            grant_type: "authorization_code",
            state: "token"
        }),
        {
            headers: {
                "Content-Type": "application/x-www-form-urlencoded"
            },
            maxRedirects: 0
        })

    return response.data as TokenResult
}
