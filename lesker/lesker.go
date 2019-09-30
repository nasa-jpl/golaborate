package lesker

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
	terminators = []byte("\r")
)

func makeSerConf(addr string) *serial.Config {
	return &serial.Config{
		Name:        addr,
		Baud:        19200,
		Size:        10,
		Parity:      serial.ParityOdd,
		StopBits:    serial.Stop1,
		ReadTimeout: 1 * time.Second}
}

// KJC300 has a serial connection and can make commands
type KJC300 struct {
	conn *serial.Port
}

// NewGuage returns a new KJC300 instance
func NewGuage(addr string) (KJC300, error) {
	cfg := makeSerConf(addr)
	conn, err := serial.OpenPort(cfg)
	tc := KJC300{conn: conn}
	return tc, err
}

// MkMsg generates a message that conforms to the custom schema used by the KJC300 gauges
func (kjc *KJC300) MkMsg(cmd string) []byte {
	return append([]byte("#01"+cmd), terminators...)
}

func (kjc *KJC300) sendAndReturnSingleFloat(msg []byte) (float64, error) {
	err := kjc.Send(msg)
	if err != nil {
		return 0, err
	}
	txt, err := kjc.Recv()
	if err != nil {
		return 0, err
	}
	f, err := strconv.ParseFloat(txt, 64)
	return f, err
}

// Send sends a command to the controller.
// If not terminated by terminators, behavior is undefined
func (kjc *KJC300) Send(cmd []byte) error {
	_, err := kjc.conn.Write(cmd)
	return err
}

// Recv data from the hardware and convert it to a string, stripping the leading *
func (kjc *KJC300) Recv() (string, error) {
	reader := bufio.NewReader(kjc.conn)
	bytes, err := reader.ReadBytes('\r')
	return strings.TrimLeft(string(bytes), "*"), err
}

// SWVersion returns the sensor ID from the controller
func (kjc *KJC300) SWVersion() (string, error) {
	msg := kjc.MkMsg("VER")
	err := kjc.Send(msg)
	if err != nil {
		return "", err
	}
	id, err := kjc.Recv()
	return id, nil
}

func (kjc *KJC300) Read() (float64, error) {
	msg := kjc.MkMsg("RD")
	err := kjc.Send(msg)
	if err != nil {
		return 0, err
	}
	resp, err := kjc.Recv()
	strs := strings.Split(resp, "_")
	protofloat := strs[1]
	return strconv.ParseFloat(protofloat, 64)
}
