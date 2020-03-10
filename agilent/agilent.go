// Package agilent provides an interface to agilent test and measurement equipment
package agilent

import (
	"time"

	"github.com/tarm/serial"
	"github.jpl.nasa.gov/HCIT/go-hcit/comm"
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
	*comm.RemoteDevice
}

// NewFunctionGenerator creates a new FunctionGenerator instance with
// the communuication set up
func NewFunctionGenerator(addr string, serial bool) *FunctionGenerator {
	term := &comm.Terminators{Rx: '\r', Tx: '\r'}
	cfg := makeSerConf(addr)
	rd := comm.NewRemoteDevice(addr, serial, term, cfg)
	return &FunctionGenerator{&rd}
}

// SetFunction configures the output function used by the generator
func (f *FunctionGenerator) SetFunction(fcn string) error {
	// FUNC: SHAP <fcn>
	return nil
}

// GetFunction returns the current function type used by the generator
func (f *FunctionGenerator) GetFunction() (string, error) {
	// FUNC?
	return "", nil
}

// SetFrequency configures the output frequency of the generator in Hz
func (f *FunctionGenerator) SetFrequency(hz float64) error {
	// FREQ <Hz>
	return nil
}

// GetFrequency returns the frequency of the generator in Hz
func (f *FunctionGenerator) GetFrequency() (float64, error) {
	// FREQ?
	return 0, nil
}

// SetVoltage configures the output voltage (Vpp) of the signal
func (f *FunctionGenerator) SetVoltage(volts float64) error {
	// VOLT <volts Vpp>; UNIT VPP
	return nil
}

// GetVoltage returns the current output votlage of the generator
func (f *FunctionGenerator) GetVoltage() (float64, error) {
	// VOLT?
	return 0, nil
}

// SetOffset configures the output voltage offset
func (f *FunctionGenerator) SetOffset(volts float64) error {
	// VOLT: OFF <volts>
	return nil
}

// GetOffset gets the current voltage offset
func (f *FunctionGenerator) GetOffset() (float64, error) {
	// VOLT: OFF?
	return 0, nil
}

// SetOutputLoad configures the adjustments inside the generator for the
// impedance of the load circuit
func (f *FunctionGenerator) SetOutputLoad(ohms float64) error {
	// OUT: LOAD <ohms>
	return nil
}

// EnableOutput enables the output on the front connector of the function generator
func (f *FunctionGenerator) EnableOutput() error {
	// OUT ON
	return nil
}

// DisableOutput disables the output on the front connector of the function generator
func (f *FunctionGenerator) DisableOutput() error {
	// OUT OFF
	return nil
}

func (f *FunctionGenerator) GetOutput() (bool, error) {
	// OUT? I'm assuming.
	return false, nil
}

// PopError gets a single error from the queue on the generator
func (f *FunctionGenerator) PopError() error {
	// SYST: ERR?
	return nil
}
