package pi

import (
	"errors"
	"math/rand"
	"sync"
	"time"

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

func NewControllerMock(pool *comm.Pool, index int, handshaking bool) PIController {
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
				nextPos = pos + randN1to1()
				converged = true
			}
			if (lastPos > pos) && (nextPos < pos) {
				nextPos = pos + randN1to1()
				converged = true
			}
			c.Lock()
			c.pos[axis] = nextPos
			if converged {
				c.moving[axis] = false
			}
			c.Unlock()
		case <-time.After(time.Hour):
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
	return "", NotImplemented
}
