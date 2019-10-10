package newport

import (
	"net/http"
	"strconv"
)

var (
	epsCommands = map[string]motionCommand{
		// Status functions
		"DP": motionCommand{
			Descr:    "Get target position",
			Route:    "target-position",
			Method:   http.MethodGet,
			UsesAxis: true},
		"DV": motionCommand{
			Descr:    "Get working speed",
			Route:    "working-speed",
			Method:   http.MethodGet,
			UsesAxis: true},
		"ID": motionCommand{
			Descr:  "Get stage model and serial number",
			Route:  "model-serial",
			Method: http.MethodGet},
		"MD": motionCommand{
			Descr:    "Get axis motion status",
			Route:    "moving",
			Method:   http.MethodGet,
			UsesAxis: true},
		"PH": motionCommand{
			Descr:    "Get hardware status",
			Route:    "hw-status",
			Method:   http.MethodGet,
			UsesAxis: true},
		"TB": motionCommand{
			Descr:    "Get error message",
			Route:    "err-msg",
			Method:   http.MethodGet,
			UsesAxis: true},
		"TE": motionCommand{
			Descr:    "Get error number",
			Route:    "err-num",
			Method:   http.MethodGet,
			UsesAxis: true},
		"TP": motionCommand{
			Descr:    "Get position",
			Route:    "position",
			Method:   http.MethodGet,
			UsesAxis: true},
		"TS": motionCommand{
			Descr:  "Get controller status",
			Route:  "controller-status",
			Method: http.MethodGet},
		"TV": motionCommand{
			Descr:    "Get velocity",
			Route:    "velocity",
			Method:   http.MethodGet,
			UsesAxis: true},
		"TX": motionCommand{
			Descr:  "Get controller activity",
			Route:  "controller-activity",
			Method: http.MethodGet},
		"VE": motionCommand{
			Descr:  "Get firmware version",
			Route:  "firmware-version",
			Method: http.MethodGet},

		// Motion & Position control
		"AB": motionCommand{
			Descr:    "Abort motion",
			Route:    "abort-motion",
			Method:   http.MethodGet,
			UsesAxis: true},
		"DH": motionCommand{
			Descr:    "Define home",
			Route:    "define-home",
			Method:   http.MethodPost,
			UsesAxis: true},
		"MT": motionCommand{
			Descr:    "Move to hardware travel limit",
			Route:    "move-hardware-travel-limit",
			Method:   http.MethodPost,
			UsesAxis: true},
		"MV": motionCommand{
			Descr:    "Move indefinitely",
			Route:    "move-indefinitely",
			Method:   http.MethodPost,
			UsesAxis: true},
		"MZ": motionCommand{
			Descr:    "Move to nearest index",
			Route:    "move-to-nearest-index",
			Method:   http.MethodPost,
			UsesAxis: true},
		"OR": motionCommand{
			Descr:    "Origin searching",
			Route:    "origin-search",
			Method:   http.MethodPost,
			UsesAxis: true},
		"PA": motionCommand{
			Descr:    "Position absolute",
			Route:    "move-absolute",
			Method:   http.MethodPost,
			UsesAxis: true},
		"PR": motionCommand{
			Descr:    "Position relative",
			Route:    "move-relative",
			Method:   http.MethodPost,
			UsesAxis: true},
		"ST": motionCommand{
			Descr:    "Stop motion",
			Route:    "stop",
			Method:   http.MethodPost,
			UsesAxis: true},

		// Trajectory definition
		// not implemented: JH, JK, JW, OL, OH, OM, SH, UF
		"AC": motionCommand{
			Descr:    "Set acceleration",
			Route:    "acceleration",
			Method:   http.MethodPost,
			UsesAxis: true},
		"AE": motionCommand{
			Descr:    "Set e-stop deceleration",
			Route:    "e-stop-acceleration",
			Method:   http.MethodPost,
			UsesAxis: true},
		"AG": motionCommand{
			Descr:    "Set deceleration",
			Route:    "deceleration",
			Method:   http.MethodPost,
			UsesAxis: true},
		"AU": motionCommand{
			Descr:    "Set maximum acceleration",
			Route:    "maximum-acceleration",
			Method:   http.MethodPost,
			UsesAxis: true},
		"BA": motionCommand{
			Descr:    "Set backlash compensation",
			Route:    "backlash-compensation",
			Method:   http.MethodPost,
			UsesAxis: true},
		"CO": motionCommand{
			Descr:    "Set linear compensation",
			Route:    "linear-compensation",
			Method:   http.MethodPost,
			UsesAxis: true},
		"VA": motionCommand{
			Descr:    "Set velocity",
			Route:    "velocity",
			Method:   http.MethodPost,
			UsesAxis: true},
		"VB": motionCommand{
			Descr:    "Set base velocity for stepper motors",
			Route:    "stepper-base-velocity",
			Method:   http.MethodPost,
			UsesAxis: true},
		"VU": motionCommand{
			Descr:    "Set maximum speed",
			Route:    "maximum-speed",
			Method:   http.MethodPost,
			UsesAxis: true},

		// Not implemented:
		// - General mode selection
		// - Motion device parameters
		// - Programming
		// - Trajectory definition
		// - Flow control & sequence
		// - Group functions
		// - Digital filters
		// - Master-slave mode definition
	}
)

// EPS301 represents an EPS301 motion controller
type EPS301 struct {
	Addr string
}

func (ep *EPS301) makeTelegram(mc motionCommand, axis int) (string, error) {
	pieces := []string{}
	if mc.UsesAxis {
		pieces = append(pieces, strconv.Itoa(axis))
	}
}
