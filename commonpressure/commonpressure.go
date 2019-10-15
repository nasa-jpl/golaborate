// Package commonpressure works with pressure sensors speaking the lesker/gp dialect.
// It contains low-level mechanisms for dealing with serial connections and higher
// level mechanisms for providing an HTTP extraction over the device.
package commonpressure

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/tarm/serial"
	"github.jpl.nasa.gov/HCIT/go-hcit/comm"
	"github.jpl.nasa.gov/HCIT/go-hcit/server"
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
	comm.RemoteDevice
	server.Server
}

// NewSensor creates a new Sensor instance
func NewSensor(addr, urlStem string, serial bool) *Sensor {
	rd := comm.NewRemoteDevice(addr, serial)
	srv := server.Server{RouteTable: make(server.RouteTable), Stem: urlStem}
	s := Sensor{RemoteDevice: rd}
	srv.RouteTable["pressure"] = s.HTTPHandler
	s.Server = srv
	return &s
}

// SerialConf returns a serial config and satisfies SerialConfigurator
func (s *Sensor) SerialConf() serial.Config {
	return *MakeSerConf(s.Addr)
}

// Send overloads RemoteDevice.Send to prepend the "#01" that is expected by the sensor
func (s *Sensor) Send(b []byte) error {
	o := make([]byte, 0, len(b)+3)
	o = append(o, []byte("#01")...)
	o = append(o, b...)
	return s.RemoteDevice.Send(o)
}

// Read polls the Sensor for the current temperature and humidity, opening and closing a connection along the way
func (s *Sensor) Read() (Pressure, error) {
	cmd := []byte("RD")
	err := s.Open()
	if err != nil {
		return Pressure{}, err
	}
	defer s.Close()
	r, err := s.SendRecv(cmd)
	if err != nil {
		return Pressure{}, err
	}
	resp := string(r)
	strs := strings.Split(resp, " ")
	f, err := strconv.ParseFloat(strs[1], 64)
	if err != nil {
		return Pressure{}, err
	}
	return Pressure{P: f}, nil
}

// HTTPHandler handles the single route served by a Sensor
func (s *Sensor) HTTPHandler(w http.ResponseWriter, r *http.Request) {
	p, err := s.Read()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	p.EncodeAndRespond(w, r)
}
