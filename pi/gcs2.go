// Package pi provides a Go interface to PI motion control systems
package pi

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.jpl.nasa.gov/bdube/golab/util"

	"github.com/tarm/serial"
	"github.jpl.nasa.gov/bdube/golab/comm"
)

/* GCS 2 primer
commands are three letters, like POS? or MOV
a command is followed by arguments.  Arguments are usually addressee-value pairs
like MOV 1 123.456 moves axis 1 to position 123.456

Queries are suffixed by ?

Axes can be addressed as 1..N or A..Z

If you send an invalid command, there is no response.
ERR? checks the error.

If you do not provide a controller number in the network, the response contains
no prefix for it.  Not sending a controller number is equivalent to sending
controller number 1.

When you do, the response is formatted as <to> <from> <msg>

So sending 4 MOV A 123.456 is == MOV A 123.456, and moves axis 1 (A) on
controller number 4 in the network.

On the query side, the response parsing is a little different.

POS? 1 begets the reply
1=0.0025210

This is functionally equal to 1 POS? 1, which is explicly axis 1 of controller 1
The reply changes to
0 1 1=0.0025210
"to address 0 (the PC), from address 1, axis 1 has pos =0.00..."
*/

// file gsc2 contains a generichttp/motion compliant implementation of this
// based on PI's GSC2 communication language
var (
	// ErrMap maps PI Error codes to the friendly strings
	ErrMap = map[int]string{
		0:    "No Error",
		1:    "Parameter syntax error",
		2:    "Unknown command",
		3:    "Command length out of limits or command buffer overrun",
		4:    "Error while scanning",
		5:    "Unallowable move attemtped on unreferenced axis, or move attempted with servo off",
		6:    "Parameter for SGA not valid",
		7:    "Position out of limits",
		8:    "Velocity out of limits",
		9:    "Attempt to set pivot point while U, V and W not all 0",
		10:   "Controller was stopped by command",
		11:   "Parameter for SST or one of the embedded scan algorithms out of range",
		12:   "Invalid axis combination for fast scan",
		13:   "Parameter for NAV out of range",
		14:   "Invalid analog channel",
		15:   "Invalid axis identifier",
		16:   "Unknown stage name",
		17:   "Parameter out of range",
		18:   "Invalid macro name",
		19:   "Error while recording macro",
		20:   "Macro not found",
		21:   "Axis has no brake",
		22:   "Axis identifier specified more than once",
		23:   "Illegal axis",
		24:   "Incorrect number of parameters",
		25:   "Invalid floating point number",
		26:   "Parameter missing",
		27:   "Soft limit out of range",
		28:   "No manual pad found",
		29:   "No more step-response values",
		30:   "No step-response values recorded",
		31:   "Axis has no reference sensor",
		32:   "Axis has no limit switch",
		33:   "No relay card installed",
		34:   "Command not allowed for selected stage(s)",
		35:   "No digital input installed",
		36:   "No digital output configured",
		37:   "No more MCM responses",
		38:   "No MCM values recorded",
		39:   "Controller number invalid",
		40:   "No joystick configured",
		41:   "Invalid axis for electronic gearing, axis can not be slave",
		42:   "Position of slave axis is out of range",
		43:   "Slave axis cannot be commanded directly when electronic gearing is enabled",
		44:   "Calibration of joystick failed",
		45:   "Referencing failed",
		46:   "OPM (Optical Power Meter) missing",
		47:   "OPM (Optical Power Meter) not initialized or cannot be initialized",
		48:   "OPM (Optical Power Meter) Communication Error",
		49:   "Move to limit switch failed",
		50:   "Attempt to reference axis with referencing disabled",
		51:   "Selected axis is controlled by joystick",
		52:   "Controller detected communication error",
		53:   "MOV! motion still in progress",
		54:   "Unknown parameter",
		55:   "No commands were recoreded with REP",
		56:   "Password invalid",
		57:   "Data Record Table does not exist",
		58:   "Source does not exist; number too low or too high",
		59:   "Source Record Table number too low or too high",
		60:   "Protected Param: current Command Level (CCL) too low",
		61:   "Command execution not possible while autozero is running",
		62:   "Autozero requires at least one linear axis",
		63:   "Initialization still in progress",
		64:   "Parameter is read-only",
		65:   "Parameter not found in non-volatile memory",
		66:   "Voltage out of limits",
		67:   "Not enough memory available for requested wav curve",
		68:   "Not enough memory available for DLL table; DLL can not be started",
		69:   "time delay larger than DLL table; DLL can not be started",
		70:   "GCS-array doesn't support different length; request arrays which have different lengths separately",
		71:   "Attempt to restart the generator while it is running in single step mode",
		72:   "MOV, MVR, STA, SVR, STE, IMP and WGO blocked when analog target is active",
		73:   "MOV, MVR, STA, SVR, STE, and IMP blocked when wave generator is active",
		100:  "PI LabVIEW driver reports error.  See source control for details",
		200:  "No stage connected to axis",
		201:  "File with axis parameter not found",
		202:  "Invalid axis parameter file",
		203:  "Backup file with axis parameters not found",
		204:  "PI internal error code 204",
		205:  "SMO with servo on",
		206:  "uudecode: incomplete header",
		207:  "uudecode: nothing to decode",
		208:  "uudecode: illegal UUE format",
		209:  "CRC32 error",
		210:  "Illegal file name (must be 8-0 format)",
		211:  "File not found on controller",
		212:  "Error writing file on controller",
		213:  "VEL command not allowed in DTR Command Mode",
		214:  "Position calculations failed",
		215:  "The connection between controller and stage may be broken",
		216:  "The connected stage has driven into a limite switch, call CLR to resume operation",
		217:  "Strut test command failed because of an unexpected strut stop",
		218:  "Position can be estimated only while MOV! is running",
		219:  "Positionw as calculated while MOV is running",
		301:  "Send buffer overflow",
		302:  "Voltage out of limits",
		304:  "Recieved command is too long",
		305:  "Error while reading/writing EEPROM",
		306:  "Error on I2C bus",
		307:  "Timeout while recieving command",
		308:  "A lengthy operation has not finished in the expected time",
		309:  "Insufficient space to store macro",
		310:  "Configuration data has old version number",
		311:  "Invalid configuration data",
		333:  "Internal hardware error",
		555:  "BasMac: unknown controller error",
		601:  "not enough memory",
		602:  "hardware voltage error",
		603:  "hardware temperature out of range",
		1000: "Too many nested macros",
		1001: "Macro already defined",
		1002: "Macro recording not activated",
		1003: "Invalid parameter for MAC",
		1004: "PI internal error code 1004",
		2000: "Controller already has a serial number",
		4000: "Sector erase failed",
		4001: "Flash program failed",
		4002: "Flash read failed",
		4003: "HW match code missing/invalid",
		4004: "FW match code missing/invalid",
		4005: "HW version missing/invalid",
		4006: "FW version missing/invalid",
		4007: "FW Update failed",
		// TODO: populate negative (interface) error codes
	}
)

