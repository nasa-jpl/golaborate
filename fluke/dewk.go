package fluke

import (
	"net/http"

	"github.jpl.nasa.gov/HCIT/go-hcit/comm"
	"github.jpl.nasa.gov/HCIT/go-hcit/server"
)

// DewK talks to a DewK 1620 temperature and humidity sensor
// and serves data HTTP routes and meta HTTP routes (route list)
type DewK struct {
	comm.RemoteDevice
	server.Server
}

// NewDewK creates a new DewK instance
func NewDewK(addr string, urlStem string, serial bool) *DewK {
	if !serial {
		addr = addr + ":10001"
	}
	rd := comm.NewRemoteDevice(addr, serial)
	srv := server.Server{RouteTable: make(server.RouteTable), Stem: urlStem}
	dk := DewK{RemoteDevice: rd}
	srv.RouteTable["temphumid"] = dk.HTTPHandler
	dk.Server = srv
	return &dk
}

// Read polls the DewK for the current temperature and humidity, opening and closing a connection along the way
func (dk *DewK) Read() (TempHumid, error) {
	cmd := []byte("read?")
	err := dk.Open()
	if err != nil {
		return TempHumid{}, err
	}
	defer dk.Close()
	resp, err := dk.SendRecv(cmd)
	if err != nil {
		return TempHumid{}, err
	}
	return ParseTHFromBuffer(resp)
}

// HTTPHandler handles the single route served by a DewK
func (dk *DewK) HTTPHandler(w http.ResponseWriter, r *http.Request) {
	th, err := dk.Read()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	th.EncodeAndRespond(w, r)
}
