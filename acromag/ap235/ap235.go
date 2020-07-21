/*Package ap235 provides an interface to Acromag AP235 16-bit waveform DAC modules

Some performance-limiting design changes are made from the C SDK provided
by acromag.  Namely: there are Go functions for each channel configuration,
and a call to any of them issues a transfer of the configuration to the board.
This will prevent the last word in performance under obscure conditions (e.g.
updating the config 10,000s of times per second) but is not generally harmful
and simplifies interfacing to the device, as there can be no "I updated the
config but forgot to write to the device" errors.

Basic usage is as followed:
 dac, err := ap235.New(0) // 0 is the 0th card in the system, in range 0~4
 if err != nil {
 	log.fatal(err)
 }
 defer dac.Close()
 // single channel, immediate mode (output immediately on write)
 // see method docs on error values, this example ignores them
 // there is a get for each set
 ch := 1
 dac.SetRange(ch, TenVSymm)
 dac.SetPowerUpVoltage(ch, ap235.MidScale)
 dac.SetClearVoltage(ch, ap235.MidScale)
 dac.SetOverTempBehavior(ch, true) // shut down if over temp
 dac.SetOverRange(ch, false) // over range not allowed
 dac.SetOutputSimultaneous(ch, false) // this is what puts it in immediate mode
 dac.Output(ch, 1.) // 1 volt as close as can be quantized.  Calibrated.
 dac.OutputDN(ch, 2000) // 2000 DN, uncalibrated

 // multi-channel, synchronized
 chs := []int{1,2,3}
 // setup
 for _, ch := range chs {
 	dac.SetRange(ch, TenVSymm)
	dac.SetPowerUpVoltage(ch, ap235.MidScale)
	dac.SetClearVoltage(ch, ap235.MidScale)
	dac.SetOverTempBehavior(ch, true)
	dac.SetOverRange(ch, false)
 	dac.SetOutputSimultaneous(ch, true)
 }
 // in your code
 dac.OutputMulit(chs, []float64{1, 2, 3} // calibrated
 dac.OutputMultiDN(chs, []uint16{1000, 2000, 3000}) // uncalibrated

*/
package ap235

/*
#cgo LDFLAGS: -lm
#include <stdlib.h>
#include "../apcommon/apcommon.h"
#include "AP235.h"
#include "shim.h"
*/
import "C"
import (
	"errors"
	"fmt"
	"math"
	"unsafe"
)

func init() {
	errCode := C.InitAPLib()
	if errCode != C.S_OK {
		panicS := fmt.Sprintf("initializing Acromag library failed with code %d", errCode)
		panic(panicS)
	}
}

// OutputScale is the output scale of the DAC at power up or clear
type OutputScale int

// OutputRange is the output range of the DAC
type OutputRange int

// TriggerMode is a triggering mode
type TriggerMode int

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

	slope = C.IDEALSLOPE
	zero  = C.IDEALZEROBTC
	maxV  = C.ENDPOINTHI
	minV  = C.ENDPOINTLO
	maxDN = C.CLIPHI
	minDN = C.CLIPLO

	// TriggerSoftware represents a software triggering mode
	TriggerSoftware TriggerMode = iota

	// TriggerTimer represents a timer (internally clocked waveform) trigger mode
	TriggerTimer

	// TriggerExternal represents a triggering mode which is externally clocked
	TriggerExternal
)

