// Package server contains misc server utilities.
package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

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
	BindRoutes(string)
	ListRoutes()
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
	RouteTable RouteTable
}

// BindRoutes binds routes on the default http server at stem+str
// for str in ListRoutes
func (s *Server) BindRoutes(stem string) {
	for str, meth := range s.RouteTable {
		http.HandleFunc(stem+"/"+str, meth)
	}

	http.HandleFunc(stem+"/"+"list-of-routes", func(w http.ResponseWriter, r *http.Request) {
		list := s.ListRoutes()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		err := json.NewEncoder(w).Encode(list)
		if err != nil {
			fstr := fmt.Sprintf("error encoding list of routes data to json %q", err)
			log.Println(fstr)
			http.Error(w, fstr, http.StatusInternalServerError)
		}
	})

	return
}

// ListRoutes returns a slice of strings that includes all of the routes bound
// by this server
func (s *Server) ListRoutes() []string {
	return s.RouteTable.ListEndpoints()
}
