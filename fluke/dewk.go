package fluke

import (
	"github.jpl.nasa.gov/HCIT/go-hcit/comm"
)

// DewK talks to a DewK 1620 temperature and humidity sensor
// and serves data HTTP routes and meta HTTP routes (route list)
type DewK struct {
	*comm.RemoteDevice
}

// NewDewK creates a new DewK instance
func NewDewK(addr string, urlStem string, serial bool) *DewK {
	if !serial {
		addr = addr + ":10001"
	}
	rd := comm.NewRemoteDevice(addr, serial, nil, nil)
	return &DewK{RemoteDevice: &rd}
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
