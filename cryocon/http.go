package cryocon

import (
	"encoding/json"
	"go/types"
	"net/http"

	"github.jpl.nasa.gov/HCIT/go-hcit/server"

	"goji.io"
	"goji.io/pat"
)

// HTTPWrapper provides HTTP bindings on top of the underlying Go interface
// BindRoutes must be called on it
type HTTPWrapper struct {
	// Sensor is the underlying sensor that is wrapped
	Monitor *TemperatureMonitor

	// RouteTable maps goji patterns to http handlers
	RouteTable map[goji.Pattern]http.HandlerFunc
}

// NewHTTPWrapper returns a new HTTP wrapper with the route table pre-configured
func NewHTTPWrapper(urlStem string, m *TemperatureMonitor) HTTPWrapper {
	w := HTTPWrapper{Monitor: m}
	rt := map[goji.Pattern]http.HandlerFunc{
		pat.Get(urlStem + "read"):     w.ReadAll,
		pat.Get(urlStem + "read/:ch"): w.ReadChan,
		pat.Get(urlStem + "version"):  w.Version,
	}
	w.RouteTable = rt
	return w
}

// ReadAll reads all the channels and returns them as an array of f64 over JSON.  Units of Celcius.
func (h *HTTPWrapper) ReadAll(w http.ResponseWriter, r *http.Request) {
	f, err := h.Monitor.ReadAllChannels()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
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
	f, err := h.Monitor.ReadChannelLetter(ch)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	hp := server.HumanPayload{T: types.Float64, Float: f}
	hp.EncodeAndRespond(w, r)
	return
}

// Version reads the version and sends it back as text/plain
func (h *HTTPWrapper) Version(w http.ResponseWriter, r *http.Request) {
	v, err := h.Monitor.Identification()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(v))
	return
}
