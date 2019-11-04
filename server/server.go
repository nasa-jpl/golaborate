// Package server contains reusable / embeddable code for http-ifying sensors and data
package server

import (
	"encoding/json"
	"fmt"
	"go/types"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

// HandleFuncer is an interface that either the http module, or an HTTP ServeMux
// or equivalent (Gin, Goji, many others) satisfy
type HandleFuncer interface {
	// HandleFunc as defined in stdlib/http
	HandleFunc(string, func(http.ResponseWriter, *http.Request))
}

// ReplyWithFile replies to the client request by serving the given file name
func ReplyWithFile(w http.ResponseWriter, r *http.Request, fn string, fldr string) {

	filePath, err := filepath.Abs(filepath.Join(fldr, fn))
	if err != nil {
		fstr := fmt.Sprintf("unable to compute abspath of file %s %s %s", fldr, fn, err)
		log.Println(fstr)
		http.Error(w, fstr, http.StatusInternalServerError)
	}

	f, err := os.Open(filePath)
	defer f.Close()
	if err != nil {
		fstr := fmt.Sprintf("source file missing %s", filePath)
		log.Println(fstr)
		http.Error(w, fstr, http.StatusNotFound)
	}

	stat, err := f.Stat()
	if err != nil {
		fstr := fmt.Sprintf("error retrieving source file stats %s", err)
		log.Println(fstr)
		http.Error(w, fstr, http.StatusNotFound)
	}
	// read some stuff to set the headers appropriately
	http.ServeContent(w, r, fn, stat.ModTime(), f)
	return
}

// HTTPBinder is an object which knows how to bind methods to HTTP routes and can list them
type HTTPBinder interface {
	BindRoutes()
	ListRoutes() []string
	URLStem() string
}

// RouteTable maps URL endpoints to
type RouteTable map[string]http.HandlerFunc

// ListEndpoints lists the endpoints in a RouteTable (the keys)
func (rt RouteTable) ListEndpoints() []string {
	routes := make([]string, 0, len(rt))
	for k := range rt {
		routes = append(routes, k)
	}
	return routes
}

// A Server holds a RouteTable and implements HTTPBinder
type Server struct {
	//RouteTable is an instance of type RouteTable
	RouteTable RouteTable

	// stem is the string returned by URLStem to satisfy HTTPBinder
	Stem string
}

// NewServer returns a new Server instance
func NewServer(stem string) Server {
	return Server{RouteTable: make(RouteTable), Stem: stem}
}

// URLStem returns the head of all URLs returned in ListRoutes
func (s *Server) URLStem() string {
	return s.Stem
}

// BindRoutes binds routes on the default http server at stem+str
// for str in ListRoutes
func (s *Server) BindRoutes() {
	stem := s.URLStem()
	for str, meth := range s.RouteTable {
		http.HandleFunc(stem+"/"+str, meth)
	}
	return
}

// ListRoutes returns a slice of strings that includes all of the routes bound
// by this server
func (s *Server) ListRoutes() []string {
	return s.RouteTable.ListEndpoints()
}

// Mainframe is the top-level struct for an actual HTTP server with many
// Server objects that map to hardware and represent "services" to the end user
type Mainframe struct {
	nodes []HTTPBinder
}

// Add adds a new server to the mainframe
func (m *Mainframe) Add(s HTTPBinder) {
	m.nodes = append(m.nodes, s)
}

// RouteGraph returns a non-recursive, depth-1 map of URL stems and their endpoints
func (m *Mainframe) RouteGraph() map[string][]string {
	routes := make(map[string][]string)
	for _, s := range m.nodes {
		stem := s.URLStem()
		if _, ok := routes[stem]; ok {
			routes[stem] = append(routes[stem], s.ListRoutes()...)
		} else {
			routes[stem] = s.ListRoutes()
		}
	}
	return routes
}

func (m *Mainframe) graphHandler(w http.ResponseWriter, r *http.Request) {
	graph := m.RouteGraph()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	err := json.NewEncoder(w).Encode(graph)
	if err != nil {
		fstr := fmt.Sprintf("error encoding route graph to json state %q", err)
		log.Println(fstr)
		http.Error(w, fstr, http.StatusInternalServerError)
	}
	return
}

// BindRoutes binds the routes for each member service
func (m *Mainframe) BindRoutes(h HandleFuncer) {
	listRouteMap := make(map[string][]string)
	for _, s := range m.nodes {
		s.BindRoutes()
		stem := s.URLStem()
		if value, ok := listRouteMap[stem]; ok { // ok, key exists, concat the lists
			listRouteMap[stem] = append(value, s.ListRoutes()...)
		} else {
			listRouteMap[stem] = s.ListRoutes()
		}
	}

	for stem, listOfRoutes := range listRouteMap {
		h.HandleFunc(stem+"/route-graph", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			err := json.NewEncoder(w).Encode(listOfRoutes)
			if err != nil {
				fstr := fmt.Sprintf("error encoding list of routes data to json %q", err)
				log.Println(fstr)
				http.Error(w, fstr, http.StatusInternalServerError)
			}
		})
	}

	h.HandleFunc("/route-graph", m.graphHandler)

	return
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

	// Float holds a float
	Float float64

	// String holds a string
	String string

	// Uint16 holds a uint16
	Uint16 uint16

	// T holds the type of data actually contained in the payload
	T types.BasicKind
}

