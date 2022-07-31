import Duration from "@icholy/duration"
import axios from "axios"
import { getAppConfig } from "../../config/config"
import { log } from "../../logger"
import { add } from "../duration"
import { fetchCode } from "./code"
import { fetchToken, Token, TokenResult } from "./token"

let token: Token

export const setToken = (newToken: Token) => {
    token = newToken
}

export const convertToken = (mieleToken: TokenResult, now = new Date()) => {
    const copy = {
        ...mieleToken
    } as any
    delete copy.expires_in

    return {
        ...copy,
        expiresAt: add(now, Duration.seconds(mieleToken.expires_in))
    } as Token
}

export const needsRefresh = (tokenToTest = token, now = new Date()) => {
    const inOneDay = add(now, Duration.days(1))
    return (tokenToTest && tokenToTest.expiresAt <= inOneDay)
}

export const login = async (now = new Date()) => {
    try {
        if (token && needsRefresh(token, now)) {
            // Refresh token
            token = convertToken(await refreshToken(token.refresh_token))
        }
    }
    catch (e) {
        log.error(`Token refresh failed. Trying to login with username/password. ${e}`)
    }

    if (!token) {
        const code = await fetchCode()
        token = convertToken(await fetchToken(code))
    }
}

export const getToken = async () => {
    if (!token || needsRefresh()) {
        await login()
    }

    return token
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
