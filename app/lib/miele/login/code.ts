import { ConfigMiele, getAppConfig } from "../../config/config"
import axios from "axios"
import { log } from "../../logger"

export const codeUrl = "https://api.mcs3.miele.com/oauth/auth"
export const fetchCode = async () => {
    log.debug("Fetching code")

    // Debug this by visiting the following URL:
    // https://api.mcs3.miele.com/thirdparty/login/?redirect_uri=/v1/&client_id=<your_client_id>&response_type=code
    //
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
    log.debug("Code location", location)

    const params = new URL(location).searchParams
    const code = params.get("code")
    if (!code) {
        throw new Error("Cannot fetch code. Code missing.")
    }
    log.debug("Code", code)
    return code
}
