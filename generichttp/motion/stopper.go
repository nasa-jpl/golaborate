package motion

import (
	"net/http"

	"github.com/go-chi/chi"
	"github.com/nasa-jpl/golaborate/generichttp"
)

// Stopper describes an interface with stop-related methods for axes
type Stopper interface {
	// Stop aborts motion of the axis
	Stop(string) error
}

// HTTPStop adds routes for the mover to the route tabler
func HTTPStop(iface Stopper, table generichttp.RouteTable) {
	table[generichttp.MethodPath{Method: http.MethodPost, Path: "/axis/{axis}/stop"}] = Stop(iface)
}

// Home returns an HTTP handler func from a mover that homes an axis
func Stop(m Stopper) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		axis := chi.URLParam(r, "axis")
		err := m.Stop(axis)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

