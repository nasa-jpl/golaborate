// Package aerotech provides wrappers around Aerotech ensemble motion controllers and enables more pleasant HTTP control
package aerotech

import (
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

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
	OKCode = byte(37) // %

	// BadReqCode is the first byte in the controller's response when the message
	// was not understood
	BadReqCode = byte(33) // !

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
	if len(raw) < 2 {
		return response{}
	}
	var r response
	var v byte
	// scan for the ok/nok code.  Assume the last one belongs to us, if there
	// are multiple (e.g. unread responses)
	// it's ok to return something invalid if there was a "read" that was not
	// flushed, this should be considered unrecoverable.
	for {
		tmp := raw[0]
		if tmp == OKCode || tmp == BadReqCode {
			raw = raw[1:]
			v = tmp
		} else {
			break
		}
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

type Status struct {
	Enabled            bool
	Homed              bool
	InPosition         bool
	MoveActive         bool
	AccelPhase         bool
	DecelPhase         bool
	PositionCapture    bool
	CurrentClamp       bool
	BrakeOutput        bool
	MotionIsCw         bool
	MasterSlaveControl bool
	CalActive          bool
	CalEnabled         bool
	JoystickControl    bool
	Homing             bool
	MasterSuppress     bool
	GantryActive       bool
	GantryMaster       bool
	AutofocusActive    bool
	CommandFilterDone  bool
	InPosition2        bool
	ServoControl       bool
	CwEOTLimit         bool
	CcwEOTLimit        bool
	HomeLimit          bool
	MarkerInput        bool
	HallAInput         bool
	HallBInput         bool
	HallCInput         bool
	SineEncoderError   bool
	CosineEncoderError bool
	ESTOPInput         bool
}

func StatusFromBitfield(b int32) Status {
	var s Status
	s.Enabled = (b>>0)&1 == 1
	s.Homed = (b>>1)&1 == 1
	s.InPosition = (b>>2)&1 == 1
	s.MoveActive = (b>>3)&1 == 1
	s.AccelPhase = (b>>4)&1 == 1
	s.DecelPhase = (b>>5)&1 == 1
	s.PositionCapture = (b>>6)&1 == 1
	s.CurrentClamp = (b>>7)&1 == 1
	s.BrakeOutput = (b>>8)&1 == 1
	s.MotionIsCw = (b>>9)&1 == 1
	s.MasterSlaveControl = (b>>10)&1 == 1
	s.CalActive = (b>>11)&1 == 1
	s.CalEnabled = (b>>12)&1 == 1
	s.JoystickControl = (b>>13)&1 == 1
	s.Homing = (b>>14)&1 == 1
	s.MasterSuppress = (b>>15)&1 == 1
	s.GantryActive = (b>>16)&1 == 1
	s.GantryMaster = (b>>17)&1 == 1
	s.AutofocusActive = (b>>18)&1 == 1
	s.CommandFilterDone = (b>>19)&1 == 1
	s.InPosition2 = (b>>20)&1 == 1
	s.ServoControl = (b>>21)&1 == 1
	s.CwEOTLimit = (b>>22)&1 == 1
	s.CcwEOTLimit = (b>>23)&1 == 1
	s.HomeLimit = (b>>24)&1 == 1
	s.MarkerInput = (b>>25)&1 == 1
	s.HallAInput = (b>>26)&1 == 1
	s.HallBInput = (b>>27)&1 == 1
	s.HallCInput = (b>>28)&1 == 1
	s.SineEncoderError = (b>>29)&1 == 1
	s.CosineEncoderError = (b>>30)&1 == 1
	s.ESTOPInput = (b>>31)&1 == 1
	return s
}

func (s Status) Bit(label string) bool {
	label = strings.ToLower(label)
	switch label {
	case "enabled":
		return s.Enabled
	case "homed":
		return s.Homed
	case "inposition":
		return s.InPosition
	case "moveactive":
		return s.MoveActive
	case "accelphase":
		return s.AccelPhase
	case "decelphase":
		return s.DecelPhase
	case "positioncapture":
		return s.PositionCapture
	case "currentclamp":
		return s.CurrentClamp
	case "brakeoutput":
		return s.BrakeOutput
	case "motioniscw":
		return s.MotionIsCw
	case "masterslavecontrol":
		return s.MasterSlaveControl
	case "calactive":
		return s.CalActive
	case "calenabled":
		return s.CalEnabled
	case "joystickcontrol":
		return s.JoystickControl
	case "homing":
		return s.Homing
	case "mastersuppress":
		return s.MasterSuppress
	case "gantryactive":
		return s.GantryActive
	case "gantrymaster":
		return s.GantryMaster
	case "autofocusactive":
		return s.AutofocusActive
	case "commandfilterdone":
		return s.CommandFilterDone
	case "inposition2":
		return s.InPosition2
	case "servocontrol":
		return s.ServoControl
	case "cweotlimit":
		return s.CwEOTLimit
	case "ccweotlimit":
		return s.CcwEOTLimit
	case "homelimit":
		return s.HomeLimit
	case "markerinput":
		return s.MarkerInput
	case "hallainput":
		return s.HallAInput
	case "hallbinput":
		return s.HallBInput
	case "hallcinput":
		return s.HallCInput
	case "sineencodererror":
		return s.SineEncoderError
	case "cosineencodererror":
		return s.CosineEncoderError
	case "estopinput":
		return s.ESTOPInput
	default:
		panic("aerotech: bit queried not present in Status bitfield")
	}
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
		return Status{}, err
	}
	i64, err := strconv.ParseInt(resp, 10, 32)
	if err != nil {
		return Status{}, err
	}
	b := int32(i64)
	return StatusFromBitfield(b), nil
}

// GetEnabled gets if the given axis is enabled or not
func (e *Ensemble) GetEnabled(axis string) (bool, error) {
	// get the status, it is a 32-bit int, which is really a bitfield
	status, err := e.GetStatus(axis)
	return status.Enabled, err
}

func (e *Ensemble) GetInPosition(axis string) (bool, error) {
	status, err := e.GetStatus(axis)
	return status.InPosition, err
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
