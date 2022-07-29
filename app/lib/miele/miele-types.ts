export type MieleDevice = {
    id: string,
    data: any
}

export enum DeviceStatus {
    UNKNOWN = -1,
    RESERVED= 0,
    OFF = 1,
    ON = 2,
    PROGRAMMED = 3,
    PROGRAMMED_WAITING_TO_START = 4,
    RUNNING = 5,
    PAUSE = 6,
    END_PROGRAMMED = 7,
    FAILURE = 8,
    PROGRAMME_INTERRUPTED =9,
    IDLE = 10,
    RINSE_HOLD = 11,
    SERVICE = 12,
    SUPERFREEZING = 13,
    SUPERCOOLING = 14,
    SUPERHEATING = 15
}

export enum Phase {
    UNKNOWN = -1,
    OFF = 0,
    NOT_RUNNING = 1792,
    REACTIVATING = 1793,
    PRE_WASH = 1794,
    MAIN_WASH = 1795,
    RINSE = 1796,
    INTERIM_RINSE = 1797,
    FINAL_RINSE = 1798,
    DRYING = 1799,
    FINISHED = 1800,
    PRE_WASH2 =1801
}
