// Package server contains reusable / embeddable code for http-ifying sensors and data
package server

import (
	"encoding/json"
	"fmt"
	"go/types"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"github.jpl.nasa.gov/HCIT/go-hcit/util"
	"goji.io"
	"goji.io/pat"
)

// ReplyWithFile replies to the client request by serving the given file name
func ReplyWithFile(w http.ResponseWriter, r *http.Request, fn string, fldr string) {
	filePath, err := filepath.Abs(filepath.Join(fldr, fn))
	if err != nil {
		fstr := fmt.Sprintf("unable to compute abspath of file %s %s %s", fldr, fn, err)
		http.Error(w, fstr, http.StatusInternalServerError)
		return
	}

	f, err := os.Open(filePath)
	defer f.Close()
	if err != nil {
		fstr := fmt.Sprintf("source file missing %s", filePath)
		http.Error(w, fstr, http.StatusNotFound)
		return
	}

	stat, err := f.Stat()
	if err != nil {
		fstr := fmt.Sprintf("error retrieving source file stats %s", err)
		http.Error(w, fstr, http.StatusNotFound)
		return
	}
	// read some stuff to set the headers appropriately
	http.ServeContent(w, r, fn, stat.ModTime(), f)
	return
}

// HTTPer is an interface which allows types to yield their route tables
// for processing
type HTTPer interface {
	RT() RouteTable
}

// RouteTable maps goji patterns to handler funcs
type RouteTable map[*pat.Pattern]http.HandlerFunc

// Endpoints returns the endpoints in the route table
func (rt RouteTable) Endpoints() []string {
	routes := make([]string, len(rt))
	idx := 0
	for key := range rt {
		routes[idx] = key.String()
		idx++
	}
	return routes
}

// Bind calls HandleFunc for each route in the table on the given mux
func (rt RouteTable) Bind(mux *goji.Mux) {
	for ptrn, meth := range rt {
		mux.HandleFunc(ptrn, meth)
	}
}

// BuildMux takes equal length slices of HTTPers and strings ("stems")
// and uses them to construct a goji mux with populated handlers.
// The mux serves a special route, route-list, which returns an
// array of strings containing all routes as JSON.
func BuildMux(https []HTTPer, strs []string) *goji.Mux {
	root := goji.NewMux()
	protomuxes := make([]string, 0, len(strs))
	leaves := make([]string, 0, len(strs))

	// pop the potential muxes off the paths
	for _, str := range strs {
		pieces := strings.Split(str, "/")
		proto := strings.Join(pieces[:len(pieces)-1], "/")
		leaf := pieces[len(pieces)-1]

		leaves = append(leaves, leaf)
		protomuxes = append(protomuxes, proto)
	}
	uniq := util.UniqueString(protomuxes)

	// now build a map of muxes and protos
	muxes := make(map[string]*goji.Mux, len(uniq))
	for _, str := range uniq {
		mux := goji.SubMux()
		root.Handle(pat.New(str+"/*"), mux)
		muxes[str] = mux
	}

	// collect all the endpoints and binx the muxes
	AllEndpoints := []string{}
	for idx := 0; idx < len(https); idx++ {
		// dump the endpoints
		h := https[idx]
		rt := h.RT()
		AllEndpoints = append(AllEndpoints, rt.Endpoints()...)

		// now bind the routes to the mux
		mux := muxes[protomuxes[idx]]
		leaf := leaves[idx]
		if leaf != "" {
			// if the leaf isn't blank, it means we need yet another mux
			newMux := goji.SubMux()
			mux.Handle(pat.New(leaf+"/*"), newMux)
			rt.Bind(newMux)
		} else {
			rt.Bind(mux)
		}
	}
	return root
}

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
