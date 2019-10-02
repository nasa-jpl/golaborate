package fluke

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/tarm/serial"
)

// MockableDewK sensor which may or may not be real
type MockableDewK interface {
	ReadAndReplyWithJSON() http.Handler
}

// DewK holds the address and connection type (TCP or serial) of a fluke sensor
type DewK struct {
	Addr, Conntype, Name string
}

// ReadAndReplyWithJSON reads the sensor over Conntype and responds with json-encoded TempHumic
func (dk *DewK) ReadAndReplyWithJSON(w http.ResponseWriter, r *http.Request) {
	var data TempHumid
	var err error
	if dk.Conntype == "TCP" { // this could be a switch if we need more than 2 types
		data, err = TCPPollDewKCh1(dk.Addr)
	} else {
		data, err = SerPollDewKCh1(dk.Addr)
	}
	if err != nil {
		fstr := fmt.Sprintf("unable to read data from DewK sensor %+v, error %q", dk, err)
		log.Println(fstr)
		http.Error(w, fstr, http.StatusInternalServerError)
		return
	}
	data.EncodeAndRespond(w, r)
	log.Printf("%s checked fluke %s, %+v", r.RemoteAddr, dk.Name, data)
	return

}

// MockDewK sensor that returns 22 +/- 1C temp and 10 +/- 1% RH
type MockDewK struct{}

// ReadAndReplyWithJSON returns 22 +/- 1C temp and 10 +/- 1% RH from a fake sensor
func (mdk *MockDewK) ReadAndReplyWithJSON(w http.ResponseWriter, r *http.Request) {
	var t, h float64
	h = 10 + rand.Float64()
	t = 22 + rand.Float64()
	th := TempHumid{T: t, H: h}
	th.EncodeAndRespond(w, r)
}

// TempHumid holds Temperature and Humidity data, T in C and H in % RH
type TempHumid struct {
	T float64 `json:"temp"`
	H float64 `json:"rh"`
}

// EncodeAndRespond Encodes the data to JSON and writes to w.
// logs errors and replies with http.Error // status 500 on error
func (th *TempHumid) EncodeAndRespond(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	err := json.NewEncoder(w).Encode(th)
	if err != nil {
		fstr := fmt.Sprintf("error encoding fluke data to json state %q", err)
		log.Println(fstr)
		http.Error(w, fstr, http.StatusInternalServerError)
	}
	return
}

// ParseTHFromBuffer converts a raw buffer looking like 21.4,6.5,0,0\r to a TempHumid object for channel 1
func ParseTHFromBuffer(buf []byte) (TempHumid, error) {
	// convert it to a string, then split on ",", and cast to float64
	str := string(buf)
	pieces := strings.SplitN(str, ",", 3)[:2] // 3 pieces potentially leaves the trailing trash, [:2] drops it
	numeric := make([]float64, 2)
	for i, v := range pieces {
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return TempHumid{}, err
		}
		numeric[i] = f
	}
	return TempHumid{T: numeric[0], H: numeric[1]}, nil
}

// TCPPollDewKCh1 reads temperature and humidity from a Fluke 1620a Thermo-Hygrometer over TCP/IP.
func TCPPollDewKCh1(ip string) (TempHumid, error) {
	// these meters communicate and port 10001.  They talk in raw TCP.
	// sending read? spits back data looking like 21.4,6.5,0,0\r
	// commas separate values.  Channels are all concat'd
	port := "10001"
	cmd := "read?\n"
	timeout := 3 * time.Second

	// open a tcp connection to the meter and send it our command
	conn, err := net.DialTimeout("tcp", ip+":"+port, timeout)
	if err != nil {
		return TempHumid{}, err
	}
	deadline := time.Now().Add(timeout)
	conn.SetReadDeadline(deadline)
	conn.SetWriteDeadline(deadline)
	defer conn.Close()
	if err != nil {
		return TempHumid{}, err
	}
	_, err = fmt.Fprintf(conn, cmd)
	if err != nil {
		return TempHumid{}, err
	}

	// make a new buffer reader and read up to \r
	reader := bufio.NewReader(conn)
	resp, err := reader.ReadBytes('\r')
	if err != nil {
		return TempHumid{}, err
	}
	return ParseTHFromBuffer(resp)
}

// SerPollDewKCh1 reads temperature and humidity from a Fluke 1620a Thermo-Hygrometer over serial.
func SerPollDewKCh1(addr string) (TempHumid, error) {
	conf := &serial.Config{
		Name:        addr,
		Baud:        9600,
		ReadTimeout: 1 * time.Second}

	conn, err := serial.OpenPort(conf)
	if err != nil {
		log.Printf("cannot open serial port %q", err)
		return TempHumid{}, err
	}
	reader := bufio.NewReader(conn)
	buf, err := reader.ReadBytes('\r')
	if err != nil {
		log.Printf("failed to read bytes from meter, %q", err)
		return TempHumid{}, err
	}
	return ParseTHFromBuffer(buf)
}
