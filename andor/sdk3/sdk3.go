/*Package sdk3 exposes control of Andor cameras in Go via their SDK, v3.

 */
package sdk3

/*
#cgo CFLAGS: -I/usr/local
#cgo LDFLAGS: -L/usr/local/lib -latcore -latutility
#include <stdlib.h>
#include <atcore.h>

*/
import "C"
import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
	"unsafe"

	"github.jpl.nasa.gov/HCIT/go-hcit/mathx"
)

const (
	// LengthOfUndefinedBuffers is how large a buffer to allocate for a Wchar
	// string when we have no way of knowing ahead of time how big it is
	// it is measured in Wchars
	LengthOfUndefinedBuffers = 255

	// WRAPVER is the andor wrapper code version.
	// Incremement this when pkg sdk3 is updated.
	WRAPVER = 4
)

// ErrFeatureNotFound is generated when a feature is looked up in the Features
// map but does not exist there
type ErrFeatureNotFound struct {
	// Feature is the specific feature not found
	Feature string
}

// Error satisfies the error interface
func (e ErrFeatureNotFound) Error() string {
	return fmt.Sprintf("feature %s not found in Features map, see go-hcit/andor/sdk3#Features for known features", e.Feature)
}

var (
	// Features maps features to "types" without using the types pkg, due to C enums
	Features = map[string]string{
		// ints
		"AccumulatedCount":        "int",
		"AOIHBin":                 "int",
		"AOIVBin":                 "int",
		"AOILeft":                 "int",
		"AOITop":                  "int",
		"AOIStride":               "int",
		"AOIHeight":               "int",
		"AOIWidth":                "int",
		"BaselineLevel":           "int",
		"BufferOverflowEvent":     "int",
		"DeviceCount":             "int",
		"DeviceVideoIndex":        "int",
		"EventsMissedEvent":       "int",
		"ExposureStartEvent":      "int",
		"ExposureEndEvent":        "int",
		"FrameCount":              "int",
		"ImageSizeBytes":          "int",
		"LUTIndex":                "int",
		"LUTValue":                "int",
		"RowNExposureEndEvent":    "int",
		"RowNExposureStartEvent":  "int",
		"SensorHeight":            "int",
		"SensorWidth":             "int",
		"TimestampClock":          "int",
		"TimestampClockFrequency": "int",

		// bools
		"CameraAcquiring":       "bool",
		"EventEnable":           "bool",
		"FullAOIControl":        "bool",
		"IOInvert":              "bool",
		"MetadataEnable":        "bool",
		"MetadataFrame":         "bool",
		"MetadataTimestamp":     "bool",
		"Overlap":               "bool", // TODO: see if enabling this fixes fast shutter problems
		"SensorCooling":         "bool",
		"SpuriousNoiseFilter":   "bool",
		"SynchronousTriggering": "bool",

		// commands
		"AcquisitionStart":    "command",
		"AcquisitionStop":     "command",
		"CameraDump":          "command",
		"SoftwareTrigger":     "command",
		"TimestampClockReset": "command",

		// floats
		"BytesPerPixel":            "float",
		"ExposureTime":             "float",
		"FrameRate":                "float",
		"MaxInterfaceTransferRate": "float",
		"PixelHeight":              "float",
		"PixelWidth":               "float",
		"ReadoutTime":              "float",
		"SensorTemperature":        "float",
		// "TargetSensorTemperature":  "float", removed 2019-11-25, deprecated by Andor

		// enums
		"AOIBinning":               "enum",
		"AOILayout":                "enum",
		"BitDepth":                 "enum",
		"CycleMode":                "enum",
		"ElectronicShutteringMode": "enum",
		"FanSpeed":                 "enum",
		"PixelEncoding":            "enum",
		"PixelReadoutRate":         "enum",
		"TemperatureControl":       "enum",
		"TemperatureStatus":        "enum",
		"TriggerMode":              "enum",
		"SimplePreAmpGainControl":  "enum",

		// strings
		"CameraModel":     "string",
		"CameraName":      "string",
		"ControllerID":    "string",
		"DriverVersion":   "string",
		"FirmwareVersion": "string",
		"InterfaceType":   "string",
		"SerialNumber":    "string",
	}
)

