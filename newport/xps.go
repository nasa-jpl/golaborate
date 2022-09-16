package newport

/*responses are formatted as:
< err code >,TheExactTextYouSentIfThereIsABadError,EndOfAPI
or
< err code>,returnVal,EndOfApi

Ex:
"-20,GroupPositionCurrentGet(Group1,double *),EndOfAPI"
(status code -20 => fatal init)
OR
"0,0.000314605934,EndOfAPI"
(status code 0, OK)
*/

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/nasa-jpl/golaborate/comm"
)

// XPS manual states that up to 84 connections are supported (XPS-D)
const xpsConcurrencyLimit = 84

type xpsResponse struct {
	errCode int
	content string
}

func parse(input string) xpsResponse {
	// first, check what the error codepoint is.
	pieces := strings.SplitN(input, ",", 2)
	if len(pieces) < 2 {
		return xpsResponse{errCode: 2, content: input}
	}
	ecode, err := strconv.Atoi(pieces[0])
	if err != nil {
		return xpsResponse{errCode: 3, content: input}
	}

	// now pop EndOfAPI
	resp := pieces[1]
	endIdx := strings.Index(resp, "EndOfAPI")
	if endIdx == -1 {
		return xpsResponse{errCode: 2, content: input}
	}
	resp = resp[:endIdx]
	if strings.HasSuffix(resp, ",") {
		resp = resp[:endIdx-1]
	}
	// now resp is just the bit we actually want
	return xpsResponse{errCode: ecode, content: resp}
}

// XPSError is a formatible error code from the XPS
type XPSError struct {
	code int
}

func intToXPSStatus(i int) XPSStatus {
	if str, ok := XPSGroupStatuses[i]; ok {
		return XPSStatus{Code: i, Text: str}
	}
	return XPSStatus{Code: i, Text: "UNKNOWN STATUS"}
}

func (e XPSError) Error() string {
	if s, ok := XPSErrorCodes[e.code]; ok {
		return fmt.Sprintf("%d - %s", e.code, s)
	}
	return fmt.Sprintf("%d - UNKNOWN ERROR CODE", e.code)
}

// XPSStatus is a struct holding a status code and its string
type XPSStatus struct {
	// Code is the status code
	Code int

	// Text is the textual version of the status
	Text string
}

// IsReady returns true if the axis status is "ready" or false if not
func (s XPSStatus) IsReady() bool {
	c := s.Code
	if (c >= 10 && c < 20) || (c == 70) || (c == 77) {
		return true
	}
	return false
}

// IsHomed returns true if the axis status is "homed" or false if not
func (s XPSStatus) IsHomed() bool {
	c := s.Code
	if (c >= 10 && c <= 18) {
		return true
	}
	return false
}

