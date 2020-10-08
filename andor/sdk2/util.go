package sdk2

import (
	"errors"
	"fmt"

	"github.jpl.nasa.gov/bdube/golab/util"
)

// Enum behaves a bit like a C enum
type Enum map[string]int

var (
	// ErrBadEnumIndex is generated when an unknown enum index is used
	ErrBadEnumIndex = errors.New("index not found in enum")

	// ErrParameterNotSet is generated when a parameter is Gotten before it is set
	ErrParameterNotSet = errors.New("parameter not set and not queryable from SDK, set to learn in wrapper")

	// AcquisitionMode maps names to the values used by the SDK
	AcquisitionMode = Enum{
		"SingleScan":    1,
		"Accumulate":    2,
		"Kinetic":       3,
		"FastKinetic":   4,
		"RunUntilAbort": 5,
	}

	// ReadoutMode maps names to the values used by the SDK
	ReadoutMode = Enum{
		"FullVerticalBinning": 0,
		"MultiTrack":          1,
		"RandomTrack":         2,
		"SingleTrack":         3,
		"Image":               4,
	}

	// TriggerMode maps names to the values used by the SDK
	TriggerMode = Enum{
		"Internal":                 0,
		"External":                 1,
		"ExternalStart":            6,
		"External Exposure (Bulb)": 7,
		"External FVB EM":          9,
		"Software":                 10,
	}

	// FilterMode maps names to the values used by the SDK
	FilterMode = Enum{
		"No Filter":           0,
		"Median":              1,
		"Level Above":         2,
		"Interquartile Range": 3,
		"Noise Threshold":     4,
	}

	// ShutterMode maps names to the values used by the SDK
	ShutterMode = Enum{
		"Auto":  0,
		"Open":  1,
		"Close": 2,
	}

	// VerticalClockVoltage maps names to the values used by the SDK
	VerticalClockVoltage = Enum{
		"Normal": 0,
		"+1":     1,
		"+2":     2,
		"+3":     3,
		"+4":     4,
	}

	// EMGainMode maps names ot the values used by the SDK
	EMGainMode = Enum{
		"Default":  0,
		"Extended": 1,
		"Linear":   2,
		"Real":     3,
	}

	//ErrCodes is a map of error codes to their string values
	ErrCodes = map[DRVError]string{
		20001: "DRV_ERROR_CODES",
		20002: "DRV_SUCCESS",
		20003: "DRV_VXD_NOT_INSTALLED",
		20004: "DRV_ERROR_SCAN",
		20005: "DRV_ERROR_CHECKSUM",
		20006: "DRV_ERROR_FILELOAD",
		20007: "DRV_UNKNOWN_FUNCTION",
		20008: "DRV_ERROR_VXD_INIT",
		20009: "DRV_ERROR_ADDRESS",
		20010: "DRV_ERROR_PAGE_LOCK",
		20011: "DRV_ERROR_PAGE_UNLOCK",
		20012: "DRV_ERROR_BOARDTEST",
		20013: "DRV_ERROR_ACK",
		20014: "DRV_ERROR_UP_FIFO",
		20015: "DRV_ERROR_PATTERN",
		// no 20016
		20017: "DRV_ACQUISITION_ERRORS",
		20018: "DRV_ACQ_BUFFER",
		20019: "DRV_ACQ_DOWNFIFO_FULL",
		20020: "DRV_PROC_UNKNOWN_INSTRUCTION",
		20021: "DRV_ILLEGAL_OP_CODE",
		20022: "DRV_KINETIC_TIME_NOT_MET",
		20023: "DRV_ACCUM_TIME_NOT_MET",
		20024: "DRV_NO_NEW_DATA",
		// no 20025
		20026: "DRV_SPOOLERROR",
		// no 20027-20032
		20033: "DRV_TEMPERATURE_CODES",
		20034: "DRV_TEMPERATURE_OFF",
		20035: "DRV_TEMPERATURE_NOT_STABILIZED",
		20036: "DRV_TEMPERATURE_STABILIZED",
		20037: "DRV_TEMPERATURE_NOT_REACHED",
		20038: "DRV_TEMPERATURE_OUT_RANGE",
		20039: "DRV_TEMPERATURE_NOT_SUPPORTED",
		20040: "DRV_TEMPERATURE_DRIFT",
		// no 20041-20048
		20049: "DRV_GENERAL_ERRORS",
		20050: "DRV_INVALID_AUX",
		20051: "DRV_COF_NOTLOADED",
		20052: "DRV_FPGAPROG",
		20053: "DRV_FLEXERROR",
		20054: "DRV_GPIBERROR",
		// no 20055-20063
		20064: "DRV_DATATYPE",
		20065: "DRV_DRIVER_ERRORS",
		20066: "DRV_P1INVALID",
		20067: "DRV_P2INVALID",
		20068: "DRV_P3INVALID",
		20069: "DRV_P4INVALID",
		20070: "DRV_INIERROR",
		20071: "DRV_COFERROR",
		20072: "DRV_ACQUIRING",
		20073: "DRV_IDLE",
		20074: "DRV_TEMPCYCLE",
		20075: "DRV_NOT_INITIALIZED",
		20076: "DRV_P5INVALID",
		20077: "DRV_P6INVALID",
		20078: "DRV_INVALID_MODE",
		20079: "DRV_INVALID_FILTER",
		20080: "DRV_I2CERRORS",
		20081: "DRV_DRV_I2CDEVNOTFOUND",
		20082: "DRV_I2CTIMEOUT",
		20083: "DRV_P7INVALID",
		// no 20084-20088
		20089: "DRV_USBERROR",
		20090: "DRV_IOCERROR",
		20091: "DRV_NOT_SUPPORTED",
		// no 20092
		20093: "DRV_USB_INTERRUPT_ENDPOINT_ERROR",
		20094: "DRV_RANDOM_TRACK_ERROR",
		20095: "DRV_INVALID_TRIGGER_MODE",
		20096: "DRV_LOAD_FIRMWARE_ERROR",
		20097: "DRV_DIVIDE_BY_ZERO_ERROR",
		20098: "DRV_INVALID_RINGEXPOSURES",
		20099: "DRV_BINNING_ERROR",
		// no 20100-20989 -- sort of. 100s come later in the manual
		20990: "DRV_ERROR_NOCAMERA",
		20991: "DRV_NOT_SUPPORTED",
		20992: "DRV_NOT_AVAILABLE",
		// no 20993-20114
		20115: "DRV_ERROR_MAP",
		20116: "DRV_ERROR_UNMAP",
		20117: "DRV_ERROR_MDL",
		20118: "DRV_ERROR_UNMDL",
		20119: "DRV_ERROR_BUFFSIZE",
		// no 20120
		20121: "DRV_ERROR_NOHANDLE",
		// no 20122-20129
		20130: "DRV_GATING_NOT_AVAILABLE",
		20131: "DRV_FPGA_VOLTAGE_ERROR",
		20100: "DRV_INVALID_AMPLIFIER",
	}

	// BeneignErrorCodes is sequence of error codes which mean
	// the status is normal
	BeneignErrorCodes = []uint{
		20002, // success
		20073, // idle
	}
)