// Binning encapsulates information about
type Binning struct {
	// H is the horizontal binning factor
	H int `json:"h"`

	// V is the vertical binning factor
	V int `json:"v"`
}

// FormatBinning converts a binning object to the SDK3 enum style.  Will
// cause an error inside the SDK if b.H != b.V, or if 0 <= b.H <= 4
func FormatBinning(b Binning) string {
	return fmt.Sprintf("%dx%d", b.H, b.V)
}

// ParseBinning converts an "HxV" string from the SDK into a binning object
func ParseBinning(sdkValue string) Binning {
	b := Binning{}
	chunks := strings.Split(sdkValue, "x")
	if len(chunks) != 2 {
		return b
	}
	// impossible for this to panic, since len must == 2
	b.H, _ = strconv.Atoi(chunks[0])
	b.V, _ = strconv.Atoi(chunks[1])
	return b
}

// AOI describes an area of interest on the camera
type AOI struct {
	// Left is the left pixel index.  1-based
	Left int `json:"left"`

	// Top is the top pixel index.  1-based
	Top int `json:"top"`

	// Width is the width in pixels
	Width int `json:"width"`

	// Height is the height in pixels
	Height int `json:"height"`
}

// Camera represents a camera from SDK3
type Camera struct {
	// buffer is written to by the SDK.
	// It must be 8-byte aligned.
	buffer []uint64

	// cptr is a C pointer to the first byte in buffer
	cptr *C.AT_U8

	// cptrsize is the 'size' of the c pointer
	cptrsize C.int

	// gptr is a Go pointer to the first byte in buffer
	gptr unsafe.Pointer

	// bufferOnQueue is a flag indicating if we have put a buffer onto the SDK's
	// queue yet
	bufferOnQueue bool

	// Handle holds the int that points to a specific camera
	Handle int

	// exposureTime is the currently programmed exposure time.
	exposureTime time.Duration

	// aoiStride is the stride of the padded data within the AOI
	aoiStride int

	// aoiWidth is the width of the AOI in pixels
	aoiWidth int

	// aoiHeight is the height of the AOI in pixels
	aoiHeight int

	// aoiLeft is the left pixel index (1-based) of the AoI
	aoiLeft int

	// aoiTop is the top pixel index (1-based) of the AOI
	aoiTop int

	// binning is a cross-compatible representation of AOIBinning
	binning Binning

	// imageSizeBytes is the size of the image buffer in bytes
	imageSizeBytes int

	// sensorWidth holds the width of the sensor in pixels
	sensorWidth int

	// sensorHeight holds the height of the sensor in pixels
	sensorHeight int

	// sdkver holds the sdk version
	sdkver string

	// fwver holds the firmware version
	fwver string

	// drvver holds the driver version
	drvver string

	// model holds the model number
	model string

	// serial holds the serial number
	serial string
}

// Open opens a connection to the camera.  Typically, a real camera
// is index 0, and there are two simulator cameras at indices 1 and 2
func Open(camIdx int) (*Camera, error) {
	c := Camera{}
	var hndle C.AT_H
	err := enrich(Error(int(C.AT_Open(C.int(camIdx), &hndle))), "AT_OPEN")
	c.Handle = int(hndle)
	return &c, err
}

// Close closes a connection to the camera
func (c *Camera) Close() error {
	return enrich(Error(int(C.AT_Close(C.AT_H(c.Handle)))), "AT_CLOSE")
}

// Allocate creates the buffer that will be populated by the SDK
// it should be called at init, and whenever the AOI or encoding changes
func (c *Camera) Allocate() error {
	sze, err := c.ImageSizeBytes()
	if err != nil {
		return err
	}
	c.buffer = make([]uint64, sze/8) // uint64 forces byte alignment, 8 bytes per uint64
	c.gptr = unsafe.Pointer(&c.buffer[0])
	c.cptr = (*C.AT_U8)(c.gptr)
	c.cptrsize = C.int(sze)
	return nil
}

// ImageSizeBytes is the size of the image buffer in bytes.  This function
// allows us to cache the value without going to the SDK for it.
// Use GetInt directly if you want to guarantee there are no desync bugs.
func (c *Camera) ImageSizeBytes() (int, error) {
	var i int
	var err error
	if c.imageSizeBytes == 0 {
		i, err = GetInt(c.Handle, "ImageSizeBytes")
		c.imageSizeBytes = i
	} else {
		i = c.imageSizeBytes
		err = nil
	}
	return i, err
}

