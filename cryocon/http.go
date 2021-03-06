package cryocon

import (
	"encoding/json"
	"go/types"
	"math"
	"net/http"

	"github.com/nasa-jpl/golaborate/generichttp"
	"goji.io/pat"
)

// HTTPWrapper provides HTTP bindings on top of the underlying Go interface
// BindRoutes must be called on it
type HTTPWrapper struct {
	// Sensor is the underlying sensor that is wrapped
	TemperatureMonitor

	// RouteTable maps goji patterns to http handlers
	RouteTable generichttp.RouteTable
}

// NewHTTPWrapper returns a new HTTP wrapper with the route table pre-configured
func NewHTTPWrapper(m TemperatureMonitor) HTTPWrapper {
	w := HTTPWrapper{TemperatureMonitor: m}
	rt := generichttp.RouteTable{
		generichttp.MethodPath{Method: http.MethodGet, Path: "/read"}:     w.ReadAll,
		generichttp.MethodPath{Method: http.MethodGet, Path: "/read/:ch"}: w.ReadChan,
		generichttp.MethodPath{Method: http.MethodGet, Path: "/version"}:  w.Version,
	}
	w.RouteTable = rt
	return w
}

// RT satisfies the HTTPer interface
func (h HTTPWrapper) RT() generichttp.RouteTable {
	return h.RouteTable
}

// ReadAll reads all the channels and returns them as an array of f64 over JSON.  Units of Celcius.  NaN (no probe) encoded as -274.
func (h *HTTPWrapper) ReadAll(w http.ResponseWriter, r *http.Request) {
	f, err := h.TemperatureMonitor.ReadAllChannels()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	for idx := 0; idx < len(f); idx++ {
		if math.IsNaN(f[idx]) {
			f[idx] = -274
		}
	}
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(f)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	return
}

// ReadChan reads a single channel A~G (or so, may expand with future hardware)
// plucked from the URL and returns the value in Celcius as JSON
func (h *HTTPWrapper) ReadChan(w http.ResponseWriter, r *http.Request) {
	ch := pat.Param(r, "ch")
	f, err := h.TemperatureMonitor.ReadChannelLetter(ch)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	hp := generichttp.HumanPayload{T: types.Float64, Float: f}
	hp.EncodeAndRespond(w, r)
	return
}

// Version reads the version and sends it back as json
func (h *HTTPWrapper) Version(w http.ResponseWriter, r *http.Request) {
	v, err := h.TemperatureMonitor.Identification()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	hp := generichttp.HumanPayload{T: types.String, String: v}
	hp.EncodeAndRespond(w, r)
	return
}
