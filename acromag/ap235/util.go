package ap235

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

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
	OperatingWaveform OperatingMode = 2 // from ap235.h

	// MAXSAMPLES is the maximum number of samples in the buffer of a single
	// channel.  It is repeated from AP235.h to avoid an unnecessary CFFI call
	MAXSAMPLES = 4096

	// MaxXferSize is the (max) number of samples to send in one DMA transfer
	MaxXferSize = MAXSAMPLES / 2
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

	// ErrIncompatibleWaveform is generated when a single output command is sent
	// to a channel configured for waveform playback
	ErrIncompatibleWaveform = errors.New("single output commands are not possible when channel is configured for waveform playback")

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

// ChannelStatus contains the status of a given DAC channel
type ChannelStatus struct {
	// Channel is the associated channel
	Channel int

	// FIFOEmpty - if true, the FIFO queue is empty
	FIFOEmpty bool

	// FIFOHalfFull - if true, the FIFO queue is half full
	FIFOHalfFull bool

	// FIFOFull - if true, the FIFO queue is full
	FIFOFull bool

	// FIFOUnderflow - if true, the FIFO queue was emptied while draining
	FIFOUnderflow bool

	// BurstSingleComplete - if true, the FIFO queue was emptied while draining for a single burst playback
	BurstSingleComplete bool

	// Busy - if true, the channel's DAC is busy
	Busy bool
}
