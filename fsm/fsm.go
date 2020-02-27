// Package fsm provides a system for operating a fast steering mirror control system at high speed
package fsm

import (
	"sync"

	"github.jpl.nasa.gov/HCIT/go-hcit/mccdaq"
)

// ControlLoop is a struct which operates a control loop
type ControlLoop struct {
	DAC *mccdaq.DAC

	sync.Mutex
}

// Update turns the crank on the control loop.  An internal lock is acquired
// and the phase (refresh rate) of the loop, i.e. call frequency of update,
// is blocked by the phase lock of the control loop.
// Any errors writing to the DAC are bubbled up
func (c *ControlLoop) Update(target float64) error {
	c.Lock()
	defer c.Unlock()
	return c.DAC.Write(1, target)
}
