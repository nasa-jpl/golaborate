// Package locker provides an HTTP middleware which allows an HTTPHandler to be locked, returning 423 (locked)
package locker

import (
	"encoding/json"
	"go/types"
	"net/http"
	"strings"

	"github.jpl.nasa.gov/HCIT/go-hcit/server"
	"goji.io/pat"
)

// Inject adds a lock route to a server.HTTPer which is used to manipulate the locker
func Inject(other server.HTTPer, l *Locker) {
	rt := other.RT()
	rt[pat.Get("/lock")] = l.HTTPGet
	rt[pat.Post("/lock")] = l.HTTPSet
}

// Locker is a type which behaves like a sync.Mutex without the blocking,
// and holds a list of routes (Goji patterns) to not protext
type Locker struct {
	isLocked bool

	// DoNotProtect is a list of paths not to apply the lock to
	DoNotProtect []string
}

// New returns a new Locker with DoNotProtect prepopulated with "lock"
func New() *Locker {
	return &Locker{DoNotProtect: []string{"lock"}}
}

// Lock the locker
func (l *Locker) Lock() {
	l.isLocked = true
}

// Unlock the locker
func (l *Locker) Unlock() {
	l.isLocked = false
}

// Locked returns true if the locker is locked
func (l *Locker) Locked() bool {
	return l.isLocked
}

// Check is an HTTP middleware that returns http.StatusLocked if Locked() is true, otherwise passes down the line
func (l *Locker) Check(next http.Handler) http.Handler {
	// return a handlerfunc wrapping a handler, middleware/generator pattern
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if l.Locked() {
			// check if the path is protected
			protected := true
			url := r.URL.Path
			for _, str := range l.DoNotProtect {
				if strings.Contains(url, str) {
					protected = false
				}
			}
			// if it is, bounce the request - locked
			if protected {
				w.WriteHeader(http.StatusLocked)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

// HTTPSet calls Lock or Unlock based on json:bool on the request body
func (l *Locker) HTTPSet(w http.ResponseWriter, r *http.Request) {
	b := server.BoolT{}
	err := json.NewDecoder(r.Body).Decode(&b)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if b.Bool {
		l.Lock()
	} else {
		l.Unlock()
	}
	w.WriteHeader(http.StatusOK)
}

// HTTPGet returns Locked() over HTTP as JSON
func (l *Locker) HTTPGet(w http.ResponseWriter, r *http.Request) {
	b := l.Locked()
	hp := server.HumanPayload{T: types.Bool, Bool: b}
	hp.EncodeAndRespond(w, r)
	return
}
