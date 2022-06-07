package pi

import (
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"sync"
	"time"
	"unicode"

	"github.jpl.nasa.gov/bdube/golab/comm"
)

const (
	piServoPeriod      = 50 * time.Microsecond // 20kHz servo rate on PI controllers circa 2014 or so and later
	piServerPeriodSec  = 50e-6                 // Period is for ticker, PeriodSec is for math
	piPositioningError = 1e-8                  // up to 10 nm on lengths, 10 nrad on angles
)

var NotImplemented = errors.New("not implemented")

type MockController struct {
	sync.Mutex
	enabled map[string]bool
	moving  map[string]bool
	homed   map[string]bool
	pos     map[string]float64
	vel     map[string]float64
}

func randN1to1() float64 {
	return rand.Float64()*2 - 1 // [0,1] => [0,2] => [-1,1]
}

func NewControllerMock(pool *comm.Pool, index int, handshaking bool) *MockController {
	return &MockController{
		enabled: make(map[string]bool),
		moving:  make(map[string]bool),
		homed:   make(map[string]bool),
		pos:     make(map[string]float64),
		vel:     make(map[string]float64)}
}

func (c *MockController) Disable(axis string) error {
	c.Lock()
	defer c.Unlock()
	if c.moving[axis] {
		return GCS2Err(53)
	} // moving
	c.enabled[axis] = false
	return nil
}

func (c *MockController) Enable(axis string) error {
	c.Lock()
	defer c.Unlock()
	c.enabled[axis] = true
	return nil
}

func (c *MockController) GetEnabled(axis string) (bool, error) {
	c.Lock()
	defer c.Unlock()
	return c.enabled[axis], nil
}

func (c *MockController) GetInPosition(axis string) (bool, error) {
	c.Lock()
	defer c.Unlock()
	return !c.moving[axis], nil
}

func (c *MockController) GetPos(axis string) (float64, error) {
	c.Lock()
	defer c.Unlock()
	return c.pos[axis], nil
}

func (c *MockController) GetVelocity(axis string) (float64, error) {
	c.Lock()
	defer c.Unlock()
	v, ok := c.vel[axis]
	if !ok {
		c.vel[axis] = 1
		v = 1
	}
	return v, nil
}

func (c *MockController) SetVelocity(axis string, v float64) error {
	c.Lock()
	defer c.Unlock()
	if c.moving[axis] {
		return GCS2Err(53)
	}
	c.vel[axis] = v
	return nil
}

func (c *MockController) Home(axis string) error {
	c.Lock()
	defer c.Unlock()
	if !c.enabled[axis] {
		return GCS2Err(5)
	}
	// up to 10 seconds to home
	secsToHome := rand.Intn(10)
	time.Sleep(time.Duration(secsToHome) * time.Second)
	c.homed[axis] = true
	return nil
}

func (c *MockController) setPosition(axis string, pos float64) {
	c.Lock()
	defer c.Unlock()
	c.pos[axis] = pos
}

// MoveAbs = public interface; moveTo = asynchronous internal interface
func (c *MockController) moveTo(axis string, pos float64) {
	tick := time.NewTicker(piServoPeriod)
	defer tick.Stop()
	v, _ := c.GetVelocity(axis)
	step := v * piServerPeriodSec
	// there is a better mock here that checks the current time and the wall time
	// when the move should be over.  It's only a few lines of code and higher
	// fidelity than this.  Beauty for another day, this M.F is BIEGE.
	for {
		select {
		case <-tick.C:
			lastPos, _ := c.GetPos(axis)
			nextPos := lastPos + step
			var converged bool

			// overshoot; -> + and -> - direction cases
			if (lastPos < pos) && (nextPos > pos) {
				nextPos = pos + randN1to1()*1e-9
				converged = true
			}
			if (lastPos > pos) && (nextPos < pos) {
				nextPos = pos + randN1to1()*1e-9
				converged = true
			}
			c.Lock()
			c.pos[axis] = nextPos
			if converged {
				c.moving[axis] = false
				c.Unlock()
				return
			} else {
				c.Unlock()
			}
		case <-time.After(24 * time.Hour):
			c.Lock()
			c.moving[axis] = false
			c.Unlock()
			return
		}
	}
}

func (c *MockController) MoveAbs(axis string, pos float64) error {
	c.Lock()
	defer c.Unlock()
	if !c.enabled[axis] {
		return GCS2Err(5)
	}
	if !c.homed[axis] {
		return GCS2Err(5)
	}
	if c.moving[axis] {
		return GCS2Err(53)
	}
	c.moving[axis] = true
	go c.moveTo(axis, pos)
	return nil
}

