package transform

// DeviceStatus mirrors lib/miele/miele-types.ts.
type DeviceStatus int

const (
	DeviceStatusUnknown                  DeviceStatus = -1
	DeviceStatusReserved                 DeviceStatus = 0
	DeviceStatusOff                      DeviceStatus = 1
	DeviceStatusOn                       DeviceStatus = 2
	DeviceStatusProgrammed               DeviceStatus = 3
	DeviceStatusProgrammedWaitingToStart DeviceStatus = 4
	DeviceStatusRunning                  DeviceStatus = 5
	DeviceStatusPause                    DeviceStatus = 6
	DeviceStatusEndProgrammed            DeviceStatus = 7
	DeviceStatusFailure                  DeviceStatus = 8
	DeviceStatusProgrammeInterrupted     DeviceStatus = 9
	DeviceStatusIdle                     DeviceStatus = 10
	DeviceStatusRinseHold                DeviceStatus = 11
	DeviceStatusService                  DeviceStatus = 12
	DeviceStatusSuperfreezing            DeviceStatus = 13
	DeviceStatusSupercooling             DeviceStatus = 14
	DeviceStatusSuperheating             DeviceStatus = 15
)

var deviceStatusNames = map[DeviceStatus]string{
	DeviceStatusUnknown:                  "UNKNOWN",
	DeviceStatusReserved:                 "RESERVED",
	DeviceStatusOff:                      "OFF",
	DeviceStatusOn:                       "ON",
	DeviceStatusProgrammed:               "PROGRAMMED",
	DeviceStatusProgrammedWaitingToStart: "PROGRAMMED_WAITING_TO_START",
	DeviceStatusRunning:                  "RUNNING",
	DeviceStatusPause:                    "PAUSE",
	DeviceStatusEndProgrammed:            "END_PROGRAMMED",
	DeviceStatusFailure:                  "FAILURE",
	DeviceStatusProgrammeInterrupted:     "PROGRAMME_INTERRUPTED",
	DeviceStatusIdle:                     "IDLE",
	DeviceStatusRinseHold:                "RINSE_HOLD",
	DeviceStatusService:                  "SERVICE",
	DeviceStatusSuperfreezing:            "SUPERFREEZING",
	DeviceStatusSupercooling:             "SUPERCOOLING",
	DeviceStatusSuperheating:             "SUPERHEATING",
}

// DeviceStatusName returns the symbolic name for a status value, or "UNKNOWN"
// for any value not in the enum. Matches the TS `DeviceStatus[status]` lookup
// where missing keys would produce `undefined`; we preserve compatibility by
// substituting "UNKNOWN".
func DeviceStatusName(s DeviceStatus) string {
	if name, ok := deviceStatusNames[s]; ok {
		return name
	}
	return "UNKNOWN"
}

// Phase mirrors lib/miele/miele-types.ts.
type Phase int

const (
	PhaseUnknown      Phase = -1
	PhaseOff          Phase = 0
	PhaseNotRunning   Phase = 1792
	PhaseReactivating Phase = 1793
	PhasePreWash      Phase = 1794
	PhaseMainWash     Phase = 1795
	PhaseRinse        Phase = 1796
	PhaseInterimRinse Phase = 1797
	PhaseFinalRinse   Phase = 1798
	PhaseDrying       Phase = 1799
	PhaseFinished     Phase = 1800
	PhasePreWash2     Phase = 1801
)

var phaseNames = map[Phase]string{
	PhaseUnknown:      "UNKNOWN",
	PhaseOff:          "OFF",
	PhaseNotRunning:   "NOT_RUNNING",
	PhaseReactivating: "REACTIVATING",
	PhasePreWash:      "PRE_WASH",
	PhaseMainWash:     "MAIN_WASH",
	PhaseRinse:        "RINSE",
	PhaseInterimRinse: "INTERIM_RINSE",
	PhaseFinalRinse:   "FINAL_RINSE",
	PhaseDrying:       "DRYING",
	PhaseFinished:     "FINISHED",
	PhasePreWash2:     "PRE_WASH2",
}

func PhaseName(p Phase) string {
	if name, ok := phaseNames[p]; ok {
		return name
	}
	return "UNKNOWN"
}