// makeSerConf makes a new serial.Config with correct parity, baud, etc, set.
func makeSerConf(addr string) *serial.Config {
	return &serial.Config{
		Name:        addr,
		Baud:        115200,
		Size:        8,
		Parity:      serial.ParityNone,
		StopBits:    serial.Stop1,
		ReadTimeout: 10 * time.Minute}
}

// Controller maps to any PI controller, e.g. E-509, E-727, C-884
type Controller struct {
	*comm.RemoteDevice

	// DV is the maximum allowed voltage delta between commands
	DV *float64
}

// NewController returns a fully configured new controller
func NewController(addr string, serial bool) *Controller {
	// \r terminators
	// terms := comm.Terminators{Rx: '\r', Tx: '\r'}
	terms := comm.Terminators{Rx: 10, Tx: 10}
	rd := comm.NewRemoteDevice(addr, serial, &terms, makeSerConf(addr))
	rd.Timeout = 10 * time.Minute
	return &Controller{RemoteDevice: &rd}
}

func (c *Controller) writeOnlyBus(msg string) error {
	err := c.Open()
	if err != nil {
		return err
	}
	c.Lock()
	defer func() {
		c.Unlock()
		c.CloseEventually()
	}()
	err = c.RemoteDevice.Send([]byte(msg))
	if err != nil {
		return err
	}
	return nil
}

// copied from aerotech/gCodeWriteOnly, L108, at commit
// 5d7de8ced55aa818fd206987016c775203ef7b53
func (c *Controller) gCodeWriteOnly(msg string, more ...string) error {
	str := strings.Join(append([]string{msg}, more...), " ")
	return c.writeOnlyBus(str)
}

func (c *Controller) readBool(cmd, axis string) (bool, error) {
	str := strings.Join([]string{cmd, axis}, " ")
	err := c.RemoteDevice.Open()
	if err != nil {
		return false, err
	}
	resp, err := c.RemoteDevice.SendRecv([]byte(str))
	if err != nil {
		return false, err
	}
	str = string(resp)
	if len(str) == 0 {
		return false, fmt.Errorf("the response from the controller was blank, is the axis label correct")
	}
	// TODO: dedup this, copied from GetPos
	parts := strings.Split(str, "=")
	// could panic here, assume the response is always intact,
	// meaning parts is of length 2
	return strconv.ParseBool(parts[1])
}

