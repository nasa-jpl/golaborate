package aerotech

import (
	"github.jpl.nasa.gov/HCIT/go-hcit/generichttp"
)

// NewHTTPWrapper returns a new HTTP wrapper with the route table pre-configured
func NewHTTPWrapper(e *Ensemble) generichttp.HTTPMotionController {
	return generichttp.NewHTTPMotionController(e)
}