// GetSensorWidth gets the width of the sensor in pixels
func (c *Camera) GetSensorWidth() (int, error) {
	var i int
	var err error
	if c.sensorWidth == 0 {
		i, err = GetInt(c.Handle, "SensorWidth")
		c.sensorWidth = i
	} else {
		i = c.sensorWidth
		err = nil
	}
	return i, err
}

// GetSensorHeight gets the height of the sensor in pixels
func (c *Camera) GetSensorHeight() (int, error) {
	var i int
	var err error
	if c.sensorHeight == 0 {
		i, err = GetInt(c.Handle, "SensorHeight")
		c.sensorHeight = i
	} else {
		i = c.sensorHeight
		err = nil
	}
	return i, err
}

// GetAOIStride is the stride of one row in the image buffer in bytes.  This
// function allows us to cache the value without going to the SDK for it.
// Use GetInt directly if you want to guarantee there are no desync bugs.
func (c *Camera) GetAOIStride() (int, error) {
	var i int
	var err error
	if c.aoiStride == 0 {
		i, err = GetInt(c.Handle, "AOIStride")
		c.aoiStride = i
	} else {
		i = c.aoiStride
		err = nil
	}
	return i, err
}

// GetAOIWidth is the width of one row in the image buffer in pixels.  This
// function allows us to cache the value without going to the SDK for it.
// Use GetInt directly if you want to guarantee there are no desync bugs.
func (c *Camera) GetAOIWidth() (int, error) {
	var i int
	var err error
	if c.aoiWidth == 0 {
		i, err = GetInt(c.Handle, "AOIWidth")
		c.aoiWidth = i
	} else {
		i = c.aoiWidth
		err = nil
	}
	return i, err
}

// GetAOIHeight is the height of one column in the image buffer in pixels.  This
// function allows us to cache the value without going to the SDK for it.
// Use GetInt directly if you want to guarantee there are no desync bugs.
func (c *Camera) GetAOIHeight() (int, error) {
	var i int
	var err error
	if c.aoiHeight == 0 {
		i, err = GetInt(c.Handle, "AOIHeight")
		c.aoiHeight = i
	} else {
		i = c.aoiHeight
		err = nil
	}
	return i, err
}

// GetAOILeft gets the left pixel of the AOI.  Starts at 1.
func (c *Camera) GetAOILeft() (int, error) {
	var i int
	var err error
	if c.aoiLeft == 0 {
		i, err = GetInt(c.Handle, "AOILeft")
		c.aoiLeft = i
	} else {
		i = c.aoiLeft
		err = nil
	}
	return i, err
}

// GetAOITop gets the top pixel index of the AOI.  Starts at 1.
func (c *Camera) GetAOITop() (int, error) {
	var i int
	var err error
	if c.aoiTop == 0 {
		i, err = GetInt(c.Handle, "AOITop")
		c.aoiTop = i
	} else {
		i = c.aoiTop
		err = nil
	}
	return i, err
}

// SetAOI updates the AOI and re-allocates the buffer.  Width and height are
// calculated from the difference of the sensor dimensions and top-left if they
// are zero
func (c *Camera) SetAOI(aoi AOI) error {
	// top
	err := SetInt(c.Handle, "AOITop", int64(aoi.Top))
	if err != nil {
		return err
	}
	c.aoiTop = aoi.Top

	// left
	err = SetInt(c.Handle, "AOILeft", int64(aoi.Left))
	if err != nil {
		return err
	}
	c.aoiLeft = aoi.Left

	width := aoi.Width
	if width == 0 { // if the width is zero, the width will span from left~end of chip
		width, err := c.GetSensorWidth()
		if err != nil {
			return err
		}
		width -= c.aoiLeft
	}
	err = SetInt(c.Handle, "AOIWidth", int64(width))
	if err != nil {
		return err
	}
	c.aoiWidth = width

	height := aoi.Height
	if height == 0 { // if the height is zero, the height will span from top~end of chip
		height, err := c.GetSensorHeight()
		if err != nil {
			return err
		}
		height -= c.aoiTop
	}
	err = SetInt(c.Handle, "AOIHeight", int64(height))
	if err != nil {
		return err
	}
	c.aoiHeight = height

	// blow the image size cache
	c.imageSizeBytes = 0
	err = c.Allocate()
	if err != nil {
		return err
	}
	err = c.QueueBuffer()
	return err
}

