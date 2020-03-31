// Package agilent provides an interface to agilent test and measurement equipment
package agilent

import (
	"strconv"
	"strings"
	"time"

	"github.com/tarm/serial"
	"github.jpl.nasa.gov/bdube/golab/comm"
	"github.jpl.nasa.gov/bdube/golab/scpi"
)

// makeSerConf makes a new serial.Config with correct parity, baud, etc, set.
func makeSerConf(addr string) *serial.Config {
	return &serial.Config{
		Name:        addr,
		Baud:        57600,
		Size:        8,
		Parity:      serial.ParityNone,
		StopBits:    serial.Stop1,
		ReadTimeout: 10 * time.Minute}
}

// FunctionGenerator is an interface to hardware of the same name
type FunctionGenerator struct {
	scpi.SCPI
}

// NewFunctionGenerator creates a new FunctionGenerator instance with
// the communuication set up
func NewFunctionGenerator(addr string, serial bool) *FunctionGenerator {
	term := &comm.Terminators{Rx: 10, Tx: 10}
	cfg := makeSerConf(addr)
	rd := comm.NewRemoteDevice(addr, serial, term, cfg)
	return &FunctionGenerator{scpi.SCPI{RemoteDevice: &rd, Handshaking: true}}
}

// SetFunction configures the output function used by the generator
func (f *FunctionGenerator) SetFunction(fcn string) error {
	// FUNC: SHAP <fcn>
	s := strings.Join([]string{"FUNC:", fcn}, "")
	return f.Write(s)
}

// GetFunction returns the current function type used by the generator
func (f *FunctionGenerator) GetFunction() (string, error) {
	// FUNC?
	return f.ReadString("FUNC:SHAP?")
}

// SetFrequency configures the output frequency of the generator in Hz
func (f *FunctionGenerator) SetFrequency(hz float64) error {
	// FREQ <Hz>
	s := strconv.FormatFloat(hz, 'G', -1, 64)
	return f.Write("FREQ", s)
}

// GetFrequency returns the frequency of the generator in Hz
func (f *FunctionGenerator) GetFrequency() (float64, error) {
	// FREQ?
	return f.ReadFloat("FREQ?")
}

// SetVoltage configures the output voltage (Vpp) of the signal
func (f *FunctionGenerator) SetVoltage(volts float64) error {
	// VOLT <volts Vpp>; UNIT VPP
	s := strconv.FormatFloat(volts, 'G', -1, 64)
	return f.Write("VOLT", s, "VPP")
}

// GetVoltage returns the current output votlage of the generator
func (f *FunctionGenerator) GetVoltage() (float64, error) {
	// VOLT?
	return f.ReadFloat("VOLT?")
}

// SetOffset configures the output voltage offset
func (f *FunctionGenerator) SetOffset(volts float64) error {
	// VOLT: OFF <volts>
	s := strconv.FormatFloat(volts, 'G', -1, 64)
	return f.Write("VOLT:OFFSET", s)
}

// GetOffset gets the current voltage offset
func (f *FunctionGenerator) GetOffset() (float64, error) {
	// VOLT: OFF?
	return f.ReadFloat("VOLT:OFFSET?")
}

// SetOutputLoad configures the adjustments inside the generator for the
// impedance of the load circuit
func (f *FunctionGenerator) SetOutputLoad(ohms float64) error {
	// OUT: LOAD <ohms>
	s := strconv.FormatFloat(ohms, 'G', -1, 64)
	return f.Write("OUTPUT: LOAD", s)
}

// SetOutput turns Output on or off
func (f *FunctionGenerator) SetOutput(on bool) error {
	predicate := "OFF"
	if on {
		predicate = "ON"
	}
	return f.Write("OUTPUT " + predicate)
}

// GetOutput returns True if the generator is currently outputting a signal
func (f *FunctionGenerator) GetOutput() (bool, error) {
	// OUT? I'm assuming.
	return f.ReadBool("OUTPUT?")
}
