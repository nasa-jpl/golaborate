package thorlabs

import (
	"fmt"

	"github.jpl.nasa.gov/HCIT/go-hcit/comm"
)

// LDC4000 represents an LDC4000 laser diode and TEC controller
type LDC4000 struct {
	*comm.RemoteDevice
}

var (
	// LDC4000Errors maps LDC4000 error codes to strings
	LDC4000Errors = map[int]string{
		-100: "COMMAND ERROR",
		-101: "INVALID CHARACTER",
		-102: "SYNTAX ERROR",
		-103: "INVALID SEPARATOR",
		-104: "DATA TYPE ERROR",
		-105: "GROUP EXECUTE TRIGGER NOT ALLOWED",
		//106, 107 skipped
		-108: "PARAMETER NOT ALLOWED",
		-109: "MISSING PARAMETER",
		-110: "COMMAND HEADER ERROR",
		-113: "UNDEFINED HEADER (UNKNOWN COMMAND)",
		-115: "UNEXPECTED NUMBER OF PARAMETERS",
		-120: "NUMERIC DATA ERROR",
		-130: "SUFFIX ERROR",
		-131: "INVALID SUFFIX",
		-151: "INVALID STRING DATA",

		-220: "PARAMETER ERROR",
		-221: "SETTINGS CONFLICT",
		-222: "DATA OUT OF RANGE",
		-230: "DATA CORRUPT OR STALE",
		-231: "DATA QUESTIONABLE",
		-240: "HARDWARE ERROR",
		-241: "HARDWARE MISSING",
		-250: "MASS STORAGE ERROR",
		-251: "MISSING MASS STORAGE",
		-252: "MISSING MEDIA",
		-253: "CORRUPT MEDIA",
		-254: "MEDIA FULL",
		-255: "DIRECTORY FULL",
		-256: "FILE NAME NOT FOUND",
		-257: "FILE NAME ERROR",
		-258: "MEDIA PROTECTED",

		-310: "SYSTEM ERROR",
		-311: "MEMORY ERROR",
		-313: "CALIBRATION MEMORY LOST",
		-314: "SAVE/RECALL MEMORY LOST",
		-315: "CONFIGURATION MEMORY LOST",
		-321: "OUT OF MEMORY",
		-330: "SELF-TEST FAILED",
		-340: "CALIBRAITON FAILURE",
		-350: "QUEUE OVERFLOW",
		-363: "INPUT BUFFER OVERRUN",

		-400: "QUERY ERROR",
		-410: "QUERY INTERRUPTED",

		3:  "INSTRUMENT IS OVERHEATED",
		20: "NOT PERMITTED WITH LD OUTPUT ON",
		22: "INTERLOCK CIRCUIT IS OPEN",
		23: "KEY SWITCH IN LOCKED POSITION",
		24: "LD OPEN CIRCUIT DETECTED",
		25: "LD-ENABLE INPUT IS DE-ASSERTED",
		26: "LD TEMPERATURE PROTECTION IS ACTIVE",
		27: "NOT PERMITTED WITH PHOTODIODE BIAS ON",
		28: "NOT PERMITTED WITH QCW MODE ON",
		30: "NOT PERMITTED WITH TEC OUTPUT ON",
		31: "WRONG TEC SOURCE OPERATING MODE",
		32: "PID AUTO-TUNE IS CURRENTLY RUNNING",
		33: "PID AUTO-TUNE VALUE ERROR",
		34: "TEC OPEN CIRCUIT DETECTED",
		35: "TEMEPRATURE SENSOR FAILURE",
		36: "TEC CABLE CONNECTION FAILURE",
	}
)

// On turns the LD on
func (ldc *LDC4000) On() error {
	cmd := "OUTPUT ON"
	return nil
}

// Off turns the LD off
func (ldc *LDC4000) Off() error {
	cmd := "OUTPUT OFF"
	return nil
}

// IsOn checks if the LDC is on or off
func (ldc *LDC4000) IsOn() (bool, error) {
	cmd := "OUTPUT?"
	return false, nil
}

// SetConstantPowerMode puts the laser into constant power mode (true) or into constant current mode (false)
func (ldc *LDC4000) SetConstantPowerMode(b bool) error {
	var cmd string
	if b {
		cmd = "SOURCE:FUNCTION:MODE POWER"
	} else {
		cmd = "SOURCE:FUNCTION:MODE CURRENT"
	}
	return nil
}

// GetConstantPowerMode gets if the laser is in constant power mode (true) or constant current mode (false)
func (ldc *LDC4000) GetConstantPowerMode() (bool, error) {
	cmd := "SOURCE:FUNCTION:MODE?"
	return false, nil
}

// SetPowerLevel sets the output power level in watts
func (ldc *LDC4000) SetPowerLevel(p float64) error {
	cmd := fmt.Sprintf("SOURCE:POWER %f.9", p)
	return nil
}

// SetCurrentLevel sets the output current in Amps
func (ldc *LDC4000) SetCurrentLevel(c float64) error {
	cmd := fmt.Sprintf("SOURCE:CURRENT %f.9", c)
	return nil
}