// GetAOI gets the AOI
func (c *Camera) GetAOI() (AOI, error) {
	// no point bailing early since these will all throw the same error if
	// they do at all
	top, err := c.GetAOITop()
	left, err := c.GetAOILeft()
	width, err := c.GetAOIWidth()
	height, err := c.GetAOIHeight()
	return AOI{Top: top, Left: left, Width: width, Height: height}, err
}

// GetSDKVersion gets the software version of the SDK
func (c *Camera) GetSDKVersion() (string, error) {
	var s string
	var err error
	if c.sdkver == "" {
		s, err = SoftwareVersion()
		c.sdkver = s
	} else {
		s = c.sdkver
		err = nil
	}
	return s, err
}

// GetBinning gets the binning
func (c *Camera) GetBinning() (Binning, error) {
	if c.binning.H == 0 {
		b := Binning{}
		// uninitialized, fetch from SDK
		s, err := GetString(c.Handle, "AOIBinning")
		if err != nil {
			return b, err
		}
		b = ParseBinning(s)
		c.binning = b
	}
	return c.binning, nil
}

// SetBinning sets the AOIBinning feature
func (c *Camera) SetBinning(b Binning) error {
	// blow the image size cache
	c.imageSizeBytes = 0
	str := FormatBinning(b)
	err := enrich(SetString(c.Handle, "AOIBinning", str), "AOIBinning")
	if err != nil {
		return err
	}
	return c.Allocate()
}

// GetFirmwareVersion gets the firmware version of the camera
func (c *Camera) GetFirmwareVersion() (string, error) {
	var s string
	var err error
	if c.fwver == "" {
		s, err = GetString(c.Handle, "FirmwareVersion")
		c.fwver = s
	} else {
		s = c.fwver
		err = nil
	}
	return s, err
}

// GetDriverVersion gets the software version of the SDK
func (c *Camera) GetDriverVersion() (string, error) {
	var s string
	var err error
	if c.drvver == "" {
		s, err = GetString(c.Handle, "DriverVersion")
		c.drvver = s
	} else {
		s = c.drvver
		err = nil
	}
	return s, err
}

// GetModel returns the model string
func (c *Camera) GetModel() (string, error) {
	var model string
	var err error
	if c.model == "" {
		model, err = GetString(c.Handle, "CameraModel")
		c.model = model
	} else {
		model = c.model
		err = nil
	}
	return model, err
}

// GetSerialNumber return the serial number
func (c *Camera) GetSerialNumber() (string, error) {
	var serial string
	var err error
	if c.serial == "" {
		serial, err = GetString(c.Handle, "SerialNumber")
		c.serial = serial
	} else {
		serial = c.serial
		err = nil
	}
	return serial, err
}

// QueueBuffer puts the Camera's internal buffer into the write queue for the SDK
// only one buffer is supported in this wrapper, though the SDK supports
// multiple buffers
func (c *Camera) QueueBuffer() error {
	if len(c.buffer) == 0 {
		return fmt.Errorf("Go buffer cannot hold entire frame, likely uninitialized, len=%d, cap=%d", len(c.buffer), cap(c.buffer))
	}
	err := Error(int(C.AT_QueueBuffer(C.AT_H(c.Handle), c.cptr, c.cptrsize)))
	if err == nil {
		c.bufferOnQueue = true
	}
	return err
}

// WaitBuffer waits for the camera to push a frame into the buffer
// errors if Queue has not been called, on timeout, or on an SDK error
func (c *Camera) WaitBuffer(timeout time.Duration) error {
	if !c.bufferOnQueue {
		return ErrBufferNotOnQueue
	}
	tout := C.uint(timeout.Nanoseconds() / 1e6)
	var (
		size C.int
		ptr  *C.AT_U8
	)
	err := Error(int(C.AT_WaitBuffer(C.AT_H(c.Handle), &ptr, &size, tout)))
	return err
}

