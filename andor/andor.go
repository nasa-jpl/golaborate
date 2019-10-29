/*Package andor exposes control of Andor cameras in Go

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
package andor

/*
#cgo CFLAGS: -I../DyLib
#cgo LDFLAGS: -L. -lkeyboard
#include <ATMCD32D.H>
*/
import "C"
import (
	"fmt"
	"time"
)

// AcquisitionMode represents a mode of acquisition to the camera.
type AcquisitionMode uint

const (
	// AcquisitionSingleScan is the single-scan acq. mode
	AcquisitionSingleScan AcquisitionMode = iota

	// AcquisitionAccumulate is a continuous acquisition mode
	AcquisitionAccumulate

	// AcquisitionKinetic is Andor's Kinetic acq. mode
	AcquisitionKinetic

	// AcquisitionFastKinetic is Andor's fast kinetic acq. mode
	AcquisitionFastKinetic

	// AcquisitionRunUntilAbort acquires until acquisition is aborted
	AcquisitionRunUntilAbort
)

// ReadoutMode represents a readout mode of the camera.
type ReadoutMode uint

const (
	// ReadoutFullVerticalBinning reads out as if the sensor were a line array
	ReadoutFullVerticalBinning ReadoutMode = iota

	// ReadoutMultiTrack is like an array of ReadoutSingleTrack
	ReadoutMultiTrack

	// ReadoutRandomTrack is like MultiTrack, but the camera sets the positions itself
	ReadoutRandomTrack

	// ReadoutSingleTrack is like FulLVerticalBinning, but for a certain
	// row index and track height
	ReadoutSingleTrack

	// ReadoutImage is the mode you probably want to operate your camera in
	ReadoutImage
)

// TriggerMode represents a mode of triggering the camera
type TriggerMode uint

const (
	// TriggerInternal uses internal triggering
	TriggerInternal TriggerMode = iota

	// TriggerExternal uses external triggering
	TriggerExternal

	_
	_
	_
	_
	// TriggerExternalStart uses rising edges from an external trigger and falling edges from an internal one
	TriggerExternalStart

	// TriggerExternalExposure is used for bulb exposure operation
	TriggerExternalExposure

	_

	// TriggerExternalFVBEM is a special mode we don't use
	TriggerExternalFVBEM

	// TriggerSoftware uses pure software triggering
	TriggerSoftware
)

// FilterMode represents a mode of filtering the data
type FilterMode uint

const (
	// FilterNoFilter defeats filtering
	FilterNoFilter FilterMode = iota

	// FilterMedian uses median filtering
	FilterMedian

	// FilterLevelAbove uses a basic threshold filter
	FilterLevelAbove

	// FilterInterquartileRange uses a sophistocated IQ range filter
	FilterInterquartileRange

	// FilterNoiseThreshold uses a filter referenced to the noise level
	FilterNoiseThreshold
)

// VerticalClockVoltage represents the a discrete voltage level determined by
// Andor for the vertical shift register
type VerticalClockVoltage uint

const (
	// VerticalClockNormal is the normal / base vertical shift register voltage
	VerticalClockNormal VerticalClockVoltage = iota

	//VerticalClockPlusOne increases the voltage by one step
	VerticalClockPlusOne

	//VerticalClockPlusTwo increases the voltage by two steps
	VerticalClockPlusTwo

	//VerticalClockPlusThree increases the voltage by three steps
	VerticalClockPlusThree

	//VerticalClockPlusFour increases the voltage by four steps
	VerticalClockPlusFour
)

// Camera represents an Andor camera
type Camera struct{}

var (
	//ErrCodes is a map of error codes to their string values
	ErrCodes = map[uint]string{
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
		20002,
	}
)

// DRVError represents a driver error and has nice formatting
type DRVError uint

func (e DRVError) Error() string {
	if s, ok := ErrCodes[e]; ok {
		return fmt.Sprintf("%v - %s", e, s)
	}
	return fmt.Sprintf("%v - UNKNOWN_ERROR_CODE", e)
}

// uintSliceContains returns true if value is in slice, otherwise false
func uintSliceContains(slice []uint, value uint) bool {
	ret := false
	for _, cmpV := range slice {
		if value == cmpV {
			ret = true
		}
	}
	return ret
}

// Error returns nil if the error code is beneign, otherwise returns
// an object which prints the error code and string value
func Error(code uint) error {
	if uintSliceContains(BeneignErrorCodes, code) {
		return nil
	}
	return DRVError(code)
}

/* this block contains functions which deal with camera initialization

 */

// Initialize initializes the camera connection with the driver library
func (c *Camera) Initialize() error {
	return nil
}

// ShutDown shuts down the camera
func (c *Camera) ShutDown() error {
	return nil
}

// GetDetector gets the detector
func (c *Camera) GetDetector() error { // need another return type
	return nil
}

// GetHardwareVersion gets the hardware version string from the camera
func (c *Camera) GetHardwareVersion() error { // need another return type
	return nil
}

// GetSoftwareVersion gets the software version from the caemra
func (c *Camera) GetSoftwareVersion() error {
	return nil
}

