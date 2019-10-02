// Package omega enables working with DPF700-series flow meters.
package omega

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

// Flow is a struct holding a single variable f used for http responses
type Flow struct {
	F float64 `json:"f"`
}

// EncodeAndRespond Encodes the data to JSON and writes to w.
// logs errors and replies with http.Error // status 500 on error
func (f *Flow) EncodeAndRespond(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	err := json.NewEncoder(w).Encode(f)
	if err != nil {
		fstr := fmt.Sprintf("error encoding pressure data to json state %q", err)
		log.Println(fstr)
		http.Error(w, fstr, http.StatusInternalServerError)
	}
	return
}

// DPF700 has a serial connection and can make commands
type DPF700 struct {
	conn *serial.Port
}

func makeSerConf(addr string) *serial.Config {
	return &serial.Config{
		Name:        addr,
		Baud:        9600,
		Size:        7,
		Parity:      serial.ParityEven,
		StopBits:    serial.Stop1,
		ReadTimeout: 1 * time.Second}
}

// NewMeter returns a new DPF700 instance
func NewMeter(addr string) (DPF700, error) {
	cfg := makeSerConf(addr)
	conn, err := serial.OpenPort(cfg)
	tc := DPF700{conn: conn}
	return tc, err
}

// ReadAndReplyWithJSON read the sensor and reply with a JSON body
func (dfp *DPF700) ReadAndReplyWithJSON(w http.ResponseWriter, r *http.Request) {
	data, err := dfp.Read()
	if err != nil {
		fstr := fmt.Sprintf("unable to read data from sensor %+v, error %q", dfp, err)
		log.Println(fstr)
		http.Error(w, fstr, http.StatusInternalServerError)
		return
	}
	f := Flow{F: data}
	f.EncodeAndRespond(w, r)
	log.Printf("%s checked sensor, %e", r.RemoteAddr, data)
	return
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
	log.Println(resp)
	return strconv.ParseFloat(resp[1:], 64)
}
