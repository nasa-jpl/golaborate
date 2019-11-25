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
	"time"
	"unsafe"

	"github.jpl.nasa.gov/HCIT/go-hcit/mathx"
)

const (
	// LengthOfUndefinedBuffers is how large a buffer to allocate for a Wchar
	// string when we have no way of knowing ahead of time how big it is
	// it is measured in Wchars
	LengthOfUndefinedBuffers = 255
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
		// "TargetSensorTemperature":  "float", removed 2019-11-25, deprecated by Andorj

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

	// ExposureTime is the currently programmed exposure time.
	ExposureTime time.Duration
}

// Allocate creates the buffer that will be populated by the SDK
// it should be called at init, and whenever the AOI or encoding changes
func (c *Camera) Allocate() error {
	sze, err := GetInt(c.Handle, "ImageSizeBytes")
	if err != nil {
		return err
	}
	c.buffer = make([]uint64, sze/8) // uint64 forces byte alignment, 8 bytes per uint64
	c.gptr = unsafe.Pointer(&c.buffer[0])
	c.cptr = (*C.AT_U8)(c.gptr)
	c.cptrsize = C.int(sze)
	return nil
}

// Open opens a connection to the camera.  camIdx 1 is the system handle,
// start at 2 for the first camera, 3 for the second, and so forth
func Open(camIdx int) (*Camera, error) {
	c := Camera{}
	var hndle C.AT_H
	err := Error(int(C.AT_Open(C.int(camIdx), &hndle)))
	c.Handle = int(hndle)
	return &c, err
}

// Close closes a connection to the camera
func (c *Camera) Close() error {
	return Error(int(C.AT_Close(C.AT_H(c.Handle))))
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

	// do the big acquisition loop
	err = c.QueueBuffer()
	if err != nil {
		return []uint16{}, err
	}
	err = IssueCommand(c.Handle, "AcquisitionStart")
	if err != nil {
		return []uint16{}, err
	}
	// wait a multiple of the shutter time, or a fixed time if that is too short for the SDK to be stable
	minWait := 100 * time.Millisecond
	Wait := minWait

	calcWait := expT * 3
	if calcWait > Wait {
		Wait = calcWait
	}
	err = c.WaitBuffer(Wait)
	if err != nil {
		return []uint16{}, err
	}
	err = IssueCommand(c.Handle, "AcquisitionStop")
	if err != nil {
		return []uint16{}, err
	}
	buf, err := c.BufferCopy()
	if err != nil {
		return []uint16{}, err
	}
	buf, err = c.UnpadBuffer(buf)
	if err != nil {
		return []uint16{}, err
	}
	ary := bytesToUint(buf)
	return ary, nil
}

// GetExposureTime gets the current exposure time as a duration
func (c *Camera) GetExposureTime() (time.Duration, error) {
	var err error
	if c.ExposureTime == time.Duration(0) { // zero value, uninitialized
		tS, err := GetFloat(c.Handle, "ExposureTime")
		// convert to ns then round to int and make a duration
		tNs := tS * 1e9
		tNsI := int(mathx.Round(tNs, 0))
		dur := time.Duration(tNsI) * time.Nanosecond
		if err == nil {
			c.ExposureTime = dur
		}
	}
	return c.ExposureTime, err
}

// SetExposureTime sets the exposure time as a duration
func (c *Camera) SetExposureTime(d time.Duration) error {
	ts := d.Seconds()
	err := SetFloat(c.Handle, "ExposureTime", ts)
	if err == nil {
		c.ExposureTime = d
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

// GetFanOn gets if the fan is currently on
func (c *Camera) GetFanOn() (bool, error) {
	speed, err := GetEnumString(c.Handle, "FanSpeed")
	return speed == "Off", err
}

// SetFanOn sets the fan on or off
func (c *Camera) SetFanOn(b bool) error {
	return SetEnumString(c.Handle, "FanSpeed", "On")
}

// BufferCopy returns a copy of the current buffer at this moment in time
// may have undefined behavior if camera is writing while you read
//
// This is a copy, so do not do this inside of a a hot loop (e.g. running at 100fps)
func (c *Camera) BufferCopy() ([]byte, error) {
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

// ShutDown shuts down the camera and 'finalizes' the SDK.
// This maintains API compatibility with SDK2
// and is not an exact match of
func (c *Camera) ShutDown() {
	return
}

// JPLNeoBootup runs the standard bootup sequence used for Andor Neo cameras in HCIT
// this should be called after Initialize
func JPLNeoBootup(c *Camera) error {
	fmt.Println("eshutter")
	err := SetEnumString(c.Handle, "ElectronicShutteringMode", "Rolling")
	if err != nil {
		return err
	}
	fmt.Println("preampgain skipped - notimplemented on simcam")
	err = SetEnumString(c.Handle, "SimplePreAmpGainControl", "16-bit (low noise & high well capacity)")
	if err != nil {
		return err
	}
	fmt.Println("fan speed")
	err = SetEnumString(c.Handle, "FanSpeed", "Off")
	if err != nil {
		return err
	}
	fmt.Println("pixel readout rate")
	err = SetEnumString(c.Handle, "PixelReadoutRate", "280 MHz")
	// err = SetEnumIndex(c.Handle, "PixelReadoutRate", 0)
	if err != nil {
		return err
	}

	fmt.Println("pixel encoding")
	// err = SetEnumIndex(c.Handle, "PixelEncoding", 2)
	err = SetEnumString(c.Handle, "PixelEncoding", "Mono16")
	strs, err := GetEnumStrings(c.Handle, "PixelEncoding")
	fmt.Println(strs)
	if err != nil {
		return err
	}

	fmt.Println("triggermode")
	err = SetEnumString(c.Handle, "TriggerMode", "Internal")
	if err != nil {
		return err
	}

	fmt.Println("metadata - skipped simcam")
	err = SetBool(c.Handle, "MetadataEnable", false)
	if err != nil {
		return err
	}

	// fmt.Println("metadatats")
	// err = SetBool(c.Handle, "MetadataTimestamp", true)
	// if err != nil {
	// 	return err
	// }

	fmt.Println("cooling")
	err = SetBool(c.Handle, "SensorCooling", false)
	if err != nil {
		return err
	}

	fmt.Println("spurriousnoise - skipped simcak")
	// err = SetBool(c.Handle, "SpuriousNoiseFilter", false)
	// if err != nil {
	// 	return err
	// }

	return nil
}

// UnpadBuffer strips padding bytes from a buffer
func (c *Camera) UnpadBuffer(buf []byte) ([]byte, error) {
	// TODO: this allocates something bigger than needed
	// can improve performance a little bit by changing this
	out := make([]byte, 0, len(buf))
	stride, err := GetInt(c.Handle, "AOIStride")
	if err != nil {
		return out, err
	}
	width, err := GetInt(c.Handle, "AOIWidth")
	if err != nil {
		return out, err
	}
	height, err := GetInt(c.Handle, "AOIHeight")
	if err != nil {
		return out, err
	}

	// TODO: generalize this to other modes besides 16-bit
	bidx := 0                    // byte index
	bpp := 2                     // bytes per pixel
	rowWidthBytes := bpp * width // width (stride) or a row in bytes
	// implicitly row major order, but seems to be from the SDK
	for row := 0; row < height; row++ {
		bytes := buf[bidx : bidx+rowWidthBytes]
		out = append(out, bytes...)
		// finally, move
		bidx += stride // stride is the padded stride
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
