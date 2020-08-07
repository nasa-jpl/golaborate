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
	"reflect"
	"strconv"
	"strings"
	"unsafe"
)

func init() {
	errCode := C.InitAPLib()
	if errCode != C.S_OK {
		panicS := fmt.Sprintf("initializing Acromag library failed with code %d", errCode)
		panic(panicS)
	}
}

// TODO: scatter_list might fuck me.  FGPA is holding some data about host memory
// and something about only 26 bits.
/// need to copy data into pcor_buf..?????????????

// OutputScale is the output scale of the DAC at power up or clear
type OutputScale int

// OutputRange is the output range of the DAC
type OutputRange int

// TriggerMode is a triggering mode
type TriggerMode int

// OperatingMode is a mode of operating the DAC for a given channel
type OperatingMode int

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
)
const (
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

	// from ap235.h
	idealZeroSB  = 0
	idealZeroBTC = 1
	idealSlope   = 2
	endpointLo   = 3
	endpointHi   = 4
	clipLo       = 5
	clipHi       = 6
	offset       = 0
	gain         = 1
)
const (
	// TriggerSoftware represents a software triggering mode
	TriggerSoftware TriggerMode = iota

	// TriggerTimer represents a timer (internally clocked waveform) trigger mode
	TriggerTimer

	// TriggerExternal represents a triggering mode which is externally clocked
	TriggerExternal

	// OperatingSingle is an operating mode corresponding to a single sample at a time
	// it is compatible with software triggering only.  It is compatible with simultaneous output.
	OperatingSingle OperatingMode = 0 // from ap235.h

	// OperatingWaveform is an operating mode corresponding to waveform output.
	// it is incompatible with the software triggering mode.
	OperatingWaveform OperatingMode = 4 // from ap235.h

	// MAXSAMPLES is the maximum number of samples in the buffer of a single
	// channel.  It is repeated from AP235.h to avoid an unnecessary CFFI call
	MAXSAMPLES = 4096

	// DMAXferSize is the (max) number of samples to send in one DMA transfer
	DMAXferSize = MAXSAMPLES / 2
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
	ErrTimerTooFast = errors.New("timer too fast: DAC cannot settle to < 1LSB before next value given.  Value accepted")

	// ErrIncompatibleOperatingTrigger is generated when the triggering and operating modes are incompatible
	ErrIncompatibleOperatingTrigger = errors.New("operating mode and trigger source are incompatible, software+single or (external|timer)+waveform are the only valid combinations.  Change accepted, state inconsistent")

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
		return -1, errors.New("invalid output range")
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

// RangeToMinMax converts a range string, <min,max> to floats.
// the input is assumed to be well formed; 0,0 is returned for badly formed inputs,
// or a panic occurs for inputs not containing a 0
func RangeToMinMax(rangeS string) (float64, float64) {
	// assume well-formed input,
	pieces := strings.Split(rangeS, ",")
	f1, _ := strconv.ParseFloat(pieces[0], 64)
	f2, _ := strconv.ParseFloat(pieces[1], 64)
	return f1, f2
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
		return -1, errors.New("triggering mode must be a member of {software, timer, external}")
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

// ValidateOperatingMode checks that an operating mode is valid
// s is a member of {'single', 'waveform'}
func ValidateOperatingMode(s string) (OperatingMode, error) {
	switch s {
	case "single":
		return OperatingSingle, nil
	case "waveform":
		return OperatingWaveform, nil
	default:
		return -1, errors.New("operating mode must be a member of {single, waveform}")
	}
}

// FormatOperatingMode formats the operating mode to either single or waveform.
func FormatOperatingMode(o OperatingMode) string {
	switch o {
	case OperatingSingle:
		return "single"
	case OperatingWaveform:
		return "waveform"
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
	cfg *C.struct_cblk235

	idealCode [8][7]float64

	// cursors hold the index into buffer
	// that corresponds to the current sample offset of each channel
	cursor [16]int

	// sample_count holds the number of samples in a supplied waveform
	// on a per-channel basis
	sampleCount [16]int

	// buffers are the sample queues for each channel, owned by C
	buffer [16][]uint16
	// cptrs holds the pointers in C to be used to free the buffers later
	cptr [16]*C.short

	cScatterInfo *[4]C.ulong

	playingBack bool
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

	cfgPtr := (*C.struct_cblk235)(C.malloc(C.sizeof_struct_cblk235))
	o.cfg = cfgPtr
	// confirmed by Kate Blanketship on Gophers slack that this
	// is a valid way to generate the pointer that C wants
	// see also: several ways to get the same address of the
	// data: https://play.golang.org/p/fpkOIT9B3BB

	o.cfg.pIdealCode = cMkCopyOfIdealData(idealCode)

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
	ptr := &o.cScatterInfo[0]
	errCode := C.Setup_board_corrected_buffer(o.cfg, &ptr)
	if errCode != 0 {
		return nil, errors.New("error reading calibration data from AP235")
	}
	// binitialize and bAP are set in Setup_board, ditto for rwcc235
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
	fmt.Println(Crng)
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
	opMode, _ := dac.GetOperatingMode(channel)
	dac.sendCfgToBoard(channel)
	if opMode == "waveform" {
		if (triggerMode != "external") && (triggerMode != "timer") {
			return ErrIncompatibleOperatingTrigger
		}
	}
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

// SetOperatingMode changes the operating mode of the DAC.
//
// Valid modes are 'single', 'waveform'.
//
// a non-nil error will be generated if the triggering mode
// for this channel is incomaptible.  The config change will
// still be made.
// err should be checked on the later of the two calls to
// SetOperatingMode and SetTriggerMode
func (dac *AP235) SetOperatingMode(channel int, mode string) error {
	o, err := ValidateOperatingMode(mode)
	if err != nil {
		return err
	}
	dac.cfg.opts._chan[C.int(channel)].OpMode = C.int(o)
	trigger, _ := dac.GetTriggerMode(channel)
	dac.sendCfgToBoard(channel)
	if mode == "waveform" {
		if (trigger != "external") && (trigger != "timer") {
			return ErrIncompatibleOperatingTrigger
		}
	}
	return nil
}

// GetOperatingMode retrieves whether the DAC is in single sample or waveform mode
func (dac *AP235) GetOperatingMode(channel int) (string, error) {
	modeC := dac.cfg.opts._chan[C.int(channel)].OpMode
	return FormatOperatingMode(OperatingMode(modeC)), nil
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
	C.cnfg235(dac.cfg, C.int(channel))
	return
}

// Output writes a voltage to a channel.
// the error is only non-nil if the value is out of range
func (dac *AP235) Output(channel int, voltage float64) error {
	// TODO: look into cd235 C function
	// this is a hack to improve code reuse, no need to allocate slices here
	vB := []float64{voltage}
	vU := []uint16{0}
	dac.calibrateData(channel, vB, vU)
	return dac.OutputDN16(channel, vU[0])
}

// OutputDN16 writes a value to the board in DN.
//
// the error is always nil
func (dac *AP235) OutputDN16(channel int, value uint16) error {
	// going to round trip, since we want to use the DAC in calibrated mode
	// convert value to a f64
	rng, _ := dac.GetRange(channel)
	min, max := RangeToMinMax(rng)
	step := (max - min) / 65535
	fV := []float64{min + step*float64(value)}

	// set FIFO configuration for this channel to 1 sample
	cCh := C.int(channel)
	dac.cfg.SampleCount[cCh] = 1
	ptr := &dac.cfg.pcor_buf[cCh][0]
	ptr2 := &dac.cfg.pcor_buf[cCh][1]
	dac.cfg.current_ptr[cCh] = ptr
	dac.cfg.head_ptr[cCh] = ptr
	dac.cfg.tail_ptr[cCh] = ptr2
	// dac.cfg.ideal_buf[cCh][0] = C.short(int16(value - 32768)) // OR with 0x8000 converts u16 to i16
	C.cd235(dac.cfg, C.int(channel), (*C.double)(&fV[0]))
	C.fifowro235(dac.cfg, cCh)
	// fmt.Println(value, dac.cfg.ideal_buf[cCh][0])
	return nil
}

// OutputMulti writes voltages to multiple output channels.
// the error is non-nil if any of these conditions occur:
//	1.  A blend of output modes (some simultaneous, some immediate)
//  2.  A command is out of range
//
// if an error is encountered in case 2, the output buffer of the DAC may be
// partially updated from proceeding valid commands.  No invalid values escape
// to the DAC output.
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

// OutputMultiDN16 is equivalent to OutputMulti, but with DNs instead of volts.
// see the docstring of OutputMulti for more information.
func (dac *AP235) OutputMultiDN16(channels []int, uint16s []uint16) error {
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
		err := dac.OutputDN16(channels[i], uint16s[i])
		if err != nil {
			return fmt.Errorf("channel %d DN %d: %w", channels[i], uint16s[i], err)
		}
	}
	if sim {
		dac.Flush()
	}
	return nil
}

// Flush writes any pending output values to the device
func (dac *AP235) Flush() {
	C.simtrig235(dac.cfg)
}

// StartWaveform starts waveform playback on all waveform channels
// the error is only non-nil if playback is already occuring
func (dac *AP235) StartWaveform() error {
	if dac.playingBack {
		return errors.New("AP235 is already playing back a waveform")
	}
	// first step is to prep DMA for each channel
	// and start the interrupt servicing thread
	// prepping DMA means
	fmt.Println("operating mode = ", dac.cfg.opts._chan[0].OpMode)
	fmt.Println("output update mode = ", dac.cfg.opts._chan[0].UpdateMode)
	fmt.Println("range mode = ", dac.cfg.opts._chan[0].Range)
	fmt.Println("power up voltage = ", dac.cfg.opts._chan[0].PowerUpVoltage)
	fmt.Println("data reset = ", dac.cfg.opts._chan[0].DataReset)
	fmt.Println("full reset = ", dac.cfg.opts._chan[0].FullReset)
	fmt.Println("trigger source = ", dac.cfg.opts._chan[0].TriggerSource)
	fmt.Println("timer divider = ", dac.cfg.TimerDivider)
	fmt.Println("interrupt source = ", dac.cfg.opts._chan[0].InterruptSource)
	C.rsts235(dac.cfg)
	fmt.Printf("Ch0 status = %02X\n", dac.cfg.ChStatus[0])
	C.start_waveform(dac.cfg)
	dac.playingBack = true
	go dac.serviceInterrupts()
	return nil
}

// StopWaveform stops playback on all channels.
// the error is non-nil only if playback is not occuring
func (dac *AP235) StopWaveform() error {
	if !dac.playingBack {
		return errors.New("AP235 is not playing back a waveform")
	}
	C.stop_waveform(dac.cfg)
	return nil
}

// need software reset?  drvr235.c, L475

// calibrateData converts a f64 value to uint16.  This is basically cd235
// len(buffer) shall == len(volts)
func (dac *AP235) calibrateData(channel int, volts []float64, buffer []uint16) {
	// see AP235 manual (PDF), page 68
	cCh := C.int(channel)
	rngS, _ := dac.GetRange(channel)    // err always nil
	rng, _ := ValidateOutputRange(rngS) // err always nil
	gainCoef := 1 + float64(dac.cfg.ogc235[cCh][rng][gain])/(65535*16)
	slopeCoef := float64(dac.cfg.pIdealCode[rng][idealSlope])
	offset := float64(dac.cfg.pIdealCode[rng][idealZeroBTC] + dac.cfg.pIdealCode[rng][offset]/16)
	gain := gainCoef * slopeCoef
	min := float64(dac.cfg.pIdealCode[rng][clipLo])
	max := float64(dac.cfg.pIdealCode[rng][clipHi])

	for i := 0; i < len(volts); i++ {
		in := volts[i]
		out := gain*in + offset // could optimize this by premuling gain and slope
		if out > max {
			out = max
		} else if out < min {
			out = min
		}
		// buffer[i] = int16(out) ^ 0x8000 // or with 0x8000 per the manual
		buffer[i] = uint16(out + 0x8000)
	}
}

// PopulateWaveform populates the waveform table for a given channel
// the error is only non-nil if the DAC is currently playing back a waveform
func (dac *AP235) PopulateWaveform(channel int, data []float64) error {
	// need to:
	// 1) convert f64 => uint16
	// 2) handle buffer, cursor, sample count
	//    sampleCount in Go, not SampleCount in C (which is <= 4096)
	// 3) free old buffers if this isn't the first time we're populating
	// 5) put the channel in FIFO_DMA mode
	// 4) do the first dma transfer
	// we do not start the background thread until waveform playback starts
	// since we only want to start the one thread, not one per channel.

	// create a buffer long enough to hold the waveform in uint16s
	if dac.playingBack {
		return errors.New("AP235 cannot change waveform table during playback")
	}

	err := dac.SetOperatingMode(channel, "waveform")
	if err != nil {
		return err // err is beneign, but force users to reconfigure DAC first
	}
	l := len(data)
	buf, cptr, err := cMkarrayU16(l)
	if err != nil {
		return err
	}
	if dac.cptr[channel] != nil {
		// free old buffer and replace
		C.aligned_free(unsafe.Pointer(dac.cptr[channel]))
	}
	dac.cptr[channel] = cptr

	// set the interrupt source for this channel (needed for DMA)
	dac.cfg.opts._chan[channel].InterruptSource = 1

	// now convert each value to a u16 and update the buffer
	dac.calibrateData(channel, data, buf) // "moves" data->buf
	dac.sampleCount[channel] = l
	dac.cursor[channel] = 0
	dac.buffer[channel] = buf
	fmt.Println(dac.buffer[channel][:100])
	C.set_DAC_sample_addresses(dac.cfg, C.int(channel))
	dac.doDMATransfer(channel)
	return nil
}

// serviceInterrupts should be run as a background goroutine;
// it advances the cursor into the data buffer for each channel
// and triggers new DMA writes to keep the DAC fed
func (dac *AP235) serviceInterrupts() {
	// the minimum recommended timer period is 0x136
	// which is (310 * 32 ns) = 9.9us
	// so this loop could happen as frequently as
	// 9.9us * 2048 samples = 20 ms
	// it's not all that hot after all.
	fmt.Println("starting to service interrupts")
	for {
		Cstatus := C.fetch_status(dac.cfg)

		fmt.Println("Cstatus =", Cstatus)
		status := uint(Cstatus)

		if status == 0 {
			return
		}
		// at least one channel requires updating
		for i := 0; i < 16; i++ { // i = channel index
			var mask uint = 1 << i
			if (mask & status) != 0 {
				dac.doDMATransfer(i)
			}
		}
		fmt.Println("refreshing interrupts")
		C.refresh_interrupt(dac.cfg, Cstatus)
	}
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
	C.Teardown_board_corrected_buffer(dac.cfg)
	errC := C.APClose(dac.cfg.nHandle)
	return enrich(errC, "APClose")
}

func (dac *AP235) doDMATransfer(channel int) {
	// TODO: check this thoroughly for off-by-one errors
	head := dac.cursor[channel]
	tailOffset := dac.sampleCount[channel] - dac.cursor[channel]
	if tailOffset > DMAXferSize {
		tailOffset = DMAXferSize
	}
	tail := head + tailOffset
	p1 := (*C.short)(unsafe.Pointer(&dac.buffer[channel][head]))
	p2 := (*C.short)(unsafe.Pointer(&dac.buffer[channel][tail]))
	fmt.Printf("doing DMA transfer %d %d %p %p\n", channel, tailOffset, p1, p2)
	C.do_DMA_transfer(dac.cfg, C.int(channel), C.uint(tailOffset), p1, p2)
}

// CMkarrayU16 allocates a []uint16 in C and returns a Go slice without copying
// as well as the pointer for freeing, and error if malloc failed.
func cMkarrayU16(size int) ([]uint16, *C.short, error) {
	cptr := C.MkDataArray(C.int(size))
	if cptr == nil {
		return nil, nil, fmt.Errorf("cMkarrayU16: cmalloc failed")
	}
	var slc []uint16
	hdr := (*reflect.SliceHeader)(unsafe.Pointer(&slc))
	hdr.Cap = size
	hdr.Len = size
	hdr.Data = uintptr(unsafe.Pointer(cptr))
	return slc, cptr, nil
}

// cMkCopyOfIdealData copies all values from idealCodes to a C owned
// array.  C.free must be called on it at a later date.
func cMkCopyOfIdealData(idealCodes [8][7]float64) *[8][7]C.double {
	cPtr := C.malloc(C.sizeof_double * 8 * 7)
	cArr := (*[8][7]C.double)(cPtr)
	for i := 0; i < 8; i++ {
		for j := 0; j < 7; j++ {
			cArr[i][j] = C.double(idealCodes[i][j])
		}
	}
	return cArr
}
