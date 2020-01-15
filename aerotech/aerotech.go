// Package aerotech provides wrappers around Aerotech ensemble motion controllers and enables more pleasant HTTP control
package aerotech

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.jpl.nasa.gov/HCIT/go-hcit/util"

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
	errClamped = errors.New("requested position violates software limits, aborted, is the axis enabled?")
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

	// Limits contains the server imposed limits on the controller
	Limits map[string]util.Limiter

	// velocities holds the velocities of the axes; the controller does not allow this to be queried, so we store it here
	velocities map[string]float64
}

// NewEnsemble returns a new Ensemble instance with the remote
// device preconfigured
func NewEnsemble(addr string, serial bool, limits map[string]util.Limiter) *Ensemble {
	// we actually need \r terminators on both sides, but ACK responses
	// are not terminated, so we strip them everywhere else.
	terms := comm.Terminators{Rx: '\n', Tx: '\n'}
	rd := comm.NewRemoteDevice(addr, false, &terms, nil)
	rd.Timeout = 60 * time.Second // long timeout for aerotech equipment
	return &Ensemble{
		RemoteDevice: &rd,
		Limits:       limits,
		velocities:   map[string]float64{}}
}

func (e *Ensemble) writeOnlyBus(msg string) error {
	err := e.Open()
	if err != nil {
		return err
	}
	e.Lock()
	defer e.Unlock()
	defer e.CloseEventually()
	err = e.Send([]byte(msg))
	if err != nil {
		return err
	}
	resp, err := getAnyResponseFromEnsemble(e.RemoteDevice, true)
	if err != nil {
		return err
	}
	// sanitize in case there is the response from a previous message here
	if len(resp) == 2 {
		resp = resp[1:] // discard the first byte (it was the old response)
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

// GetEnabled gets if the given axis is enabled or not
func (e *Ensemble) GetEnabled(axis string) (bool, error) {
	// get the status, it is a 32-bit int, which is really a bitfield
	str := fmt.Sprintf("AXISSTATUS(%s)", axis)
	resp, err := e.RemoteDevice.OpenSendRecvClose([]byte(str))
	if err != nil {
		return false, err
	}
	if resp[0] != OKCode {
		return false, ErrBadResponse{string(resp)}
	}
	if resp[0] == OKCode {
		resp = resp[1:] // expected this time
	}
	if resp[0] == OKCode {
		resp = resp[1:] // the ensemble may have a % in its write buffer still from a connection it dropped
	}

	lastByte := resp[len(resp)-1]
	return util.GetBit(lastByte, 0), nil // this might actually need to be 1 on the index
	// the very last bit in the response contains the axis enabled status
}

// Home commands the controller to home an axis
func (e *Ensemble) Home(axis string) error {
	return e.gCodeWriteOnly("HOME", axis)
}

// MoveAbs commands the controller to move an axis to an absolute position
func (e *Ensemble) MoveAbs(axis string, pos float64) error {
	set := pos
	set, err := e.clampPos(axis, pos, false)
	if err != nil {
		return err
	}
	if set != pos {
		return errClamped
	}
	posS := strconv.FormatFloat(pos, 'G', -1, 64)
	return e.gCodeWriteOnly("MOVEABS", axis, posS)
}

// MoveRel commands the controller to move an axis an incremental distance
func (e *Ensemble) MoveRel(axis string, dist float64) error {
	set := dist
	set, err := e.clampPos(axis, dist, true)
	if err != nil {
		return err
	}
	if set != dist {
		return errClamped
	}
	posS := strconv.FormatFloat(dist, 'G', -1, 64)
	return e.gCodeWriteOnly("MOVEINC", axis, posS)
}

// GetPos gets the absolute position of an axis from the controller
func (e *Ensemble) GetPos(axis string) (float64, error) {
	// this could be refactored into something like a talkReadSingleFloat
	str := fmt.Sprintf("PFBK %s", axis)
	resp, err := e.RemoteDevice.OpenSendRecvClose([]byte(str))
	if err != nil {
		return 0, err
	}
	if resp[0] != OKCode {
		return 0, ErrBadResponse{string(resp)}
	}
	// there may be a garbage OK/NOK flag in the second byte
	// if a long-running communication was cut short by TCP timeout,
	// we just discard the first byte (which came from the old message)
	// since there is nothing to do with it now (we are on a separate communication)
	if len(resp) > 2 && (resp[1] == OKCode) {
		resp = resp[1:]
	}
	str = string(resp[1:])
	return strconv.ParseFloat(str, 64)
}

// SetVelocity sets the velocity of an axis in mm/s
func (e *Ensemble) SetVelocity(axis string, vel float64) error {
	str := fmt.Sprintf("MOVEINC %s 0 %sF %.9f", axis, axis, vel)
	err := e.gCodeWriteOnly(str)
	if err != nil {
		e.velocities[axis] = vel
	}
	return err
}

// clampPos clamps an input if there is a limit set, otherwise does not modulate it
func (e *Ensemble) clampPos(axis string, input float64, relative bool) (float64, error) {
	var (
		current float64
		output  float64
		err     error = nil
	)
	if relative { // the input needs to be translated to an absolute to apply the limit.
		current, err = e.GetPos(axis)
		if err != nil {
			return input, err
		}
		input += current
	}
	if limiter, ok := e.Limits[axis]; ok {
		if (limiter.Max != 0) || (limiter.Min != 0) {
			// the alternative is uninitialized
			output = limiter.Clamp(input)
			if relative && (output != input) {
				return 0, nil
			}
		}
	} else {
		output = input
	}
	if relative {
		output -= current
	}
	// if not relative, current is zero anyway, so can always subtract it
	return output, err
}

// GetVelocity gets the velocity of an axis in mm/s
func (e *Ensemble) GetVelocity(axis string) (float64, error) {
	if vel, ok := e.velocities[axis]; ok {
		return vel, nil
	}
	return 0, errors.New("velocity not known for axis, use SetVelocity to make it known")
}

// Limit returns the limiter for an axis
func (e *Ensemble) Limit(axis string) util.Limiter {
	return e.Limits[axis]
}