// GetFrame triggers an exposure and returns the frame as a strided slice of bytes
func (c *Camera) GetFrame() ([]uint16, error) {
	if !c.bufferOnQueue {
		return nil, ErrBufferNotOnQueue
	}
	// if we have to query hardware for exposure time, there may be an error
	expT, err := c.GetExposureTime()
	if err != nil {
		return []uint16{}, err
	}

	// injected here 2019-12-03, not writable on AcqStart happens because
	// CameraAcquiring is true
	IssueCommand(c.Handle, "AcquisitionStop") // gobble any errors from this

	// do the big acquisition loop
	err = c.QueueBuffer()
	if err != nil {
		return []uint16{}, err
	}

	err = IssueCommand(c.Handle, "AcquisitionStart")
	if err != nil {
		return []uint16{}, err
	}
	err = c.WaitBuffer(expT + 1*time.Second)
	if err != nil {
		return []uint16{}, err
	}
	err = IssueCommand(c.Handle, "AcquisitionStop")
	if err != nil {
		return []uint16{}, err
	}
	buf, err := c.unpadBuffer()
	if err != nil {
		return []uint16{}, err
	}
	ary := bytesToUint(buf)
	return ary, nil
}

// Burst performs a burst by taking N images at M fps
func (c *Camera) Burst(frames int, fps float64) ([]uint16, error) {
	var holding []byte
	var output []uint16

	// get the previous framerate so we can reset to this like a good neighbor
	prevFps, err := GetFloat(c.Handle, "FrameRate")
	if err != nil {
		return output, err
	}

	prevCycle, err := GetEnumString(c.Handle, "CycleMode")
	if err != nil {
		return output, err
	}

	IssueCommand(c.Handle, "AcquisitionStop")

	err = c.QueueBuffer()
	if err != nil {
		return output, err
	}
	fmt.Println(GetBool(c.Handle, "CameraAcquiring"))
	err = SetEnumString(c.Handle, "CycleMode", "Continuous")
	if err != nil {
		return output, err
	}
	// now start acq and begin handling buffers
	err = SetFloat(c.Handle, "FrameRate", fps)
	if err != nil {
		return output, err
	}

	// get the exposure time so we know how long to wait for a buffer
	expT, err := c.GetExposureTime()
	if err != nil {
		return output, err
	}
	waitT := expT + time.Second

	err = IssueCommand(c.Handle, "AcquisitionStart")

	for idx := 0; idx < frames; idx++ {
		err = c.QueueBuffer()
		if err != nil {
			return output, err
		}
		err := c.WaitBuffer(waitT)
		if err != nil {
			return output, err
		}
		buf, err := c.unpadBuffer()
		if err != nil {
			return output, err
		}
		if idx == 0 { // on the first go around, allocate a big buffer to hold the cube
			holding = make([]byte, 0, len(buf)*frames)
		}
		// every time, add to the buffer
		holding = append(holding, buf...)
	}
	output = bytesToUint(holding)

	// finally, reset the camera to how it was when we started
	err = SetFloat(c.Handle, "FrameRate", prevFps)
	err = SetEnumString(c.Handle, "CycleMode", prevCycle)
	return output, err
}

func (c *Camera) unpadBuffer() ([]byte, error) {
	buf, err := c.Buffer()
	if err != nil {
		return []byte{}, err
	}
	stride, err := c.GetAOIStride()
	if err != nil {
		return []byte{}, err
	}
	width, err := c.GetAOIWidth()
	if err != nil {
		return []byte{}, err
	}
	height, err := c.GetAOIHeight()
	if err != nil {
		return []byte{}, err
	}
	return UnpadBuffer(buf, stride, width, height)
}

// GetExposureTime gets the current exposure time as a duration
func (c *Camera) GetExposureTime() (time.Duration, error) {
	var err error
	if c.exposureTime == time.Duration(0) { // zero value, uninitialized
		tS, err := GetFloat(c.Handle, "ExposureTime")
		// convert to ns then round to int and make a duration
		tNsI := int(math.Round(tS * 1e9)) // * 1e9 seconds -> ns
		dur := time.Duration(tNsI) * time.Nanosecond
		if err == nil {
			c.exposureTime = dur
		}
	}
	return c.exposureTime, err
}

// SetExposureTime sets the exposure time as a duration
func (c *Camera) SetExposureTime(d time.Duration) error {
	ts := d.Seconds()
	err := SetFloat(c.Handle, "ExposureTime", ts)
	if err == nil {
		c.exposureTime = d
	}
	return err
}

