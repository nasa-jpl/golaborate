/*Package sdk2 exposes control of Andor cameras in Go via their SDK, v2

This package provides much of the total C/C++ API for working with Andor cameras
but it is not totally exhaustive.  It was created to enable use of the NEO sCMOS
and iXon EMCCD cameras at JPL in a scientific / instruments context, and thus
does not fully support any of the onboard processing features, except the Set
functions used to disable them.  We do not support multiple cameras on one PC.
We do not in this package provide any recipes; this is purely a driver interface
that has some gussied up in and output types.  We mostly duplicate the API from
the C/C++ shared library, with the exception of a few grammatical cleanups.

Users are encouraged to write packages that build on this driver to build more
complex functionality.  A basic recipe for the library's usage during a session
is as follows (duplicated from the SDK manual):

 // initialize the camera
 cam := andor.Camera{...} // you need to provide some data here
 cam.Initialize()
 cam.GetDetector()
 cam.GetHardwareVersion()
 cam.GetSoftwareVersion()
 cam.GetNumberVSSpeeds()
 cam.GetVSSpeed()
 cam.GetNumberHSSpeeds()
 cam.GetHSSpeed()

 // achieve thermal stability
 cam.GetTemperatureRange()
 cam.SetTemperature()
 cam.CoolerActive(true)
 for tempNotInRange := true; tempNotInRange {
	 cam.GetTemperature()
 }

 // program exposure
 cam.SetAcquisitionMode()
 cam.SetReadoutMode()
 cam.SetShutter()
 cam.SetExposureTime()
 cam.SetTriggerMode()
 cam.SetAccumulationCycleTime()
 cam.SetNumberAccumulations()
 cam.SetNumberKinetics()
 cam.SetKineticCycleTime()
 cam.GetAcquisitionTimings()
 cam.SetHSSpeed()
 cam.SetVSSpeed()

 // take frames
 cam.StartAcquisition()
 cam.GetStatus() // TODO: replace with waitfor
 cam.GetAcquiredData()

 // shutdown
 // Note you will want to use the temperature control loop on the camera
 // to bring it back to ambient temperature at an acceptable slew rate of < 10C/min
 // and not just perform a hard shutdown to avoid damaging the camera.
 cam.ShutDown()

We do not explicitly write the parameters here, or handle returns or errors.
This is obviously long and granular and may motivate your writing an extension
library.

*/
package sdk2

