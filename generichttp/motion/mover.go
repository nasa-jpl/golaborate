package motion

import (
	"encoding/json"
	"go/types"
	"net/http"
	"strconv"

	"github.com/go-chi/chi"
	"github.com/nasa-jpl/golaborate/generichttp"
)

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
func HTTPMove(iface Mover, table generichttp.RouteTable) {
	table[generichttp.MethodPath{Method: http.MethodPost, Path: "/axis/{axis}/home"}] = Home(iface)
	table[generichttp.MethodPath{Method: http.MethodGet, Path: "/axis/{axis}/pos"}] = GetPos(iface)
	table[generichttp.MethodPath{Method: http.MethodPost, Path: "/axis/{axis}/pos"}] = SetPos(iface)
}

// GetPos returns an HTTP handler func from a mover that gets the position of an axis
func GetPos(m Mover) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		axis := chi.URLParam(r, "axis")
		pos, err := m.GetPos(axis)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		hp := generichttp.HumanPayload{T: types.Float64, Float: pos}
		hp.EncodeAndRespond(w, r)
	}
}

func popAxisRelative(r *http.Request) (string, bool, error) {
	axis := chi.URLParam(r, "axis")
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
		f := generichttp.FloatT{}
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
		axis := chi.URLParam(r, "axis")
		err := m.Home(axis)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}