// GetCooling gets if temperature control is currently active or not
func (c *Camera) GetCooling() (bool, error) {
	return GetBool(c.Handle, "SensorCooling")
}

// SetCooling sets if temperature control is currently active or not
func (c *Camera) SetCooling(b bool) error {
	return SetBool(c.Handle, "SensorCooling", b)
}

// GetTemperature gets the current temperature of the sensor in Celcius
func (c *Camera) GetTemperature() (float64, error) {
	return GetFloat(c.Handle, "SensorTemperature")
}

// GetTemperatureSetpoints gets a list of strings representing the
// temperatures the detector can currently be cooled to
func (c *Camera) GetTemperatureSetpoints() ([]string, error) {
	return GetEnumStrings(c.Handle, "TemperatureControl")
}

// GetTemperatureSetpoint gets the temp control setpoint as a string
func (c *Camera) GetTemperatureSetpoint() (string, error) {
	return GetEnumString(c.Handle, "TemperatureControl")
}

// SetTemperatureSetpoint sets the temp control point to a value that is returned by
// GetTemperatureSetpoints
func (c *Camera) SetTemperatureSetpoint(s string) error {
	return SetEnumString(c.Handle, "TemperatureControl", s)
}

// GetTemperatureStatus gets the current status of sensor cooling.  One of:
// - Cooler Off
// - Stabilised
// - Cooling
// - Drift
// - Not Stabilised
// - Fault
func (c *Camera) GetTemperatureStatus() (string, error) {
	return GetEnumString(c.Handle, "TemperatureStatus")
}

// GetFan gets if the fan is currently on
func (c *Camera) GetFan() (bool, error) {
	speed, err := GetEnumString(c.Handle, "FanSpeed")
	return speed != "Off", err
}

// SetFan sets the fan on or off
func (c *Camera) SetFan(b bool) error {
	var str string
	if b == true {
		str = "On"
	} else {
		str = "Off"
	}
	return SetEnumString(c.Handle, "FanSpeed", str)
}

// Buffer the current buffer at this moment in time.  This is technically a copy
// but go slices are allocated on the heap, so it only copies the header with
// minimal performance impact.
//
// may have undefined behavior if camera is writing while you read
func (c *Camera) Buffer() ([]byte, error) {
	// this function is needed because we use a buffer of uint64 to
	// guarantee 8-byte alignment.  We want the underlying data
	buf := []byte{}
	nbytes, err := GetInt(c.Handle, "ImageSizeBytes")
	if err != nil {
		return buf, err
	}
	hdr := (*reflect.SliceHeader)(unsafe.Pointer(&buf))
	hdr.Data = uintptr(unsafe.Pointer(&c.buffer[0]))
	hdr.Len = nbytes
	hdr.Cap = nbytes
	return buf, nil
}

// Command issues a command to this camera's handle
func (c *Camera) Command(cmd string) error {
	return IssueCommand(c.Handle, cmd)
}

