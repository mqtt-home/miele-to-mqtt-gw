export const JEST_INTEGRATION_TIMEOUT = 60 * 5 * 1000
export const JEST_DEFAULT_TIMEOUT = 5 * 1000

export const curlHealthTest = (host: string, port: number) => {
    return {
        test: `curl -f http://${host}:${port} || exit 1`,
        interval: 5_000,
        timeout: 30_000,
        retries: 20,
        startPeriod: 1_000
    }
}

export const waitFor = async (predicate: () => boolean) => {
    while (!predicate()) {
        await new Promise(resolve => setTimeout(resolve, 10))
    }
}
