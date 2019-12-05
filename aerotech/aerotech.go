// Package aerotech provides wrappers around Aerotech ensemble motion controllers and enables more pleasant HTTP control
package aerotech

import (
	"fmt"
	"strconv"
	"strings"

	"github.jpl.nasa.gov/HCIT/go-hcit/comm"
)

// the Aerotech ASCII interface may seem like nonsense, and it partially is.
// It is this way in one part due to heritage, and in another for compatibility.
// Most CNC controllers use a scripting language called Gcode.  Aerotech's ASCII
// language is almost Gcode, such that a little text replacement is all that
// is needed to convert one to the other.
//
// For example, the string "moveabs X 10.00" moves the X axis to 10.00 units
//
// the equivalent G code is "G01 X 10.00"
//
// There are optional extras, for example "moveabs X 10.00 XF 1" uses a feedrate
// of 1 unit/sec
//
// The G code is "G01 X 10.00 F 1", since G-code does not support different
// feeds for different axes (at least in the ANSI standard)
// The more functional language, for example "PFBKPROG(X)" to get the
// position feedback before calibration, gearing, and camming
// is a more modern syntax. and the two exist more or less indepdently.
//
// getting the position is not in a gcode-y syntax, probably,
// because G code does not have a standard command for that.

// The Ensemble controllers do not actually accept G code inputs.

// This wrapper is quite simple with few guarantees.

const (
	// OKCode is the first byte in the controller's response when the message
	// is acknowledged and response nominal
	OKCode = byte(37)

	// BadReqCode is the first byte in the controller's response when the message
	// was not understood
	BadReqCode = byte(33)
)

var (
	// MotionAliases maps B. Dube aliases to the way Aerotech wants them
	MotionAliases = map[string]string{
		"move-abs": "MOVEABS",
		"move-rel": "MOVEINC",
		"get-pos":  "PFBKPROG",
	}

	// CmdGCode maps commands (as aerotech knows them, not B. Dube aliases)
	// to if a command is g-code style (true) or functional style (false)
	CmdGCode = map[string]bool{
		"MOVEABS": true,
		"MOVEINC": true,
		"HOME":    true,
		"ENABLE":  true,
		"DISABLE": true,
	}
)

// ErrBadResponse is generated when a bad response comes from the controller
type ErrBadResponse struct {
	resp string
}

func (e ErrBadResponse) Error() string {
	return fmt.Sprintf("bad response, OK returns %%, got %s", e.resp)
}

// Ensemble represents an Ensemble motion controller
type Ensemble struct {
	*comm.RemoteDevice
}

// NewEnsemble returns a new Ensemble instance with the remote
// device preconfigured
func NewEnsemble(addr string, serial bool) *Ensemble {
	term := comm.Terminators{Rx: '\n', Tx: '\n'}
	rd := comm.NewRemoteDevice(addr, false, &term, nil)
	return &Ensemble{RemoteDevice: &rd}
}

func (e *Ensemble) writeOnlyBus(msg string) error {
	resp, err := e.RemoteDevice.OpenSendRecvClose([]byte(msg))
	if err != nil {
		return err
	}
	if len(resp) != 1 || resp[0] != OKCode {
		return ErrBadResponse{string(resp)}
	}
	return nil
}

func (e *Ensemble) gCodeWriteOnly(msg string, more ...string) error {
	str := strings.Join(append([]string{msg}, more...), " ")
	return e.writeOnlyBus(str)
}

// Enable commands the controller to enable an axis
func (e *Ensemble) Enable(axis string) error {
	return e.gCodeWriteOnly("ENABLE", axis)
}

// Disable commands the controller to disable an axis
func (e *Ensemble) Disable(axis string) error {
	return e.gCodeWriteOnly("DISABLE", axis)
}

// GetAxisEnabled gets if the given axis is enabled or not
func (e *Ensemble) GetAxisEnabled(axis string) (bool, error) {
	// get the status, it is a 32-bit int, which is really a bitfield
	str := fmt.Sprintf("AXISSTATUS(%s)", axis)
	resp, err := e.RemoteDevice.OpenSendRecvClose([]byte(str))
	if err != nil {
		return false, err
	}
	if resp[0] != OKCode {
		return false, ErrBadResponse{string(resp)}
	}
	i, err := strconv.Atoi(string(resp[1:]))
	return i > 0, err
	// bit0, most significant bit contains the sign
	// it also contains the axis enabled value
	// so, if the int is positive, the axis is enabled
}

// Home commands the controller to home an axis
func (e *Ensemble) Home(axis string) error {
	return e.gCodeWriteOnly("HOME", axis)
}

// MoveAbs commands the controller to move an axis to an absolute position
func (e *Ensemble) MoveAbs(axis string, pos float64) error {
	posS := strconv.FormatFloat(pos, 'G', -1, 64)
	return e.gCodeWriteOnly("MOVEABS", axis, posS)
}

// MoveRel commands the controller to move an axis an incremental distance
func (e *Ensemble) MoveRel(axis string, dist float64) error {
	posS := strconv.FormatFloat(dist, 'G', -1, 64)
	return e.gCodeWriteOnly("MOVEINC", axis, posS)
}

// GetPos gets the absolute position of an axis from the controller
func (e *Ensemble) GetPos(axis string) (float64, error) {
	// this could be refactored into something like a talkReadSingleFloat
	str := fmt.Sprintf("PFBKPROG(%s)", axis)
	resp, err := e.RemoteDevice.OpenSendRecvClose([]byte(str))
	if err != nil {
		return 0, err
	}
	if resp[0] != OKCode {
		return 0, ErrBadResponse{string(resp)}
	}
	str = string(resp[1:])
	return strconv.ParseFloat(str, 64)
}
