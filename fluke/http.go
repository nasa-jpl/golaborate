package fluke

import (
	"net/http"

	"github.jpl.nasa.gov/HCIT/go-hcit/server"

	"goji.io/pat"
)

// HTTPWrapper provides HTTP bindings on top of the underlying Go interface
// BindRoutes must be called on it
type HTTPWrapper struct {
	// Sensor is the underlying sensor that is wrapped
	DewK

	// RouteTable maps goji patterns to http handlers
	RouteTable server.RouteTable
}

// NewHTTPWrapper returns a new HTTP wrapper with the route table pre-configured
func NewHTTPWrapper(dk DewK) HTTPWrapper {
	w := HTTPWrapper{DewK: dk}
	rt := server.RouteTable{
		pat.Get("read"): w.Read,
	}
	w.RouteTable = rt
	return w
}

// RT satisfies server.HTTPer
func (h HTTPWrapper) RT() server.RouteTable {
	return h.RouteTable
}

// Read reads the temp and humidity from the DewK and sends the response back as JSON
func (h HTTPWrapper) Read(w http.ResponseWriter, r *http.Request) {
	th, err := h.DewK.Read()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	th.EncodeAndRespond(w, r)
}
