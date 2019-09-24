package serveraccess

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

// ServerStatus holds the current user, if the server is busy, and when the user
// took control
type ServerStatus struct {
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
func (stat *ServerStatus) NotifyActive(w http.ResponseWriter, r *http.Request) {
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
func (stat *ServerStatus) ReleaseActive(w http.ResponseWriter, r *http.Request) {
	log.Printf("released, %s last authed at %s, released by %s",
		stat.User,
		stat.WhenAuthed.Format(time.RFC822),
		r.RemoteAddr)

	stat = &ServerStatus{}
	w.WriteHeader(http.StatusOK)
}

// CheckActive takes a request and returns the JSON representation of stat
func (stat *ServerStatus) CheckActive(w http.ResponseWriter, r *http.Request) {
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