func (c *MockController) MoveRel(axis string, dPos float64) error {
	c.Lock()
	defer c.Unlock()
	if !c.enabled[axis] {
		return GCS2Err(5)
	}
	if !c.homed[axis] {
		return GCS2Err(5)
	}
	c.moving[axis] = true
	pos := c.pos[axis] + dPos
	go c.moveTo(axis, pos)
	return nil
}

func (c *MockController) Raw(s string) (string, error) {
	// PI GCS2 format: (TLA = Three Letter Acronym)
	// from<sp>to<sp>TLA<sp><?><sp>arg1<sp>arg2

	// err 1 = parameter syntax error
	// err 3 = command length out of bounds
	if len(s) < 4 {
		// minimum length is TLA?
		return "", GCS2Err(3)
	}

	// pi is ascii, safe to assume uniform indexing of string (bytes)
	// and runes
	u := []rune(s)
	// if FROM is specified, it does not matter to our mock
	if unicode.IsDigit(u[0]) {
		s = s[1:]
		u = u[1:]
	}
	if s[0:1] == "" {
		// next character had to be a space
		s = s[1:]
		u = u[1:]
	} else {
		return "", GCS2Err(1)
	}
	// if TO is specified, it does not matter to our mock
	if unicode.IsDigit(u[0]) {
		s = s[1:]
		u = u[1:]
	}
	if s[0:1] == "" {
		// next character had to be a space
		s = s[1:]
		u = u[1:]
	} else {
		return "", GCS2Err(1)
	}
	// now we may have skipped four characters (from, sp, to, sp); is there anything
	// left?
	if len(s) < 4 {
		// minimum length is TLA?
		return "", GCS2Err(3)
	}
	// no way to avoid nested switches.  Pull the reads up
	if s[3] == '?' {
		// read, only one argument (axis)
		if len(s) != 7 {
			// T L A <sp> ? <sp> axis == 7
			return "", GCS2Err(1)
		}
		axis := string(s[6])
		switch s[:3] {
		case "VEL":
			return returnFloatAsString(axis, c.GetVelocity)
		case "POS":
			return returnFloatAsString(axis, c.GetPos)
		case "SVA":
			return returnFloatAsString(axis, c.GetVelocity)
		case "ONT":
			return returnBoolAsString(axis, c.GetInPosition)
		case "SVO":
			return returnBoolAsString(axis, c.GetEnabled)
		default:
			return "", tlaNotUnderstoodInMock(s)
		}
	}
	// all code in the above if returns, do not need an indent
	// what remains is either 1 axis-arg pair or two
	// TODO 2022-06-06 ; decided to only support one axis-argument pair for
	// simplicity until someone has a demonstrable need for two
	if len(s) < 5 {
		// 6 = T L A <sp> axis
		return "", GCS2Err(1)
	}
	axis := string(s[4])
	// take care of argument-less commands
	switch s[:3] {
	// only one case today, maybe more in the future.  Keep a switch.
	case "FRF":
		return "", c.Home(axis)
	}
	// boolean argument commands
	if len(s) < 7 {
		// 7 = 5 + sp + 0 or 1
		return "", GCS2Err(1)
	}
	argb := s[7] == '1' // 1 = true; 0 = false
	switch s[:3] {
	case "SVO":
		if argb {
			return "", c.Enable(axis)
		} else {
			return "", c.Disable(axis)
		}
	}
	if s[:3] == "SPA" || s[:3] == "CCL" || s[:3] == "WPA" {
		return "", nil // assume user knows how to use parameter manipulation
	}
	// everything else is a float command
	argf, err := strconv.ParseFloat(s[7:], 64)
	if err != nil {
		return "", err
	}
	switch s[:3] {
	case "MOV":
		return "", c.MoveAbs(axis, argf)
	case "MVR":
		return "", c.MoveRel(axis, argf)
		// case "SVA": -- SVA not supported in the mock right now
		// return "", c.SetVoltage(axis, argf)
	}
	return "", NotImplemented
}

type tlaNotUnderstoodInMock string

func (t tlaNotUnderstoodInMock) Error() string {
	return fmt.Sprintf("command '%s' was not understood by Golab's PI/GCS2 mock", string(t))
}

func returnFloatAsString(s string, fun func(string) (float64, error)) (string, error) {
	f, err := fun(s)
	return strconv.FormatFloat(f, 'g', -1, 64), err
}

func returnBoolAsString(s string, fun func(string) (bool, error)) (string, error) {
	b, err := fun(s)
	// can clobber input now
	if b {
		s = "1"
	} else {
		s = "0"
	}
	return s, err
}
