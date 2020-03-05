// Package pi provides a Go interface to PI motion control systems
package pi

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/tarm/serial"
	"github.jpl.nasa.gov/HCIT/go-hcit/comm"
)

// file gsc2 contains a generichttp/motion compliant implementation of this
// based on PI's GSC2 communication language

// makeSerConf makes a new serial.Config with correct parity, baud, etc, set.
func makeSerConf(addr string) *serial.Config {
	return &serial.Config{
		Name:        addr,
		Baud:        115200,
		Size:        8,
		Parity:      serial.ParityNone,
		StopBits:    serial.Stop1,
		ReadTimeout: 10 * time.Minute}
}

// Controller maps to any PI controller, e.g. E-509, E-727, C-884
type Controller struct {
	*comm.RemoteDevice
}

// NewController returns a fully configured new controller
func NewController(addr string, serial bool) *Controller {
	// \r terminators
	terms := comm.Terminators{Rx: '\r', Tx: '\r'}
	rd := comm.NewRemoteDevice(addr, serial, &terms, makeSerConf(addr))
	rd.Timeout = 10 * time.Minute
	return &Controller{RemoteDevice: &rd}
}

func (c *Controller) writeOnlyBus(msg string) error {
	err := c.Open()
	if err != nil {
		return err
	}
	c.Lock()
	defer func() {
		c.Unlock()
		c.CloseEventually()
	}()
	err = c.RemoteDevice.Send([]byte(msg))
	if err != nil {
		return err
	}
	return nil
}

// copied from aerotech/gCodeWriteOnly, L108, at commit
// 5d7de8ced55aa818fd206987016c775203ef7b53
func (c *Controller) gCodeWriteOnly(msg string, more ...string) error {
	str := strings.Join(append([]string{msg}, more...), " ")
	return c.writeOnlyBus(str)
}

// MoveAbs commands the controller to move an axis to an absolute position
func (c *Controller) MoveAbs(axis string, pos float64) error {
	posS := strconv.FormatFloat(pos, 'G', -1, 64)
	return c.gCodeWriteOnly("MOV", axis, posS)
}

// MoveRel commands the controller to move an axis by a delta
func (c *Controller) MoveRel(axis string, delta float64) error {
	posS := strconv.FormatFloat(delta, 'G', -1, 64)
	return c.gCodeWriteOnly("MVR", axis, posS)
}

// GetPos returns the current position of an axis
func (c *Controller) GetPos(axis string) (float64, error) {
	// "POS? A" -> "A=+0080.4106"
	str := fmt.Sprintf("POS? %s", axis)
	resp, err := c.RemoteDevice.SendRecv([]byte(str))
	if err != nil {
		return 0, err
	}
	str = string(resp)
	parts := strings.Split(str, "=")
	// could panic here, assume the response is always intact,
	// meaning parts is of length 2
	return strconv.ParseFloat(parts[1], 64)
}

// Enable causes the controller to enable motion on a given axis
func (c *Controller) Enable(axis string) error {
	return c.gCodeWriteOnly("ONL", axis, "1")
}

// Disable causes the controller to disable motion on a given axis
func (c *Controller) Disable(axis string) error {
	return c.gCodeWriteOnly("ONL", axis, "0")
}

// Home causes the controller to move an axis to its home position
func (c *Controller) Home(axis string) error {
	return c.gCodeWriteOnly("GOH", axis)
}
