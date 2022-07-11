package motion

import (
	"encoding/json"
	"go/types"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/nasa-jpl/golaborate/generichttp"
)

// Speeder describes an interface with velocity-related methods for axes
type Speeder interface {
	// SetVelocity sets the velocity setpoint on the axis
	SetVelocity(string, float64) error

	// GetVelocity gets the velocity setpoint on the axis
	GetVelocity(string) (float64, error)
}

// HTTPSpeed adds routes for the speeder to the route table
func HTTPSpeed(iface Speeder, table generichttp.RouteTable) {
	table[generichttp.MethodPath{Method: http.MethodPost, Path: "/axis/{axis}/velocity"}] = SetVelocity(iface)
	table[generichttp.MethodPath{Method: http.MethodGet, Path: "/axis/{axis}/velocity"}] = GetVelocity(iface)
}

// SetVelocity returns an HTTP handler func which sets the velocity setpoint on an axis
func SetVelocity(s Speeder) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		axis := chi.URLParam(r, "axis")
		floatT := generichttp.FloatT{}
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
		axis := chi.URLParam(r, "axis")
		vel, err := s.GetVelocity(axis)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		hp := generichttp.HumanPayload{T: types.Float64, Float: vel}
		hp.EncodeAndRespond(w, r)
	}
}
