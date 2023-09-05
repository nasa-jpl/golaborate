package newport

import (
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/nasa-jpl/golaborate/comm"

	"github.com/tarm/serial"
)

const (
	// TxTerm is the outgoing terminator
	CarriageReturn = '\r'
	Newline        = '\n'
)

var (
	// ErrBufferWouldOverflow is generated when the buffer on the ESP controller
	// would overflow if the message was transmitted
	ErrBufferWouldOverflow = errors.New("buffer too long, maximum command length is 80 characters")

	commands = []Command{
		// Status functions
		{Cmd: "TE", Alias: "err-num", Description: "get error number", IsReadOnly: true},
		{Cmd: "TP", Alias: "get-position", Description: "get position", UsesAxis: true, IsReadOnly: true},
		{Cmd: "TS", Alias: "controller-status", Description: "get controller status", IsReadOnly: true},
		{Cmd: "TV", Alias: "get-velocity", Description: "get velocity", UsesAxis: true, IsReadOnly: true},
		{Cmd: "TX", Alias: "controller-activity", Description: "get controller activity", IsReadOnly: true},
		{Cmd: "VE", Alias: "controller-firmware", Description: "get controller firmware version", IsReadOnly: true},

		// Motion functions
		{Cmd: "AB", Alias: "abort-program", Description: "abort program", UsesAxis: true},
		{Cmd: "DH", Alias: "define-home", Description: "define home", UsesAxis: true},
		{Cmd: "MT", Alias: "move-hw-limit", Description: "move to hardware limit", UsesAxis: true},
		{Cmd: "MV", Alias: "move-indef", Description: "move indefinitely", UsesAxis: true},
		{Cmd: "OR", Alias: "origin-search", Description: "origin searching", UsesAxis: true},
		{Cmd: "PA", Alias: "move-abs", Description: "move absolute", UsesAxis: true},
		{Cmd: "PR", Alias: "move-rel", Description: "move relative", UsesAxis: true},
		{Cmd: "ST", Alias: "stop", Description: "stop motion", UsesAxis: true},

		// trajectory definition, less JH JK JW OL OH OM SH UF
		{Cmd: "AC", Alias: "set-accel", Description: "set acceleration", UsesAxis: true},
		{Cmd: "AE", Alias: "set-estop-accel", Description: "set e-stop acceleration", UsesAxis: true},
		{Cmd: "AG", Alias: "set-decel", Description: "set deceleration", UsesAxis: true},
		{Cmd: "AU", Alias: "set-max-accel", Description: "set maximum acceleration", UsesAxis: true},
		{Cmd: "BA", Alias: "set-backlash-comp", Description: "set backlash compensation on or off", UsesAxis: true},
		{Cmd: "CO", Alias: "set-linear-comp", Description: "set linear compensation on or off", UsesAxis: true},
		{Cmd: "VA", Alias: "set-velocity-linear", Description: "set velocity for linear motors", UsesAxis: true},
		{Cmd: "VB", Alias: "set-velocity-stepper", Description: "set velocity for stepper motors", UsesAxis: true},
		{Cmd: "VU", Alias: "set-max-speed", Description: "set maximum speed", UsesAxis: true},
		{Cmd: "MO", Alias: "enable-axis", Description: "Turn the motor on for an axis", UsesAxis: true},
		{Cmd: "MF", Alias: "disable-axis", Description: "turn the motor off for an axis", UsesAxis: true},

		// WS1 => the 1 is the delay after end of motion in ms;
		// a zero value causes the WS to be ignored
		{Cmd: "WS1", Alias: "wait", Description: "Wait for the axis' motion to stop", UsesAxis: true},

		// Not implemented:
		// - General mode selection,
		// - motion device parameters,
		// - programming,
		// - flow control & sequence,
		// - group functions,
		// - digital filters
		// - master-slave mode definition
	}

	// ESPErrorCodesWithoutAxes maps error codes to error strings when the errors
	// are not axis specific
	ESPErrorCodesWithoutAxes = map[int]string{
		0:  "NO ERROR DETECTED",
		3:  "OVER TEMPERATURE SHUTDOWN",
		4:  "EMERGENCY STOP ACTIVATED",
		6:  "COMMAND DOES NOT EXIST",
		7:  "PARAMETER OUT OF RANGE",
		8:  "CABLE INTERLOCK ERROR",
		9:  "AXIS NUMBER OUT OF RANGE",
		10: "EEPROM WRITE FAILED",
		11: "EEPROM READ FAILED",
		13: "GROUP NUMBER MISSING",
		14: "GROUP NUMBER OUT OF RANGE",
		15: "GROUP NUMBER NOT ASSIGNED",
		16: "GROUP NUMBER ALREADY ASSIGNED",
		17: "GROUP AXIS OUT OF RANGE",
		18: "GROUP AXIS ALREADY ASSIGNED",
		19: "GROUP AXIS DUPLICATED",
		20: "DATA ACQUISITION IS BUSY",
		21: "DATA ACQUISITION SETUP ERROR",
		22: "DATA ACQUISITION NOT ENABLED",
		23: "SERVO CYCLE (400μS) TICK FAILURE",
		25: "DOWNLOAD IN PROGRESS",
		26: "STORED PROGRAM NOT STARTED",
		27: "COMMAND NOT ALLOWED",
		28: "STORED PROGRAM FLASH AREA FULL",
		29: "GROUP PARAMETER MISSING",
		30: "GROUP PARAMETER OUT OF RANGE",
		31: "GROUP MAXIMUM VELOCITY EXCEEDED",
		32: "GROUP MAXIMUM ACCELERATION EXCEEDED",
		33: "GROUP MAXIMUM DECELERATION EXCEEDED",
		34: "GROUP MOVE NOT ALLOWED DURING MOTION",
		35: "PROGRAM NOT FOUND",
		37: "AXIS NUMBER MISSING",
		38: "COMMAND PARAMETER MISSING",
		40: "LAST COMMAND CANNOT BE REPEATED",
		41: "MAX NUMBER OF LABELS PER PROGRAM EXCEEDED",
		46: "RS-485 EXT FAULT DETECTED",
		47: "RS-485 CRC FAULT DETECTED",
		48: "CONTROLLER NUMBER OUT OF RANGE",
		49: "SCAN IN PROGRESS",
	}

	// ESPErrorCodesWithAxes maps the final two digits of an axis-specific
	// error code to a string.  The axis number is excluded from the key.
	ESPErrorCodesWithAxes = map[int]string{
		0:  "MOTOR TYPE NOT DEFINED",
		1:  "PARAMETER OUT OF RANGE",
		2:  "AMPLIFIER FAULT DETECTED",
		3:  "FOLLOWING ERROR THRESHLD EXCEEDED",
		4:  "POSITIVE HARDWARE LIMIT REACHED",
		5:  "NEGATIVE HARDWARE LIMIT REACHED",
		6:  "POSITIVE SOFTWARE LIMIT REACHED",
		7:  "NEGATIVE SOFTWARE LIMIT REACHED",
		8:  "MOTOR / STAGE NOT CONNECTED",
		9:  "FEEDBACK SIGNAL FAULT DETECTED",
		10: "MAXIMUM VELOCITY EXCEEDED",
		11: "MAXIMUM ACCELERATION EXCEEDED",
		13: "MOTOR NOT ENABLED",
		14: "MOTION IN PROGRESS",
		15: "MAXIMUM JERK EXCEEDED",
		16: "MAXIMUM DAC OFFSET EXCEEDED",
		17: "ESP CRITICAL SETTINGS ARE PROTECTED",
		18: "ESP STAGE DEVICE ERROR",
		19: "ESP STAGE DATA INVALID",
		20: "HOMING ABORTED",
		21: "MOTOR CURRENT NOT DEFINED",
		22: "UNIDRIVE COMMUNICATIONS ERROR",
		23: "UNIDRIVE NOT DETECTED",
		24: "SPEED OUT OF RANGE",
		25: "INVALID TRAJECTORY MASTER AXIS",
		26: "PARAMETER CHARGE NOT ALLOWED",
		28: "INVALID ENCODER STEP RATIO",
		29: "DIGITAL I/O INTERLOCK DETECTED",
		30: "COMMAND NOT ALLOWED DURING HOMING",
		31: "COMMAND NOT ALLOWED DUE TO GROUP ASSIGNMENT",
		32: "INVALID TRAJECTORY MODE FOR MOVING",
	}
)

