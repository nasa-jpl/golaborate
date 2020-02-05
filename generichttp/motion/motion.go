// Package motion provides an HTTP interface to motion controllers
package motion

/*
This file uses higher order / metaprogramming to efficiently bind the supported
interfaces for a motion controller, which may implement any number of them.
There are functions which consume a type that
*/
import (
	"bytes"
	"encoding/json"
	"errors"
	"go/types"
	"io/ioutil"
	"net/http"
	"strconv"

	"github.jpl.nasa.gov/HCIT/go-hcit/server"
	"github.jpl.nasa.gov/HCIT/go-hcit/util"
	"goji.io/pat"
)

var (
	errClamped = errors.New("requested position violates software limits, aborted")
)

// Enabler describes an interface with enable/disable methods for axes
type Enabler interface {
	// Enable enables an axis
	Enable(string) error

	// Disable disables an axis
	Disable(string) error

	// GetEnabled gets if an axis is enabled
	GetEnabled(string) (bool, error)
}

// HTTPEnable adds routes for the enabler to the route table
func HTTPEnable(iface Enabler, table server.RouteTable) {
	table[pat.Get("/axis/:axis/enabled")] = GetEnabled(iface)
	table[pat.Post("/axis/:axis/enabled")] = SetEnabled(iface)
}

// SetEnabled returns an HTTP handler func from an enabler that enables or disables the axis
func SetEnabled(e Enabler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		axis := pat.Param(r, "axis")
		boolT := server.BoolT{}
		err := json.NewDecoder(r.Body).Decode(&boolT)
		defer r.Body.Close()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if boolT.Bool {
			err = e.Enable(axis)
		} else {
			err = e.Disable(axis)
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		return
	}
}

// GetEnabled returns an HTTP handler func from an enabler that returns if the axis is enabled
func GetEnabled(e Enabler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		axis := pat.Param(r, "axis")
		enabled, err := e.GetEnabled(axis)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		hp := server.HumanPayload{T: types.Bool, Bool: enabled}
		hp.EncodeAndRespond(w, r)
	}
}

// Mover describes an interface with position-related methods for axes
type Mover interface {
	// GetPos gets the current position of an axis
	GetPos(string) (float64, error)

	// MoveAbs moves an axis to an absolute position
	MoveAbs(string, float64) error

	// MoveRel moves an axis a relative amount
	MoveRel(string, float64) error

	// Home homes an axis
	Home(string) error
}

// HTTPMove adds routes for the mover to the route tabler
func HTTPMove(iface Mover, table server.RouteTable) {
	table[pat.Post("/axis/:axis/home")] = Home(iface)
	table[pat.Get("/axis/:axis/pos")] = GetPos(iface)
	table[pat.Post("/axis/:axis/pos")] = SetPos(iface)
}

// GetPos returns an HTTP handler func from a mover that gets the position of an axis
func GetPos(m Mover) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		axis := pat.Param(r, "axis")
		pos, err := m.GetPos(axis)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		hp := server.HumanPayload{T: types.Float64, Float: pos}
		hp.EncodeAndRespond(w, r)
	}
}

func popAxisRelative(r *http.Request) (string, bool, error) {
	axis := pat.Param(r, "axis")
	relative := r.URL.Query().Get("relative")
	if relative == "" {
		relative = "false"
	}
	b, err := strconv.ParseBool(relative)
	return axis, b, err
}

// SetPos returns an HTTP handler func from a mover that triggers an absolute or
// relative move on an axis based on the relative query parameter
func SetPos(m Mover) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		axis, b, err := popAxisRelative(r)
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
			err = m.MoveRel(axis, f.F64)
		} else {
			err = m.MoveAbs(axis, f.F64)
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}

// Home returns an HTTP handler func from a mover that homes an axis
func Home(m Mover) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		axis := pat.Param(r, "axis")
		err := m.Home(axis)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

// Speeder describes an interface with velocity-related methods for axes
type Speeder interface {
	// SetVelocity sets the velocity setpoint on the axis
	SetVelocity(string, float64) error

	// GetVelocity gets the velocity setpoint on the axis
	GetVelocity(string) (float64, error)
}

// HTTPSpeed adds routes for the speeder to the route table
func HTTPSpeed(iface Speeder, table server.RouteTable) {
	table[pat.Post("/axis/:axis/velocity")] = SetVelocity(iface)
	table[pat.Get("/axis/:axis/velocity")] = GetVelocity(iface)
}

