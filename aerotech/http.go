package aerotech

import (
	"encoding/json"
	"go/types"
	"net/http"
	"strconv"

	"github.jpl.nasa.gov/HCIT/go-hcit/server"
	"goji.io/pat"
)

// HTTPWrapper wraps an Ensemble for control with HTTP
type HTTPWrapper struct {
	Ensemble

	RouteTable server.RouteTable
}

// NewHTTPWrapper returns a new HTTP wrapper with the route table pre-configured
func NewHTTPWrapper(e Ensemble) HTTPWrapper {
	w := HTTPWrapper{Ensemble: e}
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
	err := h.Ensemble.Enable(axis)
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
	enabled, err := h.Ensemble.GetAxisEnabled(axis)
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
	err := h.Ensemble.Home(axis)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// GetPos gets the absolute position of an axis
func (h HTTPWrapper) GetPos(w http.ResponseWriter, r *http.Request) {
	axis := pat.Param(r, "axis")
	pos, err := h.Ensemble.GetPos(axis)
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
		err = h.Ensemble.MoveRel(axis, f.F64)
	} else {
		err = h.Ensemble.MoveAbs(axis, f.F64)
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}
