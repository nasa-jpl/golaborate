package commonpressure

import (
	"bufio"
	"strconv"
	"strings"
	"time"

	"github.com/tarm/serial"
)

var (
	terminators = []byte("\r")
)

func makeSerConf(addr string) *serial.Config {
	return &serial.Config{
		Name:        addr,
		Baud:        19200,
		Size:        8,
		Parity:      serial.ParityOdd,
		StopBits:    serial.Stop1,
		ReadTimeout: 1 * time.Second}
}

// Sensor has a serial connection and can make commands
type Sensor struct {
	conn *serial.Port
}

// NewGuage returns a new Sensor instance
func NewGuage(addr string) (Sensor, error) {
	cfg := makeSerConf(addr)
	conn, err := serial.OpenPort(cfg)
	tc := Sensor{conn: conn}
	return tc, err
}

// MkMsg generates a message that conforms to the custom schema used by the Sensor gauges
func (sens *Sensor) MkMsg(cmd string) []byte {
	return append([]byte("#01"+cmd), terminators...)
}

// Send sends a command to the controller.
// If not terminated by terminators, behavior is undefined
func (sens *Sensor) Send(cmd []byte) error {
	_, err := sens.conn.Write(cmd)
	return err
}

// Recv data from the hardware and convert it to a string, stripping the leading *
func (sens *Sensor) Recv() (string, error) {
	reader := bufio.NewReader(sens.conn)
	bytes, err := reader.ReadBytes('\r')
	return strings.TrimLeft(string(bytes), "*"), err
}

// SWVersion returns the sensor ID from the controller
func (sens *Sensor) SWVersion() (string, error) {
	msg := sens.MkMsg("VER")
	err := sens.Send(msg)
	if err != nil {
		return "", err
	}
	id, err := sens.Recv()
	return id, nil
}

func (sens *Sensor) Read() (float64, error) {
	msg := sens.MkMsg("RD")
	err := sens.Send(msg)
	if err != nil {
		return 0, err
	}
	resp, err := sens.Recv()
	strs := strings.Split(resp, "_")
	protofloat := strs[1]
	return strconv.ParseFloat(protofloat, 64)
}
