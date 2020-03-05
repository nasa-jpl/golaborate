package fsm

import (
	"encoding/json"
	"go/types"
	"net/http"

	"github.jpl.nasa.gov/HCIT/go-hcit/server"
	"goji.io/pat"
)

// HTTPDisturbance is an HTTPer that exposes an HTTP interface to a disturbance
type HTTPDisturbance struct {
	d *Disturbance

	RouteTable server.RouteTable
}

// NewHTTPDisturbance creates an HTTP wrapper around a disturbance
// with pre-populated route table
func NewHTTPDisturbance(d *Disturbance) HTTPDisturbance {
	disturbance := HTTPDisturbance{d: d}
	rt := server.RouteTable{}
	rt[pat.Post("/csv")] = disturbance.AcceptCSV
	rt[pat.Post("/control")] = disturbance.Control
	rt[pat.Get("/cursor")] = disturbance.Cursor
	disturbance.RouteTable = rt
	return disturbance
}

// RT makes HTTPDisturbance conform to server.HTTPer
func (hd HTTPDisturbance) RT() server.RouteTable {
	return hd.RouteTable
}

// AcceptCSV downloads a CSV from the request and stores it in the buffer
func (hd HTTPDisturbance) AcceptCSV(w http.ResponseWriter, r *http.Request) {
	err := hd.d.LoadCSV(r.Body)
	defer r.Body.Close()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// Control issues a play, pause, resume, or stop command to the disturbance
func (hd HTTPDisturbance) Control(w http.ResponseWriter, r *http.Request) {
	str := server.StrT{}
	err := json.NewDecoder(r.Body).Decode(&str)
	defer r.Body.Close()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	switch str.Str {
	case "pause":
		hd.d.Pause()
	case "stop":
		hd.d.Stop()
	case "resume":
		hd.d.Resume()
	case "start":
		hd.d.Play()
	}
}

// Cursor sends back the current counter
// (useful after an error has stopped the loop)
func (hd HTTPDisturbance) Cursor(w http.ResponseWriter, r *http.Request) {
	hp := server.HumanPayload{T: types.Int, Int: hd.d.cursor}
	hp.EncodeAndRespond(w, r)
}
