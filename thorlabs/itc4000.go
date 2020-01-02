package thorlabs

import (
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.jpl.nasa.gov/HCIT/go-hcit/usbtmc"
)

/* unlike the remotedevice classes, this package assumes the connection to the
device is always open
*/
const (
	// TLVID is the Thorlabs vendor ID
	TLVID = 0x1313

	// LDC4001PID is the LDC4001 product ID
	LDC4001PID = 0x804a
)

// LDCError is a formatible error code from the XPS
type LDCError struct {
	code int
}

// Error satisfies stdlib error interface
func (e LDCError) Error() string {
	if s, ok := ITC4000Errors[e.code]; ok {
		return fmt.Sprintf("%d - %s", e.code, s)
	}
	return fmt.Sprintf("%d - UNKNOWN ERROR CODE", e.code)
}

// ITC4000 represents an ITC4000 laser diode and TEC controller
type ITC4000 struct {
	sync.Mutex

	dev usbtmc.USBDevice
}

// NewITC4000 creates a new ITC4000 instance absorbing the first one seen on the USB[us]
func NewITC4000() (*ITC4000, error) {
	d, err := usbtmc.NewUSBDevice(TLVID, LDC4001PID)
	return &ITC4000{dev: d}, err
}

var (
	// ITC4000Errors maps ITC4000 error codes to strings
	ITC4000Errors = map[int]string{
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

func (ldc *ITC4000) writeReadBus(cmd string) (string, error) {
	ldc.Lock()
	defer ldc.Unlock()
	err := ldc.dev.Write(append([]byte(cmd), '\n'))
	if err != nil {
		return "", err
	}
	resp, err := ldc.dev.Read()
	idx := len(resp.Data) - 1
	if resp.Data[idx] == 0x10 { // Data Link Escape may be last byte, pop it
		resp.Data = resp.Data[:idx]
		idx--
	}
	if resp.Data[idx] == '\n' { // pop trailing newline
		resp.Data = resp.Data[:idx]
	}
	return string(resp.Data), err
}

func (ldc *ITC4000) writeOnlyBus(cmd string) error {
	return ldc.dev.Write(append([]byte(cmd), '\n'))
}

// EmissionOn turns the LD on
func (ldc *ITC4000) EmissionOn() error {
	return ldc.writeOnlyBus("OUTPUT ON")
}

// EmissionOff turns the LD off
func (ldc *ITC4000) EmissionOff() error {
	return ldc.writeOnlyBus("OUTPUT OFF")
}

// EmissionIsOn checks if the LDC is on or off
func (ldc *ITC4000) EmissionIsOn() (bool, error) {
	resp, err := ldc.writeReadBus("OUTPUT?")
	return resp == "1", err
}

// SetConstantPowerMode puts the laser into constant power mode (true) or into constant current mode (false)
func (ldc *ITC4000) SetConstantPowerMode(b bool) error {
	var cmd string
	if b {
		cmd = "SOURCE:FUNCTION:MODE POWER"
	} else {
		cmd = "SOURCE:FUNCTION:MODE CURRENT"
	}
	return ldc.writeOnlyBus(cmd)
}

// GetConstantPowerMode gets if the laser is in constant power mode (true) or constant current mode (false)
func (ldc *ITC4000) GetConstantPowerMode() (bool, error) {
	resp, err := ldc.writeReadBus("SOURCE:FUNCTION:MODE?")
	return resp != "CURR", err
}

// SetPowerLevel sets the output power level in watts
func (ldc *ITC4000) SetPowerLevel(p float64) error {
	cmd := fmt.Sprintf("SOURCE:POWER %f.9", p)
	return ldc.writeOnlyBus(cmd)
}

// SetCurrent sets the output current in mA
func (ldc *ITC4000) SetCurrent(c float64) error {
	cmd := fmt.Sprintf("SOURCE:CURRENT %.9f", c/1e3)
	return ldc.writeOnlyBus(cmd)
}

// GetCurrent gets the output current in mA
func (ldc *ITC4000) GetCurrent() (float64, error) {
	resp, err := ldc.writeReadBus("SOURCE:CURRENT?")
	if err != nil {
		return 0, err
	}
	f, err := strconv.ParseFloat(resp, 64)
	return f * 1e3, err
}

// Raw sends a command and retrieves the reply if there is a question mark in the command, else returns "", err
func (ldc *ITC4000) Raw(cmd string) (string, error) {
	if !strings.Contains(cmd, "?") {
		err := ldc.writeOnlyBus(cmd)
		return "", err
	}
	return ldc.writeReadBus(cmd)
}
