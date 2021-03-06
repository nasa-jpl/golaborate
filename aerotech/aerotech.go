// Package aerotech provides wrappers around Aerotech ensemble motion controllers and enables more pleasant HTTP control
package aerotech

import (
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/nasa-jpl/golaborate/comm"
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

// Ensemble represents an Ensemble motion controller
type Ensemble struct {
	pool *comm.Pool

	timeout time.Duration

	// velocities holds the velocities of the axes; the controller does not allow this to be queried, so we store it here
	velocities map[string]float64

	asyncMode *bool
}

// NewEnsemble returns a new Ensemble instance
func NewEnsemble(addr string, connectSerial bool) *Ensemble {
	// we actually need \r terminators on both sides, but ACK responses
	// are not terminated, so we strip them everywhere else.
	maker := comm.BackingOffTCPConnMaker(addr, 3*time.Second)
	pool := comm.NewPool(1, 30*time.Second, maker)
	return &Ensemble{
		pool:       pool,
		velocities: map[string]float64{},
		timeout:    300 * time.Second}
}

func (e *Ensemble) writeReadRaw(msg string) (response, error) {
	/* this function works as follows:
	Declare some outer scope error and trial counts,
	these are the overall error and number of attempts.
	We want to constrain retry to some number of attempts.
	We do not use off-the-shelf retry, because there are
	two potential things we want to retry specifically,
	not just the overall action.

	Get a connection and try writing to it;
	if it's junk immediately trash it and try the write until it succeeds.
	Then, reusing the connection or getting a new one if it's junk,
	read for N/OK.
	*/
	var (
		resp     response
		conn     io.ReadWriter
		wrap     io.ReadWriter
		werr     error = io.EOF
		tries          = 0
		MaxTries       = 3
	)
	// enter the write attempt, clean slate.  No attempts, no connections.
	for werr != nil && tries < MaxTries {
		var err error
		conn, err = e.pool.Get()
		if err != nil {
			// error getting a connection, bail completely
			return resp, err
		}
		wrap, err = comm.NewTimeout(conn, e.timeout)
		wrap = comm.NewTerminator(wrap, Terminator, Terminator)
		if err != nil {
			// timeout unsupported, bail completely
			return resp, err
		}
		_, werr = io.WriteString(wrap, msg)
		if werr != nil {
			// write error, need to scan for the magic string "connection reset"
			errS := werr.Error()
			if strings.Contains(errS, "reset") {
				// reset by peer -- try again
				tries++
				e.pool.Destroy(conn)
				continue
				// do not need to rebuild the connection here, happens on the
				// next loop entry
			}
			// do not know how to handle other errors
			e.pool.Destroy(conn)
			return resp, werr
		}
		// succeeded in writing, continue to reading
		// can't defer connection cleanup, because we may trash
		// it in the read attempt
		break
	}
	/*
		Now we enter the second part, reading.  The state here is:

		1) we have a connection, which may or may not be junk
		2) we want to read.  The controller writes everything
			in one packet, so a single read call with a
			decent sized buffer suffices.
			Then we just evaluate the N/OK code.
			Or return any irrecoverable errors along the way.
			If the code is OK, return nil.

	*/
	n := 0
	tries = 0
	werr = io.EOF
	tcpFrameSize := 1500
	buf := make([]byte, tcpFrameSize)
	for werr != nil && tries < MaxTries {
		// no terminator wrapper here, because the OK/NO
		// codes come unterminated.  Assume one data block, as commented above.
		n, werr = wrap.Read(buf)
		if werr != nil {
			// an error, check if the connection was reset, and if so recycle
			// it and get a new one
			errS := werr.Error()
			if strings.Contains(errS, "reset") {
				tries++
				e.pool.Destroy(conn)
				// below here copy pasted from above
				var err error
				conn, err = e.pool.Get()
				if err != nil {
					// error getting a connection, bail completely
					return resp, err
				}
				wrap, err = comm.NewTimeout(conn, e.timeout)
				wrap = comm.NewTerminator(wrap, Terminator, Terminator)
				if err != nil {
					// timeout unsupported, bail completely
					return resp, err
				}
				// now we have remade the connection, reboot the loop
				continue
			}
			// do not know how to handle other errors
			e.pool.Destroy(conn)
			return resp, werr
		}
		// if we got to this point, we have an intact response
		break
	}
	// finally, (FINALLY) we can clean up the connection
	e.pool.ReturnWithError(conn, werr)
	// parse and return
	resp = parse(buf[:n])
	return resp, nil
}

// writeOnly does a write and reads the response code for OK/NOK
func (e *Ensemble) writeOnly(msg string) error {
	resp, err := e.writeReadRaw(msg)
	if err != nil {
		return err
	}
	if !resp.isOK() && resp.code != 0 { // zero code occurs when we get no response, some generations of EPAQ do not ACK cmds that only write
		return fmt.Errorf("unexpected response, expected %s got %s", string([]byte{resp.code}), resp.string())
	}
	return nil
}

// writeRead does a write and reads an ASCII response
func (e *Ensemble) writeRead(msg string) (string, error) {
	resp, err := e.writeReadRaw(msg)
	if !resp.isOK() && resp.code != 0 {
		return "", fmt.Errorf("unexpected response, expected %s got %s", string([]byte{resp.code}), resp.string())
	}
	return resp.string(), err
}

func (e *Ensemble) gCodeWriteOnly(msg string, more ...string) error {
	str := strings.Join(append([]string{msg}, more...), " ")
	return e.writeOnly(str)
}

// Enable commands the controller to enable an axis
func (e *Ensemble) Enable(axis string) error {
	return e.gCodeWriteOnly("ENABLE", axis)
}

// Disable commands the controller to disable an axis
func (e *Ensemble) Disable(axis string) error {
	return e.gCodeWriteOnly("DISABLE", axis)
}

// GetStatus returns the (unpacked) status bitfield for the given axis
func (e *Ensemble) GetStatus(axis string) (Status, error) {
	// test with an HCIT ensemble
	// AXISSTATUS(X) => %-1791485433
	// Q.E.D the integer is written in ASCII format,
	// so why did the old GetEnabled work by just popping the last byte?
	// do not ponder these things...
	resp, err := e.writeRead(fmt.Sprintf("AXISSTATUS(%s)", axis))
	if err != nil {
		return 0, err
	}
	i64, err := strconv.ParseInt(resp, 10, 32)
	if err != nil {
		return 0, err
	}
	return Status(i64), nil
}

// GetEnabled gets if the given axis is enabled or not
func (e *Ensemble) GetEnabled(axis string) (bool, error) {
	// get the status, it is a 32-bit int, which is really a bitfield
	status, err := e.GetStatus(axis)
	return status.Enabled(), err
}

// GetInPosition returns True if the axis is within the position target, "on target"
func (e *Ensemble) GetInPosition(axis string) (bool, error) {
	status, err := e.GetStatus(axis)
	return status.InPosition(), err
}

// SetSynchronous commands the controller to use synchronous motion mode
//
// The axis argument is ignored (Aerotech controllers are synchronous or not at
// the entire controller scope)
func (e *Ensemble) SetSynchronous(axis string, useSync bool) error {
	var cmd string
	if useSync {
		cmd = "WAIT MODE INPOS"
	} else {
		cmd = "WAIT MODE NOWAIT"
	}
	err := e.writeOnly(cmd)
	tmp := !useSync
	if err == nil {
		e.asyncMode = &tmp // sync is opposite to async
	}
	return err

}

// GetSynchronous queries whether the controller is in synchronous mode or not.
//
// The axis argument is ignored (Aerotech controllers are synchronous or not at
// the entire controller scope)
func (e *Ensemble) GetSynchronous(axis string) (bool, error) {
	if e.asyncMode == nil {
		return false, errors.New("synchronicity not known for axis, use SetSynchronous to make it known")
	}
	return *e.asyncMode, nil
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
	str := fmt.Sprintf("PFBK %s", axis)
	resp, err := e.writeRead(str)
	if err != nil {
		return 0, err
	}
	// at this point, we are in OK land; our response is just a string of a float
	return strconv.ParseFloat(resp, 64)
}

// SetVelocity sets the velocity of an axis in mm/s
func (e *Ensemble) SetVelocity(axis string, vel float64) error {
	str := fmt.Sprintf("MOVEINC %s 0 %sF %.9f", axis, axis, vel)
	err := e.gCodeWriteOnly(str)
	if err == nil {
		e.velocities[axis] = vel
	}
	return err
}

// GetVelocity gets the velocity of an axis in mm/s
func (e *Ensemble) GetVelocity(axis string) (float64, error) {
	if vel, ok := e.velocities[axis]; ok {
		return vel, nil
	}
	return 0, errors.New("velocity not known for axis, use SetVelocity to make it known")
}

// Raw implements ascii.Rawer
func (e *Ensemble) Raw(s string) (string, error) {
	return e.writeRead(s)
}
