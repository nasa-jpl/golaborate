/*Package sdk3 exposes control of Andor cameras in Go via their SDK, v3.

 */
package sdk3

/*
#cgo CFLAGS: -I/usr/local
#cgo LDFLAGS: -L/usr/local/lib -latcore
#include <stdlib.h>
#include <atcore.h>

*/
import "C"
import (
	"errors"
	"fmt"
	"reflect"
	"time"
	"unsafe"

	cwch "github.com/lordadamson/cgo.wchar"
)

var (
	// ErrBufferNotOnQueue is generated before a catastrophic side effect is triggered
	ErrBufferNotOnQueue = errors.New("no buffer placed on queue, this error saves you from memory corruption")

	// ErrCodes is a map of error codes (ints) to error strings
	ErrCodes = map[DRVError]string{
		0:  "AT_SUCCESS",
		1:  "AT_ERR_NOT_INITIALISED", // added _ after NOT
		2:  "AT_ERR_NOT_IMPLEMENTED",
		3:  "AT_ERR_READONLY",
		4:  "AT_ERR_NOT_READABLE",               // added _ after NOT
		5:  "AT_ERR_NOT_WRITABLE",               // added _ after NOT
		6:  "AT_ERR_OUT_OF_RANGE",               // added two _'s
		7:  "AT_ERR_INDEX_NOT_AVAILABLE",        // added two _'s
		8:  "AT_ERR_INDEX_NOT_IMPLEMENTED",      // added two _'s
		9:  "AT_ERR_EXCEEDED_MAX_STRING_LENGTH", // added three _'s
		10: "AT_ERR_CONNECTION",
		11: "AT_ERR_NO_DATA",           // added a _
		12: "AT_ERR_INVALID_HANDLE",    // added a _
		13: "AT_ERR_TIMED_OUT",         // added a _
		14: "AT_ERR_BUFFER_FULL",       // added a _
		15: "AT_ERR_INVALID_SIZE",      // added a _
		16: "AT_ERR_INVALID_ALIGNMENT", // added a _
		17: "AT_ERR_COMM",
		18: "AT_ERR_STRING_NOT_AVAILABLE",   // added two _
		19: "AT_ERR_STRING_NOT_IMPLEMENTED", // added two _
		20: "AT_ERR_NULL_FEATURE",
		21: "AT_ERR_NULL_HANDLE",
		22: "AT_ERR_NULL_IMPLEMENTED_VAR",
		23: "AT_ERR_NULL_READABLE_VAR",
		24: "AT_ERR_NULL_READONLY_VAR",
		25: "AT_ERR_NULL_WRITABLE_VAR",
		26: "AT_ERR_NULL_MIN_VALUE", // added a _
		27: "AT_ERR_NULL_MAX_VALUE", // added a _
		28: "AT_ERR_NULL_VALUE",
		29: "AT_ERR_NULL_STRING",
		30: "AT_ERR_NULL_COUNT_VAR",
		31: "AT_ERR_NULL_IS_AVAILABLE_VAR",  // added a _
		32: "AT_ERR_NULL_MAX_STRING_LENGTH", // aded two _
		33: "ATT_ERR_NULL_EV_CALLBACK",      // added a _
		34: "AT_ERR_NULL_QUEUE_PTR",
		35: "AT_ERR_NULL_WAIT_PTR",
		36: "AT_ERR_NULL_PTR_SIZE",    // added a _
		37: "AT_ERR_NO_MEMORY",        // added a _
		38: "AT_ERR_DEVICE_IN_USE",    // added two _
		39: "AT_ERR_DEVICE_NOT_FOUND", // added two _

		100: "AT_ERR_HARDWARE_OVERFLOW",
	}

	// IntegerFeatures is a map of strings to true
	// this just gives us faster lookup in exchange
	// for a little messier godoc.
	IntegerFeatures = map[string]bool{
		"AccumulatedCount":        true,
		"AOIHBin":                 true,
		"AOIVBin":                 true,
		"AOILeft":                 true,
		"AOITop":                  true,
		"AOIStride":               true,
		"AOIWidth":                true,
		"BaselineLevel":           true,
		"BufferOverflowEvent":     true,
		"DeviceCount":             true,
		"DeviceVideoIndex":        true,
		"EventsMissedEvent":       true,
		"ExposureStartEvent":      true,
		"ExposureEndEvent":        true,
		"FrameCount":              true,
		"ImageSizeBytes":          true,
		"LUTIndex":                true,
		"LUTValue":                true,
		"RowNExposureEndEvent":    true,
		"RowNExposureStartEvent":  true,
		"SensorHeight":            true,
		"SensorWidth":             true,
		"TimestampClock":          true,
		"TimestampClockFrequency": true,
	}

	// BoolFeatures is a map of strings to true
	// this just gives faster lookup in exchange
	// for a little messier godoc.
	// The value of the map means the feature is
	// valid, and is not the value of the feature
	BoolFeatures = map[string]bool{
		"CameraAcquiring":       true,
		"EventEnable":           true,
		"FullAOIControl":        true,
		"IOInvert":              true,
		"MetadataEnable":        true,
		"MetadataFrame":         true,
		"MetadataTimestamp":     true,
		"Overlap":               true, // TODO: see if enabling this fixes fast shutter problems
		"SensorCooling":         true,
		"SpuriousNoiseFilter":   true,
		"SynchronousTriggering": true,
	}

	// CommandFeatures is a map of strings to true
	// this just gives faster lookup in exchange
	// for a little messier godoc.
	CommandFeatures = map[string]bool{
		"AcquisitionStart":    true,
		"AcquisitionStop":     true,
		"CameraDump":          true,
		"SoftwareTrigger":     true,
		"TimestampClockReset": true,
	}

	// FloatFeatures is a map of strings to true
	// this just gives faster lookup in exchange
	// for a little messier godoc.
	FloatFeatures = map[string]bool{
		"BytesPerPixel":            true,
		"ExposureTime":             true,
		"FrameRate":                true,
		"MaxInterfaceTransferRate": true,
		"PixelHeight":              true,
		"PixelWidth":               true,
		"ReadoutTime":              true,
		"SensorTemperature":        true,
		"TargetSensorTemperature":  true,
	}
)