// HardwareVersion is a struct holding hardware versions
type HardwareVersion struct {
	// PCB version
	PCB uint

	// Decode Flex 10K file version
	Decode uint

	dummy1 uint
	dummy2 uint

	// CameraFirmwareVersion Version # of camera firmware
	CameraFirmwareVersion uint

	// CameraFirmwareBuild
	CameraFirmwareBuild uint
}

// SoftwareVersion is a struct holding software versions
type SoftwareVersion struct {
	// EPROM version
	EPROM uint

	// COF version
	COF uint

	// DriverRevision
	DriverRevision uint

	// DriverVersion
	DriverVersion uint

	// DLLRevision
	DLLRevision uint

	// DLLVersion
	DLLVersion uint
}

// AcquisitionTimings holds various acquisition timing parameters
type AcquisitionTimings struct {
	// Exposure is the exposure time in seconds
	Exposure float64

	// Accumulation is the charge accumulation cycle time in seconds
	Accumulation float64

	// Kinetic is the kinetic cycle time in seconds
	Kinetic float64
}

// Status is a camera status.  They are also error codes
type Status uint

const (
	// WRAPVER is the wrapper version around andor SDK v2
	WRAPVER = 4

	// StatusIdle is IDLE waiting on instructions
	StatusIdle Status = 20073

	// StatusTempCycle executing temperature cycle
	StatusTempCycle Status = 20074

	// StatusAcquiring Acquisition in progress
	StatusAcquiring Status = 20072

	// StatusAccumTimeNotMet unable to meet accumulate cycle time
	StatusAccumTimeNotMet Status = 20023

	// StatusKineticTimeNotMet unable to meet kinetic cycle time
	StatusKineticTimeNotMet Status = 20022

	// StatusDriverError unable to communicate with card
	StatusDriverError Status = 20013

	// StatusAcqBufferOverflow buffer overflow at ISA slot
	StatusAcqBufferOverflow Status = 20018

	// StatusSpoolError buffer overflow at spool buffer
	StatusSpoolError Status = 20026
)

// DRVError represents a driver error and has nice formatting
type DRVError uint

func (e DRVError) Error() string {
	if s, ok := ErrCodes[e]; ok {
		return fmt.Sprintf("%d - %s", e, s)
	}
	return fmt.Sprintf("%v - UNKNOWN_ERROR_CODE", e)
}

// Error returns nil if the error code is beneign, otherwise returns
// an object which prints the error code and string value
func Error(code uint) error {
	if util.UintSliceContains(BeneignErrorCodes, code) {
		return nil
	}
	return DRVError(code)
}

// BeneignThermal returns true if the status code is a beneign thermal one
func BeneignThermal(err error) bool {
	if err == nil {
		return true
	}
	if drv, ok := err.(DRVError); ok {
		if (20033 < drv) && (20041 > drv) {
			return true
		}
		return false
	}
	return false
}
