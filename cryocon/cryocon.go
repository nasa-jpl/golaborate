// Package cryocon provides utilities for working with temperature sensors
// Supports model 12, 14, 18i and maybe more
package cryocon

import (
	"fmt"
	"strconv"
	"strings"

	"github.jpl.nasa.gov/HCIT/go-hcit/comm"
	"github.jpl.nasa.gov/HCIT/go-hcit/temperature"
)

const (
	sep = ":"
)

// parseTempToC converts a response string looking like "250.123124;K" into
// the same temperature in C, or errors on malformed input
func parseTempToC(resp string) (float64, error) {
	// -- on an unused channel, or ".."
	if strings.Contains(resp, "--") || strings.Contains(resp, "..") {
		return 0, nil
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
// Cryocon Model 12/14 Rev <fimware rev code><hardware rev code
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

// ReadAllChannels reads all of the temperature channels of the sensor in C
func (tm *TemperatureMonitor) ReadAllChannels() ([]float64, error) {
	// make the output sequence
	out := make([]float64, 0)

	var err error = nil
	err = tm.Open()
	if err != nil {
		return out, err
	}
	defer tm.CloseEventually()
	ch := 0
	for {
		cmd := []byte(fmt.Sprintf("INP %d:TEMP?;UNIT?", ch))
		resp, err := tm.SendRecv(cmd)
		fmt.Println(err)
		if err != nil {
			break
		}
		rs := string(resp)
		fmt.Println(rs, err)
		if rs == "NAK" {
			break // past the last channel
		}
		t, err := parseTempToC(rs)
		if err != nil {
			break
		}
		out = append(out, t)
		ch++
		fmt.Println(ch, t)
	}
	return out, err
}