var (
	// ErrSimultaneousOutput is generated when a device in simultaneous output mode is issued
	// an Output command that is accepted for next flush but not executed.
	ErrSimultaneousOutput = errors.New("device is in simultaneous output mode: accepted but not written")

	// ErrVoltageTooLow is generated when a too low voltage is commanded
	ErrVoltageTooLow = errors.New("commanded voltage below lower limit")

	// ErrVoltageTooHigh is generated when a too high voltage is commanded
	ErrVoltageTooHigh = errors.New("commanded voltage above upper limit")

	// ErrTimerTooFast is generated when a timer is running too fast
	ErrTimerTooFast = errors.New("timer too fast: DAC cannot settle to < 1LSB before next value given")

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
	// the range channel option is an index into the outer element
	idealCode = [8][7]float64{
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

	// StatusCodes is the status codes defined by AP235.h
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

// ValidateOutputRange ensures that an output range is valid
// s is formatted as "<low>,<high>"
func ValidateOutputRange(s string) (OutputRange, error) {
	switch s {
	case "-10,10":
		return TenVSymm, nil
	case "0,10":
		return TenVPos, nil
	case "-5,5":
		return FiveVSymm, nil
	case "0,5":
		return FiveVPos, nil
	case "-2.5,7.5":
		return N2_5To7_5V, nil
	case "-3,3":
		return ThreeVSymm, nil
	case "0,16":
		return SixteenVPos, nil
	case "0,20":
		return TwentyVPos, nil
	default:
		return 0, errors.New("invalid output range")
	}
}

// FormatOutputRange converts an output range to a CSV of low,high
func FormatOutputRange(o OutputRange) string {
	switch o {
	case TenVSymm:
		return "-10,10"
	case TenVPos:
		return "0,10"
	case FiveVSymm:
		return "-5,5"
	case FiveVPos:
		return "0,5"
	case N2_5To7_5V:
		return "-2.5,7.5"
	case ThreeVSymm:
		return "-3,3"
	case SixteenVPos:
		return "0,16"
	case TwentyVPos:
		return "0,20"
	default:
		return ""
	}
}

// ValidateTriggerMode ensures that a triggering mode is valid
// s is a member of {software, timer, external}
func ValidateTriggerMode(s string) (TriggerMode, error) {
	switch s {
	case "software":
		return TriggerSoftware, nil
	case "timer":
		return TriggerTimer, nil
	case "external":
		return TriggerExternal, nil
	default:
		return -1, fmt.Errorf("triggering mode must be a member of {software, timer, external}")
	}
}

// FormatTriggerMode converts a trigger mode to a string representation,
// which is a member of {software, timer, external}
func FormatTriggerMode(t TriggerMode) string {
	switch t {
	case TriggerSoftware:
		return "software"
	case TriggerTimer:
		return "timer"
	case TriggerExternal:
		return "external"
	default:
		return ""
	}
}

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

// AP235 is an acromag 16-bit DAC of the same type
type AP235 struct {
	cfg C.struct_cblk235
}

// New creates a new instance and opens the connection to the DAC
func New(deviceIndex int) (*AP235, error) {
	var (
		o    AP235
		out  = &o
		addr *C.struct_mapap235
		cs   = C.CString(C.DEVICE_NAME) // untyped constant in C needs enforcement in Go
	)
	defer C.free(unsafe.Pointer(cs))
	// confirmed by Kate Blanketship on Gophers slack that this
	// is a valid way to generate the pointer that C wants
	// see also: several ways to get the same address of the
	// data: https://play.golang.org/p/fpkOIT9B3BB
	cptr := (*[8][7]C.double)(unsafe.Pointer(&idealCode))
	o.cfg.pIdealCode = cptr

	// open the board, initialize it, get its address, and populate its config
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
	errC = C.GetAPAddress2(o.cfg.nHandle, &addr)
	err = enrich(errC, "GetAPAddress")
	if err != nil {
		return out, err
	}
	o.cfg.brd_ptr = addr
	o.cfg.pAP = C.GetAP(o.cfg.nHandle)
	if o.cfg.pAP == nil {
		return out, fmt.Errorf("unable to get a pointer to the acropack module")
	}

	// assign the buffer pointer
	C.Setup_board_corrected_buffer(&o.cfg)
	// binitialize and bAP are set in Setup_board
	return out, nil
}

// SetRange configures the output range of the DAC
// this function only returns an error if the range is not allowed
// rngS is specified as in ValidateOutputRange
func (dac *AP235) SetRange(channel int, rngS string) error {
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
func (dac *AP235) GetRange(channel int) (string, error) {
	Crng := dac.cfg.opts._chan[C.int(channel)].Range
	return FormatOutputRange(OutputRange(Crng)), nil
}

// SetPowerUpVoltage configures the voltage set on the DAC at power up
// The error is only non-nil if the scale is invalid
func (dac *AP235) SetPowerUpVoltage(channel int, scale OutputScale) error {
	if scale < ZeroScale || scale > FullScale {
		return fmt.Errorf("output scale %d is not allowed", scale)
	}
	dac.cfg.opts._chan[C.int(channel)].PowerUpVoltage = C.int(scale)
	dac.sendCfgToBoard(channel)
	return nil
}

// GetPowerUpVoltage retrieves the voltage of the DAC at power up.
// the error is always nil
func (dac *AP235) GetPowerUpVoltage(channel int) (OutputScale, error) {
	Cpwr := dac.cfg.opts._chan[C.int(channel)].PowerUpVoltage
	return OutputScale(Cpwr), nil
}

// SetClearVoltage sets the voltage applied at the output when the device is cleared
// the error is only non-nil if the voltage is invalid
func (dac *AP235) SetClearVoltage(channel int, scale OutputScale) error {
	if scale < ZeroScale || scale > FullScale {
		return fmt.Errorf("output scale %d is not allowed", scale)
	}
	dac.cfg.opts._chan[C.int(channel)].ClearVoltage = C.int(scale)
	dac.sendCfgToBoard(channel)
	return nil
}

// GetClearVoltage gets the voltage applied at the output when the device is cleared
// The error is always nil
func (dac *AP235) GetClearVoltage(channel int) (OutputScale, error) {
	Cpwr := dac.cfg.opts._chan[C.int(channel)].ClearVoltage
	return OutputScale(Cpwr), nil
}

// SetOverTempBehavior sets the behavior of the device when an over temp
// is detected.  Shutdown == true -> shut down the board on overtemp
// the error is always nil
func (dac *AP235) SetOverTempBehavior(channel int, shutdown bool) error {
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
func (dac *AP235) GetOverTempBehavior(channel int) (bool, error) {
	Cint := dac.cfg.opts._chan[C.int(channel)].ThermalShutdown
	return Cint == 1, nil
}

// SetOverRange configures if the DAC is allowed to exceed output limits by 5%
// allowed == true allows the DAC to operate slightly beyond limits
// the error is always nil
func (dac *AP235) SetOverRange(channel int, allowed bool) error {
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
func (dac *AP235) GetOverRange(channel int) (bool, error) {
	Cint := dac.cfg.opts._chan[C.int(channel)].OverRange
	return Cint == 1, nil
}

// SetTriggerMode configures the DAC for a given triggering mode
// the error is only non-nil if the trigger mode is invalid
func (dac *AP235) SetTriggerMode(channel int, triggerMode string) error {
	tm, err := ValidateTriggerMode(triggerMode)
	if err != nil {
		return err
	}
	dac.cfg.opts._chan[C.int(channel)].TriggerSource = C.int(tm)
	dac.sendCfgToBoard(channel)
	return nil
}

// GetTriggerMode retrieves the current triggering mode
// the error is always nil
func (dac *AP235) GetTriggerMode(channel int) (string, error) {
	tm := dac.cfg.opts._chan[C.int(channel)].TriggerSource
	return FormatTriggerMode(TriggerMode(tm)), nil
}

// SetTriggerDirection if the DAC's trigger is input (false) or output (true)
// the error is always nil.
func (dac *AP235) SetTriggerDirection(b bool) error {
	var i int // init to zero value, false->0
	if b {
		i = 1
	}
	dac.cfg.TriggerDirection = C.uint32_t(i)
	// dac.sendCfgToBoard() TODO: need to send to board?
	return nil
}

// GetTriggerDirection returns true if the DAC's trigger is output, false if it is input
// the error is always nil
func (dac *AP235) GetTriggerDirection() (bool, error) {
	ci := dac.cfg.TriggerDirection
	var b bool
	if ci == 1 {
		b = true
	}
	return b, nil
}

// SetClearOnUnderflow configures the DAC to clear output on an underflow if true
// the error is always nil
func (dac *AP235) SetClearOnUnderflow(channel int, b bool) error {
	var i int // init to zero value, false->0
	if b {
		i = 1
	}
	dac.cfg.opts._chan[C.int(channel)].UnderflowClear = C.int(i)
	dac.sendCfgToBoard(channel)
	return nil
}

// GetClearOnUnderflow configures the DAC to clear output on an underflow if true
// the error is always nil
func (dac *AP235) GetClearOnUnderflow(channel int) (bool, error) {
	ci := dac.cfg.opts._chan[C.int(channel)].UnderflowClear
	var b bool
	if ci == 1 {
		b = true
	}
	return b, nil
}

// SetOutputSimultaneous configures the DAC to simultaneous mode or async mode
// this function will always return nil.
func (dac *AP235) SetOutputSimultaneous(channel int, simultaneous bool) error {
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
func (dac *AP235) GetOutputSimultaneous(channel int) (bool, error) {
	i := int(dac.cfg.opts._chan[C.int(channel)].UpdateMode)
	return i == 1, nil
}

// SetTimerPeriod sets the timer period,
// the time between repetitions of the timer clock
//
// if the period is too short, it is still used but an error is generated.
// the accuracy of the output may be compromised operating in this regime.
//
// no other errors can be generated.
func (dac *AP235) SetTimerPeriod(nanoseconds uint32) error {
	tdiv := nanoseconds / 32
	dac.cfg.TimerDivider = C.uint32_t(tdiv)
	if tdiv < 310 {
		return ErrTimerTooFast
	}
	return nil
}

// GetTimerPeriod retrieves the timer period in nanoseconds
//
// the error is always nil
func (dac *AP235) GetTimerPeriod() (uint32, error) {
	return uint32(dac.cfg.TimerDivider) * 32, nil
}

// sendCfgToBoard updates the configuration on the board
func (dac *AP235) sendCfgToBoard(channel int) {
	C.cnfg235(&dac.cfg, C.int(channel))
	return
}

// Output writes a voltage to a channel.
// the error is only non-nil if the value is out of range
func (dac *AP235) Output(channel int, voltage float64) error {
	// TODO: look into cd235 C function
	// this function copied and unchanged from AP236; same logic,
	// only difference is in OutputDN
	rngS, _ := dac.GetRange(channel)    // output range
	rng, _ := ValidateOutputRange(rngS) // from the board, the range cannot be invalid; ignore error

	slp := idealCode[rng][slope]   // slope
	zro := idealCode[rng][zero]    // zero DN
	mindn := idealCode[rng][minDN] // min value allowed
	maxdn := idealCode[rng][maxDN] // max value allowed
	minvolt := idealCode[rng][minV]
	maxvolt := idealCode[rng][maxV]
	dn := math.Round(voltage*slp + zro)
	if voltage < minvolt || dn < mindn {
		return ErrVoltageTooLow
	} else if voltage > maxvolt || dn > maxdn {
		return ErrVoltageTooHigh
	}
	dnU := uint16(dn)
	return dac.OutputDN(channel, dnU)
}

// OutputDN writes a value to the board in DN.
// Value is of type interface{} for compatibility but must be a uint16
// or an error will be generated
func (dac *AP235) OutputDN(channel int, value interface{}) error {
	v, ok := (value).(uint16)
	if !ok {
		return fmt.Errorf("output value is not a uint16")
	}
	// set FIFO configuration for this channel to 1 sample
	cCh := C.int(channel)
	cfg := dac.cfg
	cfg.SampleCount[cCh] = 1
	ptr := &cfg.ideal_buf[cCh][0]
	cfg.current_ptr[cCh] = ptr
	cfg.head_ptr[cCh] = ptr

	// unwrapped version
	// this is not legal.  The runtime may move what ptr points to
	// in general, but when this exists in one expression,
	// the expression in its entirety is valid
	// ptr2 := unsafe.Pointer(ptr)
	// ptr3 := uintptr(ptr2) + 1
	// cfg.tail_ptr[cCh] = (*C.short)(unsafe.Pointer(ptr3))
	ptr2 := unsafe.Pointer(uintptr(unsafe.Pointer(ptr)) + 1)
	cfg.tail_ptr[cCh] = (*C.short)(ptr2)
	cfg.ideal_buf[cCh][0] = C.short(int16(v - 32768)) // TODO: commented out here; 0x8000 overflows int16 ^ 0x8000) // BTC to straight binary via ^
	C.fifowro235(&dac.cfg, cCh)
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
func (dac *AP235) OutputMulti(channels []int, voltages []float64) error {
	// how this is different to AP236:
	// AP236 is immediate output.  Write output -> it happens.
	// AP235 is waveform and has three triggering modes for each
	// channel:
	// 1.  software
	// 2.  timer
	// 3.  exterinal input
	// ensure channels are homogeneous
	for i := 0; i < len(channels); i++ { // old for is faster than range, this code may be hot
		tm, _ := dac.GetTriggerMode(channels[i])
		if tm != "software" {
			return fmt.Errorf("trigger mode must be software.  Channel %d was %s",
				channels[i], tm)
		}
	}
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

// OutputMultiDN is equivalent to OutputMulti, but with DNs instead of volts.
// see the docstring of OutputMulti for more information.
func (dac *AP235) OutputMultiDN(channels []int, uint16s []interface{}) error {
	// how this is different to AP236:
	// AP236 is immediate output.  Write output -> it happens.
	// AP235 is waveform and has three triggering modes for each
	// channel:
	// 1.  software
	// 2.  timer
	// 3.  exterinal input
	// ensure channels are homogeneous
	for i := 0; i < len(channels); i++ { // old for is faster than range, this code may be hot
		tm, _ := dac.GetTriggerMode(channels[i])
		if tm != "software" {
			return fmt.Errorf("trigger mode must be software.  Channel %d was %s",
				channels[i], tm)
		}
	}
	sim, _ := dac.GetOutputSimultaneous(channels[0])
	for i := 0; i < len(channels); i++ { // old for is faster than range, this code may be hot
		sim2, _ := dac.GetOutputSimultaneous(channels[i])
		if sim2 != sim {
			return fmt.Errorf("mixture of output modes used, must be homogeneous.  Channel %d != channel %d",
				channels[i], channels[0])
		}
	}
	for i := 0; i < len(channels); i++ {
		err := dac.OutputDN(channels[i], uint16s[i])
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
func (dac *AP235) Flush() {
	C.simtrig235(&dac.cfg)
}

// Clear soft resets the DAC, clearing the output but not configuration
// the error is always nil
func (dac *AP235) Clear(channel int) error {
	dac.cfg.opts._chan[C.int(channel)].DataReset = C.int(1)
	dac.sendCfgToBoard(channel)
	dac.cfg.opts._chan[C.int(channel)].DataReset = C.int(0)
	return nil
}

// Reset completely clears both data and configuration for a channel
// the error is always nil
func (dac *AP235) Reset(channel int) error {
	dac.cfg.opts._chan[C.int(channel)].FullReset = C.int(1)
	dac.sendCfgToBoard(channel)
	dac.cfg.opts._chan[C.int(channel)].FullReset = C.int(0)
	return nil
}

// Close the dac, freeing hardware.
func (dac *AP235) Close() error {
	errC := C.APClose(dac.cfg.nHandle)
	return enrich(errC, "APClose")
}
