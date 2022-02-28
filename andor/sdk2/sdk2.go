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
complex functionality.  An example of this is in the same repository,
cmd/andorhttp2, which wraps the camera in an HTTP server.

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
	"image"
	"log"
	"reflect"
	"strconv"
	"time"
	"unsafe"

	"github.com/astrogo/fitsio"
	"github.jpl.nasa.gov/bdube/golab/generichttp/camera"
	"github.jpl.nasa.gov/bdube/golab/util"
)

// Camera represents an Andor camera
type Camera struct {
	// nil values cause get functions to bounce
	// tempSetpoint is the temperature the TEC is set to
	tempSetpoint *string

	// exposureTime is the length of time the exposure is set to
	exposureTime *time.Duration

	vsAmplitude *string //vs ==> vertical shift

	acquisitionMode *string

	readoutMode *string

	triggerMode *string

	filterMode *string

	// fanOn holds the status of the fan
	fanOn *bool

	// aoi holds the AOI parameters
	aoi *camera.AOI

	// binning holds the binning parameters
	bin *camera.Binning

	// emgainmode holds the EM gain mode
	emgainMode *string

	// shutter holds if the shutter is currently open
	shutter *bool

	emAdvanced *bool

	baselineClamp *bool

	// shutterAuto holds if the shutter is in automatic mode or manual
	shutterAuto *bool

	// shutterSpeed indicates the opening AND closing time of the shutter
	shutterSpeed *time.Duration

	// adchannel holds the selected A/D channel
	adchannel *int

	// frameTransfer holds if the camera is in frame transfer mode
	frameTransfer *bool
}

func boolOptionHelper() map[string]interface{} {
	return map[string]interface{}{
		"type": "bool",
	}
}

func enumOptionHelper(e Enum) map[string]interface{} {
	return map[string]interface{}{
		"type":    "enum",
		"options": enumKeys(e),
	}
}

type noptsFunc func() (int, error)
type rangeOptFunc func() (int, int, error)
type intOptFunc func(int) (int, error)
type optFunc func(int) (float64, error)

func intOptionHelper(rangefunc rangeOptFunc) map[string]interface{} {
	min, max, err := rangefunc()
	if err != nil {
		return nil
	}
	return map[string]interface{}{
		"type": "int",
		"min":  min,
		"max":  max,
	}
}