var (
	// XPSErrorCodes maps XPS error integers to strings
	XPSErrorCodes = map[int]string{
		0: "SUCCESS",

		-115: "HARDWARE FUNCTION NOT SUPPORTED",
		-113: "BOTH ENDS OF RUNS ACTIVATED",
		-112: "EXCITATION SIGNAL INITIALIZATION",
		-111: "GATHERING BUFFER FULL",
		-110: "NOT ALLOWED FOR GANTRY",
		-109: "NEED TO BE HOMED AT LEAST ONCE",
		-108: "SOCKET CLOSED BY ADMIN",
		-107: "NEED ADMINISTRATOR RIGHTS",
		-106: "WRONG USERNAME OR PASSWORD",
		-105: "SCALING CALIBRATION",
		-104: "PID TUNING INITIALIZATION",
		-103: "SIGNAL POINTS NOT ENOUGH",
		-102: "RELAY FEEDBACK TEST SIGNAL NOISY",
		-101: "RELAY FEEDBACK TEST NO OSCILLATION",
		-100: "INTERNAL ERROR",
		-99:  "FATAL EXTERNAL MODULE LOAD",
		-98:  "OPTIONAL EXTERNAL MODULE UNLOAD",
		-97:  "OPTIONAL EXTERNAL MODULE LOAD",
		-96:  "OPTIONAL EXTERNAL MODULE KILL",
		-95:  "OPTIONAL EXTERNAL MODULE EXECUTE",
		-94:  "OPTIONAL EXTERNAL MODULE FILE",

		-85: "HOME SEARCH GANTRY TOLERANCE ERROR",
		-83: "EVENT ID UNDEFINED",
		-82: "EVENT BUFFER FULL",
		-81: "ACTIONS NOT CONFIGURED",
		-80: "EVENTS NOT CONFIGURED",

		-75: "TRAJ TIME",
		-74: "READ FILE PARAMETER KEY",
		-73: "END OF FILE",
		-72: "TRAJ INIITALIZATION",
		-71: "MSG QUEUE",
		-70: "TRAJ FINAL VELOCITY",
		-69: "TRAJ ACC LIMIT",
		-68: "TRAJ VEL LIMIT",
		// no -67
		-66: "TRAJ EMPTY",
		-65: "TRAJ ELEM LINE",
		-64: "TRAJ ELEM SWEEP",
		-63: "TRAJ ELEM RADIUS",
		-62: "TRAJ ELEM TYPE",
		-61: "READ FILE",
		-60: "WRITE FILE",

		-51: "SPIN OUT OF RANGE",
		-50: "MOTOR INITIALIZATION ERROR",
		-49: "GROUP HOME SEARCH ZM ERROR",
		-48: "BASE VELOCITY",
		-47: "WRONG TCL TASKNAME",
		-46: "NOT ALLOWED BACKLASH",
		-45: "END OF RUN",
		-44: "SLAVE",
		-43: "GATHERING RUNNING",
		-42: "JOB OUT OF RANGE",
		-41: "SLAVE CONFIGURATION",
		-40: "MNEMO EVENT",
		-39: "NMEMO ACTION",
		-38: "TCL INTERPRETOR",
		-37: "TCL SCRIPT KILL",
		-36: "UNKNOWN TCL FILE",
		-35: "TRAVEL LIMITS",
		// no -34
		-33: "GROUP MOTION DONE TIMEOUT",
		-32: "GATHERING NOT CONFIGURED",
		-31: "HOME OUT OF RANGE",
		-30: "GATHERING NOT STARTED",
		-29: "MNEMOTYPEGATHERING",
		-28: "GROUP HOME SEARCH TIMEOUT",
		-27: "GROUP ABORT MOTION",
		-26: "EMERGENCY SIGNAL",
		-25: "FOLLOWING ERROR",
		-24: "UNCOMPATIBLE",
		-23: "POSITION COMPARE NOT SET",
		-22: "NOT ALLOWED ACTION",
		-21: "IN INITIALIZATION",
		-20: "FATAL INIT",
		-19: "GROUP NAME",
		-18: "POSITIONER NAME",
		-17: "PARAMETER OUT OF RANGE",
		-16: "WRONG TYPE UNSIGNEDINT",
		-15: "WRONG TYPE INT",
		-14: "WRONG TYPE DOUBLE",
		-13: "WRONG TYPE CHAR",
		-12: "WRONG TYPE BOOL",
		-11: "WRONG TYPE BIT WORD",
		-10: "WRONG TYPE",
		-9:  "WRONG PARAMETER NUMBER",
		-8:  "WRONG OBJECT TYPE",
		-7:  "WRONG FORMAT",
		// no -6
		-5: "POSITIONER ERROR",
		-4: "UNKNOWN COMMAND",
		-3: "STRING TOO LONG",
		-2: "TCP TIMEOUT",
		-1: "BUSY SOCKET",
		1:  "TCL INTERPRETOR ERROR",
		2:  "RESPONSE INCOMPLETE", // below here is custom and not part of the XPS "spec"
		3:  "ERROR PARSING ERROR CODE",
	}

	// XPSGroupStatuses maps status ints to their strings for XPS groups
	//
	// if i < 10 || i == 50, things are not initialized.
	// 10 < i < 20 OK/ready.
	// 20 <= i < 40, disabled
	// 10 <= i < 18, homed/ready
	XPSGroupStatuses = map[int]string{
		0:  "Not initialized state",
		1:  "Not initialized state due to an emergency brake : see positioner status",
		2:  "Not initialized state due to an emergency stop : see positioner status",
		3:  "Not initialized state due to a following error during homing",
		4:  "Not initialized state due to a following error",
		5:  "Not initialized state due to a homing timeout",
		6:  "Not initialized state due to a motion done timeout during homing",
		7:  "Not initialized state due to a KillAll command",
		8:  "Not initialized state due to an end of run after homing",
		9:  "Not initialized state due to an encoder calibration error",
		10: "Ready state due to an AbortMove command",
		11: "Ready state from homing",
		12: "Ready state from motion",
		13: "Ready state due to a MotionEnable command",
		14: "Ready state from slave",
		15: "Ready state from jogging",
		16: "ready state from analog tracking",
		17: "Ready state from trajectory",
		18: "Ready state from spinning",
		// 19 skipped
		20: "Disable state",
		21: "Disabled state due a following error on ready state",
		22: "Disabled state due to a following error during motion",
		23: "Disabled state due to a motion done timeout during moving",
		24: "Disabled state due to a following error on slave state",
		25: "Disabled state due to a following error on jogging state",
		26: "Disabled state due to a following error during trajectory",
		27: "Disabled state due to a motion done timeout during trajectory",
		28: "Disabled state due to a following error during analog tracking",
		29: "Disabled state due to a slave error during motion",
		30: "Disabled state due to a slave error on slave state",
		31: "Disabled state due to a slave error on jogging state",
		32: "Disabled state due to a slave error during trajectory",
		33: "Disabled state due to a slave error during analog tracking",
		34: "Disabled state due to a slave error on ready state",
		35: "Disabled state due to a following error on spinning state",
		36: "Disabled state due to a slave error on spinning state",
		37: "Disabled state due to a following error on auto-tuning",
		38: "Disabled state due to a slave error on auto-tuning",
		// 39 skipped
		40: "Emergency braking",
		41: "Motor initialization state",
		42: "Not referenced state",
		43: "Homing state",
		44: "moving state",
		45: "Trajectory state",
		46: "slave state due to a SlaveEnable command",
		47: "Jogging state due to a JogEnable command",
		48: "Analog tracking state due to a TrackingEnable command",
		49: "Analog interpolated encoder calibration state",
		50: "Not initialized state due to a mechanical zero inconsistency during homing",
		51: "Spinning state due to a SpinParametersSet command",
		// 52~62 skipped
		63: "Not initialized state due to a motor initialization error",
		64: "Referencing state",
		// 65 skipped
		66: "Not initialized state due to a perpendicularity error homing",
		67: "Not initialized state due to a master/slave error during homing",
		68: "Auto-tuning state",
		69: "Scaling calibration state",
		70: "Ready state from auto-tuning",
		71: "Not initialized state from scaling calibration",
		72: "Not initialized state due to a scaling calibration error",
		73: "Excitation signal generation state",
		74: "Disable state due to a following error on excitation signal generation state",
		75: "Disable state due to a master/slave error on excitation signal generation state",
		76: "Disable state due to an emergency stop on excitation signal generation state",
		77: "Ready state from excitation signal generation",
	}
)

