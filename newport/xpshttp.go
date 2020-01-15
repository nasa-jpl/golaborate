package newport

import (
	"github.jpl.nasa.gov/HCIT/go-hcit/generichttp"
)

// NewXPSHTTPWrapper creates a new HTTP wrapper around an XPS controller
func NewXPSHTTPWrapper(xps *XPS) generichttp.HTTPMotionController {
	return generichttp.NewHTTPMotionController(xps)
}
