// Package aerotech provides wrappers around Aerotech ensemble motion controllers and enables more pleasant HTTP control
package aerotech

import (
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.jpl.nasa.gov/bdube/golab/util"

	"github.jpl.nasa.gov/bdube/golab/comm"
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

	// Terminator is the request terminator used
	Terminator = '\n'
)

type response struct {
	code byte
	body []byte
}

func (r response) isOK() bool {
	return r.code == OKCode
}

func (r response) string() string {
	return string(r.body)
}

func parse(raw []byte) response {
	var r response
	var v byte
	// scan for the ok/nok code.  Assume the last one belongs to us, if there
	// are multiple (e.g. unread responses)
	// it's ok to return something invalid if there was a "read" that was not
	// flushed, this should be considered unrecoverable.
	for v = raw[0]; v == OKCode || v == BadReqCode; {
		raw = raw[1:]
	}
	r.code = v
	// strip any terminators
	for raw[len(raw)-1] == Terminator {
		raw = raw[:len(raw)-1]
	}
	r.body = raw
	return r
}

// ErrBadResponse is generated when a bad response comes from the controller
type ErrBadResponse struct {
	resp string
}

func (e ErrBadResponse) Error() string {
	return fmt.Sprintf("bad response, OK returns %%, got %s", e.resp)
}

// Ensemble represents an Ensemble motion controller
type Ensemble struct {
	pool *comm.Pool

	timeout time.Duration

	// velocities holds the velocities of the axes; the controller does not allow this to be queried, so we store it here
	velocities map[string]float64
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
		wrap, err = comm.NewTimeout(wrap, e.timeout)
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
				wrap, err = comm.NewTimeout(wrap, e.timeout)
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
	if !resp.isOK() {
		return fmt.Errorf("unexpected response, expected %s got %s", string([]byte{resp.code}), resp.string())
	}
	return nil
}

// writeRead does a write and reads an ASCII response
func (e *Ensemble) writeRead(msg string) (string, error) {
	resp, err := e.writeReadRaw(msg)
	if !resp.isOK() {
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

// GetEnabled gets if the given axis is enabled or not
func (e *Ensemble) GetEnabled(axis string) (bool, error) {
	// get the status, it is a 32-bit int, which is really a bitfield
	str := fmt.Sprintf("AXISSTATUS(%s)", axis)
	resp, err := e.writeRead(str)
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
	str := fmt.Sprintf("PFBK %s", axis)
	resp, err := e.writeRead(str)
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
