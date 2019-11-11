package sdk3

import (
	"errors"
	"fmt"

	cwch "github.com/lordadamson/cgo.wchar"
)

/*
#cgo CFLAGS: -I/usr/local
#cgo LDFLAGS: -L/usr/local/lib -latcore
#include <stdlib.h>
#include <atcore.h>

*/
import "C"

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
		"AOIHeight":               true,
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
		return fmt.Sprintf("%d - %s", e, s)
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
	return GetInt(int(C.AT_HANDLE_SYSTEM), "DeviceCount")
}

// SoftwareVersion returns the software (SDK) version
// InitializeLibrary must be called first
func SoftwareVersion() (string, error) {
	return GetString(int(C.AT_HANDLE_SYSTEM), "SoftwareVersion")
}

// SetInt sets an integer
func SetInt(handle int, feature string, val int64) error {
	cstr, err := cwch.FromGoString(feature)
	if err != nil {
		return err
	}
	str := (*C.AT_WC)(cstr.Pointer())
	return Error(int(C.AT_SetInt(C.AT_H(handle), str, C.AT_64(val))))
}

// GetInt gets an integer
func GetInt(handle int, feature string) (int, error) {
	cstr, err := cwch.FromGoString(feature)
	if err != nil {
		return 0, err
	}
	str := (*C.AT_WC)(cstr.Pointer())
	var out C.AT_64
	errCode := int(C.AT_GetInt(C.AT_H(handle), str, &out))
	return int(out), Error(errCode)
}

// GetIntMax gets the max value an integer can be set to
func GetIntMax(handle int, feature string) (int, error) {
	// this code is identical to GetInt except for the C call
	cstr, err := cwch.FromGoString(feature)
	if err != nil {
		return 0, err
	}
	str := (*C.AT_WC)(cstr.Pointer())

	var out C.AT_64
	errCode := int(C.AT_GetIntMax(C.AT_H(handle), str, &out))
	return int(out), Error(errCode)
}

// GetIntMin gets the min value an integer can be set to
func GetIntMin(handle int, feature string) (int, error) {
	// this code is identical to GetInt except for the C call
	cstr, err := cwch.FromGoString(feature)
	if err != nil {
		return 0, err
	}
	str := (*C.AT_WC)(cstr.Pointer())

	var out C.AT_64
	errCode := int(C.AT_GetIntMin(C.AT_H(handle), str, &out))
	return int(out), Error(errCode)
}

// SetFloat sets a floating point value
func SetFloat(handle int, feature string, value float64) error {
	cstr, err := cwch.FromGoString(feature)
	if err != nil {
		return err
	}
	str := (*C.AT_WC)(cstr.Pointer())

	return Error(int(C.AT_SetFloat(C.AT_H(handle), str, C.double(value))))
}

// GetFloat gets a floating point value
func GetFloat(handle int, feature string) (float64, error) {
	cstr, err := cwch.FromGoString(feature)
	if err != nil {
		return 0, err
	}
	str := (*C.AT_WC)(cstr.Pointer())

	var out C.double
	errCode := int(C.AT_GetFloat(C.AT_H(handle), str, &out))
	return float64(out), Error(errCode)
}

// GetFloatMax gets the maximum of a floating point value
func GetFloatMax(handle int, feature string) (float64, error) {
	// this code is identical to GetFloat except for the C call
	cstr, err := cwch.FromGoString(feature)
	if err != nil {
		return 0, err
	}
	str := (*C.AT_WC)(cstr.Pointer())

	var out C.double
	errCode := int(C.AT_GetFloatMax(C.AT_H(handle), str, &out))
	return float64(out), Error(errCode)
}

// GetFloatMin gets the minimum of a floating point value
func GetFloatMin(handle int, feature string) (float64, error) {
	// this code is identical to GetFloat except for the C call
	cstr, err := cwch.FromGoString(feature)
	if err != nil {
		return 0, err
	}
	str := (*C.AT_WC)(cstr.Pointer())

	var out C.double
	errCode := int(C.AT_GetFloatMin(C.AT_H(handle), str, &out))
	return float64(out), Error(errCode)
}

// SetBool sets a boolean feature
func SetBool(handle int, feature string, tru bool) error {
	// this code is identical to GetFloat except for the C call
	cstr, err := cwch.FromGoString(feature)
	if err != nil {
		return err
	}
	str := (*C.AT_WC)(cstr.Pointer())
	return Error(int(C.AT_SetBool(C.AT_H(handle), str, boolToAT(tru))))
}

// GetBool gets the value of a boolean feature
func GetBool(handle int, feature string) (bool, error) {
	// this code is identical to GetFloat except for the C call
	cstr, err := cwch.FromGoString(feature)
	if err != nil {
		return false, err
	}
	str := (*C.AT_WC)(cstr.Pointer())
	var b C.AT_BOOL
	errCode := int(C.AT_GetBool(C.AT_H(handle), str, &b))
	return atToBool(b), Error(errCode)
}

