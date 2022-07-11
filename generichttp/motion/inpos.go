package motion

import (
	"go/types"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/nasa-jpl/golaborate/generichttp"
)

// InpositionQueryer is a type which can query whether an axis is in position
type InPositionQueryer interface {
	// GetInPosition returns True if the axis is in position
	GetInPosition(string) (bool, error)
}

// GetInPosition returns an http.HandlerFunc for i.GetInPosition
func GetInPosition(i InPositionQueryer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		axis := chi.URLParam(r, "axis")
		enabled, err := i.GetInPosition(axis)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		hp := generichttp.HumanPayload{T: types.Bool, Bool: enabled}
		hp.EncodeAndRespond(w, r)
	}
}

// HTTPInPosition adds routes for InPosition to the route table
func HTTPInPosition(iface InPositionQueryer, table generichttp.RouteTable) {
	table[generichttp.MethodPath{Method: http.MethodGet, Path: "/axis/{axis}/inposition"}] = GetInPosition(iface)
}
