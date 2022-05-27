// Package pi provides a Go interface to PI motion control systems
package pi

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.jpl.nasa.gov/bdube/golab/comm"
)

/* GCS 2 primer
commands are three letters, like POS? or MOV
a command is followed by arguments.  Arguments are usually addressee-value pairs
like MOV 1 123.456 moves axis 1 to position 123.456

Queries are suffixed by ?

One command per line.

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

// file gsc2 contains a generichttp/motion compliant implementation around GCS2

// ControllerNetwork is a network of daisy chained controllers
type ControllerNetwork struct {
	pool        *comm.Pool
	Controllers map[int]PIController
}

// NewNetwork creates a controller network with a shared pool
func NewNetwork(addr string, serial bool) *ControllerNetwork {
	maker := comm.BackingOffTCPConnMaker(addr, 3*time.Second)
	pool := comm.NewPool(1, 30*time.Second, maker)
	return &ControllerNetwork{pool: pool, Controllers: map[int]PIController{}}
}

// Add adds a controller to the network and returns it
func (n *ControllerNetwork) Add(index int, handshaking, mock bool) PIController {
	var c PIController
	if !mock {
		c = NewController(n.pool, index, handshaking)
	} else {
		c = NewControllerMock(n.pool, index, handshaking)
	}
	n.Controllers[index] = c
	return c
}

// PIController is a superset of several generichttp interfaces
type PIController interface {
	Enabler
	Mover
	Speeder
	InPositionQueryer
	RawCommunicator
}

// Controller maps to any PI controller, e.g. E-509, E-727, C-884
type Controller struct {
	index int

	pool *comm.Pool

	// Timeout controls how long to wait for.
	Timeout time.Duration

	// Handshaking controls if commands check for errors.  Higher throughput can
	// be achieved without error checking in exchange for reduced safety
	Handshaking bool

	// DV is the maximum allowed voltage delta between commands
	DV *float64
}

// NewController returns a new motion controller
// addr is the location to send to, e.g. 192.168.100.2106.
//
// index is the controller index in the daisy chain.  In a single controller
// network, index=1.
//
// handshaking=true will check for errors after all commnads.  False does no error
// checking.
func NewController(pool *comm.Pool, index int, handshaking bool) *Controller {
	return &Controller{
		index:       index,
		pool:        pool,
		Handshaking: handshaking,
		Timeout:     30 * time.Second,
	}
}

// write writes command(s) to the controller.  The controller index
// is automatically prepended.  Commands with a ? in them will be rejected,
// as they are queries.
func (c *Controller) write(msgs ...string) error {
	for i := range msgs {
		msg := msgs[i]
		if strings.Contains(msg, "?") && !strings.Contains(msg, "WAC") {
			return errors.New("pi/gcs2: command contains a query in write-only operation")
		}
	}
	conn, err := c.pool.Get()
	if err != nil {
		return err
	}
	defer func() { c.pool.ReturnWithError(conn, err) }()
	var wrap io.ReadWriter
	wrap, err = comm.NewTimeout(conn, c.Timeout)
	if err != nil {
		return err
	}
	wrap = comm.NewTerminator(wrap, '\n', '\n')

	for i := range msgs {
		msg := msgs[i]
		msg = strconv.Itoa(c.index) + " " + msg
		_, err = io.WriteString(wrap, msg)
		if err != nil {
			return err
		}
	}
	if c.Handshaking {
		msg := strconv.Itoa(c.index) + " ERR?"
		_, err = io.WriteString(wrap, msg)
		// error response will look like 0 1 nnnn which is six bytes, ten is enough
		buf := make([]byte, 10)
		n, err := wrap.Read(buf)
		if err != nil {
			return err
		}
		pieces := bytes.Split(buf[:n], []byte{' '})
		eCode := string(pieces[len(pieces)-1])
		errCode, err := strconv.Atoi(eCode)
		if err != nil {
			return err
		}
		return GCS2Err(errCode)
	}
	return nil
}

// write sends a request for data to the controller.  The controller index
// is automatically prepended.  Commands with a ? in them will be rejected,
// as they are queries.  The response is returned, after stripping the prefix
// and suffix (~= "0 1" and \n)
func (c *Controller) query(msg string) ([]byte, error) {
	// setup
	if !strings.Contains(msg, "?") {
		return nil, errors.New("query lacks a question mark")
	}
	conn, err := c.pool.Get()
	if err != nil {
		return nil, err
	}
	defer func() { c.pool.ReturnWithError(conn, err) }()
	var wrap io.ReadWriter
	wrap, err = comm.NewTimeout(conn, c.Timeout)
	if err != nil {
		return nil, err
	}
	wrap = comm.NewTerminator(wrap, '\n', '\n')

	// prepend controller ID and send query
	msg = strconv.Itoa(c.index) + " " + msg
	_, err = io.WriteString(wrap, msg)
	if err != nil {
		return nil, err
	}
	buf := make([]byte, tcpFrameSize)
	n, err := wrap.Read(buf)
	if err != nil {
		return nil, err
	}
	pieces := bytes.SplitN(buf[:n], []byte{' '}, 3)
	fromAddr, err := strconv.Atoi(string(pieces[1]))
	if err != nil {
		return nil, errors.New("pi/gcs2: could not parse controller ID from response")
	}
	if fromAddr != c.index {
		return nil, errors.New("pi/gcs2: response received was not from the expected controller")
	}
	return pieces[2], nil
}

func (c *Controller) readBool(cmd, axis string) (bool, error) {
	str := strings.Join([]string{cmd, axis}, " ")
	resp, err := c.query(str)
	if err != nil {
		return false, err
	}
	resp = stripAxis(axis, resp)
	return resp[0] == '1', nil
}

func (c *Controller) readFloat(cmd, axis string) (float64, error) {
	str := strings.Join([]string{cmd, axis}, " ")
	resp, err := c.query(str)
	if err != nil {
		return 0, err
	}
	resp = stripAxis(axis, resp)
	return strconv.ParseFloat(string(resp), 64)
}

// MoveAbs commands the controller to move an axis to an absolute position
func (c *Controller) MoveAbs(axis string, pos float64) error {
	// want to wait this long before reading position to wait for convergence
	start := time.Now()
	msg := fmt.Sprintf("MOV %s %.9f", axis, pos)
	err := c.write(msg)
	end := time.Now()
	dur := end.Sub(start)
	if err != nil {
		return err
	}

	const maxChecks = 10000
	for checks := 0; checks < maxChecks; checks++ {
		time.Sleep(dur) // avoid thrashing the controller
		b, err := c.readBool("ONT?", axis)
		if err != nil {
			return err
		}
		if !b {
			// not on target
			if checks == maxChecks-1 {
				lastPos, err := c.GetPos(axis)
				if err != nil {
					return err
				}
				return fmt.Errorf("pi/gsc2: stage position did not converge after %d checks, last value %f for target %f", maxChecks, lastPos, pos)
			}
		}
		break
	}
	// position converged
	return nil
}

// MoveRel commands the controller to move an axis by a delta
func (c *Controller) MoveRel(axis string, delta float64) error {
	// want to wait this long before reading position to wait for convergence
	start := time.Now()
	msg := fmt.Sprintf("MVR %s %.9f", axis, delta)
	err := c.write(msg)
	end := time.Now()
	dur := end.Sub(start)
	if err != nil {
		return err
	}

	const maxChecks = 10000
	for checks := 0; checks < maxChecks; checks++ {
		time.Sleep(dur) // avoid threashing the controller
		b, err := c.readBool("ONT?", axis)
		if err != nil {
			return err
		}
		if !b {
			// not on target
			if checks == maxChecks-1 {
				lastPos, err := c.GetPos(axis)
				if err != nil {
					return err
				}
				return fmt.Errorf("pi/gsc2: stage position did not converge after %d checks, last value %f during relative move of %f", maxChecks, lastPos, delta)
			}
		}
		break
	}
	// position converged
	return nil
}

// for sync move macro, it is as simple as
// MAC BEG movwai
// MOV $1 $2
// WAC ONT? $1 = 1
// MAC END
// ... but this does not render the "reply when move done" behavior that may be
// desired :\
// only solution is polling

// GetPos returns the current position of an axis
func (c *Controller) GetPos(axis string) (float64, error) {
	return c.readFloat("POS?", axis)
}

// GetInPosition returns True if axis is in position
func (c *Controller) GetInPosition(axis string) (bool, error) {
	return c.readBool("ONT?", axis)
}

// SetVelocity returns the velocity of an axis
func (c *Controller) SetVelocity(axis string, v float64) error {
	return c.write(fmt.Sprintf("VEL %s %.9f", axis, v))
}

// GetVelocity returns the velocity of an axis
func (c *Controller) GetVelocity(axis string) (float64, error) {
	return c.readFloat("VEL?", axis)
}

// Enable causes the controller to enable motion on a given axis
func (c *Controller) Enable(axis string) error {
	return c.write(fmt.Sprintf("SVO %s 1", axis))
}

// Disable causes the controller to disable motion on a given axis
func (c *Controller) Disable(axis string) error {
	return c.write(fmt.Sprintf("SVO %s 0", axis))
}

// GetEnabled returns True if the given axis is enabled
func (c *Controller) GetEnabled(axis string) (bool, error) {
	return c.readBool("SVO?", axis)
}

// Home causes the controller to move an axis to its home position
func (c *Controller) Home(axis string) error {
	return c.write(fmt.Sprintf("FRF %s", axis))
}

// SetVoltage sets the voltage on an axis
func (c *Controller) SetVoltage(axis string, volts float64) error {
	msg := fmt.Sprintf("SVA %s %.9f", axis, volts)
	return c.write(msg)
}

// GetVoltage returns the voltage on an axis
func (c *Controller) GetVoltage(axis string) (float64, error) {
	return c.readFloat("SVA?", axis)
}

// Raw implements generichttp/ascii.RawCommunicator
func (c *Controller) Raw(s string) (string, error) {
	if strings.Contains(s, "?") {
		resp, err := c.query(s)
		return string(resp), err
	}
	err := c.write(s)
	return "", err
}
