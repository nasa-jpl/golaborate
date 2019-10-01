package commonpressure

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/tarm/serial"
)

var (
	terminators = []byte("\r")
)

// MakeSerConf makes a new serial.Config with correct parity, baud, etc, set.
func MakeSerConf(addr string) *serial.Config {
	return &serial.Config{
		Name:        addr,
		Baud:        19200,
		Size:        8,
		Parity:      serial.ParityNone,
		StopBits:    serial.Stop1,
		ReadTimeout: 1 * time.Second}
}

// Pressure is a struct holding a single variable P used for http responses
type Pressure struct {
	P float64 `json:"p"`
}

// EncodeAndRespond Encodes the data to JSON and writes to w.
// logs errors and replies with http.Error // status 500 on error
func (press *Pressure) EncodeAndRespond(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	err := json.NewEncoder(w).Encode(press)
	if err != nil {
		fstr := fmt.Sprintf("error encoding pressure data to json state %q", err)
		log.Println(fstr)
		http.Error(w, fstr, http.StatusInternalServerError)
	}
	return
}

// Sensor has a serial connection and can make commands
type Sensor struct {
	Conn *serial.Port
}

// NewGauge returns a new Sensor instance
func NewGauge(addr string) (Sensor, error) {
	cfg := MakeSerConf(addr)
	conn, err := serial.OpenPort(cfg)
	tc := Sensor{Conn: conn}
	return tc, err
}

// ReadAndReplyWithJSON read the sensor and reply with a JSON body
func (sens *Sensor) ReadAndReplyWithJSON(w http.ResponseWriter, r *http.Request) {
	data, err := sens.Read()
	if err != nil {
		fstr := fmt.Sprintf("unable to read data from sensor %+v, error %q", sens, err)
		log.Println(fstr)
		http.Error(w, fstr, http.StatusInternalServerError)
		return
	}
	p := Pressure{P: data}
	p.EncodeAndRespond(w, r)
	log.Printf("%s checked sensor, %e", r.RemoteAddr, data)
	return
}

// MkMsg generates a message that conforms to the custom schema used by the Sensor gauges
func (sens *Sensor) MkMsg(cmd string) []byte {
	return append([]byte("#01"+cmd), terminators...)
}

// Send sends a command to the controller.
// If not terminated by terminators, behavior is undefined
func (sens *Sensor) Send(cmd []byte) error {
	_, err := sens.Conn.Write(cmd)
	return err
}

// Recv data from the hardware and convert it to a string, stripping the leading *
func (sens *Sensor) Recv() (string, error) {
	reader := bufio.NewReader(sens.Conn)
	bytes, err := reader.ReadBytes('\r')
	bytes = bytes[1 : len(bytes)-2] // drop first char, "*", and last "\r"
	return string(bytes), err
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
	strs := strings.Split(resp, " ")
	protofloat := strings.TrimRight(strs[1], "\r")
	return strconv.ParseFloat(protofloat, 64)
}