// XPSErr converts an error code to something that implements the error interface
func XPSErr(code int) error {
	if code == 0 {
		return nil
	}
	return XPSError{code}
}

/*XPS represents an XPS series motion controller.

Note that the programming manual has a lot of socket numbers sprinkled around.
We do not see any here because the embedded pool manages that for us.  The
controller supports up to 100 concurrent sockets.

While newport markets the XPS as a more versatile and consistent
(vis-a-vis communication) product than the older ESP line, this is not really
true in the author of this package's opinion.  For example, there is no function
that returns the number of positioners in a group, yet to move a positioner it
must belong to a group, and when you get the position of the group you must
supply the number of positioners to query for.  Consequently, a best practice
emerges to simply put each positioner in its own group, and not use the group
functionality at all.  This practice eliminates the ability to work with groups
the way they are used in most other motion controllers, which is a shame.

There is also no use of terminators for datagrams; the controller scans inputs
for valid entries and sends them back to enable synchronization.

You also send commands formatted as C/++ source code, which is presumably
compiled or interpreted by the controller.
*/
type XPS struct {
	pool *comm.Pool
}

// NewXPS makes a new XPS instance
func NewXPS(addr string) *XPS {
	maker := comm.BackingOffTCPConnMaker(addr, 3*time.Second)
	pool := comm.NewPool(xpsConcurrencyLimit, 30*time.Second, maker)
	return &XPS{pool: pool}
}