// GetNumberVSSpeeds gets the number of vertical shift register speeds available
func (c *Camera) GetNumberVSSpeeds() error { // need another return type
	return nil
}

// GetVSSpeed gets the vertical shift register speed
func (c *Camera) GetVSSpeed() error { // need another return type
	return nil
}

// GetFastestRecommendedVSSpeed gets the fastest vertical shift register speed
// that does not require changing the vertical clock voltage.  It returns
// the fastest vertical clock speed's intcode and the actual speed in microseconds
func (c *Camera) GetFastestRecommendedVSSpeed() (int, float64, error) {
	return 0, 0., nil
}

// SetVSSpeed sets the vertical shift register speed
func (c *Camera) SetVSSpeed() error { // need another argument type
	return nil
}

// SetVSAmplitude sets the vertical shift register voltage
func (c *Camera) SetVSAmplitude(vcv VerticalClockVoltage) error {
	return nil
}

// GetNumberHSSpeeds gets the number of horizontal shift register speeds available
func (c *Camera) GetNumberHSSpeeds() error { // need another return type
	return nil
}

// GetHSSpeed gets the horizontal shift speed
func (c *Camera) GetHSSpeed() error { // need another return type
	return nil
}

// SetHSSpeed sets the horizontal shift speed
func (c *Camera) SetHSSpeed() error { // need another argument type
	return nil
}

/* the above deals with camera initialization, the below deals with temperature regulation.

 */

// SetCoolerActive toggles the cooler on (true) or off (false)
func (c *Camera) SetCoolerActive(b bool) error {
	// call CoolerON or CoolerOFF as needed
	return nil
}

// GetCoolerActive gets if the cooler is currently engaged
func (c *Camera) GetCoolerActive() (bool, error) {
	return true, nil
}

// GetTemperatureRange gets ...
func (c *Camera) GetTemperatureRange() error { // need another return type
	return nil
}

// GetTemperature gets the current temperature in degrees celcius
func (c *Camera) GetTemperature() (float64, error) {
	return 0, nil
}

// SetTemperature sets the temperature setpoint in degrees celcius
func (c *Camera) SetTemperature(t float64) error {
	return nil
}

/* the above deals with thermal management, the below deals with acquisition programming

 */

// SetAcquisitionMode sets the acquisition mode of the camera
func (c *Camera) SetAcquisitionMode(am AcquisitionMode) error {
	return nil
}

// SetReadoutMode sets the readout mode of the camera
func (c *Camera) SetReadoutMode(rm ReadoutMode) error {
	return nil
}

// SetShutter sets the shutter mode of the camera TODO: check this
func (c *Camera) SetShutter() error {
	return nil
}

// SetExposureTime sets the exposure time of the camera in seconds
func (c *Camera) SetExposureTime(t float64) error {
	return nil
}

// SetTriggerMode sets the trigger mode of the camera
func (c *Camera) SetTriggerMode(tm TriggerMode) error {
	return nil
}

// SetAccumulationCycleTime sets the accumulation cycle time of the camera in seconds
func (c *Camera) SetAccumulationCycleTime(t float64) error {
	return nil
}

// SetNumberAccumulations sets the number of accumulaions
func (c *Camera) SetNumberAccumulations(i uint) error {
	return nil
}

// SetNumberKinetics sets the number of kinetics
func (c *Camera) SetNumberKinetics(i uint) error {
	return nil
}

// SetKineticCycleTime sets the kinetic cycle time
func (c *Camera) SetKineticCycleTime(t float64) error {
	return nil
}

// GetAcquisitionTimings gets the acquisition timings
func (c *Camera) GetAcquisitionTimings() error {
	return nil
}

// StartAcquisition starts the camera acquiring charge for an image
func (c *Camera) StartAcquisition() error {
	return nil
}

// GetStatus gets the status while the camera is acquiring data
func (c *Camera) GetStatus() error {
	return nil
}

// GetAcquiredData gets the acquired data / frame
func (c *Camera) GetAcquiredData() error {
	return nil
}

// AbortAcquisition aborts the current acquisition if one is active
func (c *Camera) AbortAcquisition() error {
	return nil
}

// WaitForAcquisition sleeps while waiting for the acquisition completed signal
// from the SDK
func (c *Camera) WaitForAcquisition(t time.Duration) error {
	i64 := t.Milliseconds()
	errCode := C.WaitForAcquisitionTimeOut(i64)
	return Error(errCode)
}

/* the previous section deals with acquisition, the below deals with processing.

 */

// FilterSetMode sets the filtering mode of the camera
func (c *Camera) FilterSetMode(fm FilterMode) error {
	return nil
}

// SetBaselineClamp toggles the baseline clamp feature of the camera on (true) or off (false)
func (c *Camera) SetBaselineClamp(b bool) error {
	return nil
}

/* the previous section deals with processing, the below deals with EMCCD features.

 */

// GetEMCCDGain gets the current EMCCD gain
func (c *Camera) GetEMCCDGain() error { // need another return type
	return nil
}

// GetEMGainRange gets the min and max EMCCD gain settings for the current gain
// mode and temperature of the sensor
func (c *Camera) GetEMGainRange() (int, int, error) {
	return 0, 0, error
}
