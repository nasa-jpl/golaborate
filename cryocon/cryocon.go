// Package cryocon provides utilities for working with temperature sensors
// Supports model 12, 14, 18i and maybe more
package cryocon

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.jpl.nasa.gov/HCIT/go-hcit/comm"
	"github.jpl.nasa.gov/HCIT/go-hcit/temperature"
)

const (
	sep = ":"
)

// parseTempToC converts a response string looking like "250.123124;K" into
// the same temperature in C, or errors on malformed input.
// Unpopulated channel responses ("--", "..") return NaN
func parseTempToC(resp string) (float64, error) {
	// -- on an unused channel, or ".."
	if strings.Contains(resp, "--") || strings.Contains(resp, "..") {
		return math.NaN(), nil
	}
	pieces := strings.Split(string(resp), ";")
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
	*comm.RemoteDevice
}

// NewTemperatureMonitor creates a new temperature monitor instance
func NewTemperatureMonitor(addr string) *TemperatureMonitor {
	term := comm.Terminators{Rx: '\n', Tx: '\n'}
	rd := comm.NewRemoteDevice(addr, false, &term, nil)
	return &TemperatureMonitor{RemoteDevice: &rd}
}

// Identification returns the identifying information from the monitor.
// it looks something like:
//
// Cryocon Model 12/14 Rev <fimware rev code><hardware rev code>
func (tm *TemperatureMonitor) Identification() (string, error) {
	cmd := []byte("*IDN?")
	err := tm.Open()
	if err != nil {
		return "", err
	}
	defer tm.CloseEventually()
	resp, err := tm.SendRecv(cmd)
	return string(resp), err
}

// SendRecv wraps the underlying RemoteDevice to pick the CR off the end
func (tm *TemperatureMonitor) SendRecv(b []byte) ([]byte, error) {
	resp, err := tm.RemoteDevice.SendRecv(b)
	if err != nil {
		return []byte{}, err
	}
	return resp[:len(resp)-1], nil // slice up to does not include the second index, this chops off one byte
}

// ReadChannelLetter reads the temperature on a given channel in C, where the
// channel is something like "A"
func (tm *TemperatureMonitor) ReadChannelLetter(ch string) (float64, error) {
	cmd := []byte("INP " + ch + ":TEMP?;UNIT?")
	err := tm.Open()
	if err != nil {
		return 0, err
	}
	defer tm.CloseEventually()
	resp, err := tm.SendRecv(cmd)
	if err != nil {
		return 0, err
	}
	return parseTempToC(string(resp))
}

// ReadAllChannels reads all of the temperature channels of the sensor in C.
// any unpopulated channels are NaN.
func (tm *TemperatureMonitor) ReadAllChannels() ([]float64, error) {
	err := tm.Open()
	if err != nil {
		return []float64{}, err
	}
	defer tm.CloseEventually()

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
		cmd := []byte(fmt.Sprintf("INP %d:TEMP?;UNIT?", ch))
		resp, err := tm.SendRecv(cmd)
		if err != nil {
			break
		}
		rs := string(resp)
		if rs == "NAK" {
			break // past the last channel
		}
		t, err := parseTempToC(rs)
		if err != nil {
			break
		}
		out[ch] = t
	}
	return out, err
}