// DRVError represents a driver error
type DRVError int

func (e DRVError) Error() string {
	if s, ok := ErrCodes[e]; ok {
		return fmt.Sprintf("%v - %s", e, s)
	}
	return fmt.Sprintf("%v - UNKNOWN_ERROR_CODE", e)
}

// Error returns nil on beneign error codes or returns an error object on non-beneign ones
func Error(code int) error {
	if code == 0 {
		return nil
	}
	return DRVError(code)
}

func boolToAT(b bool) C.AT_BOOL {
	if b {
		return C.AT_TRUE
	}
	return C.AT_FALSE
}

func atToBool(b C.AT_BOOL) bool {
	if b == C.AT_TRUE {
		return true
	}
	return false
}

// InitializeLibrary calls the function of the same name in the Andor SDK
func InitializeLibrary() error {
	return Error(int(C.AT_InitialiseLibrary()))
}

// FinalizeLibrary calls the function of the same name in the Andor SDK
func FinalizeLibrary() {
	C.AT_FinaliseLibrary()
}

// DeviceCount returns the number of devices (cameras) found by the SDK
// InitializeLibrary must be called first
func DeviceCount() (int, error) {
	return GetInt(C.AT_HANDLE_SYSTEM, "DeviceCount")
}

// SoftwareVersion returns the software (SDK) version
// InitializeLibrary must be called first
func SoftwareVersion() (string, error) {
	return GetString(C.AT_HANDLE_SYSTEM, "SoftwareVersion")
}

// SetInt sets an integer
func SetInt(handle C.AT_H, feature string, val int64) error {
	cstr, err := cwch.FromGoString(feature)
	if err != nil {
		return err
	}
	str := (*C.AT_WC)(cstr.Pointer())
	return Error(int(C.AT_SetInt(handle, str, C.AT_64(val))))
}

// GetInt gets an integer
func GetInt(handle C.AT_H, feature string) (int, error) {
	cstr, err := cwch.FromGoString(feature)
	if err != nil {
		return 0, err
	}
	str := (*C.AT_WC)(cstr.Pointer())
	var out C.AT_64
	errCode := int(C.AT_GetInt(handle, str, &out))
	return int(out), Error(errCode)
}

