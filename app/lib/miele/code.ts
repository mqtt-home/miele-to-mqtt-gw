import { ConfigMiele, getAppConfig } from "../config/config";

export const fetchCode = async (config: ConfigMiele = getAppConfig().miele) => {
    const response = await fetch("https://api.mcs3.miele.com/oauth/auth", {
        body: new URLSearchParams({
            email: config.username,
            password: config.password,
            redirect_uri: "/v1/",
            state: "login",
            response_type: "code",
            client_id: config["client-id"],
            vgInformationSelector: "de-DE"
        }),
        headers: {
            "Content-Type": "application/x-www-form-urlencoded"
        },
        method: "POST",
        redirect: "manual"
    })

    if (response.status !== 302) {
        throw new Error(`Cannot fetch code. Unexpected response status (${response.status}).`)
    }

    const location = response.headers.get("location")
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
