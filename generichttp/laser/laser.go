// Package laser exposes control of laser controllers over HTTP
package laser

import (
	"encoding/json"
	"go/types"
	"net/http"

	"github.jpl.nasa.gov/HCIT/go-hcit/server"
	"goji.io/pat"
)

// Controller is a basic interface for laser controllers
type Controller interface {
	// EmissionOn turns emission On
	EmissionOn() error

	// EmissionOff turns emission Off
	EmissionOff() error

	// EmissionIsOne returns if the laser is emitting
	EmissionIsOn() (bool, error)

	// SetCurrent sets the output current setpoint of the controller
	SetCurrent(float64) error

	// GetCurrent retrieves the output current setpoint of the controller
	GetCurrent() (float64, error)
}

// HTTPLaserController wraps a LaserController in an HTTP route table
type HTTPLaserController struct {
	// Ctl is the underlying laser controller
	Ctl Controller

	// RouteTable maps URLs to functions
	RouteTable server.RouteTable
}

// NewHTTPLaserController returns a new HTTP wrapper around an existing laser controller
func NewHTTPLaserController(ctl Controller) HTTPLaserController {
	h := HTTPLaserController{Ctl: ctl}
	rt := server.RouteTable{
		pat.Get("/emission"):  h.GetEmission,
		pat.Post("/emission"): h.SetEmission,
		pat.Get("/current"):   h.GetCurrent,
		pat.Post("/current"):  h.SetCurrent,
	}
	h.RouteTable = rt
	return h
}

// RT safisfies the server.HTTPer interface
func (h HTTPLaserController) RT() server.RouteTable {
	return h.RouteTable
}

// SetEmission turns emission on or off based on a json payload
func (h *HTTPLaserController) SetEmission(w http.ResponseWriter, r *http.Request) {
	bT := server.BoolT{}
	err := json.NewDecoder(r.Body).Decode(&bT)
	defer r.Body.Close()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if bT.Bool {
		err = h.Ctl.EmissionOn()
	} else {
		err = h.Ctl.EmissionOff()
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// GetEmission returns json {'bool': <T/F>} with the current emission status
func (h *HTTPLaserController) GetEmission(w http.ResponseWriter, r *http.Request) {
	b, err := h.Ctl.EmissionIsOn()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	bT := server.HumanPayload{Bool: b, T: types.Bool}
	bT.EncodeAndRespond(w, r)
}

// SetCurrent sets the output current of the controller in mA
func (h *HTTPLaserController) SetCurrent(w http.ResponseWriter, r *http.Request) {
	fT := server.FloatT{}
	err := json.NewDecoder(r.Body).Decode(&fT)
	defer r.Body.Close()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	err = h.Ctl.SetCurrent(fT.F64)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// GetCurrent gets the output current of the controller in mA
func (h *HTTPLaserController) GetCurrent(w http.ResponseWriter, r *http.Request) {
	f, err := h.Ctl.GetCurrent()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	fT := server.HumanPayload{Float: f, T: types.Float64}
	fT.EncodeAndRespond(w, r)
}