// GetIntMax gets the max value an integer can be set to
func GetIntMax(handle C.AT_H, feature string) (int, error) {
	// this code is identical to GetInt except for the C call
	cstr, err := cwch.FromGoString(feature)
	if err != nil {
		return 0, err
	}
	str := (*C.AT_WC)(cstr.Pointer())

	var out C.AT_64
	errCode := int(C.AT_GetIntMax(handle, str, &out))
	return int(out), Error(errCode)
}

// GetIntMin gets the min value an integer can be set to
func GetIntMin(handle C.AT_H, feature string) (int, error) {
	// this code is identical to GetInt except for the C call
	cstr, err := cwch.FromGoString(feature)
	if err != nil {
		return 0, err
	}
	str := (*C.AT_WC)(cstr.Pointer())

	var out C.AT_64
	errCode := int(C.AT_GetIntMin(handle, str, &out))
	return int(out), Error(errCode)
}

// SetFloat sets a floating point value
func SetFloat(handle C.AT_H, feature string, value float64) error {
	cstr, err := cwch.FromGoString(feature)
	if err != nil {
		return err
	}
	str := (*C.AT_WC)(cstr.Pointer())

	return Error(int(C.AT_SetFloat(handle, str, C.double(value))))
}

// GetFloat gets a floating point value
func GetFloat(handle C.AT_H, feature string) (float64, error) {
	cstr, err := cwch.FromGoString(feature)
	if err != nil {
		return 0, err
	}
	str := (*C.AT_WC)(cstr.Pointer())

	var out C.double
	errCode := int(C.AT_GetFloat(handle, str, &out))
	return float64(out), Error(errCode)
}

// GetFloatMax gets the maximum of a floating point value
func GetFloatMax(handle C.AT_H, feature string) (float64, error) {
	// this code is identical to GetFloat except for the C call
	cstr, err := cwch.FromGoString(feature)
	if err != nil {
		return 0, err
	}
	str := (*C.AT_WC)(cstr.Pointer())

	var out C.double
	errCode := int(C.AT_GetFloatMax(handle, str, &out))
	return float64(out), Error(errCode)
}

// GetFloatMin gets the minimum of a floating point value
func GetFloatMin(handle C.AT_H, feature string) (float64, error) {
	// this code is identical to GetFloat except for the C call
	cstr, err := cwch.FromGoString(feature)
	if err != nil {
		return 0, err
	}
	str := (*C.AT_WC)(cstr.Pointer())

	var out C.double
	errCode := int(C.AT_GetFloatMin(handle, str, &out))
	return float64(out), Error(errCode)
}

// SetBool sets a boolean feature
func SetBool(handle C.AT_H, feature string, tru bool) error {
	// this code is identical to GetFloat except for the C call
	cstr, err := cwch.FromGoString(feature)
	if err != nil {
		return err
	}
	str := (*C.AT_WC)(cstr.Pointer())
	return Error(int(C.AT_SetBool(handle, str, boolToAT(tru))))
}

// GetBool gets the value of a boolean feature
func GetBool(handle C.AT_H, feature string) (bool, error) {
	// this code is identical to GetFloat except for the C call
	cstr, err := cwch.FromGoString(feature)
	if err != nil {
		return false, err
	}
	str := (*C.AT_WC)(cstr.Pointer())
	var b C.AT_BOOL
	errCode := int(C.AT_GetBool(handle, str, &b))
	return atToBool(b), Error(errCode)
}

// SetString sets the value of a string
func SetString(handle C.AT_H, feature, value string) error {
	cstr, err := cwch.FromGoString(feature)
	if err != nil {
		return err
	}
	featstr := (*C.AT_WC)(cstr.Pointer())

	cstr, err = cwch.FromGoString(value)
	if err != nil {
		return err
	}
	valstr := (*C.AT_WC)(cstr.Pointer())

	return Error(int(C.AT_SetString(handle, featstr, valstr)))
}

// GetStringMaxLength returns the length of a string, use this to determine how big
// of a cgo.widechar string to allocate
func GetStringMaxLength(handle C.AT_H, feature string) (int, error) {
	cstr, err := cwch.FromGoString(feature)
	if err != nil {
		return 0, err
	}
	str := (*C.AT_WC)(cstr.Pointer())

	var len C.int
	errCode := int(C.AT_GetStringMaxLength(handle, str, &len))
	return int(len), Error(errCode)
}

