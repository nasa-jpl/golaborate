// Package generichttp defines interfaces for generic devices
// and an extensible type that wraps them in an HTTP interface
package generichttp

import (
	"encoding/json"
	"fmt"
	"go/types"
	"net/http"
	"sort"
	"strings"

	"github.jpl.nasa.gov/bdube/golab/util"
	"goji.io"
	"goji.io/pat"
)

// all of the following types are followed with a capital T for homogenaeity and
// avoiding clashes with builtins

// StrT is a struct with a single Str field
type StrT struct {
	Str string `json:"str"`
}

// FloatT is a struct with a single F64 field
type FloatT struct {
	F64 float64 `json:"f64"`
}

// UintT is a struct with a single Int field
type UintT struct {
	Int uint16 `json:"int"`
}

// ByteT is a struct with a single Int field
type ByteT struct {
	Int byte `json:"int"` // we won't distinguish between bytes and ints for users
}

// IntT is a struct with a single Int field
type IntT struct {
	Int int `json:"int"`
}

// Uint32T is a struct with a single int field.  It almost looks like IntT
type Uint32T struct {
	Uint uint32 `json:"uint"`
}

// BufferT is a struct with a single Int field
type BufferT struct {
	Int []byte `json:"int"`
}

// BoolT is a sutrct with a single Bool field
type BoolT struct {
	Bool bool `json:"bool"`
}

// HumanPayload is a struct containing the basic types NKT devices may work with
type HumanPayload struct {
	// Bool holds a binary value
	Bool bool

	// Buffer holds raw bytes
	Buffer []byte

	// Byte holds a single byte
	Byte byte

	// Int holds an int
	Int int

	// Float holds a float
	Float float64

	// String holds a string
	String string

	// Uint16 holds a uint16
	Uint16 uint16

	// T holds the type of data actually contained in the payload
	T types.BasicKind
}

// EncodeAndRespond converts the humanpayload to a smaller struct with only one
// field and writes it to w as JSON.
func (hp *HumanPayload) EncodeAndRespond(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	switch hp.T {
	case types.Bool:
		obj := BoolT{Bool: hp.Bool}

		// the logic from err to the closing brace is copy pasted a bunch in here
		err := json.NewEncoder(w).Encode(obj)
		if err != nil {
			fstr := fmt.Sprintf("error encoding %+v hp to JSON, %q", hp, err)
			http.Error(w, fstr, http.StatusInternalServerError)
		}
	// skip bytes case, unhandled in Unpack
	case types.Byte:
		obj := ByteT{Int: hp.Byte} // Byte -> int for consistency with uints

		err := json.NewEncoder(w).Encode(obj)
		if err != nil {
			fstr := fmt.Sprintf("error encoding %+v hp to JSON, %q", hp, err)
			http.Error(w, fstr, http.StatusInternalServerError)
		}
	case types.Int:
		obj := IntT{Int: hp.Int} // Byte -> int for consistency with uints

		err := json.NewEncoder(w).Encode(obj)
		if err != nil {
			fstr := fmt.Sprintf("error encoding %+v hp to JSON, %q", hp, err)
			http.Error(w, fstr, http.StatusInternalServerError)
		}
	case types.Float64:
		obj := FloatT{F64: hp.Float}

		err := json.NewEncoder(w).Encode(obj)
		if err != nil {
			fstr := fmt.Sprintf("error encoding %+v hp to JSON, %q", hp, err)
			http.Error(w, fstr, http.StatusInternalServerError)
		}
	case types.String:
		obj := StrT{Str: hp.String}

		err := json.NewEncoder(w).Encode(obj)
		if err != nil {
			fstr := fmt.Sprintf("error encoding %+v hp to JSON, %q", hp, err)
			http.Error(w, fstr, http.StatusInternalServerError)
		}
	case types.Uint16:
		obj := UintT{Int: hp.Uint16}
		err := json.NewEncoder(w).Encode(obj)
		if err != nil {
			fstr := fmt.Sprintf("error encoding %+v hp to JSON, %q", hp, err)
			http.Error(w, fstr, http.StatusInternalServerError)
		}

	}
}

// GetFloat calls a float-getting function and returns the response
// as json {'f64': value}
func GetFloat(fcn func() (float64, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		f, err := fcn()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		hp := HumanPayload{T: types.Float64, Float: f}
		hp.EncodeAndRespond(w, r)
		return
	}
}

// SetFloat parses a JSON input of {'f64': value} and
// calls fcn with it
func SetFloat(fcn func(float64) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		f := FloatT{}
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
		hp := HumanPayload{T: types.Int, Int: i}
		hp.EncodeAndRespond(w, r)
		return
	}
}

// SetInt parses a JSON input of {'int': value} and
// calls fcn with it
func SetInt(fcn func(int) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		f := IntT{}
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
		hp := HumanPayload{T: types.String, String: s}
		hp.EncodeAndRespond(w, r)
		return
	}
}

// SetString parses a JSON input of {'int': value} and
// calls fcn with it
func SetString(fcn func(string) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s := StrT{}
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
		hp := HumanPayload{T: types.Bool, Bool: b}
		hp.EncodeAndRespond(w, r)
		return
	}
}

// SetBool parses a JSON input of {'bool': value} and
// calls fcn with it
func SetBool(fcn func(bool) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		b := BoolT{}
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

// HTTPer is an interface which allows types to yield their route tables
// for processing
type HTTPer interface {
	RT() RouteTable
}

// RouteTable maps goji patterns to handler funcs
type RouteTable map[*pat.Pattern]http.HandlerFunc

// MethodPath is a path and method
type MethodPath struct {
	Method, Path string
}

// RouteTable2 is like RouteTable, but agnostic to the router backend instead
// of tied to Goji
type RouteTable2 map[MethodPath]http.HandlerFunc

// Endpoints returns the endpoints in the route table
func (rt RouteTable) Endpoints() []string {
	routes := make([]string, len(rt))
	idx := 0
	for key := range rt {
		routes[idx] = key.String()
		idx++
	}
	routes = util.UniqueString(routes)
	sort.Strings(routes)
	return routes
}

// EndpointsHTTP returns a function that encodes the endpoint list to a ResponseWriter
func (rt RouteTable) EndpointsHTTP() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		endpts := rt.Endpoints()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		err := json.NewEncoder(w).Encode(endpts)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

// Bind calls HandleFunc for each route in the table on the given mux.
// It also binds the endpoints route if it is not in the table already
func (rt RouteTable) Bind(mux *goji.Mux) {
	for ptrn, meth := range rt {
		mux.HandleFunc(ptrn, meth)
	}
	pg := pat.Get("/endpoints")
	if _, exists := rt[pg]; !exists {
		mux.HandleFunc(pg, rt.EndpointsHTTP())
	}
}

// SubMuxSanitize takes any string and ensures it begins with / and ends with /*
func SubMuxSanitize(str string) string {
	if !strings.HasPrefix(str, "/") {
		str = "/" + str
	}
	if !strings.HasSuffix(str, "/") {
		str += "/"
	}
	if !strings.HasSuffix(str, "*") {
		str += "*"
	}
	return str
}