// Command describes a command
type Command struct {
	Cmd         string `json:"cmd"`
	Alias       string `json:"alias"`
	Description string `json:"description"`
	UsesAxis    bool   `json:"usesAxis"`
	IsReadOnly  bool   `json:"isReadOnly"`
}

// ErrCommandNotFound is generated when a command is unknown to the newport module
type ErrCommandNotFound struct {
	Cmd string
}

func (e ErrCommandNotFound) Error() string {
	return fmt.Sprintf("command %s not found", e.Cmd)
}

// ErrAliasNotFound is generated when an alias is unknown to the newport module
type ErrAliasNotFound struct {
	Alias string
}

func (e ErrAliasNotFound) Error() string {
	return fmt.Sprintf("alias %s not found", e.Alias)
}

func commandFromCmd(cmd string) (Command, error) {
	for _, c := range commands {
		if c.Cmd == cmd {
			return c, nil
		}
	}
	return Command{}, ErrCommandNotFound{cmd}
}

func commandFromAlias(alias string) (Command, error) {
	for _, c := range commands {
		if c.Alias == alias {
			return c, nil
		}
	}
	return Command{}, ErrAliasNotFound{alias}
}

func commandFromCmdOrAlias(cmdAlias string) (Command, error) {
	var cmd Command
	cmd, err := commandFromCmd(cmdAlias)
	if err != nil {
		cmd, err = commandFromAlias(cmdAlias)
	}
	return cmd, err
}

