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

	// shutterAuto holds if the shutter is in automatic mode or manual
	shutterAuto *bool

	// adchannel holds the selected A/D channel
	adchannel *int

	// frameTransfer holds if the camera is in frame transfer mode
	frameTransfer *bool
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

// GetVSSpeed gets the vertical shift register speed in microseconds
func (c *Camera) GetVSSpeed(idx int) (float64, error) {
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

// GetHSSpeed gets the horizontal shift speed
func (c *Camera) GetHSSpeed(ch int, idx int) (float64, error) {
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
func (c *Camera) SetHSSpeed(idx int) error {
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
//
// Implementing a 32-bit function is left for the future
func (c *Camera) GetAcquiredData() ([]int32, error) {
	elements := 1024 * 1024
	buf := make([]int32, elements)
	ptr := (*C.at_32)(unsafe.Pointer(&buf[0]))
	errCode := uint(C.GetAcquiredData(ptr, C.uint(1024*1024)))
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
func (c *Camera) SetAOI(a camera.AOI) error {
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

// SetFilterMode sets the filtering mode of the camera
func (c *Camera) SetFilterMode(fm string) error {
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
	return Error(errCode)
}

// GetFrameSize gets the W, H of a frame as recorded in the strided buffer
func (c *Camera) GetFrameSize() (int, int, error) {
	aoi, err := c.GetAOI()
	if err != nil {
		return 0, 0, err
	}
	return aoi.Width, aoi.Height, nil
}

// GetFrame returns a frame from the camera as a strided buffer
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
	// this block will spam the camera for if it is acquiring for up (tExp + 5 seconds)
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
	var (
		err error
	)
	if b {
		err = c.setShutter(true, "Auto", time.Millisecond, time.Millisecond)
	} else {
		err = c.setShutter(true, "Close", time.Millisecond, time.Millisecond)
	}
	if err == nil {
		c.shutterAuto = &b
	}
	return err
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
		"EMGainMode":          c.SetEMGainMode}
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
		"ADChannel": c.SetADChannel,
		"EMGain":    c.SetEMGain,
		"HSSpeed":   c.SetHSSpeed,
		"VSSpeed":   c.SetVSSpeed}
	var errs []error
	for k, v := range settings {
		switch k {
		case "VSAmplitude", "AcquisitionMode", "ReadoutMode", "TemperatureSetpoint", "FilterMode", "TriggerMode", "EMGainMode":
			str := v.(string)
			f := strFuncs[k]
			err := f(str)
			errs = append(errs, err)
		case "ShutterOpen", "ShutterAuto", "FanOn", "EMGainAdvanced", "SensorCooling", "BaselineClamp", "FrameTransferMode":
			b := v.(bool)
			f := boolFuncs[k]
			err := f(b)
			errs = append(errs, err)
		case "ADChannel", "EMGain", "HSSpeed", "VSSpeed":
			i := int(v.(float64))
			f := intFuncs[k]
			err := f(i)
			errs = append(errs, err)
		default:
			return fmt.Errorf("Configuration parameter %s with value %v not understood or unavailble", k, v)
		}
	}
	return util.MergeErrors(errs)
}
