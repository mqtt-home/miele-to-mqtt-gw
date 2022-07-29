import { ConfigMiele, getAppConfig } from "../config/config";

export type TokenResult = {
    access_token: string,
    refresh_token: string,
    token_type: string,
    expires_in: number
}

export const fetchToken = async (code: string, config: ConfigMiele = getAppConfig().miele) => {
    const response = await fetch("https://api.mcs3.miele.com/thirdparty/token", {
        body: new URLSearchParams({
            "client_id": config["client-id"],
            client_secret: config["client-secret"],
            code,
            redirect_uri: "/v1/devices",
            grant_type: "authorization_code",
            state: "token"
        }),
        headers: {
            "Content-Type": "application/x-www-form-urlencoded"
        },
        method: "POST",
        redirect: "manual"
    })

    const json = await response.json()
    return json as TokenResult
}
