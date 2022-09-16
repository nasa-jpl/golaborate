// Package motion provides an HTTP interface to motion controllers
package motion

/*
This file uses higher order / metaprogramming to efficiently bind the supported
interfaces for a motion controller, which may implement any number of them.
There are functions which consume a type that
*/
import (
	"github.com/nasa-jpl/golaborate/generichttp"
	"github.com/nasa-jpl/golaborate/generichttp/ascii"
)

// Controller is used for the HTTP interface, which will check if the concrete
// type satisfies the other interfaces in this package and inject their routes
// automaticlaly
type Controller interface {
	// Mover - all Controllers must be Movers
	Mover
}

// HTTPMotionController wraps a motion controller with HTTP
type HTTPMotionController struct {
	Controller

	RouteTable generichttp.RouteTable
}

// NewHTTPMotionController returns a new HTTP wrapper with the route table pre-configured
func NewHTTPMotionController(c Controller) HTTPMotionController {
	w := HTTPMotionController{Controller: c}
	rt := generichttp.RouteTable{}
	if rawer, ok := (c).(ascii.RawCommunicator); ok {
		ascii.InjectRawComm(rt, rawer)
	}
	HTTPMove(c, rt)
	if enabler, ok := (c).(Enabler); ok {
		HTTPEnable(enabler, rt)
	}
	if speeder, ok := (c).(Speeder); ok {
		HTTPSpeed(speeder, rt)
	}
	if initializer, ok := (c).(Initializer); ok {
		HTTPInitialize(initializer, rt)
	}
	if syncer, ok := (c).(SynchronizationController); ok {
		HTTPSync(syncer, rt)
	}
	if inposer, ok := (c).(InPositionQueryer); ok {
		HTTPInPosition(inposer, rt)
	}
	if homequerier, ok := (c).(HomeQuerier); ok {
		HTTPHomeQuery(homequerier, rt)
	}
	w.RouteTable = rt
	return w
}

// RT satisfies the HTTPer interface
func (h HTTPMotionController) RT() generichttp.RouteTable {
	return h.RouteTable
}