func floatEnumOptionHelper(nfunc noptsFunc, optfunc optFunc) (map[string]interface{}, error) {
	n, err := nfunc()
	if err != nil {
		return nil, err
	}
	opts := make([]float64, n)
	for i := 0; i < n; i++ {
		opt, err := optfunc(i)
		if err != nil {
			return nil, err
		}
		opts[i] = opt
	}
	return map[string]interface{}{
		"type":    "floatEnum",
		"options": opts,
	}, nil
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
func (c *Camera) GetNumberVSSpeeds() (int, error) {
	var speeds C.int
	errCode := uint(C.GetNumberVSSpeeds(&speeds))
	return int(speeds), Error(errCode)
}

func (c *Camera) GetVSAmplitude() (string, error) {
	if c.vsAmplitude == nil {
		return "", ErrParameterNotSet
	}
	return *c.vsAmplitude, nil
}

func (c *Camera) GetVSAmplitudeType() (int, error) {
	if c.vsAmplitude == nil {
		return -1, ErrParameterNotSet
	}
	outputAmpType, ok := VerticalClockVoltage[*c.vsAmplitude]
	if !ok {
		return -1, ErrBadEnumIndex
	}
	return outputAmpType, nil
}

// GetVSSpeed gets the vertical shift register speed in microseconds
func (c *Camera) GetVSSpeed(idx int) (float64, error) {
	var f C.float
	errCode := uint(C.GetVSSpeed(C.int(idx), &f))
	return float64(f), Error(errCode)
}

// GetVSSpeedRange gets the vertical shift register speed in microseconds
func (c *Camera) GetVSSpeedRange() (float64, float64, error) {
	max, err := c.GetNumberVSSpeeds()
	if err != nil {
		return 0, 0, err
	}
	return 0, float64(max), nil
}

// GetVSSpeedOption gets the vertical shift register speed in microseconds
func (c *Camera) GetVSSpeedOption(idx int) (float64, error) {
	var f C.float
	errCode := uint(C.GetVSSpeed(C.int(idx), &f))
	return float64(f), Error(errCode)
}

// GetVSSpeedOptions gets the vertical shift register speed in microseconds
/*func (c *Camera) GetVSSpeedOptions() (map[string]interface{}, error) {
	ary, err := c.floatEnumOptions(c.GetNumberVSSpeeds, c.GetVSSpeed)
	if err != nil {
		return nil, err
	}
	return ary, nil
}*/

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
func (c *Camera) SetVSSpeed(idx int) error {
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
func (c *Camera) GetNumberHSSpeeds(ch int) (int, error) {
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

// GetHSSpeedOption gets the horizontal shift speed in Megahertz
// adcCh is an ADC channel
// outputAmpType is the output amplifier type; 0 => EM gain; 1 => conventional
// idx is the enum index
func (c *Camera) GetHSSpeedOption(adcCh, outputAmpType, idx int) (float64, error) {
	cch := C.int(adcCh)
	ctyp := C.int(outputAmpType)
	cidx := C.int(idx)
	var ret C.float

	errCode := uint(C.GetHSSpeed(cch, ctyp, cidx, &ret))
	return float64(ret), Error(errCode)
}

// GetHSSpeed gets the horizontal shift speed in Megahertz
// idx is the enum index
//
// ADC channel obtain from the last set
// Output amplifier type; 0 => EM gain; 1 => conventional, obtained from last
func (c *Camera) GetHSSpeed(idx int) (float64, error) {
	if c.adchannel == nil {
		return -1, ErrBadEnumIndex
	}
	outputAmpType, ok := VerticalClockVoltage[*c.vsAmplitude]
	if !ok {
		return -1, ErrBadEnumIndex
	}

	cch := *c.adchannel
	ctyp := outputAmpType
	cidx := idx

	return c.GetHSSpeedOption(cch, ctyp, cidx)
}

// SetHSSpeedIndex sets the horizontal shift speed
// outputAmpType is the output amplifier type; 0 => EM gain; 1 => conventional
// idx is the enum index
func (c *Camera) SetHSSpeedIndex(outputAmpType, idx int) error {
	ctyp := C.int(outputAmpType)
	cidx := C.int(idx)

	errCode := uint(C.SetHSSpeed(ctyp, cidx))
	return Error(errCode)
}

// SetHSSpeed sets the horizontal shift speed
// idx is the enum index
// NOTE:  The output amplitude type is the obtained from the last set
func (c *Camera) SetHSSpeed(idx int) error {

	if c.vsAmplitude == nil {
		return ErrParameterNotSet
	}
	outputAmpType, ok := VerticalClockVoltage[*c.vsAmplitude]
	if !ok {
		return ErrBadEnumIndex
	}

	return c.SetHSSpeedIndex(outputAmpType, idx)
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
func (c *Camera) GetTemperatureRange() (int, int, error) {
	var min, max C.int
	errCode := uint(C.GetTemperatureRange(&min, &max))
	return int(min), int(max), Error(errCode)
}

// GetTemperature gets the current temperature in degrees celcius.  The real type is int, but we use float for SD3 compatibility
func (c *Camera) GetTemperature() (float64, error) {
	var temp C.int
	errCode := uint(C.GetTemperature(&temp))
	err := Error(errCode)
	ret := float64(int(temp))
	if BeneignThermal(err) {
		return ret, nil
	}
	return ret, err
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

//GetTemperatureSetpointInfo Return possible options for temp setpoint
func (c *Camera) GetTemperatureSetpointInfo() (map[string]interface{}, error) {
	ret := map[string]interface{}{}
	min, max, err := c.GetTemperatureRange()
	if err != nil {
		return nil, err
	}
	ret["type"] = "string"
	ret["min"] = strconv.Itoa(min)
	ret["max"] = strconv.Itoa(max)

	return ret, nil
}

// GetTemperatureSetpoints returns an array of the MIN and MAX temperatures.
// Any integer intermediate is valid
func (c *Camera) GetTemperatureSetpoints() ([]string, error) {
	min, max, err := c.GetTemperatureRange()
	if err != nil {
		return []string{}, err
	}
	minS := strconv.Itoa(min)
	maxS := strconv.Itoa(max)
	return []string{minS, maxS}, nil
}

// GetTemperatureStatus queries the status of the cooling system.
// it is not implemented in the SDK but is required to satisfy the thermalcontroller
// interface
func (c *Camera) GetTemperatureStatus() (string, error) {
	return "", errors.New("andor/sdk2: GetTemperatureStatus not implemented on iXON EMCCD")
}

// SetFan allows the fan to be turned on or off.
// this is not a 1:1 mimic of SDk2, since it is binary
// on (HIGH) or off (OFF)
func (c *Camera) SetFan(on bool) error {
	var in C.int
	if on {
		in = 0
	} else {
		in = 2
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
	err := Error(errCode)
	if err == nil {
		c.acquisitionMode = &am
	}
	return err
}

func (c *Camera) GetAcquisitionMode() (string, error) {
	if c.acquisitionMode == nil {
		return "", ErrParameterNotSet
	}
	return *c.acquisitionMode, nil
}

// SetReadoutMode sets the readout mode of the camera.  We rename this from SetReadMode in the actual driver
func (c *Camera) SetReadoutMode(rm string) error {
	i, ok := ReadoutMode[rm]
	if !ok {
		return ErrBadEnumIndex
	}
	errCode := uint(C.SetReadMode(C.int(i)))
	err := Error(errCode)
	if err == nil {
		c.readoutMode = &rm
	}
	return err
}

// GetReadoutMode returns the current read mode
func (c *Camera) GetReadoutMode() (string, error) {
	if c.readoutMode == nil {
		return "", ErrParameterNotSet
	}
	return *c.readoutMode, nil
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
	err := Error(errCode)
	if err == nil {
		c.triggerMode = &tm
	}
	return err
}

// GetTriggerMode returns the current trigger mode
func (c *Camera) GetTriggerMode() (string, error) {
	if c.triggerMode == nil {
		return "", ErrParameterNotSet
	}
	return *c.triggerMode, nil
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

// AbortAcquisition aborts the current acquisition if one is active
func (c *Camera) AbortAcquisition() error {
	errCode := uint(C.AbortAcquisition())
	return Error(errCode)
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
func (c *Camera) GetAcquiredData() ([]int32, error) {
	// primary consumer of this code is andorhttp2, which explicitly inits
	// the frame size to the detector area
	// getframesize may fail otherwise, but we should have failed prior
	// to starting data acquisition (even earlier than data transfer, this
	// function) anyway
	w, h, err := c.GetFrameSize()
	if err != nil {
		return nil, err
	}
	elements := w * h
	buf := make([]int32, elements)
	ptr := (*C.at_32)(unsafe.Pointer(&buf[0]))
	errCode := uint(C.GetAcquiredData(ptr, C.uint(elements)))
	return buf, Error(errCode)
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
	err := Error(errCode)
	if err == nil {
		c.adchannel = &ch
	}
	return err
}

// GetADChannel returns the currently selected AD channel
func (c *Camera) GetADChannel() (int, error) {
	if c.adchannel == nil {
		return 0, ErrParameterNotSet
	}
	return *c.adchannel, nil
}

// GetADChannelRange returns the currently selected AD channel
func (c *Camera) GetADChannelRange() (int, int, error) {

	max, err := c.GetNumberADChannels()
	if err != nil {
		return 0, 0, err
	}
	return 0, max - 1, nil
}

// GetNumberPreAmpGains returns the number of preamplifier gain settings available
func (c *Camera) GetNumberPreAmpGains() (int, error) {
	var num C.int
	errCode := uint(C.GetNumberPreAmpGains(&num))
	return int(num), Error(errCode)
}

// GetPreAmpGain returns the preamp gain (multiplier) associated
// with a given index
func (c *Camera) GetPreAmpGain(idx int) (float64, error) {
	var f C.float
	errCode := uint(C.GetPreAmpGain(C.int(idx), &f))
	return float64(f), Error(errCode)
}

// GetPreAmpGainText returns a text description of a given preamp gain
func (c *Camera) GetPreAmpGainText(idx int) (string, error) {
	var buf [30]C.char
	errCode := uint(C.GetPreAmpGainText(C.int(idx), &buf[0], 30))
	return C.GoString(&buf[0]), Error(errCode)
}

// GetPreAmpOptions
func (c *Camera) GetPreAmpOptions() (map[string]interface{}, error) {
	i, err := c.GetNumberPreAmpGains()
	if err != nil {
		return nil, err
	}
	ret := map[string]interface{}{}
	ret["min"] = 0
	ret["max"] = i - 1
	ret["type"] = "int"
	return ret, err
}

// SetPreAmpGain sets the preamp gain for the given AD channel
// to the specified index
func (c *Camera) SetPreAmpGain(idx int) error {
	errCode := uint(C.SetPreAmpGain(C.int(idx)))
	return Error(errCode)
}

// SetFrameTransferMode puts the camera into frame transfer mode when
// the argument is true
func (c *Camera) SetFrameTransferMode(useFrameTransfer bool) error {
	var mode C.int
	if useFrameTransfer {
		mode = 1
	} else {
		mode = 0
	}
	errCode := uint(C.SetFrameTransferMode(mode))
	err := Error(errCode)
	if err == nil {
		c.frameTransfer = &useFrameTransfer
	}
	return err
}

// GetFrameTransferMode returns true if the camera is in frame transfer mode
func (c *Camera) GetFrameTransferMode() (bool, error) {
	if c.frameTransfer == nil {
		return false, ErrParameterNotSet
	}
	return *c.frameTransfer, nil
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
		c.aoi = &camera.AOI{Left: hstart, Top: vstart, Width: hend - hstart + 1, Height: vend - vstart + 1}
	}
	return err
}

// SetAOI sets the AoI used by the camera
func (c *Camera) SetAOI(a camera.AOI) error {
	bin, err := c.GetBinning() // trigger error if we have no knowledge
	if err != nil {
		return err
	}
	return c.SetImage(bin.H, bin.V, a.Left, a.Right(), a.Top, a.Bottom())
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

// SetFilterMode sets the filtering mode of the camera
func (c *Camera) SetFilterMode(fm string) error {
	i, ok := FilterMode[fm]
	if !ok {
		return ErrBadEnumIndex
	}
	errCode := uint(C.Filter_SetMode(C.uint(i)))
	err := Error(errCode)
	if err == nil {
		c.filterMode = &fm
	}
	return err
}

// GetFilterMode returns the current filter mode
func (c *Camera) GetFilterMode() (string, error) {
	if c.filterMode == nil {
		return "", ErrParameterNotSet
	}
	return *c.filterMode, nil
}

// SetBaselineClamp toggles the baseline clamp feature of the camera on (true) or off (false)
func (c *Camera) SetBaselineClamp(b bool) error {
	on := 0
	if b {
		on = 1
	}
	errCode := uint(C.SetBaselineClamp(C.int(on)))
	err := Error(errCode)
	if err == nil {
		c.baselineClamp = &b
	}
	return err
}

// GetBaselineClamp returns the EM advanced
func (c *Camera) GetBaselineClamp() (bool, error) {
	if c.baselineClamp == nil {
		return false, ErrParameterNotSet
	}
	return *c.baselineClamp, nil
}

/* the previous section deals with processing, the below deals with EMCCD features.

 */

// GetEMGain gets the current EMCCD gain
func (c *Camera) GetEMGain() (int, error) {
	var mult C.int
	errCode := uint(C.GetEMCCDGain(&mult))
	return int(mult), Error(errCode)
}

// SetEMGain sets the EMCCD gain.  The precise behavior depends on the current
// gain mode, see SetEMGainMode, GetEMGainMode
func (c *Camera) SetEMGain(fctr int) error {
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
	err := Error(errCode)
	if err == nil {
		c.emgainMode = &gm
	}
	return err
}

// GetEMGainMode returns the current EM gain mode
func (c *Camera) GetEMGainMode() (string, error) {
	if c.emgainMode == nil {
		return "", ErrParameterNotSet
	}
	return *c.emgainMode, nil
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
	err := Error(errCode)
	if err == nil {
		c.emAdvanced = &b
	}
	return err
}

// GetEMAdvanced returns the EM advanced
func (c *Camera) GetEMAdvanced() (bool, error) {
	if c.emAdvanced == nil {
		return false, ErrParameterNotSet
	}
	return *c.emAdvanced, nil
}

// GetFrameSize gets the W, H of a frame as recorded in the strided buffer
func (c *Camera) GetFrameSize() (int, int, error) {
	aoi, err := c.GetAOI()
	if err != nil {
		return 0, 0, err
	}
	return aoi.Width, aoi.Height, nil
}

// GetFrame returns a frame from the camera
func (c *Camera) GetFrame() (image.Image, error) {
	ret := &image.Gray16{}
	c.AbortAcquisition() // always clear out in case of dangling acq

	tExp, err := c.GetExposureTime()
	if err != nil {
		return ret, err
	}

	w, h, err := c.GetFrameSize()
	if err != nil {
		return ret, err
	}

	err = c.StartAcquisition()
	if err != nil {
		return ret, err
	}

	err = c.WaitForAcquisition(tExp + 3*time.Second)
	if err != nil {
		return ret, err
	}
	stat, err := c.GetStatus()
	// sometimes the SDK frees you from sleep even though the camera is still in acq
	// this block will spam the camera for if it is acquiring for up (tExp + 15 seconds)
	// to "guarantee" we aren't trapped in a bad state.
	if stat == StatusAcquiring {
		deadline := time.Now().Add(tExp + 15*time.Second)
		tSleep := 1 * time.Millisecond
		for stat == StatusAcquiring && time.Now().Before(deadline) {
			stat, err = c.GetStatus()
			time.Sleep(tSleep)
			tSleep *= 2
		}
	}
	buf, err := c.GetAcquiredData()
	if err != nil {
		return ret, err
	}

	l := len(buf)
	b2 := make([]uint16, l)
	for idx := 0; idx < l; idx++ {
		b2[idx] = uint16(buf[idx])
	}
	var b3 []byte
	hdr := (*reflect.SliceHeader)(unsafe.Pointer(&b3))
	hdr.Data = uintptr(unsafe.Pointer(&b2[0]))
	hdr.Len = len(b2) * 2
	hdr.Cap = cap(b2) * 2

	ret.Pix = b3
	ret.Stride = w * 2
	ret.Rect = image.Rect(0, 0, w, h)
	err = c.AbortAcquisition()
	return ret, err
}

// Burst takes a chunk of pictures and sends htem on a channel
func (c *Camera) Burst(frames int, fps float64, ch chan<- image.Image) error {
	return fmt.Errorf("not implemented")
}

// GetSerialNumber returns the serial number as an integer
func (c *Camera) GetSerialNumber() (int, error) {
	var i C.int
	errCode := uint(C.GetCameraSerialNumber(&i))
	return int(i), Error(errCode)
}

// SetShutter sets the shutter parameters of the camera.
// ttlHi sends output TTL high signal to open shutter, else sends TTL low signal
func (c *Camera) setShutter(ttlHi bool, mode string, opening, closing time.Duration) error {
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

// SetShutter opens the shutter (true) or closes it (false)
func (c *Camera) SetShutter(b bool) error {
	var inp string
	if b {
		inp = "Open"
	} else {
		inp = "Close"
	}
	err := c.setShutter(true, inp, time.Millisecond, time.Millisecond)
	if err == nil {
		c.shutter = &b
	}
	return err
}

// GetShutter returns true if the shutter is currently open
func (c *Camera) GetShutter() (bool, error) {
	if c.shutter == nil {
		return false, ErrParameterNotSet
	}
	return *c.shutter, nil
}

// SetShutterAuto puts the camera into or out of automatic shutering mode,
// in which the camera itself controls the signaling and timing of the shutter.
//
// b=false will put the shutter into a manual configuration mode and close it.
func (c *Camera) SetShutterAuto(b bool) error {
	log.Println("SetShutterAuto: incoming shutterSpeed is", c.shutterSpeed)
	if c.shutterSpeed == nil {
		return fmt.Errorf("cannot set auto shutter without first using SetShutterSpeed")
	}
	var (
		err error
	)
	if b {
		err = c.setShutter(true, "Auto", *c.shutterSpeed, *c.shutterSpeed)
	} else {
		err = c.setShutter(true, "Close", *c.shutterSpeed, *c.shutterSpeed)
	}
	if err == nil {
		c.shutterAuto = &b
	}
	return err
}

// SetShutterSpeed sets the shutter opening AND closing time for the camera
func (c *Camera) SetShutterSpeed(t time.Duration) error {
	var (
		err  error
		auto bool
	)
	if c.shutterAuto == nil {
		auto = false
	} else {
		auto = *c.shutterAuto
	}
	if auto {
		err = c.setShutter(true, "Auto", t, t)
	} else {
		var (
			b   bool
			inp string = "Close"
		)
		if c.shutter == nil {
			b = false
		} else {
			b = *c.shutter
		}
		if b {
			inp = "Open"
		}
		err = c.setShutter(true, inp, t, t)
	}
	if err == nil {
		c.shutterSpeed = &t
	}
	return err
}

// GetShutterSpeed retrieves if the shutter is in automatic (camera managed) or
// manual (user managed) mode.
func (c *Camera) GetShutterSpeed() (time.Duration, error) {
	if c.shutterSpeed == nil {
		return 0, ErrParameterNotSet
	}
	return *c.shutterSpeed, nil
}

// GetShutterAuto retrieves if the shutter is in automatic (camera managed) or
// manual (user managed) mode.
func (c *Camera) GetShutterAuto() (bool, error) {
	if c.shutterAuto == nil {
		return false, ErrParameterNotSet
	}
	return *c.shutterAuto, nil
}

// CollectHeaderMetadata satisfies generichttp/camera and makes a stack of FITS cards
func (c *Camera) CollectHeaderMetadata() []fitsio.Card {
	// grab all the shit we care about from the camera so we can fill out the header
	// plow through errors, no need to bail early
	aoi, err := c.GetAOI()
	texp, err := c.GetExposureTime()
	camsn, err := c.GetSerialNumber()
	fan, err := c.GetFan()
	tsetpt, err := c.GetTemperatureSetpoint()
	temp, err := c.GetTemperature()
	bin, err := c.GetBinning()
	if err != nil {
		bin = camera.Binning{}
	}
	binS := bin.HxV()

	var metaerr string
	if err != nil {
		metaerr = err.Error()
	} else {
		metaerr = ""
	}
	now := time.Now()
	ts := fmt.Sprintf("%d-%02d-%02dT%02d:%02d:%02d",
		now.Year(),
		now.Month(),
		now.Day(),
		now.Hour(),
		now.Minute(),
		now.Second())

	return []fitsio.Card{
		/* andor-http header format includes:
		- header format tag
		- server version
		- sdk software version
		- driver version
		- camera firmware version

		- camera model
		- camera serial number

		- aoi top, left, top, bottom
		- binning

		- fan on/off
		- thermal setpoint
		- thermal status
		- fpa temperature
		*/
		// header to the header
		{Name: "HDRVER", Value: "EMCCD-1", Comment: "header version"},
		{Name: "WRAPVER", Value: WRAPVER, Comment: "server library code version"},
		{Name: "METAERR", Value: metaerr, Comment: "error encountered gathering metadata"},
		{Name: "CAMMODL", Value: "Andor iXon Ultra 888", Comment: "camera model"},
		{Name: "CAMSN", Value: camsn, Comment: "camera serial number"},
		{Name: "BITDEPTH", Value: 14, Comment: "2^BITDEPTH is the maximum possible DN"},

		// timestamp
		{Name: "DATE", Value: ts}, // timestamp is standard and does not require comment

		// exposure parameters
		{Name: "EXPTIME", Value: texp.Seconds(), Comment: "exposure time, seconds"},

		// thermal parameters
		{Name: "FAN", Value: fan, Comment: "on (true) or off"},
		{Name: "TEMPSETP", Value: tsetpt, Comment: "Temperature setpoint"},
		{Name: "TEMPER", Value: temp, Comment: "FPA temperature (Celcius)"},
		// aoi parameters
		{Name: "AOIL", Value: aoi.Left, Comment: "1-based left pixel of the AOI"},
		{Name: "AOIT", Value: aoi.Top, Comment: "1-based top pixel of the AOI"},
		{Name: "AOIW", Value: aoi.Width, Comment: "AOI width, px"},
		{Name: "AOIH", Value: aoi.Height, Comment: "AOI height, px"},
		{Name: "AOIB", Value: binS, Comment: "AOI Binning, HxV"}}
}
func (c *Camera) SetFeature(feature string, v interface{}) error {
	type fStrErr func(string) error
	type fBoolErr func(bool) error
	type fIntErr func(int) error
	strFuncs := map[string]fStrErr{
		"VSAmplitude":         c.SetVSAmplitude,
		"AcquisitionMode":     c.SetAcquisitionMode,
		"ReadoutMode":         c.SetReadoutMode,
		"TemperatureSetpoint": c.SetTemperatureSetpoint,
		"FilterMode":          c.SetFilterMode,
		"TriggerMode":         c.SetTriggerMode,
		"EMGainMode":          c.SetEMGainMode,
	}
	boolFuncs := map[string]fBoolErr{
		"ShutterOpen":       c.SetShutter,
		"ShutterAuto":       c.SetShutterAuto,
		"FanOn":             c.SetFan,
		"EMGainAdvanced":    c.SetEMAdvanced,
		"SensorCooling":     c.SetCooling,
		"BaselineClamp":     c.SetBaselineClamp,
		"FrameTransferMode": c.SetFrameTransferMode,
	}
	intFuncs := map[string]fIntErr{
		"ADChannel":  c.SetADChannel,
		"EMGain":     c.SetEMGain,
		"HSSpeed":    c.SetHSSpeed,
		"VSSpeed":    c.SetVSSpeed,
		"PreAmpGain": c.SetPreAmpGain,
	}

	var err error
	if f, ok := strFuncs[feature]; ok {
		err = f(v.(string))
	} else if f, ok := boolFuncs[feature]; ok {
		err = f(v.(bool))
	} else if f, ok := intFuncs[feature]; ok {
		i := int(v.(float64))
		err = f(i)
	} else {
		return fmt.Errorf("Feature [%s] with value [%v] not understood or unavailble", feature, v)
	}

	return err
}

// GetFeatureInfo For numerical features, it returns the min and max values.  For enum
// features, it returns the possible strings that can be used
func (c *Camera) GetFeatureInfo(feature string) (map[string]interface{}, error) {

	switch feature {
	case "AcquisitionMode":
		return enumOptionHelper(AcquisitionMode), nil
	case "VSAmplitude":
		return enumOptionHelper(VerticalClockVoltage), nil
	case "ReadoutMode":
		return enumOptionHelper(ReadoutMode), nil
	case "FilterMode":
		return enumOptionHelper(FilterMode), nil
	case "TriggerMode":
		return enumOptionHelper(TriggerMode), nil
	case "EMGainMode":
		return enumOptionHelper(EMGainMode), nil
	case "VSSpeed":
		return floatEnumOptionHelper(c.GetNumberVSSpeeds, c.GetVSSpeedOption)
	case "ShutterOpen", "ShutterAuto", "FanOn", "EMGainAdvanced", "SensorCooling", "BaselineClamp", "FrameTransferMode":
		return boolOptionHelper(), nil
	case "ADChannel":
		return intOptionHelper(c.GetADChannelRange), nil
	case "EMGain":
		return intOptionHelper(c.GetEMGainRange), nil
	case "TemperatureSetpoint":
		return c.GetTemperatureSetpointInfo()
	default:
		return nil, ErrFeatureNotFound{feature}
	}
}

func (c *Camera) GetFeature(feature string) (interface{}, error) {
	type fStrErr func() (string, error)
	type fBoolErr func() (bool, error)
	type fIntErr func() (int, error)
	type fIntIdxErr func(idx int) (float64, error)
	strFuncs := map[string]fStrErr{
		"VSAmplitude":         c.GetVSAmplitude,
		"AcquisitionMode":     c.GetAcquisitionMode,
		"ReadoutMode":         c.GetReadoutMode,
		"TemperatureSetpoint": c.GetTemperatureSetpoint,
		"FilterMode":          c.GetFilterMode,
		"TriggerMode":         c.GetTriggerMode,
		"EMGainMode":          c.GetEMGainMode,
	}
	boolFuncs := map[string]fBoolErr{
		"ShutterOpen":       c.GetShutter,
		"ShutterAuto":       c.GetShutterAuto,
		"FanOn":             c.GetFan,
		"EMGainAdvanced":    c.GetEMAdvanced,
		"SensorCooling":     c.GetCooling,
		"BaselineClamp":     c.GetBaselineClamp,
		"FrameTransferMode": c.GetFrameTransferMode,
	}
	intFuncs := map[string]fIntErr{
		"ADChannel": c.GetADChannel,
		"EMGain":    c.GetEMGain,
	}

	/*intIdxFuncs := map[string]fIntIdxErr{
		"VSSpeed":    c.GetVSSpeed,
		"PreAmpGain": c.GetPreAmpGain,
		"GetHSSpeed": c.GetHSSpeed,
	}*/

	if f, ok := strFuncs[feature]; ok {
		return f()
	} else if f, ok := boolFuncs[feature]; ok {
		return f()
	} else if f, ok := intFuncs[feature]; ok {
		return f()
	} else {
		return nil, fmt.Errorf("Feature [%s] unknown", feature)
	}
}

func (c *Camera) GetFeatureIdx(index int, feature string) (interface{}, error) {
	type fIntIdxErr func(idx int) (float64, error)

	intIdxFuncs := map[string]fIntIdxErr{
		"VSSpeed":    c.GetVSSpeed,
		"PreAmpGain": c.GetPreAmpGain,
		"GetHSSpeed": c.GetHSSpeed,
	}

	if f, ok := intIdxFuncs[feature]; ok {
		return f(index)
	}

	return nil, fmt.Errorf("Feature [%s] unknown", feature)
}

// Configure sets many values for the camera at once
func (c *Camera) Configure(settings map[string]interface{}) error {
	type fStrErr func(string) error
	type fBoolErr func(bool) error
	type fIntErr func(int) error
	strFuncs := map[string]fStrErr{
		"VSAmplitude":         c.SetVSAmplitude,
		"AcquisitionMode":     c.SetAcquisitionMode,
		"ReadoutMode":         c.SetReadoutMode,
		"TemperatureSetpoint": c.SetTemperatureSetpoint,
		"FilterMode":          c.SetFilterMode,
		"TriggerMode":         c.SetTriggerMode,
		"EMGainMode":          c.SetEMGainMode,
	}
	boolFuncs := map[string]fBoolErr{
		"ShutterOpen":       c.SetShutter,
		"ShutterAuto":       c.SetShutterAuto,
		"FanOn":             c.SetFan,
		"EMGainAdvanced":    c.SetEMAdvanced,
		"SensorCooling":     c.SetCooling,
		"BaselineClamp":     c.SetBaselineClamp,
		"FrameTransferMode": c.SetFrameTransferMode,
	}
	intFuncs := map[string]fIntErr{
		"ADChannel":  c.SetADChannel,
		"EMGain":     c.SetEMGain,
		"HSSpeed":    c.SetHSSpeed,
		"VSSpeed":    c.SetVSSpeed,
		"PreAmpGain": c.SetPreAmpGain,
	}
	var errs []error
	for k, v := range settings {
		var err error
		if f, ok := strFuncs[k]; ok {
			err = f(v.(string))
		} else if f, ok := boolFuncs[k]; ok {
			err = f(v.(bool))
		} else if f, ok := intFuncs[k]; ok {
			i := int(v.(float64))
			err = f(i)
		} else {
			return fmt.Errorf("Configuration parameter %s with value %v not understood or unavailble", k, v)
		}
		errs = append(errs, err)
	}
	return util.MergeErrors(errs)
}