// EndianInterface is safisfied by both encoding/binary.BigEndian and LittleEndian
type EndianInterface interface {
	Uint16([]byte) uint16
}

// UnpackBinary converts the raw data from a register into a HumanPayload
func UnpackBinary(b []byte, typ types.BasicKind, endian EndianInterface) HumanPayload {
	var hp HumanPayload
	if len(b) == 0 {
		return HumanPayload{}
	}
	switch typ {
	case types.Uint16:
		v := endian.Uint16(b)
		hp = HumanPayload{Uint16: v}
	case types.Bool:
		v := uint8(b[0]) == 1
		hp = HumanPayload{Bool: v}
	case types.String:
		v := string(b)
		hp = HumanPayload{String: v}
	case types.Byte:
		v := b[0]
		hp = HumanPayload{Byte: v}
	default: // default is 10x superres floating point value
		v := endian.Uint16(b)
		hp = HumanPayload{Float: float64(v) / 10.0}
	}

	hp.T = typ
	return hp
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
			log.Println(fstr)
			http.Error(w, fstr, http.StatusInternalServerError)
		}
	// skip bytes case, unhandled in Unpack
	case types.Byte:
		obj := ByteT{Int: hp.Byte} // Byte -> int for consistency with uints

		err := json.NewEncoder(w).Encode(obj)
		if err != nil {
			fstr := fmt.Sprintf("error encoding %+v hp to JSON, %q", hp, err)
			log.Println(fstr)
			http.Error(w, fstr, http.StatusInternalServerError)
		}
	case types.Float64:
		obj := FloatT{F64: hp.Float}

		err := json.NewEncoder(w).Encode(obj)
		if err != nil {
			fstr := fmt.Sprintf("error encoding %+v hp to JSON, %q", hp, err)
			log.Println(fstr)
			http.Error(w, fstr, http.StatusInternalServerError)
		}
	case types.String:
		obj := StrT{Str: hp.String}

		err := json.NewEncoder(w).Encode(obj)
		if err != nil {
			fstr := fmt.Sprintf("error encoding %+v hp to JSON, %q", hp, err)
			log.Println(fstr)
			http.Error(w, fstr, http.StatusInternalServerError)
		}
	case types.Uint16:
		obj := UintT{Int: hp.Uint16}
		err := json.NewEncoder(w).Encode(obj)
		if err != nil {
			fstr := fmt.Sprintf("error encoding %+v hp to JSON, %q", hp, err)
			log.Println(fstr)
			http.Error(w, fstr, http.StatusInternalServerError)
		}

	}
}

// BadMethod returns an error if the request method is not GET or POST
func BadMethod(w http.ResponseWriter, r *http.Request) {
	fstr := fmt.Sprintf("%s queried %s with bad method %s, must be either GET or POST", r.RemoteAddr, r.URL, r.Method)
	log.Println(fstr)
	http.Error(w, fstr, http.StatusMethodNotAllowed)
}