// SetString sets the value of a string
func SetString(handle int, feature, value string) error {
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

	return Error(int(C.AT_SetString(C.AT_H(handle), featstr, valstr)))
}

// GetStringMaxLength returns the length of a string, use this to determine how big
// of a cgo.widechar string to allocate
func GetStringMaxLength(handle int, feature string) (int, error) {
	cstr, err := cwch.FromGoString(feature)
	if err != nil {
		return 0, err
	}
	str := (*C.AT_WC)(cstr.Pointer())

	var len C.int
	errCode := int(C.AT_GetStringMaxLength(C.AT_H(handle), str, &len))
	return int(len), Error(errCode)
}

// GetString returns the string value of a feature
// we deviate from SDK3 API by using GetStringMaxLength internally
// and avoid users having to deal with allocating C.wchar_t buffers
func GetString(handle int, feature string) (string, error) {
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
	errCode := int(C.AT_GetString(C.AT_H(handle), strc, outp, C.int(size)))

	str, err := stro.GoString()
	if err != nil {
		return "", err
	}
	return str, Error(errCode)
}

// GetEnumIndex gets the currently selected index into the enum behind feature
func GetEnumIndex(handle int, feature string) (int, error) {
	cstr, err := cwch.FromGoString(feature)
	if err != nil {
		return 0, err
	}
	strc := (*C.AT_WC)(cstr.Pointer())
	var out C.int
	errCode := int(C.AT_GetEnumIndex(C.AT_H(handle), strc, &out))
	return int(out), Error(errCode)
}

// GetEnumCount gets the number of items in the enum behind a feature
func GetEnumCount(handle int, feature string) (int, error) {
	// this function is identical to GetEnumIndex except for the C call
	cstr, err := cwch.FromGoString(feature)
	if err != nil {
		return 0, err
	}
	strc := (*C.AT_WC)(cstr.Pointer())
	var out C.int
	errCode := int(C.AT_GetEnumCount(C.AT_H(handle), strc, &out))
	return int(out), Error(errCode)
}

// GetEnumStringByIndex gets the string value of an enum at a given index
func GetEnumStringByIndex(handle int, feature string, idx int) (string, error) {
	cstr, err := cwch.FromGoString(feature)
	if err != nil {
		return "", err
	}
	strc := (*C.AT_WC)(cstr.Pointer())

	// we don't know how long the strings will be, so allocate a reasonable
	// length buffer.  This opens us to segfaults in the future if the buffer
	// is too small, or performance hits if it is too big.
	// we'll start with 64 bytes
	buf := cwch.NewWcharString(LengthOfUndefinedBuffers)
	strb := (*C.AT_WC)(buf.Pointer())
	errCode := int(C.AT_GetEnumStringByIndex(C.AT_H(handle), strc, C.int(idx), strb, C.int(LengthOfUndefinedBuffers)))
	gostr, err := buf.GoString()
	if err != nil {
		return "", err
	}
	return gostr, Error(errCode)
}

// GetEnumStrings gets the string values that are valid for an enum
func GetEnumStrings(handle int, feature string) ([]string, error) {
	count, err := GetEnumCount(handle, feature)
	if err != nil {
		return []string{}, err
	}
	values := make([]string, count)
	for idx := 0; idx < count; idx++ {
		str, err := GetEnumStringByIndex(handle, feature, idx)
		if err != nil {
			return values, err
		}
		values[idx] = str
	}
	return values, nil
}

// SetEnumIndex sets the value of a feature to an index in the backing enum
func SetEnumIndex(handle int, feature string, idx int) error {
	cstr, err := cwch.FromGoString(feature)
	if err != nil {
		return err
	}
	strc := (*C.AT_WC)(cstr.Pointer())
	errCode := int(C.AT_SetEnumIndex(C.AT_H(handle), strc, C.int(idx)))
	return Error(errCode)
}

// SetEnumString sets the value of a feature to a string that is a valid member
// of the backing enum
func SetEnumString(handle int, feature, value string) error {
	cstr, err := cwch.FromGoString(feature)
	if err != nil {
		return err
	}
	strc := (*C.AT_WC)(cstr.Pointer())

	cstr2, err := cwch.FromGoString(value)
	if err != nil {
		return err
	}
	strb := (*C.AT_WC)(cstr2.Pointer())
	errCode := int(C.AT_SetEnumString(C.AT_H(handle), strc, strb))
	return Error(errCode)
}

// IssueCommand sends a command to the SDK
func IssueCommand(handle int, feature string) error {
	cstr, err := cwch.FromGoString(feature)
	if err != nil {
		return err
	}
	strc := (*C.AT_WC)(cstr.Pointer())
	return Error(int(C.AT_Command(C.AT_H(handle), strc)))
}
