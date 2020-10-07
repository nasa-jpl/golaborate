package pi

import (
	"bytes"
	"fmt"
	"time"

	"github.com/tarm/serial"
)

const tcpFrameSize = 1500

var (
	// ErrMap maps PI Error codes to the friendly strings
	ErrMap = map[int]string{
		0:    "No Error",
		1:    "Parameter syntax error",
		2:    "Unknown command",
		3:    "Command length out of limits or command buffer overrun",
		4:    "Error while scanning",
		5:    "Unallowable move attempted on unreferenced axis, or move attempted with servo off",
		6:    "Parameter for SGA not valid",
		7:    "Position out of limits",
		8:    "Velocity out of limits",
		9:    "Attempt to set pivot point while U, V and W not all 0",
		10:   "Controller was stopped by command",
		11:   "Parameter for SST or one of the embedded scan algorithms out of range",
		12:   "Invalid axis combination for fast scan",
		13:   "Parameter for NAV out of range",
		14:   "Invalid analog channel",
		15:   "Invalid axis identifier",
		16:   "Unknown stage name",
		17:   "Parameter out of range",
		18:   "Invalid macro name",
		19:   "Error while recording macro",
		20:   "Macro not found",
		21:   "Axis has no brake",
		22:   "Axis identifier specified more than once",
		23:   "Illegal axis",
		24:   "Incorrect number of parameters",
		25:   "Invalid floating point number",
		26:   "Parameter missing",
		27:   "Soft limit out of range",
		28:   "No manual pad found",
		29:   "No more step-response values",
		30:   "No step-response values recorded",
		31:   "Axis has no reference sensor",
		32:   "Axis has no limit switch",
		33:   "No relay card installed",
		34:   "Command not allowed for selected stage(s)",
		35:   "No digital input installed",
		36:   "No digital output configured",
		37:   "No more MCM responses",
		38:   "No MCM values recorded",
		39:   "Controller number invalid",
		40:   "No joystick configured",
		41:   "Invalid axis for electronic gearing, axis can not be slave",
		42:   "Position of slave axis is out of range",
		43:   "Slave axis cannot be commanded directly when electronic gearing is enabled",
		44:   "Calibration of joystick failed",
		45:   "Referencing failed",
		46:   "OPM (Optical Power Meter) missing",
		47:   "OPM (Optical Power Meter) not initialized or cannot be initialized",
		48:   "OPM (Optical Power Meter) Communication Error",
		49:   "Move to limit switch failed",
		50:   "Attempt to reference axis with referencing disabled",
		51:   "Selected axis is controlled by joystick",
		52:   "Controller detected communication error",
		53:   "MOV! motion still in progress",
		54:   "Unknown parameter",
		55:   "No commands were recoreded with REP",
		56:   "Password invalid",
		57:   "Data Record Table does not exist",
		58:   "Source does not exist; number too low or too high",
		59:   "Source Record Table number too low or too high",
		60:   "Protected Param: current Command Level (CCL) too low",
		61:   "Command execution not possible while autozero is running",
		62:   "Autozero requires at least one linear axis",
		63:   "Initialization still in progress",
		64:   "Parameter is read-only",
		65:   "Parameter not found in non-volatile memory",
		66:   "Voltage out of limits",
		67:   "Not enough memory available for requested wav curve",
		68:   "Not enough memory available for DLL table; DLL can not be started",
		69:   "time delay larger than DLL table; DLL can not be started",
		70:   "GCS-array doesn't support different length; request arrays which have different lengths separately",
		71:   "Attempt to restart the generator while it is running in single step mode",
		72:   "MOV, MVR, STA, SVR, STE, IMP and WGO blocked when analog target is active",
		73:   "MOV, MVR, STA, SVR, STE, and IMP blocked when wave generator is active",
		100:  "PI LabVIEW driver reports error.  See source control for details",
		200:  "No stage connected to axis",
		201:  "File with axis parameter not found",
		202:  "Invalid axis parameter file",
		203:  "Backup file with axis parameters not found",
		204:  "PI internal error code 204",
		205:  "SMO with servo on",
		206:  "uudecode: incomplete header",
		207:  "uudecode: nothing to decode",
		208:  "uudecode: illegal UUE format",
		209:  "CRC32 error",
		210:  "Illegal file name (must be 8-0 format)",
		211:  "File not found on controller",
		212:  "Error writing file on controller",
		213:  "VEL command not allowed in DTR Command Mode",
		214:  "Position calculations failed",
		215:  "The connection between controller and stage may be broken",
		216:  "The connected stage has driven into a limite switch, call CLR to resume operation",
		217:  "Strut test command failed because of an unexpected strut stop",
		218:  "Position can be estimated only while MOV! is running",
		219:  "Positionw as calculated while MOV is running",
		301:  "Send buffer overflow",
		302:  "Voltage out of limits",
		304:  "Recieved command is too long",
		305:  "Error while reading/writing EEPROM",
		306:  "Error on I2C bus",
		307:  "Timeout while recieving command",
		308:  "A lengthy operation has not finished in the expected time",
		309:  "Insufficient space to store macro",
		310:  "Configuration data has old version number",
		311:  "Invalid configuration data",
		333:  "Internal hardware error",
		555:  "BasMac: unknown controller error",
		601:  "not enough memory",
		602:  "hardware voltage error",
		603:  "hardware temperature out of range",
		1000: "Too many nested macros",
		1001: "Macro already defined",
		1002: "Macro recording not activated",
		1003: "Invalid parameter for MAC",
		1004: "PI internal error code 1004",
		2000: "Controller already has a serial number",
		4000: "Sector erase failed",
		4001: "Flash program failed",
		4002: "Flash read failed",
		4003: "HW match code missing/invalid",
		4004: "FW match code missing/invalid",
		4005: "HW version missing/invalid",
		4006: "FW version missing/invalid",
		4007: "FW Update failed",
		// TODO: populate negative (interface) error codes
	}
)

// GCS2Status encapsulates a status (error) code from a PI controller
// and its logic
type GCS2Status struct {
	code int
}

// GCS2Err converts an error code to something that implements the error interface
func GCS2Err(code int) error {
	if code == 0 {
		return nil
	}
	return GCS2Status{code}
}

func (e GCS2Status) Error() string {
	if s, ok := ErrMap[e.code]; ok {
		return fmt.Sprintf("%d - %s", e.code, s)
	}
	return fmt.Sprintf("%d - UNKNOWN ERROR CODE", e.code)
}

// makeSerConf makes a new serial.Config with correct parity, baud, etc, set.
func makeSerConf(addr string) *serial.Config {
	return &serial.Config{
		Name:        addr,
		Baud:        115200,
		Size:        8,
		Parity:      serial.ParityNone,
		StopBits:    serial.Stop1,
		ReadTimeout: 10 * time.Minute}
}

func stripAxis(axis string, message []byte) []byte {
	return bytes.TrimPrefix(message, []byte(axis+"="))
}
