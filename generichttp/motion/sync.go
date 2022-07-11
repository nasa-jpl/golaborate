package motion

import (
	"encoding/json"
	"go/types"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/nasa-jpl/golaborate/generichttp"
)

// SynchronizationController is a type which can control whether an axis is
// in synchronous mode or not
type SynchronizationController interface {
	// SetSynchronous places axis (string) in sync mode
	SetSynchronous(string, bool) error

	// GetSynchronous queries whether axis (string) is in sync mode
	GetSynchronous(string) (bool, error)
}

// SetSynchronous returns an http.HandlerFunc for s.SetSynchronous
func SetSynchronous(s SynchronizationController) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		axis := chi.URLParam(r, "axis")
		boolT := generichttp.BoolT{}
		err := json.NewDecoder(r.Body).Decode(&boolT)
		defer r.Body.Close()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		err = s.SetSynchronous(axis, boolT.Bool)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		return
	}
}

// GetSynchronous returns an http.HandlerFunc for s.GetSynchronous
func GetSynchronous(s SynchronizationController) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		axis := chi.URLParam(r, "axis")
		enabled, err := s.GetSynchronous(axis)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		hp := generichttp.HumanPayload{T: types.Bool, Bool: enabled}
		hp.EncodeAndRespond(w, r)
	}
}

// HTTPInitialize adds routes for synchronization to the route table
func HTTPSync(iface SynchronizationController, table generichttp.RouteTable) {
	table[generichttp.MethodPath{Method: http.MethodGet, Path: "/axis/{axis}/synchronous"}] = GetSynchronous(iface)
	table[generichttp.MethodPath{Method: http.MethodPost, Path: "/axis/{axis}/synchronous"}] = SetSynchronous(iface)
}
