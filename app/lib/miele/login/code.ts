import { ConfigMiele, getAppConfig } from "../../config/config"
import axios from "axios"

export const fetchCode = async (config: ConfigMiele = getAppConfig().miele) => {
    const response = await axios.post(
        "https://api.mcs3.miele.com/oauth/auth",
        new URLSearchParams({
            email: config.username,
            password: config.password,
            redirect_uri: "/v1/",
            state: "login",
            response_type: "code",
            client_id: config["client-id"],
            vgInformationSelector: "de-DE"
        }),
        {
            headers: {
                "Content-Type": "application/x-www-form-urlencoded"
            },
            validateStatus: (status) => status === 302,
            maxRedirects: 0
        })

    const location = response.headers.location
    if (!location) {
        throw new Error("Cannot fetch code. Location missing.")
    }
    const params = new URL(location).searchParams
    const code = params.get("code")
    if (!code) {
        throw new Error("Cannot fetch code. Code missing.")
    }
    return code
}
