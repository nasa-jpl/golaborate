package aerotech

import (
	"github.jpl.nasa.gov/HCIT/go-hcit/generichttp"
	"github.jpl.nasa.gov/HCIT/go-hcit/server"
)

// HTTPWrapper wraps an Ensemble for control with HTTP
type HTTPWrapper struct {
	// Ensemble is the embedded controller
	*Ensemble

	// HTTPWrapper is the embedded generic motion control wrapper
	HTTPWrapper generichttp.HTTPMotionController
}

// NewHTTPWrapper returns a new HTTP wrapper with the route table pre-configured
func NewHTTPWrapper(e *Ensemble) HTTPWrapper {
	basic := generichttp.NewHTTPMotionController(e)
	return HTTPWrapper{Ensemble: e, HTTPWrapper: basic}
}

// RT satisfies the HTTPer interface
func (h HTTPWrapper) RT() server.RouteTable {
	return h.HTTPWrapper.RouteTable
}