// SetVelocity returns an HTTP handler func which sets the velocity setpoint on an axis
func SetVelocity(s Speeder) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		axis := pat.Param(r, "axis")
		floatT := server.FloatT{}
		err := json.NewDecoder(r.Body).Decode(&floatT)
		defer r.Body.Close()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		err = s.SetVelocity(axis, floatT.F64)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		return
	}
}

// GetVelocity returns an HTTP handler func which gets the velocity setpoint on an axis
func GetVelocity(s Speeder) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		axis := pat.Param(r, "axis")
		vel, err := s.GetVelocity(axis)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		hp := server.HumanPayload{T: types.Float64, Float: vel}
		hp.EncodeAndRespond(w, r)
	}
}

// Initializer is a type which may initialize an axis
type Initializer interface {
	// Initialize an axis, engaging the control electronic controls
	Initialize(string) error
}

// HTTPInitialize adds routes for initialization to the route table
func HTTPInitialize(i Initializer, table server.RouteTable) {
	table[pat.Post("/axis/:axis/initialize")] = Initialize(i)
}

// Initialize returns an HTTP handler func that calls Initialize for an axis
func Initialize(i Initializer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		axis := pat.Param(r, "axis")
		err := i.Initialize(axis)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}

// LimitMiddleware is a type that can impose axis-specific limits on motion
// it returns a boolean "notOK" that indicates if the limit would be violated
// by a motion, stopping the chain of handling calls
type LimitMiddleware struct {
	// Limits contains the server imposed limits on the controller
	Limits map[string]util.Limiter

	// Mov is a reference to the mover, used to query axis positions
	Mov Mover
}

// Check verifies if a motion would violate the axis limit, if it exists,
// and if it does, responds with StatusBadRequest
// otherwise, flows control to the next handler
func (l *LimitMiddleware) Check(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// get the axis to move, and if the motion is relative
		axis, relative, err := popAxisRelative(r)
		// bail as early as possible if we don't have a limit for this axis
		limiter, ok := l.Limits[axis]
		if !ok {
			next.ServeHTTP(w, r)
			return
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		// get the command
		f := server.FloatT{}
		// downstream functions might want the body...
		// read it all here, then "paste" it back with ioutil
		bodyContent, _ := ioutil.ReadAll(r.Body)
		defer r.Body.Close()
		r.Body = ioutil.NopCloser(bytes.NewBuffer(bodyContent))
		err = json.NewDecoder(bytes.NewReader(bodyContent)).Decode(&f)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		cmd := f.F64
		if relative {
			// in the relative case, shift the command by currPos
			currPos, err := l.Mov.GetPos(axis)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			cmd += currPos
		}
		ok = limiter.Check(cmd)
		if !ok {
			http.Error(w, errClamped.Error(), http.StatusBadRequest)
			return
		}
		// at this point, all checks have passed and we can move on
		next.ServeHTTP(w, r)
	})
}

// Inject places a /axis/:axis/limits route on the table of the HTTPer
func (l LimitMiddleware) Inject(h server.HTTPer) {
	h.RT()[pat.Get("/axis/:axis/limits")] = Limits(l)
}

// Limits returns an HTTP handler func that returns the limits for an axis
func Limits(l LimitMiddleware) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		axis := pat.Param(r, "axis")
		lim, ok := l.Limits[axis]
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		var err error
		if !ok {
			err = json.NewEncoder(w).Encode(nil)
		} else {
			err = json.NewEncoder(w).Encode(lim)
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
}

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

	RouteTable server.RouteTable
}

// NewHTTPMotionController returns a new HTTP wrapper with the route table pre-configured
func NewHTTPMotionController(c Controller) HTTPMotionController {
	w := HTTPMotionController{Controller: c}
	rt := server.RouteTable{}
	// the interface{}().(foo); ok syntax is an awful go-ism to test if c implements foo
	HTTPMove(c, rt)
	if enabler, ok := interface{}(c).(Enabler); ok {
		HTTPEnable(enabler, rt)
	}
	if speeder, ok := interface{}(c).(Speeder); ok {
		HTTPSpeed(speeder, rt)
	}
	if initializer, ok := interface{}(c).(Initializer); ok {
		HTTPInitialize(initializer, rt)
	}
	w.RouteTable = rt
	return w
}

// RT satisfies the HTTPer interface
func (h HTTPMotionController) RT() server.RouteTable {
	return h.RouteTable
}