func (xps *XPS) openReadWriteClose(cmd string) (xpsResponse, error) {
	var resp xpsResponse
	conn, err := xps.pool.Get()
	if err != nil {
		return resp, err
	}
	defer func() { xps.pool.ReturnWithError(conn, err) }()
	msg := []byte(cmd)
	n, err := conn.Write(msg)
	if err != nil {
		return resp, err
	} else if n != len(msg) {
		return resp, errors.New("XPS did not accept the entire message")
	}
	// apparently the XPS always writes everything in one packet.
	// I sure hope so, otherwise we will get data loss since NACKs don't have "EndOfAPI"
	// and there is nothing to scan for.  Really garbage interface on their part.
	buf := make([]byte, 1500)
	n, err = conn.Read(buf)
	if err != nil {
		return resp, err
	}
	buf = buf[:n]
	resp = parse(string(buf))
	return resp, nil
}

// Enable enables the axis
func (xps *XPS) Enable(axis string) error {
	cmd := fmt.Sprintf("GroupMotionEnable(%s)", axis)
	resp, err := xps.openReadWriteClose(cmd)
	if err != nil {
		return err
	}
	return XPSErr(resp.errCode)
}

// Disable disables the axis
func (xps *XPS) Disable(axis string) error {
	cmd := fmt.Sprintf("GroupMotionDisable(%s)", axis)
	resp, err := xps.openReadWriteClose(cmd)
	if err != nil {
		return err
	}
	return XPSErr(resp.errCode)
}

// GetStatus gets the current status of an axis (group) from the controller
func (xps *XPS) GetStatus(axis string) (XPSStatus, error) {
	cmd := fmt.Sprintf("GroupStatusGet(%s, int *)", axis)
	resp, err := xps.openReadWriteClose(cmd)
	if err != nil {
		return XPSStatus{}, err
	}
	i, err := strconv.Atoi(resp.content)
	if err != nil {
		return XPSStatus{}, err
	}
	return intToXPSStatus(i), nil
}

// GetEnabled gets if the axis is enabled
func (xps *XPS) GetEnabled(axis string) (bool, error) {
	// todo: look at GroupMotionStatusGet
	cmd := fmt.Sprintf("GroupStatusGet(%s, int *)", axis)
	resp, err := xps.openReadWriteClose(cmd)
	if err != nil {
		return false, err
	}
	i, err := strconv.Atoi(resp.content)
	if err != nil {
		return false, err
	}
	status := intToXPSStatus(i)
	return status.IsReady(), nil
	// return false, errors.New("XPS controllers do not have a way to query if motion is enabled")
}

// GetPos gets the absolute position of an axis
func (xps *XPS) GetPos(axis string) (float64, error) {
	cmd := fmt.Sprintf("GroupPositionCurrentGet(%s, double *)", axis)
	resp, err := xps.openReadWriteClose(cmd)
	if err != nil {
		return 0, err
	}
	if resp.errCode != 0 {
		return 0, XPSErr(resp.errCode)
	}
	return strconv.ParseFloat(resp.content, 64)
}