// CollectHeaderMetadata satisfies generichttp/camera and makes a stack of FITS cards
func (c *Camera) CollectHeaderMetadata(c *Camera) []fitsio.Card {
	// grab all the shit we care about from the camera so we can fill out the header
	// plow through errors, no need to bail early
	aoi, err := c.GetAOI()
	texp, err := c.GetExposureTime()
	sdkver, err := c.GetSDKVersion()
	drvver, err := c.GetDriverVersion()
	firmver, err := c.GetFirmwareVersion()
	cammodel, err := c.GetModel()
	camsn, err := c.GetSerialNumber()
	fan, err := c.GetFan()
	tsetpt, err := c.GetTemperatureSetpoint()
	tstat, err := c.GetTemperatureStatus()
	temp, err := c.GetTemperature()
	bin, err := c.GetBinning()
	binS := FormatBinning(bin)

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
		- go-hcit andor version
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
		fitsio.Card{Name: "HDRVER", Value: "3", Comment: "header version"},
		fitsio.Card{Name: "WRAPVER", Value: WRAPVER, Comment: "server library code version"},
		fitsio.Card{Name: "SDKVER", Value: sdkver, Comment: "sdk version"},
		fitsio.Card{Name: "DRVVER", Value: drvver, Comment: "driver version"},
		fitsio.Card{Name: "FIRMVER", Value: firmver, Comment: "camera firmware version"},
		fitsio.Card{Name: "METAERR", Value: metaerr, Comment: "error encountered gathering metadata"},
		fitsio.Card{Name: "CAMMODL", Value: cammodel, Comment: "camera model"},
		fitsio.Card{Name: "CAMSN", Value: camsn, Comment: "camera serial number"},

		// timestamp
		fitsio.Card{Name: "DATE", Value: ts}, // timestamp is standard and does not require comment

		// exposure parameters
		fitsio.Card{Name: "EXPTIME", Value: texp.Seconds(), Comment: "exposure time, seconds"},

		// thermal parameters
		fitsio.Card{Name: "FAN", Value: fan, Comment: "on (true) or off"},
		fitsio.Card{Name: "TEMPSETP", Value: tsetpt, Comment: "Temperature setpoint"},
		fitsio.Card{Name: "TEMPSTAT", Value: tstat, Comment: "TEC status"},
		fitsio.Card{Name: "TEMPER", Value: temp, Comment: "FPA temperature (Celcius)"},
		// aoi parameters
		fitsio.Card{Name: "AOIL", Value: aoi.Left, Comment: "1-based left pixel of the AOI"},
		fitsio.Card{Name: "AOIT", Value: aoi.Top, Comment: "1-based top pixel of the AOI"},
		fitsio.Card{Name: "AOIW", Value: aoi.Width, Comment: "AOI width, px"},
		fitsio.Card{Name: "AOIH", Value: aoi.Height, Comment: "AOI height, px"},
		fitsio.Card{Name: "AOIB", Value: binS, Comment: "AOI Binning, HxV"},

		// needed for uint16 encoding
		fitsio.Card{Name: "BZERO", Value: 32768},
		fitsio.Card{Name: "BSCALE", Value: 1.0}}
}

// Configure takes a map of interfaces and calls Set_xxx for each, where
// xxx is Bool, Int, etc.
func (c *Camera) Configure(settings map[string]interface{}) error {
	errs := []error{}
	for k, v := range settings {
		typs := Features[k]
		var err error
		err = nil
		switch typs {
		case "int":
			// values will unmarshal to unsized ints, assert to int then cast
			// to i64
			err = SetInt(c.Handle, k, int64(v.(int)))
		case "float":
			err = SetFloat(c.Handle, k, v.(float64))
		case "bool":
			err = SetBool(c.Handle, k, v.(bool))
		case "enum":
			err = SetEnumString(c.Handle, k, v.(string))
		default:
			err = fmt.Errorf("value %v for key %s is not of type int, float64, bool, or string", v, k)
		}
		errs = append(errs, err)
	}
	strs := []string{}
	for idx := 0; idx < len(errs); idx++ {
		err := errs[idx]
		if err != nil {
			strs = append(strs, err.Error())
		}
	}
	err := fmt.Errorf(strings.Join(strs, "\n"))
	if err.Error() == "" {
		return nil
	}
	return err
}

// UnpadBuffer strips padding bytes from a buffer
func UnpadBuffer(buf []byte, aoistride, aoiwidth, aoiheight int) ([]byte, error) {
	// TODO: this allocates something bigger than needed
	// can improve performance a little bit by changing this
	out := make([]byte, 0, len(buf))

	// TODO: generalize this to other modes besides 16-bit
	bidx := 0                       // byte index
	bpp := 2                        // bytes per pixel
	rowWidthBytes := bpp * aoiwidth // width (stride) or a row in bytes
	// implicitly row major order, but seems to be from the SDK
	for row := 0; row < aoiheight; row++ {
		bytes := buf[bidx : bidx+rowWidthBytes]
		out = append(out, bytes...)
		// finally, move
		bidx += aoistride // stride is the padded stride
	}
	return out, nil
}

func bytesToUint(b []byte) []uint16 {
	ary := []uint16{}
	hdr := (*reflect.SliceHeader)(unsafe.Pointer(&ary))
	hdr.Data = uintptr(unsafe.Pointer(&b[0]))
	hdr.Len = len(b) / 2
	hdr.Cap = cap(b) / 2
	return ary
}
