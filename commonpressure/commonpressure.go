// Package commonpressure works with pressure sensors speaking the lesker/gp dialect.
// It contains low-level mechanisms for dealing with serial connections and higher
// level mechanisms for providing an HTTP extraction over the device.
package commonpressure

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.jpl.nasa.gov/HCIT/go-hcit/util"

	"github.com/tarm/serial"
)

const (
	termination = '\r'
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

// Sensor has an address and connection type and can make commands
type Sensor struct {
	Addr, ConnType string
}

// NewSensor returns a new Sensor instance
func NewSensor(addr, connType string) Sensor {
	return Sensor{
		Addr:     addr,
		ConnType: connType}
}

func (sens *Sensor) mkConn() (io.ReadWriteCloser, error) {
	switch sens.ConnType {
	case "TCP":
		return util.TCPSetup(sens.Addr, 3*time.Second)

	case "serial":
		cfg := MakeSerConf(sens.Addr)
		return serial.OpenPort(cfg)
	default:
		return nil, errors.New("ConnType must be TCP or serial")
	}

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
	return append([]byte("#01"+cmd), termination)
}

// SendRecv sends a command and receive an ASCII response that has already had SOT/EOT trimmed
func (sens *Sensor) SendRecv(cmd []byte) (string, error) {
	conn, err := sens.mkConn()
	if err != nil {
		return "", err
	}
	defer conn.Close()
	_, err = conn.Write(cmd)
	if err != nil {
		return "", err
	}
	reader := bufio.NewReader(conn)
	bytes, err := reader.ReadBytes(termination)
	bytes = bytes[1 : len(bytes)-2] // drop first char, "*", and last "\r"
	return string(bytes), err
}

// SWVersion returns the sensor ID from the controller
func (sens *Sensor) SWVersion() (string, error) {
	msg := sens.MkMsg("VER")
	return sens.SendRecv(msg)
}

func (sens *Sensor) Read() (float64, error) {
	msg := sens.MkMsg("RD")
	resp, err := sens.SendRecv(msg)
	if err != nil {
		return 0.0, err
	}
	fmt.Println(resp)
	strs := strings.Split(resp, " ")
	protofloat := strings.TrimRight(strs[1], "\r")
	return strconv.ParseFloat(protofloat, 64)
}

// BindRoutes binds HTTP routes to the methods of the pressure sensor.  This implements server.HTTPBinder.
// ex: BindRoutes("/dst") produces the following routes:
// /dst/pressure [GET] temperature and humidity, resp looks like {"P": 0.0004}  <- 4 millitorr
func (sens *Sensor) BindRoutes(stem string) {
	http.HandleFunc(stem+"/pressure", sens.ReadAndReplyWithJSON)
}
