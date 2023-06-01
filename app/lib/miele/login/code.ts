import { ConfigMiele, getAppConfig } from "../../config/config"
import axios from "axios"

export const codeUrl = "https://api.mcs3.miele.com/oauth/auth"
// https://api.mcs3.miele.com/thirdparty/login/?redirect_uri=http://localhost:3000&client_id=76012106-c8c0-4901-8ff4-b3bf32696523&response_type=code&state=login&vgInformationSelector=de-DE
export const fetchCode = async () => {
    const config: ConfigMiele = getAppConfig().miele
    const response = await axios.post(
        codeUrl,
        new URLSearchParams({
            email: config.username,
            password: config.password,
            redirect_uri: "/v1/",
            state: "login",
            response_type: "code",
            client_id: config["client-id"],
            vgInformationSelector: config["country-code"]
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
