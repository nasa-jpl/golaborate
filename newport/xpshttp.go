package newport

import (
	"net/http"

	"github.jpl.nasa.gov/HCIT/go-hcit/server"

	"github.jpl.nasa.gov/HCIT/go-hcit/motion"
	"goji.io/pat"
)

// XPSHTTPWrapper is an HTTP wrapper around an XPS motion controller.
//
// The API is a superset of the generic motion controller interface
type XPSHTTPWrapper struct {
	// XPS is the embedded XPS controller
	*XPS

	motion.HTTPWrapper
}

// NewXPSHTTPWrapper creates a new HTTP wrapper around an XPS controller
func NewXPSHTTPWrapper(xps *XPS) XPSHTTPWrapper {
	basic := motion.NewHTTPWrapper(xps)
	w := XPSHTTPWrapper{XPS: xps, HTTPWrapper: basic}
	basic.RouteTable[pat.Post("/axis/:axis/initialize")] = w.Initialize
	return w
}

// Initialize initializes the specified axis
func (h XPSHTTPWrapper) Initialize(w http.ResponseWriter, r *http.Request) {
	axis := pat.Param(r, "axis")
	err := h.XPS.Initialize(axis)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// RT satisfies server.HTTPer
func (h XPSHTTPWrapper) RT() server.RouteTable {
	return h.HTTPWrapper.RouteTable
}
