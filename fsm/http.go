package fsm

import (
	"encoding/json"
	"go/types"
	"net/http"
	"time"

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
	rt[pat.Get("/dt")] = disturbance.Getdt
	rt[pat.Post("/dt")] = disturbance.Setdt
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

	w.WriteHeader(http.StatusOK)
}

// Cursor sends back the current counter
// (useful after an error has stopped the loop)
func (hd HTTPDisturbance) Cursor(w http.ResponseWriter, r *http.Request) {
	hp := server.HumanPayload{T: types.Int, Int: hd.d.Cursor}
	hp.EncodeAndRespond(w, r)
}

// Getdt returns the time delta in seconds as fp64 over HTTP
func (hd HTTPDisturbance) Getdt(w http.ResponseWriter, r *http.Request) {
	hp := server.HumanPayload{T: types.Float64, Float: hd.d.PL.Interval.Seconds()}
	hp.EncodeAndRespond(w, r)
	return
}

// Setdt sets the time delta in seconds as fp64 over hTTP
func (hd HTTPDisturbance) Setdt(w http.ResponseWriter, r *http.Request) {
	f := server.FloatT{}
	err := json.NewDecoder(r.Body).Decode(&f)
	defer r.Body.Close()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	ns := int64(f.F64 * 1e9)
	tD := time.Duration(ns)
	hd.d.PL.Interval = tD
	w.WriteHeader(http.StatusOK)
	return
}
