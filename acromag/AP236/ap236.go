/*Package ap236 provides an interface to Acromag AP236 16-bit DAC modules

 */
package ap236

/*
#cgo LDFLAGS: -lm
#include <stdlib.h>
#include "../apcommon/apcommon.h"
#include "AP236.h"
*/
import "C"
import (
	"errors"
	"fmt"
)

func init() {
	errCode := C.InitAPLib()
	if errCode != C.S_OK {
		panicS := fmt.Sprintf("initializing Acromag library failed with code %d", errCode)
		panic(panicS)
	}
}

// OutputScale is the output scale of the DAC at power up
type OutputScale int

// OutputRange is the output range of the DAC
type OutputRange int

// ThermalMode describes if the DAC shuts down when > 150C die temperature
type ThermalMode int

const (
	// ZeroScale represents a power up zero scale signal.
	// if the DAC is configured to -10 to 10V,
	// this powers up at -10V
	// likewise, if it is 0 to 10V, it powers up at 0V
	ZeroScale OutputScale = iota

	// MidScale boots the DAC at half of its output range
	MidScale

	// FullScale boots the DAC at its maximum output value
	FullScale

	// TenVSymm is a -10 to +10V output range
	TenVSymm OutputRange = iota
	// TenVPos is a 0 to 10V output range
	TenVPos
	// FiveVSymm is a -5 to 5V output range
	FiveVSymm
	//FiveVPos is a 0 to 5V output range
	FiveVPos
	// N2_5To7_5V is a -2.5 to 7.5V output range
	N2_5To7_5V
	// ThreeVSymm is a -3 to +3V output range
	ThreeVSymm
	// SixteenVPos is an output range of 0-16V, this requires an external voltage source
	SixteenVPos
	// TwentyVPos is an output range of 0-20V, this requires an external voltage source
	TwentyVPos
)

var (
	// ErrSimultaneousOutput is thrown when a device in simultaneous output mode is issued
	// an Output command that is accepted for next flush but not executed.
	ErrSimultaneousOutput = errors.New("device is in simultaneous output mode: accepted but not written")

	// IdealCode is the array from drvr236.c L60-L85
	// its inner elements, by index:
	// 0 - zero value DN, straight binary
	// 1 - zero value DN, two's complement
	// 2 - slope, DN/V
	// 3 - low voltage
	// 4 - high voltage
	// 5 - low DN
	// 6 - high DN
	// outer elements, by index
	// 0 - -10 to 10V
	// 1 - 0 to 10V
	// 2 - -5 to 5V
	// 3 - 0 to 5V
	// 4 - 0 to 5V
	// 5 - -2.5 to 7.5V
	// 6 - -3 to 3V
	// 7 - 0 to 16V
	// 8 - 0 to 20V
	IdealCode = [8][7]float64{
		/* IdealZeroSB, IdealZeroBTC, IdealSlope, -10 to 10V, cliplo, cliphi */
		{32768.0, 0.0, 3276.8, -10.0, 10.0, -32768.0, 32767.0},

		/* IdealZeroSB, IdealZeroBTC, IdealSlope,   0 to 10V, cliplo, cliphi */
		{0.0, -32768.0, 6553.6, 0.0, 10.0, -32768.0, 32767.0},

		/* IdealZeroSB, IdealZeroBTC, IdealSlope,  -5 to  5V, cliplo, cliphi */
		{32768.0, 0.0, 6553.6, -5.0, 5.0, -32768.0, 32767.0},

		/* IdealZeroSB, IdealZeroBTC, IdealSlope,   0 to  5V, cliplo, cliphi */
		{0.0, -32768.0, 13107.2, 0.0, 5.0, -32768.0, 32767.0},

		/* IdealZeroSB, IdealZeroBTC, IdealSlope, -2.5 to 7.5V, cliplo, cliphi */
		{16384.0, -16384.0, 6553.6, -2.5, 7.5, -32768.0, 32767.0},

		/* IdealZeroSB, IdealZeroBTC, IdealSlope,  -3 to  3V, cliplo, cliphi */
		{32768.0, 0.0, 10922.67, -3.0, 3.0, -32768.0, 32767.0},

		/* IdealZeroSB, IdealZeroBTC, IdealSlope, 0V to +16V, cliplo, cliphi */
		{0.0, -32768.0, 4095.9, 0.0, 16.0, -32768.0, 32767.0},

		/* IdealZeroSB, IdealZeroBTC, IdealSlope, 0V to +20V, cliplo, cliphi */
		{0.0, -32768.0, 3276.8, 0.0, 20.0, -32768.0, 32767.0},
	}

	// StatusCodes is the status codes defined by AP236.h
	// copied here to avoid C types as keys
	StatusCodes = map[int]string{
		0x8000: "ERROR",           // general
		0x8001: "OUT OF MEMORY",   // out of memory status value
		0x8002: "OUT OF APs",      // all AP spots have been taken
		0x8003: "INVALID HANDLE",  // no AP exists for this handle
		0x8006: "NOT INITIALIZED", // Pmc not initialized
		0x8007: "NOT IMPLEMENTED", // func is not implemented
		0x8008: "NO INTERRUPTS",   // unable to handle interrupts
		0x0000: "OK",              // no true error
	}
)