func makeTelegram(c Command, axis string, write bool, data float64) string {
	var pieces []string
	if c.UsesAxis {
		pieces = append(pieces, axis)
	}
	pieces = append(pieces, c.Cmd)
	if c.IsReadOnly || !write {
		pieces = append(pieces, "?")
	} else {
		pieces = append(pieces, strconv.FormatFloat(data, 'g', -1, 64))
	}
	return strings.Join(pieces, "")
}

func makeTelegramPlural(c []Command, axes []string, write []bool, data []float64) string {
	telegrams := make([]string, 0, len(c))
	for idx, c := range c {
		telegrams = append(telegrams, makeTelegram(c, axes[idx], write[idx], data[idx]))
	}
	return strings.Join(telegrams, ";")
}

// makeSerConf makes a new serial.Config with correct parity, baud, etc, set.
func makeSerConf(addr string) *serial.Config {
	return &serial.Config{
		Name:        addr,
		Baud:        19200,
		Size:        8,
		Parity:      serial.ParityNone,
		StopBits:    serial.Stop1,
		ReadTimeout: 1 * time.Second}
}

// ESP301 represents an ESP301 motion controller.
type ESP301 struct {
	pool *comm.Pool
}

// NewESP301 makes a new ESP301 motion controller instance
func NewESP301(addr string, connectSerial bool) *ESP301 {
	var maker comm.CreationFunc
	if connectSerial {
		maker = comm.SerialConnMaker(makeSerConf(addr))
	} else {
		maker = comm.BackingOffTCPConnMaker(addr, 1*time.Second)
	}
	p := comm.NewPool(1, time.Minute, maker)
	return &ESP301{pool: p}
}

// RawCommand sends a command directly to the motion controller (with EOT appended) and returns the response as-is
func (esp *ESP301) RawCommand(cmd string) (string, error) {
	// set up the connection
	conn, err := esp.pool.Get()
	if err != nil {
		return "", err
	}
	defer func() { esp.pool.ReturnWithError(conn, err) }()
	wrapper := comm.NewTerminator(conn, CarriageReturn, CarriageReturn)

	// acquire an almost imperceptible amount of parallel performance here
	// the message will be in flight or processed by the ESP while we
	// check if we should read.
	_, err = io.WriteString(wrapper, cmd)
	if err != nil {
		return "", err
	}
	if strings.Contains(cmd, "?") {
		buf := make([]byte, 80) // 80 byte max on Rx side, assume symmetry from ESP
		n, err := wrapper.Read(buf)
		if err != nil {
			return "", err
		}
		return string(buf[:n]), nil
	}
	return "", err
}

// Enable enables an axis
func (esp *ESP301) Enable(axis string) error {
	cmd := fmt.Sprintf("%sMO", axis)
	_, err := esp.RawCommand(cmd)
	return err
}

// Disable disables an axis
func (esp *ESP301) Disable(axis string) error {
	cmd := fmt.Sprintf("%sMF", axis)
	_, err := esp.RawCommand(cmd)
	return err
}

// GetEnabled returns if an axis is enabled.  This may not be truthy if the controller
// threw an error, check the errors if you get disabled errors and this reports true
func (esp *ESP301) GetEnabled(axis string) (bool, error) {
	cmd, _ := commandFromAlias("enable-axis")
	tele := makeTelegram(cmd, axis, false, 0)
	resp, err := esp.RawCommand(tele)
	if err != nil {
		return false, err
	}
	return strconv.ParseBool(resp)
}

// SetVelocity sets the velocity setpoint for an axis
func (esp *ESP301) SetVelocity(axis string, vel float64) error {
	c, _ := commandFromAlias("set-velocity-linear")
	tele := makeTelegram(c, axis, true, vel)
	_, err := esp.RawCommand(tele)
	return err
}

