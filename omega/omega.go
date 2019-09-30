package omega

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
		Size:        7,
		Parity:      serial.ParityEven,
		StopBits:    serial.Stop1,
		ReadTimeout: 1 * time.Second}
}

// DPF700 has a serial connection and can make commands
type DPF700 struct {
	conn *serial.Port
}

// NewMeter returns a new DPF700 instance
func NewMeter(addr string) (DPF700, error) {
	cfg := makeSerConf(addr)
	conn, err := serial.OpenPort(cfg)
	tc := DPF700{conn: conn}
	return tc, err
}

// MkMsg generates a message that conforms to the custom schema used by the DPF700 gauges
func (dfp *DPF700) MkMsg(cmd string) []byte {
	return append([]byte("@U?"+cmd), terminators...)
}

// Send sends a command to the controller.
// If not terminated by terminators, behavior is undefined
func (dfp *DPF700) Send(cmd []byte) error {
	_, err := dfp.conn.Write(cmd)
	return err
}

// Recv data from the hardware and convert it to a string, stripping the leading *
func (dfp *DPF700) Recv() (string, error) {
	reader := bufio.NewReader(dfp.conn)
	bytes, err := reader.ReadBytes('\r')
	return strings.TrimLeft(string(bytes), "*"), err
}

func (dfp *DPF700) Read() (float64, error) {
	msg := dfp.MkMsg("V")
	err := dfp.Send(msg)
	if err != nil {
		return 0, err
	}
	resp, err := dfp.Recv()
	return strconv.ParseFloat(resp[1:], 64)
}
