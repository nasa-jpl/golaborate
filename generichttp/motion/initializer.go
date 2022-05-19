package motion

import (
	"net/http"

	"github.com/go-chi/chi"
	"github.jpl.nasa.gov/bdube/golab/generichttp"
)

// Initializer is a type which may initialize an axis
type Initializer interface {
	// Initialize an axis, engaging the control electronic controls
	Initialize(string) error
}

// HTTPInitialize adds routes for initialization to the route table
func HTTPInitialize(i Initializer, table generichttp.RouteTable) {
	table[generichttp.MethodPath{Method: http.MethodPost, Path: "/axis/{axis}/initialize"}] = Initialize(i)
}

// Initialize returns an HTTP handler func that calls Initialize for an axis
func Initialize(i Initializer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		axis := chi.URLParam(r, "axis")
		err := i.Initialize(axis)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}