// GetVelocity returns the velocity setpoint for an axis
func (esp *ESP301) GetVelocity(axis string) (float64, error) {
	c, _ := commandFromAlias("set-velocity-linear")
	tele := makeTelegram(c, axis, false, 0)
	resp, err := esp.RawCommand(tele)
	if err != nil {
		return 0, err
	}
	return strconv.ParseFloat(resp, 64)
}

// GetPos gets the absolute position of an axis in controller units (usually mm)
func (esp *ESP301) GetPos(axis string) (float64, error) {
	c, _ := commandFromAlias("get-position")
	tele := makeTelegram(c, axis, false, 0)
	resp, err := esp.RawCommand(tele)
	if err != nil {
		return 0, err
	}
	return strconv.ParseFloat(resp, 64)
}

// MoveAbs sets the absolute position of an axis in controller units (usually mm)
func (esp *ESP301) MoveAbs(axis string, pos float64) error {
	c, _ := commandFromAlias("move-abs")
	c2, _ := commandFromAlias("wait")
	cmds := []Command{c, c2}
	axes := []string{axis, axis}
	write := []bool{true, true}
	data := []float64{pos, 0}
	// tele := makeTelegram(c, axis, true, pos)
	tele := makeTelegramPlural(cmds, axes, write, data)
	_, err := esp.RawCommand(tele)
	return err
}

// MoveRel triggers a relative motion of an axis in controller units
func (esp *ESP301) MoveRel(axis string, pos float64) error {
	c, _ := commandFromAlias("move-rel")
	c2, _ := commandFromAlias("wait")
	cmds := []Command{c, c2}
	axes := []string{axis, axis}
	write := []bool{true, true}
	data := []float64{pos, 0}
	// tele := makeTelegram(c, axis, true, pos)
	tele := makeTelegramPlural(cmds, axes, write, data)
	_, err := esp.RawCommand(tele)
	return err
}

// Home homes an axis.
// mode 6, negative limit switch + home mark
func (esp *ESP301) Home(axis string) error {
	cmd, _ := commandFromAlias("origin-search")
	tele := makeTelegram(cmd, axis, true, 1)
	_, err := esp.RawCommand(tele)
	return err
}

// Wait waits for motion to cease and then returns nil
func (esp *ESP301) Wait(axis string) error {
	cmd, _ := commandFromAlias("wait")
	tele := makeTelegram(cmd, axis, true, 0)
	fmt.Println(tele)
	return nil
}

// SetFollowingErrorConfiguration sets the "following error" configuration
func (esp *ESP301) SetFollowingErrorConfiguration(axis string, enableChecking, disableMotorPowerOnError, abortMotionOnError bool) error {
	// this could be cleaner, but it is rare we need to pack bits into bytes
	bits := [8]bool{
		enableChecking,
		disableMotorPowerOnError,
		abortMotionOnError,
		false,
		false,
		false,
		false,
		false}
	b := byte(0)
	for idx := uint(0); idx < 8; idx++ {
		if bits[idx] {
			b |= 1 << idx
		}
	}
	msg := fmt.Sprintf("%sZF0%XH", axis, b)
	resp, err := esp.RawCommand(msg)
	fmt.Println(resp)
	return err
}

// ReadErrors reads all error from the controller and returns a slice of the
// error messages, which may be empty if there are no errors.  The slice may be
// partially filled if a communication error is encountered while reading the
// sequence of errors.
func (esp *ESP301) ReadErrors() ([]string, error) {
	var errs []string
	cmd := "TB?"
	for {
		resp, err := esp.RawCommand(cmd)
		if err != nil {
			return errs, err
		}
		if resp[0] == '0' {
			break
		}
		pieces := strings.Split(resp, ",")
		axis := -1
		if l := len(pieces); l >= 1 {
			lcode := len(pieces[0])
			var (
				mapV  map[int]string
				icode int
				err   error
			)
			if lcode > 2 {
				mapV = ESPErrorCodesWithAxes
				icode, err = strconv.Atoi(pieces[0][lcode-2:]) // pop the axis off
				axis, err = strconv.Atoi(pieces[0][:lcode-2])
			} else {
				mapV = ESPErrorCodesWithoutAxes
				icode, err = strconv.Atoi(pieces[0])
			}
			if err != nil {
				return errs, err
			}
			// at this stage, we have a map of icode=> error message and an axis
			// number, which may be the special value of -1, indicating no axis
			// now we concatenate the axis number and the error string and push
			// onto the error stack
			errS := mapV[icode]
			if axis != -1 {
				errS = fmt.Sprintf("AXIS %d ", axis) + errS
			}
			errs = append(errs, errS)

		} else {
			return errs, fmt.Errorf("expected CSV from motion controller with at least 1 element, got %d", l)
		}
	}
	return errs, nil
}
