// Package agilent provides an interface to agilent test and measurement equipment
package agilent

import (
	"fmt"
	"strconv"
	"strings"
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
	term := &comm.Terminators{Rx: 10, Tx: 10}
	cfg := makeSerConf(addr)
	rd := comm.NewRemoteDevice(addr, serial, term, cfg)
	return &FunctionGenerator{&rd}
}

func (f *FunctionGenerator) writeOnlyBus(cmds ...string) error {
	err := f.RemoteDevice.Open()
	if err != nil {
		return err
	}
	defer f.CloseEventually()
	s := strings.Join(cmds, " ")
	return f.RemoteDevice.Send([]byte(s))
}

func (f *FunctionGenerator) readString(cmds ...string) (string, error) {
	f.writeOnlyBus(cmds...)
	resp, err := f.RemoteDevice.Recv()
	if err != nil {
		return "", err
	}
	return string(resp), nil
}

func (f *FunctionGenerator) readFloat(cmds ...string) (float64, error) {
	f.writeOnlyBus(cmds...)
	resp, err := f.RemoteDevice.Recv()
	if err != nil {
		return 0, err
	}
	return strconv.ParseFloat(string(resp), 64)
}

func (f *FunctionGenerator) readBool(cmds ...string) (bool, error) {
	f.writeOnlyBus(cmds...)
	resp, err := f.RemoteDevice.Recv()
	if err != nil {
		return false, err
	}
	return strconv.ParseBool(string(resp))
}

// SetFunction configures the output function used by the generator
func (f *FunctionGenerator) SetFunction(fcn string) error {
	// FUNC: SHAP <fcn>
	s := strings.Join([]string{"FUNC:", fcn}, "")
	return f.writeOnlyBus(s)
}

// GetFunction returns the current function type used by the generator
func (f *FunctionGenerator) GetFunction() (string, error) {
	// FUNC?
	return f.readString("FUNC:SHAP?")
}

// SetFrequency configures the output frequency of the generator in Hz
func (f *FunctionGenerator) SetFrequency(hz float64) error {
	// FREQ <Hz>
	s := strconv.FormatFloat(hz, 'G', -1, 64)
	return f.writeOnlyBus("FREQ", s)
}

// GetFrequency returns the frequency of the generator in Hz
func (f *FunctionGenerator) GetFrequency() (float64, error) {
	// FREQ?
	return f.readFloat("FREQ?")
}

// SetVoltage configures the output voltage (Vpp) of the signal
func (f *FunctionGenerator) SetVoltage(volts float64) error {
	// VOLT <volts Vpp>; UNIT VPP
	s := strconv.FormatFloat(volts, 'G', -1, 64)
	return f.writeOnlyBus("VOLT", s, "VPP")
}

// GetVoltage returns the current output votlage of the generator
func (f *FunctionGenerator) GetVoltage() (float64, error) {
	// VOLT?
	return f.readFloat("VOLT?")
}

// SetOffset configures the output voltage offset
func (f *FunctionGenerator) SetOffset(volts float64) error {
	// VOLT: OFF <volts>
	s := strconv.FormatFloat(volts, 'G', -1, 64)
	return f.writeOnlyBus("VOLT:OFFSET", s)
}

// GetOffset gets the current voltage offset
func (f *FunctionGenerator) GetOffset() (float64, error) {
	// VOLT: OFF?
	return f.readFloat("VOLT:OFFSET?")
}

// SetOutputLoad configures the adjustments inside the generator for the
// impedance of the load circuit
func (f *FunctionGenerator) SetOutputLoad(ohms float64) error {
	// OUT: LOAD <ohms>
	s := strconv.FormatFloat(ohms, 'G', -1, 64)
	return f.writeOnlyBus("OUTPUT: LOAD", s)
}

// EnableOutput enables the output on the front connector of the function generator
func (f *FunctionGenerator) EnableOutput() error {
	// OUT ON
	return f.writeOnlyBus("OUTPUT ON")
}

// DisableOutput disables the output on the front connector of the function generator
func (f *FunctionGenerator) DisableOutput() error {
	// OUT OFF
	return f.writeOnlyBus("OUTPUT OFF")
}

// GetOutput returns True if the generator is currently outputting a signal
func (f *FunctionGenerator) GetOutput() (bool, error) {
	// OUT? I'm assuming.
	return f.readBool("OUTPUT?")
}

// PopError gets a single error from the queue on the generator
func (f *FunctionGenerator) PopError() error {
	// SYST: ERR?
	s, err := f.readString("SYSTem:ERRor?") // unclear why the case needs to be this way
	if err != nil {
		return err
	}
	return fmt.Errorf(s)
}
