package newport

import (
	"net/http"
	"strconv"
)

type command struct {
	Cmd         string `json:"cmd"`
	Alias       string `json:"alias"`
	Description string `json:"description"`
	UsesAxis    bool   `json:"usesAxis"`
	IsRead      bool   `json:"isRead"`
}

var (
	commands = []command{
		// Status functions
		{Cmd: "TE", Alias: "err-num", Description: "get error number", IsRead: true},
		{Cmd: "TP", Alias: "position", Description: "get position", UsesAxis: true, IsRead: true},
		{Cmd: "TS", Alias: "controller-status", Description: "get controller status"},
		{Cmd: "TV", Alias: "velocity", Description: "get velocity", UsesAxis: true, IsRead: true},
		{Cmd: "TX", Alias: "controller-activity", Description: "get controller activity", IsRead: true},
		{Cmd: "VE", Alias: "controller-firmware", Description: "get controller firmware version", IsRead: true},
	}
)

// REWRITE THE CODE BELOW.
// MAKE SEVERAL SEQUENCES OF COMMANDS, BY CATEGORY (STATUS, MOTION, ETC)
// EXPOSE A /RAW-COMMAND ROUTE THAT ALLOWS USERS TO DO WHATEVER THEY WANT
// EXPOSE A /HELP ROUTE THAT RETURNS THE SEQUENCES OF COMMANDS
// WHICH INCLUDE HELPFUL INFORMATION, DESCRIPTION ETC.
// EXPOSE A FEW ROUTES LIKE:
// - /AXIS/IDX/ABSPOS   [GET] RETRIEVE ABSOLUTE POSITION [POST] SET POSITION
// - /AXIS/IDX/RELPOS   [GET] RETRIEVE RELATIVE POSITION [POST] SET RELATIVE POSITION
// - /AXIS/IDX/VELOCITY [GET] / [POST] AS ABOVE
// - /AXIS/IDX/ACCELERATION
// - /AXIS/IDX/DECELERATION
var (
	epsCommands = map[string]motionCommand{
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
