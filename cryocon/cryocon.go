// Package cryocon provides utilities for working with temperature sensors
// Supports model 12, 14, 18i and maybe more
package cryocon

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/nasa-jpl/golaborate/comm"
	"github.com/nasa-jpl/golaborate/scpi"
	"github.com/nasa-jpl/golaborate/temperature"
)

// parseTempToC converts a response string looking like "250.123124;K" into
// the same temperature in C, or errors on malformed input.
// Unpopulated channel responses ("--", "..") return NaN
func parseTempToC(resp string) (float64, error) {
	// -- on an unused channel, or ".."
	if strings.Contains(resp, "--") || strings.Contains(resp, "..") || strings.Contains(resp, "       ") {
		return math.NaN(), nil
	}
	pieces := strings.Split(resp, ";")
	unit := pieces[1]
	T, err := strconv.ParseFloat(pieces[0], 64)
	if err != nil {
		return 0, err
	}

	switch unit {
	case "K":
		t := temperature.Kelvin(T)
		return float64(temperature.K2C(t)), nil
	case "C":
		return T, nil
	case "F":
		t := temperature.Fahrenheit(T)
		return float64(temperature.F2C(t)), nil
	}
	return 0, fmt.Errorf("err, do not know how to convert unit %s to Celcius", unit)
}

// TemperatureMonitor models a Model 12 or Model 14 temperature monitor
type TemperatureMonitor struct {
	s scpi.SCPI
}

// NewTemperatureMonitor creates a new temperature monitor instance
func NewTemperatureMonitor(addr string) *TemperatureMonitor {
	maker := comm.BackingOffTCPConnMaker(addr, time.Second)
	pool := comm.NewPool(1, 10*time.Second, maker)
	return &TemperatureMonitor{scpi.SCPI{Pool: pool}}
}

// Identification returns the identifying information from the monitor.
// it looks something like:
//
// Cryocon Model 12/14 Rev <firmware rev code><hardware rev code>
func (tm *TemperatureMonitor) Identification() (string, error) {
	return tm.s.ReadString("*IDN?")
}

// ReadChannelLetter reads the temperature on a given channel in C, where the
// channel is something like "A"
func (tm *TemperatureMonitor) ReadChannelLetter(ch string) (float64, error) {
	s, err := tm.s.ReadString("INP", ch, ":TEMP?;UNIT?")
	if err != nil {
		return 0, err
	}
	return parseTempToC(s)
}

// ReadAllChannels reads all of the temperature channels of the sensor in C.
// any unpopulated channels are NaN.
func (tm *TemperatureMonitor) ReadAllChannels() ([]float64, error) {
	// the behavior of the sensor for too large a channel index
	// depends on firmware, JPL has FW 1~3, so we need a different solution
	// here, we look for the model number in the identification, and if not found
	// use a max of 8
	id, err := tm.Identification()
	if err != nil {
		return []float64{}, err
	}
	maxCh := 8
	if strings.Contains(id, "Model 18") {
		maxCh = 8
	} else if strings.Contains(id, "Model 12") {
		maxCh = 2
	} else if strings.Contains(id, "Model 14") {
		maxCh = 4
	}

	out := make([]float64, maxCh)
	for ch := 0; ch < maxCh; ch++ {
		resp, err := tm.s.ReadString("INP", strconv.Itoa(ch), ":TEMP?;UNIT?")
		if err != nil {
			return out, err
		}
		if resp == "NAK" {
			break // past the last channel
		}
		t, err := parseTempToC(resp)
		if err != nil {
			return out, err
		}
		out[ch] = t
	}
	return out, err
}

//// ReadResistance indicates the resistance sensed for a given channel in Ohms
//func (tm *TemperatureMonitor) ReadResistance(channel string) (float64, error) {
//	// INP <ch>:SENPr?
//}
