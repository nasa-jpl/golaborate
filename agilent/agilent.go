// Package agilent provides an interface to agilent test and measurement equipment
package agilent

import (
	"errors"
	"github.jpl.nasa.gov/bdube/golab/util"
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
func NewFunctionGenerator(addr string, connectSerial bool) *FunctionGenerator {
	var maker comm.CreationFunc
	if connectSerial {
		maker = comm.SerialConnMaker(makeSerConf(addr))
	} else {
		maker = comm.BackingOffTCPConnMaker(addr, time.Second)
	}
	pool := comm.NewPool(1, time.Hour, maker)
	return &FunctionGenerator{scpi.SCPI{Pool: pool, Handshaking: true}}
}

// SetFunction configures the output function used by the generator
func (f *FunctionGenerator) SetFunction(fcn string) error {
	// FUNC: SHAP <fcn>
	s := strings.Join([]string{":FUNC:SHAP", fcn}, " ")
	return f.Write(s)
}

// GetFunction returns the current function type used by the generator
func (f *FunctionGenerator) GetFunction() (string, error) {
	// FUNC?
	return f.ReadString(":FUNC:SHAP?")
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

// SetArbTable uploads an arbitrary functiont able to the generator
// for the 33250A, the length must be < 2^16 elements
func (f *FunctionGenerator) SetWaveform(data []uint16) error {
	if len(data) > 65535 {
		return errors.New("data too large, len must be <= 65535")
	}
	prev := f.SCPI.Handshaking
	defer func() { f.SCPI.Handshaking = prev }()
	f.SCPI.Handshaking = false

	floats := util.UintToFloat(data, 2047, 4095)
	csv := util.Float64SliceToCSV(floats, 'G', 5)
	b := []byte("DATA VOLATILE," + csv + "\n")
	conn, err := f.SCPI.Pool.Get()
	if err != nil {
		return err
	}
	defer func() { f.SCPI.Pool.ReturnWithError(conn, err) }()
	chunkSize := 64
	for i := 0; i < len(b); i += chunkSize {
		i2 := i + chunkSize
		if i2 > len(b) {
			i2 = len(b)
		}
		_, err = conn.Write(b[i:i2])
		if err != nil {
			return err
		}
		time.Sleep(50 * time.Millisecond)
	}

	return err
}
