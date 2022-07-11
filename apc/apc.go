// Package apc provides utilities for communicating with APC Smart-UPS battery backups
package apc

import (
	"time"

	"github.com/nasa-jpl/golaborate/comm"
	"github.com/tarm/serial"
)

// makeSerConf makes a new serial.Config with correct parity, baud, etc, set.
func makeSerConf(addr string) *serial.Config {
	return &serial.Config{
		Name:        addr,
		Baud:        2400,
		Size:        8,
		Parity:      serial.ParityNone,
		StopBits:    serial.Stop1,
		ReadTimeout: 1 * time.Second}
}

// UPS represents an uninterruptible power supply
type UPS struct {
	*comm.RemoteDevice
}

// NewUPS makes a new UPS instance
func NewUPS(addr string, serial bool) *UPS {
	rd := comm.NewRemoteDevice(addr, serial, nil, makeSerConf(addr))
	rd.Timeout = 3 * time.Second
	return &UPS{RemoteDevice: &rd}
}

// Temperature is the internal temperature of the battery backup
func (u *UPS) Temperature() (float64, error) {
	return 0, nil
}

// SelfTest triggers disconnect of mains.  Returns true if the test executed
func (u *UPS) SelfTest() (bool, error) {
	return true, nil
}

// Runtime returns how long the UPS can operate on the current supply
func (u *UPS) Runtime() (time.Duration, error) {
	// cmd = 'j'
	return time.Duration(0), nil
}

// Status returns the status bitfield of the UPS
func (u *UPS) Status() (Status, error) {
	// cmd = 'Q'
	return Status{}, nil
}

// Status is a decoded version of the UPS' status bitfield
type Status struct {
	// ReplaceBattery indicates the battery is spent if true
	ReplaceBattery bool `json:"replaceBattery"`

	// LowBattery indicates the battery will run out soon
	LowBattery bool `json:"lowBattery"`

	// Overloaded indicates the battery backup is subject to excess load
	Overloaded bool `json:"overloaded"`

	// OnBattery indicates that the mains connection is disrupted
	OnBattery bool `json:"onBattery"`

	// OnMains indicates the system is on mains power
	OnMains bool `json:"onMains"`

	// SmartBoost indicates APC's "SmartBoost" mode is engaged
	SmartBoost bool `json:"smartBoost"`

	// SmartTrim indicates APC's "SmartTrim" is engaged
	SmartTrim bool `json:"smartTrim"`

	// RunTimeCalibrating indicates the remaining operation time on battery is currently beign calibrated
	RunTimeCalibrating bool `json:"runtimeCalibrating"`
}