// GetString returns the string value of a feature
// we deviate from SDK3 API by using GetStringMaxLength internally
// and avoid users having to deal with allocating C.wchar_t buffers
func GetString(handle C.AT_H, feature string) (string, error) {
	cstr, err := cwch.FromGoString(feature)
	if err != nil {
		return "", err
	}
	strc := (*C.AT_WC)(cstr.Pointer())

	size, err := GetStringMaxLength(handle, feature)
	if err != nil {
		return "", err
	}
	// this line may be a source of bugs
	stro := cwch.NewWcharString(size)
	outp := (*C.AT_WC)(stro.Pointer())
	errCode := int(C.AT_GetString(handle, strc, outp, C.int(size)))

	str, err := stro.GoString()
	if err != nil {
		return "", err
	}
	return str, Error(errCode)
}

// Camera represents a camera from SDK3
type Camera struct {
	// buffer is written to by the SDK.
	// It must be 8-byte aligned.
	buffer []uint64

	// cptr is a C pointer to the first byte in buffer
	cptr *C.AT_U8

	// gptr is a Go pointer to the first byte in buffer
	gptr unsafe.Pointer

	// bufferOnQueue is a flag indicating if
	bufferOnQueue bool

	// Resolution holds the sensor resolution (H, W)
	Resolution [2]int

	// handle holds the C int that points to a specific camera
	handle C.AT_H
}

// npx is shorthand for c.Resolution[0] * c.Resolution[1]
func (c *Camera) npx() int {
	return c.Resolution[0] * c.Resolution[1]
}

// Open opens a connection to the camera
func (c *Camera) Open(camIdx int) error {
	return Error(int(C.AT_Open(C.int(camIdx), &c.handle)))
}

// Close closes a connection to the camera
func (c *Camera) Close() error {
	return Error(int(C.AT_Close(c.handle)))
}

// Queue puts the Camera's internal buffer into the write queue for the SDK
// only one buffer is supported in this wrapper, though the SDK supports
// multiple buffers
func (c *Camera) Queue() error {
	npx := c.npx()
	if len(c.buffer) == 0 || cap(c.buffer) < npx {
		return fmt.Errorf("Go buffer cannot hold entire frame, likely uninitialized, len=%d, cap=%d, npx=%d", len(c.buffer), cap(c.buffer), npx)
	}

	// with help from Bryan C. Mills on slack@gophers
	var (
		ptr     *C.AT_U8
		ptrSize C.int
	)
	// birth of c.cptr:
	// c.buffer[0] <- this is a Go byte
	// unsafe.Pointer(...) <- this is a Go pointer
	// (*C.AT_U8) <- this casts to a C pointer to a C byte
	c.cptr = (*C.AT_U8)(unsafe.Pointer(&c.buffer[0]))
	ptrSize = C.int(npx)
	c.bufferOnQueue = true

	return Error(int(C.AT_QueueBuffer(c.handle, ptr, ptrSize)))
}

// WaitBuffer waits for the camera to push a frame into the buffer
// errors if Queue has not been called, on timeout, or on an SDK error
func (c *Camera) WaitBuffer(timeout time.Duration) error {
	if !c.bufferOnQueue {
		return ErrBufferNotOnQueue
	}
	tout := C.uint(timeout.Nanoseconds() / 1e6)
	var ptrSize C.int
	return Error(int(C.AT_WaitBuffer(c.handle, &c.cptr, &ptrSize, tout)))
}

// ShutDown shuts down the camera and 'finalizes' the SDK.
// This maintains API compatibility with SDK2
// and is not an exact match of
func (c *Camera) ShutDown() {
	return
}

// ExtCamera is an extension of the Camera type with some helper and 'macro'
// functions
type ExtCamera struct {
	Camera
}

// LastFrame returns the current state of the buffer cast to int32
// This is not in the SDK3, but is something we provide to build
// better interfaces.
//
// This function does not copy and uses unsafe mechanisms internally,
// buyer beware.
func (c *ExtCamera) LastFrame() (*[]int32, error) {
	if !c.bufferOnQueue {
		return nil, ErrBufferNotOnQueue
	}
	npx := c.npx()
	aryint32 := []int32{}
	hdr := (*reflect.SliceHeader)(unsafe.Pointer(&aryint32)) // TODO: point to buffer
	hdr.Data = uintptr(unsafe.Pointer(&aryint32[0]))
	hdr.Len = npx
	hdr.Cap = npx
	return &aryint32, nil
}
