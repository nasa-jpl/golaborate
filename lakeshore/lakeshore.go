/*Package lakeshore provides tools for working with Lakeshore 332 temperature controllers.

 */
package lakeshore

import (
	"bufio"
	"strconv"
	"strings"
	"time"

	"github.com/tarm/serial"
)

// per the Lakeshore 332 manual, the temperature controller serial interface
// uses the following schema:

// baud 300, 1200, or 9600
// 10 bits per character, 1 start 7 data, 1 parity, 1 stop
// odd parity
// terminator CRLR
// < 20 commands per second

// command messages look like <command><space><parameter data><terminators>
// query messages look like <query mnemonic><?><space><parameter data><terminators>
var (
	terminators = []byte("\r\n")
)

func makeSerConf(addr string) *serial.Config {
	return &serial.Config{
		Name:        addr,
		Baud:        9600,
		Size:        10,
		Parity:      serial.ParityOdd,
		StopBits:    serial.Stop1,
		ReadTimeout: 1 * time.Second}
}

// TempController has a serial connection and can make commands
type TempController struct {
	conn *serial.Port
}

// NewController returns a new TempController instance
func NewController(addr string) (TempController, error) {
	cfg := makeSerConf(addr)
	conn, err := serial.OpenPort(cfg)
	tc := TempController{conn: conn}
	return tc, err
}

// MkMsg generates a message that conforms to the IEEE standard
func (tc *TempController) MkMsg(cmd, parameter string) []byte {
	return append([]byte(cmd+"? "+parameter), terminators...)
}

func (tc *TempController) sendAndReturnSingleFloat(msg []byte) (float64, error) {
	err := tc.Send(msg)
	if err != nil {
		return 0, err
	}
	txt, err := tc.Recv()
	if err != nil {
		return 0, err
	}
	f, err := strconv.ParseFloat(txt, 64)
	return f, err
}

// Send sends a command to the controller.
// If not terminated by terminators, behavior is undefined
func (tc *TempController) Send(cmd []byte) error {
	_, err := tc.conn.Write(cmd)
	return err
}

// Recv data from the hardware and convert it to a string
func (tc *TempController) Recv() (string, error) {
	reader := bufio.NewReader(tc.conn)
	bytes, err := reader.ReadBytes('\n')
	return string(bytes), err
}

// ID returns the sensor ID from the controller
func (tc *TempController) ID() (string, error) {
	msg := tc.MkMsg("*IDN?", "")
	err := tc.Send(msg)
	if err != nil {
		return "", err
	}
	id, err := tc.Recv()
	return id, nil
}

// ReadChannel reads temperature in C from the given channel
func (tc *TempController) ReadChannel(cnl string) (float64, error) {
	msg := tc.MkMsg("CRDG?", cnl)
	return tc.sendAndReturnSingleFloat(msg)
}

// HeaterOutput reads the heater output in %
func (tc *TempController) HeaterOutput() (float64, error) {
	msg := tc.MkMsg("HTR?", "")
	return tc.sendAndReturnSingleFloat(msg)
}

// HeaterStatus reads the heater status
func (tc *TempController) HeaterStatus() (string, error) {
	msg := tc.MkMsg("HTRST?", "")
	err := tc.Send(msg)
	if err != nil {
		return "", err
	}
	status, err := tc.Recv()
	if err != nil {
		return "", err
	}
	switch status {
	case "0":
		status = "OK"
	case "1":
		status = "OPEN"
	case "2":
		status = "SHORT"
	}
	return status, nil
}

// HeaterSetpoint reads the heater setpoint
func (tc *TempController) HeaterSetpoint(loop string) (float64, error) {
	msg := tc.MkMsg("SETP?", loop)
	return tc.sendAndReturnSingleFloat(msg)
}

// PID reads the PID constants from the controller:
// kP - linear / proportional term
// kI - integral term
// kD - derivative term
func (tc *TempController) PID() ([]float64, error) {
	msg := tc.MkMsg("PID?", "")
	err := tc.Send(msg)
	if err != nil {
		return []float64{0, 0, 0}, err
	}
	txt, err := tc.Recv()
	if err != nil {
		return []float64{0, 0, 0}, err
	}
	pieces := strings.Split(txt, ",")
	numeric := make([]float64, 3)
	for i, v := range pieces {
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return numeric, err
		}
		numeric[i] = f
	}
	return numeric, nil
}
