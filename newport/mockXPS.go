package newport

import (
	"errors"
	"math"
	"math/rand"
	"sync"
	"time"
)

const (
	xpsServoPeriod      = 250 * time.Microsecond // 4kHz servo rate on XPS controllers
	xpsServerPeriodSec  = 250e-6                 // Period is for ticker, PeriodSec is for math
	xpsPositioningError = 1e-7                   // up to 100 nm on lengths, 100 nrad on angles
	floatCmpTol         = 1e-12
)

var NotImplemented = errors.New("not implemented")

type MockController struct {
	sync.Mutex
	sem     chan struct{}
	enabled map[string]bool
	moving  map[string]bool
	homed   map[string]bool
	stop    map[string]bool
	pos     map[string]float64
	vel     map[string]float64
}

func randN1to1() float64 {
	return rand.Float64()*2 - 1 // [0,1] => [0,2] => [-1,1]
}

func NewControllerMock(addr string) *MockController {
	return &MockController{
		sem:     make(chan struct{}, xpsConcurrencyLimit),
		enabled: make(map[string]bool),
		moving:  make(map[string]bool),
		homed:   make(map[string]bool),
		stop:    make(map[string]bool),
		pos:     make(map[string]float64),
		vel:     make(map[string]float64)}
}

func (c *MockController) semAcq() {
	c.sem <- struct{}{}
}

func (c *MockController) semRelease() {
	<-c.sem
}

func (c *MockController) Disable(axis string) error {
	c.Lock()
	defer c.Unlock()
	c.semAcq()
	defer c.semRelease()
	if c.moving[axis] {
		return XPSErr(-22)
	} // moving
	c.enabled[axis] = false
	return nil
}

func (c *MockController) Enable(axis string) error {
	c.Lock()
	defer c.Unlock()
	c.semAcq()
	defer c.semRelease()
	c.enabled[axis] = true
	return nil
}

func (c *MockController) GetEnabled(axis string) (bool, error) {
	c.Lock()
	defer c.Unlock()
	c.semAcq()
	defer c.semRelease()
	return c.enabled[axis], nil
}

func (c *MockController) Homed(axis string) (bool, error) {
	c.Lock()
	defer c.Unlock()
	c.semAcq()
	defer c.semRelease()
	return c.homed[axis], nil
}

// func (c *MockController) GetInPosition(axis string) (bool, error) {
// 	c.Lock()
// 	defer c.Unlock()
// 	return !c.moving[axis], nil
// }

func (c *MockController) GetPos(axis string) (float64, error) {
	c.Lock()
	defer c.Unlock()
	c.semAcq()
	defer c.semRelease()
	return c.pos[axis], nil
}

func (c *MockController) GetVelocity(axis string) (float64, error) {
	c.Lock()
	defer c.Unlock()
	c.semAcq()
	defer c.semRelease()
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
	c.semAcq()
	defer c.semRelease()
	if c.moving[axis] {
		return XPSErr(-22)
	}
	c.vel[axis] = v
	return nil
}

func (c *MockController) Home(axis string) error {
	c.Lock()
	defer c.Unlock()
	c.semAcq()
	defer c.semRelease()
	if !c.enabled[axis] {
		return XPSErr(-50)
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
	c.semAcq()
	defer c.semRelease()
	c.pos[axis] = pos
}

// MoveAbs = public interface; moveTo = asynchronous internal interface
func (c *MockController) moveTo(axis string, pos float64) {
	c.Lock()
	c.stop[axis] = false
	c.Unlock()
	currPos, _ := c.GetPos(axis)
	if approxEqual(currPos, pos, floatCmpTol) {
		return
	}
	tick := time.NewTicker(xpsServoPeriod)
	defer tick.Stop()
	v, _ := c.GetVelocity(axis)
	// posErr is negative when we need to increase our distance
	posErr := pos - currPos
	step := v * xpsServerPeriodSec
	if math.Signbit(posErr) {
		step = -step
	}
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
				nextPos = pos + randN1to1()*xpsPositioningError
				converged = true
			}
			if (lastPos > pos) && (nextPos < pos) {
				nextPos = pos + randN1to1()*xpsPositioningError
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

		// Abort is Stop is called
		c.Lock()
		if c.stop[axis] {
			c.moving[axis] = false
			c.stop[axis] = false
			c.Unlock()
			return
		} else {
			c.Unlock()
		}
	}
}

func (c *MockController) MoveAbs(axis string, pos float64) error {
	c.Lock()
	c.semAcq()
	defer c.semRelease()
	if !c.enabled[axis] {
		c.Unlock()
		return XPSErr(-50)
	}
	if !c.homed[axis] {
		c.Unlock()
		return XPSErr(-109)
	}
	if c.moving[axis] {
		c.Unlock()
		return XPSErr(-22)
	}
	c.moving[axis] = true
	c.Unlock()
	// this might be buggy?  The goroutine that called MoveAbs is blocking,
	// but others are not prevented from doing anything (lock released) while
	// the move is happening.  There is an error on already moving, so I think
	// not buggy.
	c.moveTo(axis, pos)
	return nil
}

func (c *MockController) MoveRel(axis string, dPos float64) error {
	c.Lock()
	c.semAcq()
	defer c.semRelease()
	if !c.enabled[axis] {
		c.Unlock()
		return XPSErr(-50)
	}
	if !c.homed[axis] {
		c.Unlock()
		return XPSErr(-109)
	}
	if c.moving[axis] {
		c.Unlock()
		return XPSErr(-22)
	}
	c.moving[axis] = true
	c.Unlock()
	pos := c.pos[axis] + dPos
	c.moveTo(axis, pos)
	return nil
}

func (c *MockController) Stop(axis string) error {
	c.Lock()
	c.semAcq()
	defer c.semRelease()
	if c.moving[axis] {
		c.stop[axis] = true
	}
	c.Unlock()
	return nil
}
func (c *MockController) Raw(s string) (string, error) {
	return "", NotImplemented
}

func approxEqual(a, b, atol float64) bool {
	d := b - a
	return math.Abs(d) < atol
}