// Home homes the axis
func (xps *XPS) Home(axis string) error {
	cmd := fmt.Sprintf("GroupHomeSearch(%s)", axis)
	resp, err := xps.openReadWriteClose(cmd)
	if err != nil {
		return err
	}
	return XPSErr(resp.errCode)
}

// GetHomed gets if the axis is homed
func (xps *XPS) GetHomed(axis string) (bool, error) {
	cmd := fmt.Sprintf("GroupStatusGet(%s, int *)", axis)
	resp, err := xps.openReadWriteClose(cmd)
	if err != nil {
		return false, err
	}
	i, err := strconv.Atoi(resp.content)
	if err != nil {
		return false, err
	}
	status := intToXPSStatus(i)
	return status.IsHomed(), nil
	// return false, errors.New("XPS controllers do not have a way to query if motion is enabled")
}

// Initialize initializes the axis
func (xps *XPS) Initialize(axis string) error {
	cmd := fmt.Sprintf("GroupInitialize(%s)", axis)
	resp, err := xps.openReadWriteClose(cmd)
	if err != nil {
		return err
	}
	return XPSErr(resp.errCode)
}

// MoveAbs moves an axis to an absolute position
func (xps *XPS) MoveAbs(axis string, pos float64) error {
	cmd := fmt.Sprintf("GroupMoveAbsolute(%s,%f)", axis, pos)
	resp, err := xps.openReadWriteClose(cmd)
	if err != nil {
		return err
	}
	return XPSErr(resp.errCode)
}

// MoveRel moves the axis a relative distance
func (xps *XPS) MoveRel(axis string, pos float64) error {
	cmd := fmt.Sprintf("GroupMoveRelative(%s,%f)", axis, pos)
	resp, err := xps.openReadWriteClose(cmd)
	if err != nil {
		return err
	}
	return XPSErr(resp.errCode)
}

// GetVelocity retrieves the velocity setpoint for an axis
func (xps *XPS) GetVelocity(axis string) (float64, error) {
	axis += ".Pos"
	cmd := fmt.Sprintf("PositionerSGammaParametersGet(%s, double *, double *, double *, double *)", axis)
	resp, err := xps.openReadWriteClose(cmd)
	if err != nil {
		return 0, err
	}
	if resp.errCode != 0 {
		return 0, XPSErr(resp.errCode)
	}
	// return is CSV, we only want the first parameter
	chunks := strings.Split(resp.content, ",")
	return strconv.ParseFloat(chunks[0], 64)
}

// SetVelocity sets the velocity setpoint for an axis
func (xps *XPS) SetVelocity(axis string, vel float64) error {
	axis += ".Pos"
	cmd := fmt.Sprintf("PositionerSGammaParametersGet(%s, double *, double *, double *, double *)", axis)
	resp, err := xps.openReadWriteClose(cmd)
	if err != nil {
		return err
	}
	if resp.errCode != 0 {
		return XPSErr(resp.errCode)
	}
	// return is CSV, we only want to change the first parameter
	chunks := strings.Split(resp.content, ",")
	s := strconv.FormatFloat(vel, 'G', -1, 64)
	chunks[0] = s
	unchunked := strings.Join(chunks, ",")
	cmd = fmt.Sprintf("PositionerSGammaParametersSet(%s, %s)", axis, unchunked)
	resp, err = xps.openReadWriteClose(cmd)
	if err != nil {
		return err
	}
	if resp.errCode != 0 {
		return XPSErr(resp.errCode)
	}
	return nil

}

// Raw implements ascii.Rawer
func (xps *XPS) Raw(s string) (string, error) {
	resp, err := xps.openReadWriteClose(s)
	if err != nil {
		return "", err
	}
	if resp.errCode != 0 {
		return "", XPSErr(resp.errCode)
	}
	return resp.content, nil
}
