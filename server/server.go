package server

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/spf13/viper"
)

// Status holds the current user, if the server is busy, and when the user
// took control
type Status struct {
	User       string
	Busy       bool
	WhenAuthed time.Time
}

// AuthRequest is a passthrough struct allowing a User variale to be extracted
// from JSON
type AuthRequest struct {
	User string `json:"user"`
}

// NotifyActive takes POST requests with json like {"user": "foo"} and
// updates stat with it.  It logs errors and returns 404/BadRequest or
// returns 200/OK for a valid request
func (stat *Status) NotifyActive(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	decoder := json.NewDecoder(r.Body)
	var dat AuthRequest
	err := decoder.Decode(&dat)
	if err != nil {
		fstr := fmt.Sprintf("/notify-error cannot decode request, need JSON field \"body\" %s", err)
		log.Println(fstr)
		http.Error(w, fstr, http.StatusBadRequest)
	} else {
		stat.User = dat.User
		stat.Busy = true
		stat.WhenAuthed = time.Now()
		w.WriteHeader(http.StatusOK)
		log.Printf("user %s notified at %s from %s",
			stat.User,
			stat.WhenAuthed.Format(time.RFC822),
			r.RemoteAddr)
	}
	return
}

// ReleaseActive takes a request, does nothing with its contents, clears stat
// responds with 200/OK, and logs that control was released
func (stat *Status) ReleaseActive(w http.ResponseWriter, r *http.Request) {
	log.Printf("released, %s last authed at %s, released by %s",
		stat.User,
		stat.WhenAuthed.Format(time.RFC822),
		r.RemoteAddr)

	stat = &Status{}
	w.WriteHeader(http.StatusOK)
}

// CheckActive takes a request and returns the JSON representation of stat
func (stat *Status) CheckActive(w http.ResponseWriter, r *http.Request) {
	enc := json.NewEncoder(w)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	err := enc.Encode(stat)
	if err != nil {
		fstr := fmt.Sprintf("/check-active error encoding server state %s", err)
		log.Println(fstr)
		http.Error(w, fstr, http.StatusBadRequest)
	} else {

		log.Printf("activity checked from %s", r.RemoteAddr)
	}
	return
}

// ParseCleanup extracts a boolean value (cleanup) from URL query parameter "cleanup"
func ParseCleanup(w http.ResponseWriter, r *http.Request) bool {
	// if true, delete the file after serving it
	cleanupStr := r.URL.Query().Get("cleanup")
	if cleanupStr == "" {
		cleanupStr = "false"
	}
	cleanup, ok := strconv.ParseBool(cleanupStr)
	if ok != nil {
		fstr := fmt.Sprintf("cleanup URL parameter error, given %s, cannot be converted to float", cleanupStr)
		log.Println(fstr)
		http.Error(w, fstr, http.StatusBadRequest)
	}

	return cleanup
}

// ParseFilename extracts a string from URL query parameter "filename"
// or defaults to "tmp.dat" if one is not given
func ParseFilename(w http.ResponseWriter, r *http.Request) string {
	filename := r.URL.Query().Get("filename")
	if filename == "" {
		filename = "tmp.dat"
	}
	return filename
}

// ReplyWithFile replies to the client request by serving the given file name
func ReplyWithFile(w http.ResponseWriter, r *http.Request) {
	filename := ParseFilename(w, r)

	fldr := viper.GetString("zygoFileFolder")
	if viper.GetBool("spoofFile") {
		filename = "test.txt"
		fldr = "."
	}
	filePath, err := filepath.Abs(filepath.Join(fldr, filename))
	if err != nil {
		fstr := fmt.Sprintf("unable to compute abspath of file %s %s %s", fldr, filename, err)
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

	// read some stuff to set the headers appropriately
	filename = filepath.Base(filePath)
	fStat, _ := f.Stat()
	fSize := strconv.FormatInt(fStat.Size(), 10) // base 10 int
	cDistStr := fmt.Sprintf("attachment; filename=\"%s\"", filename)
	w.Header().Set("Content-Disposition", cDistStr)
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", fSize)
	w.WriteHeader(200)

	// copy the file to the client and print to the log
	io.Copy(w, f)

	if ParseCleanup(w, r) {
		err := os.Remove(filePath)
		if err != nil {
			log.Println(err)
			http.Error(w, fmt.Sprintf("%s", err), http.StatusInternalServerError)
		}
	}
	return
}
