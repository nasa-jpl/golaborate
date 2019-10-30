package newport

import (
	"fmt"

	"github.jpl.nasa.gov/HCIT/go-hcit/comm"
	"github.jpl.nasa.gov/HCIT/go-hcit/server"
	"github.jpl.nasa.gov/HCIT/go-hcit/util"
)

var (
	// XPSErrorCodes maps XPS error integers to strings
	XPSErrorCodes = map[int]string{
		0:    "SUCCESS",
		-115: "ERR_HARDWARE_FUNCTION_NOT_SUPPORTED",
		-113: "ERR_BOTH_ENDS_OF_RUNS_ACTIVATED",
		-112: "ERR_EXCITATION_SIGNAL_INITIALIZATION",
		-111: "ERR_GATHERING_BUFFER_FULL",
		-110: "ERR_NOT_ALLOWED_FOR_GANTRY",
		-109: "ERR_NEED_TO_BE_HOMED_AT_LEAST_ONCE",
		-108: "ERR_SOCKET_CLOSED_BY_ADMIN",
		-107: "ERR_NEED_ADMINISTRATOR_RIGHTS",
		-106: "ERR_WRONG_USERNAME_OR_PASSWORD",
		-105: "ERR_SCALING_CALIBRATION",
		-104: "ERR_PID_TUNING_INITIALIZATION",
		-103: "ERR_SIGNAL_POINTS_NOT_ENOUGH",
		-102: "ERR_RELAY_FEEDBACK_TEST_SIGNAL_NOISY",
		-101: "ERR_RELAY_FEEDBACK_TEST_NO_OSCILLATION",
		-100: "ERR_INTERNAL_ERROR",
		-99:  "ERR_FATAL_EXTERNAL_MODULE_LOAD",
		-98:  "ERR_OPTIONAL_EXTERNAL_MODULE_UNLOAD",
		-97:  "ERR_OPTIONAL_EXTERNAL_MODULE_LOAD",
		-96:  "ERR_OPTIONAL_EXTERNAL_MODULE_KILL",
		-95:  "ERR_OPTIONAL_EXTERNAL_MODULE_EXECUTE",
		-94:  "ERR_OPTIONAL_EXTERNAL_MODULE_FILE",
		-85:  "ERR_HOME_SEARCH_GANTRY_TOLERANCE_ERROR",
		-83:  "ERR_EVENT_ID_UNDEFINED",
		-82:  "ERR_EVENT_BUFFER_FULL",
		-81:  "ERR_ACTIONS_NOT_CONFIGURED",
		-80:  "ERR_EVENTS_NOT_CONFIGURED",
		-75:  "ERR_TRAJ_TIME",
		-74:  "ERR_READ_FILE_PARAMETER_KEY",
		-73:  "ERR_END_OF_FILE",
		-72:  "ERR_TRAJ_INIITALIZATION",
		-71:  "ERR_MSG_QUEUE",
		-70:  "ERR_TRAJ_FINAL_VELOCITY",
		-69:  "ERR_TRAJ_ACC_LIMIT",
		-68:  "ERR_TRAJ_VEL_LIMIT",
		// no -67
		-66: "ERR_TRAJ_EMPTY",
		-65: "ERR_TRAJ_ELEM_LINE",
		-64: "ERR_TRAJ_ELEM_SWEEP",
		-63: "ERR_TRAJ_ELEM_RADIUS",
		-62: "ERR_TRAJ_ELEM_TYPE",
		-61: "ERR_READ_FILE",
		-60: "ERR_WRITE_FILE",
		-51: "ERR_SPIN_OUT_OF_RANGE",
		-50: "ERR_MOTOR_INITIALIZATION_ERROR",
		-49: "ERR_GROUP_HOME_SEARCH_ZM_ERROR",
		-48: "ERR_BASE_VELOCITY",
		-47: "ERR_WRONG_TCL_TASKNAME",
		-46: "ERR_NOT_ALLOWED_BACKLASH",
		-45: "ERR_END_OF_RUN",
		-44: "ERR_SLAVE",
		-43: "ERR_GATHERING_RUNNING",
		-42: "ERR_JOB_OUT_OF_RANGE",
		-41: "ERR_SLAVE_CONFIGURATION",
		-40: "ERR_MNEMO_EVENT",
		-39: "ERR_NMEMO_ACTION",
		-38: "ERR_TCL_INTERPRETOR",
		-37: "ERR_TCL_SCRIPT_KILL",
		-36: "ERR_UNKNOWN_TCL_FILE",
		-35: "ERR_TRAVEL_LIMITS",
		// no -34
		-33: "ERR_GROUP_MOTION_DONE_TIMEOUT",
		-32: "ERR_GATHERING_NOT_CONFIGURED",
		-31: "ERR_HOME_OUT_OF_RANGE",
		-30: "ERR_GATHERING_NOT_STARTED",
		-29: "ERR_MNEMOTYPEGATHERING",
		-28: "ERR_GROUP_HOME_SEARCH_TIMEOUT",
		-27: "ERR_GROUP_ABORT_MOTION",
		-26: "ERR_EMERGENCY_SIGNAL",
		-25: "ERR_FOLLOWING_ERROR",
		-24: "ERR_UNCOMPATIBLE",
		-23: "ERR_POSITION_COMPARE_NOT_SET",
		-22: "ERR_NOT_ALLOWED ACTION",
		-21: "ERR_IN_INITIALIZATION",
		-20: "ERR_FATAL_INIT",
		-19: "ERR_GROUP_NAME",
		-18: "ERR_POSITIONER_NAME",
		-17: "ERR_PARAMETER_OUT_OF_RANGE",
		-16: "ERR_WRONG_TYPE_UNSIGNEDINT",
		-15: "ERR_WRONG_TYPE_INT",
		-14: "ERR_WRONG_TYPE_DOUBLE",
		-13: "ERR_WRONG_TYPE_CHAR",
		-12: "ERR_WRONG_TYPE_BOOL",
		-11: "ERR_WRONG_TYPE_BIT_WORD",
		-10: "ERR_WRONG_TYPE",
		-9:  "ERR_WRONG_PARAMETER_NUMBER",
		-8:  "ERR_WRONG_OBJECT_TYPE",
		-7:  "ERR_WRONG_FORMAT",
		// no -6
		-5: "ERR_POSITIONER_ERROR",
		-4: "ERR_UNKNOWN_COMMAND",
		-3: "ERR_STRING_TOO_LONG",
		-2: "ERR_TCP_TIMEOUT",
		-1: "ERR_BUSY_SOCKET",
		1:  "ERR_TCL_INTERPRETOR_ERROR",
	}
)

