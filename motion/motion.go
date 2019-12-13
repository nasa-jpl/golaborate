// Package motion contains an abstract interface for a motion controller
// and HTTP wrapper layer.
package motion

import (
	"encoding/json"
	"go/types"
	"net/http"
	"strconv"

	"github.jpl.nasa.gov/HCIT/go-hcit/server"
	"goji.io/pat"
)

// Controller describes a set of methods on a rudimentary motion controller
type Controller interface {
	// Enable enables an axis
	Enable(string) error

	// Disable disables an axis
	Disable(string) error

	// GetEnabled gets if an axis is enabled
	GetEnabled(string) (bool, error)

	// GetPos gets the current position of an axis
	GetPos(string) (float64, error)

	// MoveAbs moves an axis to an absolute position
	MoveAbs(string, float64) error

	// MoveRel moves an axis a relative amount
	MoveRel(string, float64) error

	// Home homes an axis
	Home(string) error
}

// HTTPWrapper wraps a motion controller with HTTP
type HTTPWrapper struct {
	Controller

	RouteTable server.RouteTable
}

// NewHTTPWrapper returns a new HTTP wrapper with the route table pre-configured
func NewHTTPWrapper(c Controller) HTTPWrapper {
	w := HTTPWrapper{Controller: c}
	rt := server.RouteTable{
		// enable/disable
		pat.Get("/axis/:axis/enabled"):  w.GetAxisEnabled,
		pat.Post("/axis/:axis/enabled"): w.SetAxisEnabled,

		// home
		pat.Post("/axis/:axis/home"): w.HomeAxis,

		// position
		pat.Get("/axis/:axis/pos"):  w.GetPos,
		pat.Post("/axis/:axis/pos"): w.SetPos,
	}
	w.RouteTable = rt
	return w
}

// RT satisfies the HTTPer interface
func (h HTTPWrapper) RT() server.RouteTable {
	return h.RouteTable
}

// SetAxisEnabled enables or disables an axis
func (h HTTPWrapper) SetAxisEnabled(w http.ResponseWriter, r *http.Request) {
	axis := pat.Param(r, "axis")
	err := h.Controller.Enable(axis)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	return
}

// GetAxisEnabled gets if an axis is enabled or disabled
func (h HTTPWrapper) GetAxisEnabled(w http.ResponseWriter, r *http.Request) {
	axis := pat.Param(r, "axis")
	enabled, err := h.Controller.GetEnabled(axis)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	hp := server.HumanPayload{T: types.Bool, Bool: enabled}
	hp.EncodeAndRespond(w, r)
}

// HomeAxis homes an axis
func (h HTTPWrapper) HomeAxis(w http.ResponseWriter, r *http.Request) {
	axis := pat.Param(r, "axis")
	err := h.Controller.Home(axis)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// GetPos gets the absolute position of an axis
func (h HTTPWrapper) GetPos(w http.ResponseWriter, r *http.Request) {
	axis := pat.Param(r, "axis")
	pos, err := h.Controller.GetPos(axis)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	hp := server.HumanPayload{T: types.Float64, Float: pos}
	hp.EncodeAndRespond(w, r)
}

// SetPos sets the position of an axis, and takes a rel query parameter
// to adjust in a relative, rather than absolute, manner
func (h HTTPWrapper) SetPos(w http.ResponseWriter, r *http.Request) {
	axis := pat.Param(r, "axis")
	relative := r.URL.Query().Get("relative")
	if relative == "" {
		relative = "false"
	}
	b, err := strconv.ParseBool(relative)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	f := server.FloatT{}
	err = json.NewDecoder(r.Body).Decode(&f)
	defer r.Body.Close()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if b {
		err = h.Controller.MoveRel(axis, f.F64)
	} else {
		err = h.Controller.MoveAbs(axis, f.F64)
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}
