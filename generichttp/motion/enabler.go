package motion

import (
	"encoding/json"
	"go/types"
	"net/http"

	"github.com/go-chi/chi"
	"github.jpl.nasa.gov/bdube/golab/generichttp"
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
func HTTPEnable(iface Enabler, table generichttp.RouteTable) {
	table[generichttp.MethodPath{Method: http.MethodGet, Path: "/axis/{axis}/enabled"}] = GetEnabled(iface)
	table[generichttp.MethodPath{Method: http.MethodPost, Path: "/axis/{axis}/enabled"}] = SetEnabled(iface)
}

// SetEnabled returns an HTTP handler func from an enabler that enables or disables the axis
func SetEnabled(e Enabler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		axis := chi.URLParam(r, "axis")
		boolT := generichttp.BoolT{}
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
		axis := chi.URLParam(r, "axis")
		enabled, err := e.GetEnabled(axis)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		hp := generichttp.HumanPayload{T: types.Bool, Bool: enabled}
		hp.EncodeAndRespond(w, r)
	}
}
