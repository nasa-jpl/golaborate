// Package generichttp defines interfaces for generic devices
// and an extensible type that wraps them in an HTTP interface
package generichttp

import (
	"encoding/json"
	"go/types"
	"net/http"

	"github.jpl.nasa.gov/bdube/golab/server"
)

// GetFloat calls a float-getting function and returns the response
// as json {'f64': value}
func GetFloat(fcn func() (float64, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		f, err := fcn()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		hp := server.HumanPayload{T: types.Float64, Float: f}
		hp.EncodeAndRespond(w, r)
		return
	}
}

// SetFloat parses a JSON input of {'f64': value} and
// calls fcn with it
func SetFloat(fcn func(float64) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		f := server.FloatT{}
		err := json.NewDecoder(r.Body).Decode(&f)
		defer r.Body.Close()
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		err = fcn(f.F64)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}

// GetInt calls an int-getting function and returns the response
// as json {'int': value}
func GetInt(fcn func() (int, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		i, err := fcn()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		hp := server.HumanPayload{T: types.Int, Int: i}
		hp.EncodeAndRespond(w, r)
		return
	}
}

// SetInt parses a JSON input of {'int': value} and
// calls fcn with it
func SetInt(fcn func(int) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		f := server.IntT{}
		err := json.NewDecoder(r.Body).Decode(&f)
		defer r.Body.Close()
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		err = fcn(f.Int)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}

// GetString calls a string-getting function and returns the response
// as json {'str': value}
func GetString(fcn func() (string, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s, err := fcn()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		hp := server.HumanPayload{T: types.String, String: s}
		hp.EncodeAndRespond(w, r)
		return
	}
}

// SetString parses a JSON input of {'int': value} and
// calls fcn with it
func SetString(fcn func(string) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s := server.StrT{}
		err := json.NewDecoder(r.Body).Decode(&s)
		defer r.Body.Close()
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		err = fcn(s.Str)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}

// GetBool calls a bool-getting function and returns the response
// as json {'bool': value}
func GetBool(fcn func() (bool, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		b, err := fcn()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		hp := server.HumanPayload{T: types.Bool, Bool: b}
		hp.EncodeAndRespond(w, r)
		return
	}
}

// SetBool parses a JSON input of {'bool': value} and
// calls fcn with it
func SetBool(fcn func(bool) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		b := server.BoolT{}
		err := json.NewDecoder(r.Body).Decode(&b)
		defer r.Body.Close()
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		err = fcn(b.Bool)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}