/*
#cgo CFLAGS: -I/usr/local
#cgo LDFLAGS: -L/usr/local/lib -landor
#include <stdlib.h>
#include <atmcdLXd.h>
*/
import "C"
import (
	"errors"
	"fmt"
	"strconv"
	"time"
	"unsafe"

	"github.jpl.nasa.gov/HCIT/go-hcit/generichttp/camera"
	"github.jpl.nasa.gov/HCIT/go-hcit/util"
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

// Camera represents an Andor camera
type Camera struct {
	// nil values cause get functions to bounce
	// tempSetpoint is the temperature the TEC is set to
	tempSetpoint *string

	// exposureTime is the length of time the exposure is set to
	exposureTime *time.Duration

	// fanOn holds the status of the fan
	fanOn *bool

	// aoi holds the AOI parameters
	aoi *camera.AOI

	// binning holds the binning parameters
	binning *camera.Binning
}

var (
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

/* this block contains functions which deal with camera initialization

 */

// Initialize initializes the camera connection with the driver library
func Initialize(iniPath string) error {
	cstr := C.CString(iniPath)
	defer C.free(unsafe.Pointer(cstr))
	errCode := uint(C.Initialize(cstr))
	return Error(errCode)
}

// ShutDown shuts down the camera
// this doesn't mimic the SDK 1:1, but the error can only be DRV_SUCCESS
// so we spare the user dealing with errors
func (c *Camera) ShutDown() {
	C.ShutDown()
}

// GetDetector gets the detector
func (c *Camera) GetDetector() (int, int, error) { // need another return type
	var x, y C.int
	errCode := uint(C.GetDetector(&x, &y))
	return int(x), int(y), Error(errCode)
}

// GetHardwareVersion gets the hardware version string from the camera
func (c *Camera) GetHardwareVersion() (HardwareVersion, error) { // need another return type
	var pcb, decode, dummy1, dummy2, camfw, cambuild C.uint
	errCode := uint(C.GetHardwareVersion(&pcb, &decode, &dummy1, &dummy2, &camfw, &cambuild))
	s := HardwareVersion{
		PCB:                   uint(pcb),
		Decode:                uint(decode),
		dummy1:                uint(dummy1),
		dummy2:                uint(dummy2),
		CameraFirmwareVersion: uint(camfw),
		CameraFirmwareBuild:   uint(cambuild)}
	return s, Error(errCode)
}

// GetSoftwareVersion gets the software version from the caemra
func (c *Camera) GetSoftwareVersion() (SoftwareVersion, error) {
	var eprom, coffile, vxdrev, vxdver, dllrev, dllver C.uint
	errCode := uint(C.GetSoftwareVersion(&eprom, &coffile, &vxdrev, &vxdver, &dllrev, &dllver))
	s := SoftwareVersion{
		EPROM:          uint(eprom),
		COF:            uint(coffile),
		DriverRevision: uint(vxdrev),
		DriverVersion:  uint(vxdver),
		DLLRevision:    uint(dllrev),
		DLLVersion:     uint(dllver),
	}
	return s, Error(errCode)
}

// GetNumberVSSpeeds gets the number of vertical shift register speeds available
func (c *Camera) GetNumberVSSpeeds() (int, error) { // need another return type
	var speeds C.int
	errCode := uint(C.GetNumberVSSpeeds(&speeds))
	return int(speeds), Error(errCode)
}

// GetVSSpeed gets the vertical shift register speed
func (c *Camera) GetVSSpeed(idx int) (float64, error) { // need another return type
	var f C.float
	errCode := uint(C.GetVSSpeed(C.int(idx), &f))
	return float64(f), Error(errCode)
}

// GetFastestRecommendedVSSpeed gets the fastest vertical shift register speed
// that does not require changing the vertical clock voltage.  It returns
// the fastest vertical clock speed's intcode and the actual speed in microseconds
func (c *Camera) GetFastestRecommendedVSSpeed() (int, float64, error) {
	var idx C.int
	var speed C.float
	errCode := uint(C.GetFastestRecommendedVSSpeed(&idx, &speed))
	return int(idx), float64(speed), Error(errCode)
}

// SetVSSpeed sets the vertical shift register speed
func (c *Camera) SetVSSpeed(idx int) error { // need another argument type
	errCode := uint(C.SetVSSpeed(C.int(idx)))
	return Error(errCode)
}

// SetVSAmplitude sets the vertical shift register voltage
func (c *Camera) SetVSAmplitude(vcv string) error {
	i, ok := VerticalClockVoltage[vcv]
	if !ok {
		return ErrBadEnumIndex
	}
	cint := C.int(i)
	errCode := uint(C.SetVSAmplitude(cint))
	return Error(errCode)
}

// GetNumberHSSpeeds gets the number of horizontal shift register speeds available
func (c *Camera) GetNumberHSSpeeds(ch int) (int, error) { // need another return type
	// var emint int
	// if emMode {
	// 	emint = 0
	// } else {
	// 	emint = 1
	// }
	// commented out, mode 1 never applies?
	cch := C.int(ch)
	ctyp := C.int(0)
	var ret C.int

	errCode := uint(C.GetNumberHSSpeeds(cch, ctyp, &ret))
	return int(ret), Error(errCode)
}

// GetHSSpeed gets the horizontal shift speed
func (c *Camera) GetHSSpeed(ch int, idx int) (float64, error) { // need another return type
	// var emint int
	// if emMode {
	// 	emint = 0
	// } else {
	// 	emint = 1
	// }
	// commented out, mode 1 never applies?
	cch := C.int(ch)
	ctyp := C.int(0)
	cidx := C.int(idx)
	var ret C.float

	errCode := uint(C.GetHSSpeed(cch, ctyp, cidx, &ret))
	return float64(ret), Error(errCode)
}

// SetHSSpeed sets the horizontal shift speed
func (c *Camera) SetHSSpeed(idx int) error { // need another argument type
	// var emint int
	// if emMode {
	// 	emint = 0
	// } else {
	// 	emint = 1
	// }
	// commented out -- seems we are always using 0 / are not in NIR (for 1)
	ctyp := C.int(0)
	cidx := C.int(idx)

	errCode := uint(C.SetHSSpeed(ctyp, cidx))
	return Error(errCode)
}

/* the above deals with camera initialization, the below deals with temperature regulation.

 */

// SetCooling toggles the cooler on (true) or off (false)
// NOTE:
// 1. When the temperature control is switched off, the temperature of the
//    sensor is gradually raised to 0C to ensure no thermal stresses are
//    set up in the sensor.  Classic & ICCD only.
// 2. When closing down the program via ShutDown, you must ensure that the
//    temperature of the detector is above -20C, otherwise calling ShutDown
//    while the detector is still cooled will cause the temperature to rise
//    faster than certified.
func (c *Camera) SetCooling(b bool) error {
	var cerr C.uint
	if b {
		cerr = C.CoolerON()
	} else {
		cerr = C.CoolerOFF()
	}
	return Error(uint(cerr))
}

// GetCooling gets if the cooler is currently engaged
func (c *Camera) GetCooling() (bool, error) {
	var ret C.int
	errCode := uint(C.IsCoolerOn(&ret))
	return int(ret) == 1, Error(errCode)
}

// GetTemperatureRange gets the valid range of temperatures
// in which the detector can be cooled
// returns (min, max, error)
func (c *Camera) GetTemperatureRange() (int, int, error) { // need another return type
	var min, max C.int
	errCode := uint(C.GetTemperatureRange(&min, &max))
	return int(min), int(max), Error(errCode)
}

// GetTemperature gets the current temperature in degrees celcius.  The real type is int, but we use float for SD3 compatibility
func (c *Camera) GetTemperature() (float64, error) {
	var temp C.int
	errCode := uint(C.GetTemperature(&temp))
	return float64(int(temp)), Error(errCode) // TODO: there may be a potential optimization float64 directly
}

// SetTemperatureSetpoint assigns a setpoint to the camera's TEC
func (c *Camera) SetTemperatureSetpoint(t string) error {
	tI, err := strconv.Atoi(t)
	if err != nil {
		return err
	}
	errCode := uint(C.SetTemperature(C.int(tI)))
	err = Error(errCode)
	if err == nil {
		c.tempSetpoint = &t
	}
	return err
}

// GetTemperatureSetpoint returns the setpoint of the camera's TEC
func (c *Camera) GetTemperatureSetpoint() (string, error) {
	if c.tempSetpoint == nil {
		return "", ErrParameterNotSet
	}
	return *c.tempSetpoint, nil
}

// GetTemperatureSetpoints returns an array of the MIN and MAX temperatures.
// Any integer intermediate is valid
func (c *Camera) GetTemperatureSetpoints() ([]string, error) {
	min, max, err := c.GetTemperatureRange()
	if err != nil {
		return []string{}, err
	}
	minS := strconv.Itoa(min) // if one errors, assume the other would too,
	maxS := strconv.Itoa(max) // skip one error check
	return []string{minS, maxS}, nil
}

// GetTemperatureStatus gets the status of the TEC subsystem on the camera
func (c *Camera) GetTemperatureStatus() (string, error) {
	// this is pasted from the GetTemperature function with minor modification
	var temp C.int
	errCode := uint(C.GetTemperature(&temp))
	return ErrCodes[DRVError(errCode)], Error(errCode)
}

// SetFan allows the fan to be turned on or off.
// this is not a 1:1 mimic of SDk2, since it is binary
// on (HIGH) or off (OFF)
func (c *Camera) SetFan(on bool) error {
	var in C.int
	if on {
		in = C.int(0)
	} else {
		in = C.int(2)
	}
	errCode := uint(C.SetFanMode(in))
	err := Error(errCode)
	if err == nil {
		c.fanOn = &on
	}
	return err
}

// GetFan returns if the fan is turned on or not, as commanded
// the SDK may override and make the return not a true mimic of reality
// even if the value is false.  A true value should always mimic reality.
func (c *Camera) GetFan() (bool, error) {
	if c.fanOn == nil {
		return false, ErrParameterNotSet
	}
	return *c.fanOn, nil
}

/* the above deals with thermal management, the below deals with acquisition programming

 */

// SetAcquisitionMode sets the acquisition mode of the camera
func (c *Camera) SetAcquisitionMode(am string) error {
	i, ok := AcquisitionMode[am]
	if !ok {
		return ErrBadEnumIndex
	}
	errCode := uint(C.SetAcquisitionMode(C.int(i)))
	return Error(errCode)
}

// SetReadoutMode sets the readout mode of the camera.  We rename this from SetReadMode in the actual driver
func (c *Camera) SetReadoutMode(rm string) error {
	i, ok := ReadoutMode[rm]
	if !ok {
		return ErrBadEnumIndex
	}
	errCode := uint(C.SetReadMode(C.int(i)))
	return Error(errCode)
}

// SetShutter sets the shutter parameters of the camera.
// ttlHi sends output TTL high signal to open shutter, else sends TTL low signal
func (c *Camera) SetShutter(ttlHi bool, mode string, opening, closing time.Duration) error {
	i, ok := ShutterMode[mode]
	if !ok {
		return ErrBadEnumIndex
	}
	ot := opening.Milliseconds()
	ct := closing.Milliseconds()
	ttl := 0
	if ttlHi {
		ttl = 1
	}
	errCode := uint(C.SetShutter(C.int(ttl), C.int(i), C.int(ot), C.int(ct)))
	return Error(errCode)
}

// SetExposureTime sets the exposure time of the camera in seconds
func (c *Camera) SetExposureTime(t time.Duration) error {
	tS := t.Seconds()
	errCode := uint(C.SetExposureTime(C.float(tS)))
	err := Error(errCode)
	if err == nil {
		c.exposureTime = &t
	}
	return err
}

// GetExposureTime returns the current exposure time
func (c *Camera) GetExposureTime() (time.Duration, error) {
	if c.exposureTime == nil {
		return 0, ErrParameterNotSet
	}
	return *c.exposureTime, nil
}

// SetTriggerMode sets the trigger mode of the camera
func (c *Camera) SetTriggerMode(tm string) error {
	i, ok := TriggerMode[tm]
	if !ok {
		return ErrBadEnumIndex
	}
	errCode := uint(C.SetTriggerMode(C.int(i)))
	return Error(errCode)
}

// SetAccumulationCycleTime sets the accumulation cycle time of the camera in seconds
func (c *Camera) SetAccumulationCycleTime(t float64) error {
	errCode := uint(C.SetAccumulationCycleTime(C.float(t)))
	return Error(errCode)
}

// SetNumberAccumulations sets the number of accumulaions
func (c *Camera) SetNumberAccumulations(i uint) error {
	errCode := uint(C.SetNumberAccumulations(C.int(i)))
	return Error(errCode)
}

// SetNumberKinetics sets the number of kinetics
func (c *Camera) SetNumberKinetics(i uint) error {
	errCode := uint(C.SetNumberKinetics(C.int(i)))
	return Error(errCode)
}

// SetKineticCycleTime sets the kinetic cycle time
func (c *Camera) SetKineticCycleTime(t float64) error {
	errCode := uint(C.SetKineticCycleTime(C.float(t)))
	return Error(errCode)
}

// GetAcquisitionTimings gets the acquisition timings
func (c *Camera) GetAcquisitionTimings() (AcquisitionTimings, error) {
	var exp, acc, kin C.float
	errCode := uint(C.GetAcquisitionTimings(&exp, &acc, &kin))
	at := AcquisitionTimings{}
	at.Exposure = float64(exp)
	at.Accumulation = float64(acc)
	at.Kinetic = float64(kin)
	return at, Error(errCode)
}

// StartAcquisition starts the camera acquiring charge for an image
func (c *Camera) StartAcquisition() error {
	errCode := uint(C.StartAcquisition())
	return Error(errCode)
}

// GetStatus gets the status while the camera is acquiring data
func (c *Camera) GetStatus() (Status, error) {
	var stat C.int
	errCode := uint(C.GetStatus(&stat))
	return Status(uint(stat)), Error(errCode)
}

// GetAcquiredData gets the acquired data / frame
//
// Implementing a 32-bit function is left for the future
func (c *Camera) GetAcquiredData() ([]int32, error) {
	elements := 1024 * 1024
	buf := make([]int32, elements)
	ptr := (*C.at_32)(unsafe.Pointer(&buf[0]))
	errCode := uint(C.GetAcquiredData(ptr, C.uint(1024*1024)))
	return buf, Error(errCode)
}

// AbortAcquisition aborts the current acquisition if one is active
func (c *Camera) AbortAcquisition() error {
	errCode := uint(C.AbortAcquisition())
	return Error(errCode)
}

// WaitForAcquisition sleeps while waiting for the acquisition completed signal
// from the SDK
func (c *Camera) WaitForAcquisition(t time.Duration) error {
	i64 := t.Milliseconds()
	errCode := uint(C.WaitForAcquisitionTimeOut(C.int(i64)))
	return Error(errCode)
}

// GetBitDepth gets the number of bits of dynamic range for a given AD channel
func (c *Camera) GetBitDepth(ch uint) (uint, error) {
	var depth C.int
	errCode := uint(C.GetBitDepth(C.int(ch), &depth))
	return uint(depth), Error(errCode)
}

// GetNumberADChannels returns the number of discrete A/D channels available
func (c *Camera) GetNumberADChannels() (int, error) {
	var chans C.int
	errCode := uint(C.GetNumberADChannels(&chans))
	return int(chans), Error(errCode)
}

// SetADChannel sets the AD channel to use until it is changed again or the
// camera is powered off
func (c *Camera) SetADChannel(ch int) error {
	errCode := uint(C.SetADChannel(C.int(ch)))
	return Error(errCode)
}

// GetMaximumExposure gets the maximum exposure time supported in the current
// configuration in seconds
func (c *Camera) GetMaximumExposure() (float64, error) {
	var f C.float
	errCode := uint(C.GetMaximumExposure(&f))
	return float64(f), Error(errCode)
}

// SetImage wraps the SDK exactly and controls AoI and binning
func (c *Camera) SetImage(hbin, vbin, hstart, hend, vstart, vend int) error {
	errCode := uint(C.SetImage(C.int(hbin), C.int(vbin), C.int(hstart), C.int(hend), C.int(vstart), C.int(vend)))
	err := Error(errCode)
	if err == nil {
		c.bin = &camera.Binning{H: hbin, V: vbin}
		c.aoi = &camera.AOI{Left: hstart, Top: vstart, Width: hend - hstart, Height: vend - vstart}
	}
	return err
}

// GetAOI returns the current AOI in use by the camera
func (c *Camera) GetAOI() (camera.AOI, error) {
	if c.aoi == nil {
		return camera.AOI{}, ErrParameterNotSet
	}
	return *c.aoi, nil
}

// GetBinning returns the current binning used by the camera
func (c *Camera) GetBinning() (camera.Binning, error) {
	if c.bin == nil {
		return camera.Binning{}, ErrParameterNotSet
	}
	return *c.bin, nil
}

// SetBinning sets the binning used by the camera
func (c *Camera) SetBinning(b camera.Binning) error {
	aoi, err := c.GetAOI() // trigger error if we have no knowledge
	if err != nil {
		return err
	}
	return c.SetImage(b.H, b.V, aoi.Left, aoi.Right(), aoi.Top, aoi.Bottom())
}

// SetAOI sets the AoI used by the camera
func (c *Camera) SetAOI(a AOI) error {
	bin, err := c.GetBinning() // trigger error if we have no knowledge
	if err != nil {
		return err
	}
	return c.SetImage(bin.H, bin.V, a.Left, a.Right(), a.Top, a.Bottom())
}

// GetMaximumBinning returns the maximum binning factor usable.
// if horizontal is true, the returned value is for the horizontal dimension.
// if horizontal is false, the returned value is for the vertical dimension.
func (c *Camera) GetMaximumBinning(rm string, horizontal bool) (int, error) {
	i, ok := ReadoutMode[rm]
	if !ok {
		return 0, ErrBadEnumIndex
	}
	var maxbin C.int
	horz := 1
	if horizontal {
		horz = 0
	}
	errCode := uint(C.GetMaximumBinning(C.int(i), C.int(horz), &maxbin))
	return int(maxbin), Error(errCode)
}

/* the previous section deals with acquisition, the below deals with processing.

 */

// FilterSetMode sets the filtering mode of the camera
func (c *Camera) FilterSetMode(fm string) error {
	i, ok := FilterMode[fm]
	if !ok {
		return ErrBadEnumIndex
	}
	errCode := uint(C.Filter_SetMode(C.uint(i)))
	return Error(errCode)
}

// SetBaselineClamp toggles the baseline clamp feature of the camera on (true) or off (false)
func (c *Camera) SetBaselineClamp(b bool) error {
	on := 0
	if b {
		on = 1
	}
	errCode := uint(C.SetBaselineClamp(C.int(on)))
	return Error(errCode)
}

/* the previous section deals with processing, the below deals with EMCCD features.

 */

// GetEMCCDGain gets the current EMCCD gain
func (c *Camera) GetEMCCDGain() (int, error) {
	var mult C.int
	errCode := uint(C.GetEMCCDGain(&mult))
	return int(mult), Error(errCode)
}

// SetEMCCDGain sets the EMCCD gain.  The precise behavior depends on the current
// gain mode, see SetEMGainMode, GetEMGainMode
func (c *Camera) SetEMCCDGain(fctr int) error {
	errCode := uint(C.SetEMCCDGain(C.int(fctr)))
	return Error(errCode)
}

// GetEMGainRange gets the min and max EMCCD gain settings for the current gain
// mode and temperature of the sensor
func (c *Camera) GetEMGainRange() (int, int, error) {
	var low, high C.int
	errCode := uint(C.GetEMGainRange(&low, &high))
	return int(low), int(high), Error(errCode)
}

// SetEMGainMode sets the current EMCCD gain mode
func (c *Camera) SetEMGainMode(gm string) error {
	i, ok := EMGainMode[gm]
	if !ok {
		return ErrBadEnumIndex
	}
	errCode := uint(C.SetEMGainMode(C.int(i)))
	return Error(errCode)
}

// SetEMAdvanced allows setting of the EM gain setting to values higher than
// 300x.  Using this setting with more than 10s of photons per pixel per readout
// will lead to advanced ageing of the sensor.
func (c *Camera) SetEMAdvanced(b bool) error {
	enabled := 0
	if b {
		enabled = 1
	}
	errCode := uint(C.SetEMAdvanced(C.int(enabled)))
	return Error(errCode)
}

// GetFrameSize gets the W, H of a frame as recorded in the strided buffer
func (c *Camera) GetFrameSize() (int, int, error) {
	return 1024, 1024, nil // TODO: actual impl
}

// GetFrame returns a frame from the camera as a strided buffer
func (c *Camera) GetFrame() ([]uint16, error) {
	err := c.StartAcquisition()
	if err != nil {
		return []uint16{}, err
	}
	tExp, err := c.GetExposureTime()
	if err != nil {
		return []uint16{}, err
	}
	err = c.WaitForAcquisition(tExp + time.Second)
	if err != nil {
		return []uint16{}, err
	}
	buf, err := c.GetAcquiredData()
	b2 := make([]uint16, len(buf))
	l := len(buf)
	for idx := 0; idx < l; idx++ {
		b2[idx] = uint16(buf[idx])
	}
	return b2, nil
}

// Burst takes a chunk of pictures and returns them as one contiguous buffer
func (c *Camera) Burst(frames int, fps float64) ([]uint16, error) {
	return []uint16{}, fmt.Errorf("not implemented")
}