// XPSErr is a fancy Error() wrapper around error codes
type XPSErr int

// Error implements the error interface
func (e XPSErr) Error() string {
	if s, ok := XPSErrorCodes[int(e)]; ok {
		return fmt.Sprintf("%d - %s", e, s)
	}
	return fmt.Sprintf("%d - ERROR_UNKNOWN_TO_GO-HCIT", e)
}

// XPSError converts an error code to something that implements the error interface
func XPSError(code int) error {
	if code == 0 {
		return nil
	}
	return XPSErr(code)
}

// popError pulls the error code off of a raw response if it is present
// and returns the code as an int and the trimmed string
func popError(resp string) (int, string) {
	return 0, ""
}

/*XPS represents an XPS series motion controller.

While newport markets the XPS as a more versatile and consistent
(vis-a-vis communication) product than the older ESP line, this is not really
true in the author of this package's opinion.  For example, there is no function
that returns the number of positioners in a group, yet to move a positioner it
must belong to a group, and when you get the position of the group you must
supply the number of positioners to query for.  Consequently, a best practice
emerges to simply put each positioner in its own group, and not use the group
functionality at all.  This practice eliminates the ability to work with groups
the way they are used in most other motion controllers, which is a shame.
*/
type XPS struct {
	*comm.RemoteDevice
	server.Server
}

// NewXPS makes a new XPS instance
func NewXPS(addr, urlStem string) *XPS {
	rd := comm.NewRemoteDevice(addr, false, nil, makeSerConf(addr))
	srv := server.NewServer(urlStem)
	xps := XPS{RemoteDevice: &rd}
	xps.Server = srv
	return &xps
}

// GroupMoveAbsolute moves a group to an absolute position
func (xps *XPS) GroupMoveAbsolute(gid string, pos []float64) error {
	fstr := util.Float64SliceToCSV(pos, 'G', 9)
	cmd := fmt.Sprintf("GroupMoveAbsolute(%s,%s)", gid, fstr)
	fmt.Println(cmd)
	return nil
}

// GroupPositionCurrentGet gets the current absolute position of a group.
// Note that we hard-code "nbElements" to 1, since there is no way to
// query for that information, so the user will not know unless they are the
// only person who has ever assigned group elements on the controller.
func (xps *XPS) GroupPositionCurrentGet(gid string) ([]float64, error) {
	cmd := fmt.Sprintf("GroupPositionCurrentGet(%s, double *)", gid)
	fmt.Println(cmd)
	return []float64{0}, nil
}
