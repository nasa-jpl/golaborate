// Package server contains misc server utilities.
package server

import (
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
