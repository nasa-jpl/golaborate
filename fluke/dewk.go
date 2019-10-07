package fluke

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
)

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

// BindRoutes binds HTTP routes to the methods of the DewK.  This implements server.HTTPBinder.
// ex: BindRoutes("/zygo-table") produces the following routes:
// /zygo-table/temphumid [GET] temperature and humidity, resp looks like {"T": 21.64, "RH": 6.1}
func (dk *DewK) BindRoutes(stem string) {
	http.HandleFunc(stem+"/temphumid", dk.ReadAndReplyWithJSON)
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