func (c *Controller) readFloat(cmd, axis string) (float64, error) {
	// "POS? A" -> "A=+0080.4106"
	// use VOL? if you want voltage
	str := strings.Join([]string{cmd, axis}, " ")
	err := c.RemoteDevice.Open()
	if err != nil {
		return 0, err
	}
	resp, err := c.RemoteDevice.SendRecv([]byte(str))
	if err != nil {
		return 0, err
	}
	str = string(resp)
	if len(str) == 0 {
		return 0, fmt.Errorf("the response from the controller was blank, is the axis enabled (online, as PI says)")
	}
	parts := strings.Split(str, "=")
	// could panic here, assume the response is always intact,
	// meaning parts is of length 2
	return strconv.ParseFloat(parts[1], 64)
}

// MoveAbs commands the controller to move an axis to an absolute position
func (c *Controller) MoveAbs(axis string, pos float64) error {
	posS := strconv.FormatFloat(pos, 'G', -1, 64)
	return c.gCodeWriteOnly("MOV", axis, posS)
}

// MoveRel commands the controller to move an axis by a delta
func (c *Controller) MoveRel(axis string, delta float64) error {
	posS := strconv.FormatFloat(delta, 'G', -1, 64)
	return c.gCodeWriteOnly("MVR", axis, posS)
}

// GetPos returns the current position of an axis
func (c *Controller) GetPos(axis string) (float64, error) {
	return c.readFloat("POS?", axis)
}

// Enable causes the controller to enable motion on a given axis
func (c *Controller) Enable(axis string) error {
	return c.gCodeWriteOnly("ONL", axis, "1")
}

// Disable causes the controller to disable motion on a given axis
func (c *Controller) Disable(axis string) error {
	return c.gCodeWriteOnly("ONL", axis, "0")
}

// GetEnabled returns True if the given axis is enabled
func (c *Controller) GetEnabled(axis string) (bool, error) {
	return c.readBool("ONL?", axis)
}

// Home causes the controller to move an axis to its home position
func (c *Controller) Home(axis string) error {
	return c.gCodeWriteOnly("GOH", axis)
}

// MultiAxisMoveAbs sends a single command to the controller to move three axes
func (c *Controller) MultiAxisMoveAbs(axes []string, positions []float64) error {
	pieces := make([]string, 2*len(axes))
	idx := 0
	for i := 0; i < len(axes); i++ {
		pieces[idx] = axes[i]
		idx++
		pieces[idx] = strconv.FormatFloat(positions[i], 'G', -1, 64)
		idx++
	}
	return c.gCodeWriteOnly("MOV", pieces...)
}

// SetVoltage sets the voltage on an axis
func (c *Controller) SetVoltage(axis string, volts float64) error {
	posS := strconv.FormatFloat(volts, 'G', -1, 64)
	return c.gCodeWriteOnly("SVA", axis, posS)
}

// GetVoltage returns the voltage on an axis
func (c *Controller) GetVoltage(axis string) (float64, error) {
	return c.readFloat("SVA?", axis)
}

// MultiAxisSetVoltage sets the voltage for multiple axes
func (c *Controller) MultiAxisSetVoltage(axes []string, voltages []float64) error {
	// copied from MultiAxisMoveAbs, not DRY
	pieces := make([]string, 2*len(axes))
	idx := 0
	for i := 0; i < len(axes); i++ {
		pieces[idx] = axes[i]
		idx++
		pieces[idx] = strconv.FormatFloat(voltages[i], 'G', -1, 64)
		idx++
	}
	return c.gCodeWriteOnly("SVA", pieces...)
}

// SetVoltageSafe sets the voltage, but first does a query and enforces that
// |c.DV| is not exceeded.  If it is, the output is clamped and no error generated
func (c *Controller) SetVoltageSafe(axis string, voltage float64) error {
	v, err := c.GetVoltage(axis)

	if err != nil {
		return err
	}
	if c.DV != nil {
		dV := *c.DV
		voltage = util.Clamp(voltage, v-dV, v+dV)
	}
	return c.SetVoltage(axis, voltage)
}

// PopError returns the last error from the controller
func (c *Controller) PopError() error {
	resp, err := c.OpenSendRecvClose([]byte("ERR?"))
	if err != nil {
		return err
	}
	s := string(resp)
	if s != "0" {
		return fmt.Errorf(s)
	}
	return nil
}
