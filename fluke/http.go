package fluke

import (
	"net/http"

	"goji.io"
	"goji.io/pat"
)

// HTTPWrapper provides HTTP bindings on top of the underlying Go interface
// BindRoutes must be called on it
type HTTPWrapper struct {
	// Sensor is the underlying sensor that is wrapped
	Monitor *DewK

	// RouteTable maps goji patterns to http handlers
	RouteTable map[goji.Pattern]http.HandlerFunc
}

// NewHTTPWrapper returns a new HTTP wrapper with the route table pre-configured
func NewHTTPWrapper(urlStem string, m *DewK) HTTPWrapper {
	w := HTTPWrapper{Monitor: m}
	rt := map[goji.Pattern]http.HandlerFunc{
		pat.Get(urlStem + "read"): w.Read,
	}
	w.RouteTable = rt
	return w
}

// Read reads the temp and humidity from the DewK and sends the response back as JSON
func (h *HTTPWrapper) Read(w http.ResponseWriter, r *http.Request) {
	th, err := h.Monitor.Read()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	th.EncodeAndRespond(w, r)
}
