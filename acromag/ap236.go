package acromag

/*
#cgo LDFLAGS: -lm
#include "apcommon.h"
#include "AP236.h"
#include "shim236.h"
*/
import "C"
import (
	"errors"
	"fmt"
	"unsafe"
)

// AP236 is an acromag 16-bit DAC of the same type
type AP236 struct {
	cfg *C.struct_cblk236
}

// NewAP236 creates a new instance and opens the connection to the DAC
func NewAP236(deviceIndex int) (*AP236, error) {
	var (
		o    AP236
		out  = &o
		addr *C.struct_map236
		cs   = C.CString(C.DEVICE_NAME) // untyped constant in C needs enforcement in Go
	)
	defer C.free(unsafe.Pointer(cs))
	o.cfg = (*C.struct_cblk236)(C.malloc(C.sizeof_struct_cblk236))
	o.cfg.pIdealCode = cMkCopyOfIdealData(idealCode)
	errC := C.APOpen(C.int(deviceIndex), &o.cfg.nHandle, cs)
	err := enrich(errC, "APOpen")
	if err != nil {
		return out, err
	}
	errC = C.APInitialize(o.cfg.nHandle)
	err = enrich(errC, "APInitialize")
	if err != nil {
		return out, err
	}
	errC = C.GetAPAddress236(o.cfg.nHandle, &addr)
	err = enrich(errC, "GetAPAddress")
	if err != nil {
		return out, err
	}
	o.cfg.brd_ptr = addr
	o.cfg.bInitialized = C.TRUE
	o.cfg.bAP = C.TRUE
	errC = C.Setup_board_cal(o.cfg)
	if errC != 0 {
		return out, errors.New("error getting offset and gain coefs from AP236")
	}
	return out, nil
}

// SetRange configures the output range of the DAC
// this function only returns an error if the range is not allowed
// rngS is specified as in ValidateOutputRange
func (dac *AP236) SetRange(channel int, rngS string) error {
	rng, err := ValidateOutputRange(rngS)
	if err != nil {
		return err
	}
	Crng := C.int(rng)
	dac.cfg.opts._chan[C.int(channel)].Range = Crng
	dac.sendCfgToBoard(channel)
	return nil
}

// GetRange returns the output range of the DAC in volts.
// The error value is always nil; the API looks
// this way for symmetry with Set
func (dac *AP236) GetRange(channel int) (string, error) {
	Crng := dac.cfg.opts._chan[C.int(channel)].Range
	return FormatOutputRange(OutputRange(Crng)), nil
}

// SetPowerUpVoltage configures the voltage set on the DAC at power up
// The error is only non-nil if the scale is invalid
func (dac *AP236) SetPowerUpVoltage(channel int, scale OutputScale) error {
	if scale < ZeroScale || scale > FullScale {
		return fmt.Errorf("output scale %d is not allowed", scale)
	}
	dac.cfg.opts._chan[C.int(channel)].PowerUpVoltage = C.int(scale)
	dac.sendCfgToBoard(channel)
	return nil
}

// GetPowerUpVoltage retrieves the voltage of the DAC at power up.
// the error is always nil
func (dac *AP236) GetPowerUpVoltage(channel int) (OutputScale, error) {
	Cpwr := dac.cfg.opts._chan[C.int(channel)].PowerUpVoltage
	return OutputScale(Cpwr), nil
}

// SetClearVoltage sets the voltage applied at the output when the device is cleared
// the error is only non-nil if the voltage is invalid
func (dac *AP236) SetClearVoltage(channel int, scale OutputScale) error {
	if scale < ZeroScale || scale > FullScale {
		return fmt.Errorf("output scale %d is not allowed", scale)
	}
	dac.cfg.opts._chan[C.int(channel)].ClearVoltage = C.int(scale)
	dac.sendCfgToBoard(channel)
	return nil
}

// GetClearVoltage gets the voltage applied at the output when the device is cleared
// The error is always nil
func (dac *AP236) GetClearVoltage(channel int) (OutputScale, error) {
	Cpwr := dac.cfg.opts._chan[C.int(channel)].ClearVoltage
	return OutputScale(Cpwr), nil
}

// SetOverTempBehavior sets the behavior of the device when an over temp
// is detected.  Shutdown == true -> shut down the board on overtemp
// the error is always nil
func (dac *AP236) SetOverTempBehavior(channel int, shutdown bool) error {
	i := 0
	if shutdown {
		i = 1
	}
	dac.cfg.opts._chan[C.int(channel)].ThermalShutdown = C.int(i)
	dac.sendCfgToBoard(channel)
	return nil
}

// GetOverTempBehavior returns true if the device will shut down when over temp
// the error is always nil
func (dac *AP236) GetOverTempBehavior(channel int) (bool, error) {
	Cint := dac.cfg.opts._chan[C.int(channel)].ThermalShutdown
	return Cint == 1, nil
}

// SetOverRange configures if the DAC is allowed to exceed output limits by 5%
// allowed == true allows the DAC to operate slightly beyond limits
// the error is always nil
func (dac *AP236) SetOverRange(channel int, allowed bool) error {
	i := 0
	if allowed {
		i = 1
	}
	dac.cfg.opts._chan[C.int(channel)].OverRange = C.int(i)
	dac.sendCfgToBoard(channel)
	return nil
}