// enrich returns a new error and decorates with the procedure called
// if the status is OK, nil is returned
func enrich(errC C.APSTATUS, procedure string) error {
	i := int(errC)
	v, ok := StatusCodes[i]
	if !ok {
		return fmt.Errorf("unknown error code")
	}
	if v == "OK" {
		return nil
	}
	return fmt.Errorf("%b: %s encountered at call to %s", i, v, procedure)
}

// AP236 is an acromag 16-bit DAC of the same type
type AP236 struct {
	cfg C.struct_cblk236
}

// NewAP236 creates a new instance and opens the connection to the DAC
func NewAP236(deviceIndex int) (*AP236, error) {
	var (
		o    AP236
		out  = &o
		addr *C.struct_map236
	)
	// confirmed by Kate Blanketship on Gophers slack that this
	// is a valid way to generate the pointer that C wants
	// see also: several ways to get the same address of the
	// data: https://play.golang.org/p/fpkOIT9B3BB
	o.cfg.pIdealCode = &IdealCode
	// 0 is the device index, TODO: allow this to be specified
	errC := C.APOpen(deviceIndex, &o.cfg.nHandle, C.DEVICE_NAME)
	err := enrich(errC, "APOpen")
	if err != nil {
		return out, err
	}
	errC = C.APInitialize(o.cfg.nHandle)
	err = enrich(errC, "APInitialize")
	if err != nil {
		return out, err
	}
	C.GetAPAddress(o.cfg.nHandle, &addr)
	o.cfg.brd_ptr = *addr
	o.cfg.bInitialized = C.TRUE
	o.cfg.bAP = C.TRUE
	return out, nil
}

// SetRange configures the output range of the DAC
func (dac *AP236) SetRange(rng OutputRange) error {
	if rng < TenVSymm || rng > TwentyVPos {
		return fmt.Errorf("output range %d is not allowed", rng)
	}
	return nil // TODO: impl
}

// GetRange returns the output range of the DAC in volts
func (dac *AP236) GetRange() (OutputRange, error) {
	return 0, nil // TODO: impl
}

// SetPowerUpVoltage configures the voltage set on the DAC at power up
func (dac *AP236) SetPowerUpVoltage(scale OutputScale) error {
	if scale < ZeroScale || scale > FullScale {
		return fmt.Errorf("output scale %d is not allowed", scale)
	}
	return nil // TODO: impl
}

// GetPowerUpVoltage retrieves the output range of the DAC
func (dac *AP236) GetPowerUpVoltage() (OutputScale, error) {
	var out OutputScale
	return out, nil // TODO: impl
}

// SetClearVoltage sets the voltage applied at the output when the device is cleared
func (dac *AP236) SetClearVoltage(scale OutputScale) error {
	if scale < ZeroScale || scale > FullScale {
		return fmt.Errorf("output scale %d is not allowed", scale)
	}
	return nil // TODO: impl
}

// GetClearVoltage gets the voltage applied at the output when the device is cleared
func (dac *AP236) GetClearVoltage() (OutputScale, error) {
	var out OutputScale
	return out, nil // TODO: impl
}

// SetOverTempBehavior sets the behavior of the device when an over temp
// is detected.  Shutdown == true -> shut down the board on overtemp
func (dac *AP236) SetOverTempBehavior(shutdown bool) error {
	return nil // TODO: impl
}

// GetOverTempBehavior returns true if the device will shut down when over temp
func (dac *AP236) GetOverTempBehavior() (bool, error) {
	return false, nil // TODO: impl
}

// SetOverRange configures if the DAC is allowed to exceed output limits by 5%
// allowed == true allows the DAC to operate slightly beyond limits
func (dac *AP236) SetOverRange(allowed bool) error {
	return nil // TODO: impl
}

// GetOverRange returns true if the DAC output is allowed to exceed nominal by 5%
func (dac *AP236) GetOverRange() (bool, error) {
	return false, nil // TODO: impl
}

// SetOutputSimultaneous configures the DAC to simultaneous mode or async mode
func (dac *AP236) SetOutputSimultaneous(simultaneous bool) error {
	return nil // TODO: impl
}

// GetOutputSimultaneous returns true if the DAC is in simultaneous output mode
func (dac *AP236) GetOutputSimultaneous() (bool, error) {
	return false, nil // TODO: impl
}

// Output writes a voltage to a channel.
//
func (dac *AP236) Output(channel int, voltage float64) error {
	return nil // TODO: impl
}

// Flush writes any pending output values to the device
func (dac *AP236) Flush() error {
	return nil // TODO: impl
}

// Clear soft resets the DAC, clearing the output but not configuration
func (dac *AP236) Clear() error {
	return nil // TODO: impl
}