// GetOverRange returns true if the DAC output is allowed to exceed nominal by 5%
// the error is always nil
func (dac *AP236) GetOverRange(channel int) (bool, error) {
	Cint := dac.cfg.opts._chan[C.int(channel)].OverRange
	return Cint == 1, nil
}

// SetOutputSimultaneous configures the DAC to simultaneous mode or async mode
// this function will always return nil.
func (dac *AP236) SetOutputSimultaneous(channel int, simultaneous bool) error {
	sim := 0
	if simultaneous {
		sim = 1
	}
	// opts.chan -> opts._chan; cgo rule to replace go identifier
	dac.cfg.opts._chan[C.int(channel)].UpdateMode = C.int(sim)
	dac.sendCfgToBoard(channel)
	return nil
}

// GetOutputSimultaneous returns true if the DAC is in simultaneous output mode
// the error value is always nil
func (dac *AP236) GetOutputSimultaneous(channel int) (bool, error) {
	i := int(dac.cfg.opts._chan[C.int(channel)].UpdateMode)
	return i == 1, nil
}

// sendCfgToBoard updates the configuration on the board
func (dac *AP236) sendCfgToBoard(channel int) {
	C.cnfg236(dac.cfg, C.int(channel))
	return
}

// Output writes a voltage to a channel.
// the error is only non-nil if the value is out of range
func (dac *AP236) Output(channel int, voltage float64) error {
	// TODO: look into cd236 C function
	C.cd236(dac.cfg, C.int(channel), C.double(voltage))
	C.wro236(dac.cfg, C.int(channel), (C.word)(dac.cfg.cor_buf[channel]))
	return nil
	// return dac.OutputDN16(channel, dac.calibrateData(channel, voltage))
}

// OutputDN16 writes a value to the board in DN.
// the error is always nil
func (dac *AP236) OutputDN16(channel int, value uint16) error {
	rng, _ := dac.GetRange(channel)
	min, max := RangeToMinMax(rng)
	step := (max - min) / 65535
	fV := min + step*float64(value)
	C.cd236(dac.cfg, C.int(channel), C.double(fV))
	C.wro236(dac.cfg, C.int(channel), (C.word)(dac.cfg.cor_buf[channel]))
	return nil
}

// OutputMulti writes voltages to multiple output channels.
// the error is non-nil if any of these conditions occur:
//	1.  A blend of output modes (some simultaneous, some immediate)
//  2.  A command is out of range
//
// if an error is encountered in case 2, the output buffer of the DAC may be
// partially updated from proceeding valid commands.  No invalid values escape
// to the DAC.
//
// The device is flushed after writing if the channels are simultaneous output.
//
// passing zero length slices will cause a panic.  Slices must be of equal length.
func (dac *AP236) OutputMulti(channels []int, voltages []float64) error {
	// ensure channels are homogeneous
	sim, _ := dac.GetOutputSimultaneous(channels[0])
	for i := 0; i < len(channels); i++ { // old for is faster than range, this code may be hot
		sim2, _ := dac.GetOutputSimultaneous(channels[i])
		if sim2 != sim {
			return fmt.Errorf("mixture of output modes used, must be homogeneous.  Channel %d != channel %d",
				channels[i], channels[0])
		}
	}
	for i := 0; i < len(channels); i++ {
		err := dac.Output(channels[i], voltages[i])
		if err != nil {
			return fmt.Errorf("channel %d voltage %f: %w", channels[i], voltages[i], err)
		}
	}
	if sim {
		dac.Flush()
	}
	return nil
}

// OutputMultiDN16 is equivalent to OutputMulti, but with DNs instead of volts.
// see the docstring of OutputMulti for more information.
func (dac *AP236) OutputMultiDN16(channels []int, uint16s []uint16) error {
	// ensure channels are homogeneous
	sim, _ := dac.GetOutputSimultaneous(channels[0])
	for i := 0; i < len(channels); i++ { // old for is faster than range, this code may be hot
		sim2, _ := dac.GetOutputSimultaneous(channels[i])
		if sim2 != sim {
			return fmt.Errorf("mixture of output modes used, must be homogeneous.  Channel %d != channel %d",
				channels[i], channels[0])
		}
	}
	for i := 0; i < len(channels); i++ {
		err := dac.OutputDN16(channels[i], uint16s[i])
		if err != nil {
			return fmt.Errorf("channel %d DN %f: %w", channels[i], uint16s[i], err)
		}
	}
	if sim {
		dac.Flush()
	}
	return nil
}

// Flush writes any pending output values to the device
func (dac *AP236) Flush() {
	C.simtrig236(dac.cfg)
}

// Clear soft resets the DAC, clearing the output but not configuration
// the error is always nil
func (dac *AP236) Clear(channel int) error {
	dac.cfg.opts._chan[C.int(channel)].DataReset = C.int(1)
	dac.sendCfgToBoard(channel)
	dac.cfg.opts._chan[C.int(channel)].DataReset = C.int(0)
	return nil
}

// Reset completely clears both data and configuration for a channel
// the error is always nil
func (dac *AP236) Reset(channel int) error {
	dac.cfg.opts._chan[C.int(channel)].FullReset = C.int(1)
	dac.sendCfgToBoard(channel)
	dac.cfg.opts._chan[C.int(channel)].FullReset = C.int(0)
	return nil
}

// Close the dac, freeing hardware.
func (dac *AP236) Close() error {
	errC := C.APClose(dac.cfg.nHandle)
	return enrich(errC, "APClose")
}
