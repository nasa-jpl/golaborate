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
func Inject(other server.HTTPer, l ManipulableLock) {
	rt := other.RT()
	if al, ok := (l).(*AxisLocker); ok {
		rt[pat.Get("/axis/:axis/lock")] = al.HTTPGet
		rt[pat.Post("/axis/:axis/lock")] = al.HTTPSet
	} else {
		rt[pat.Get("/lock")] = l.HTTPGet
		rt[pat.Post("/lock")] = l.HTTPSet
	}
}

// ManipulableLock describes a lock that can be checked and manipulated
type ManipulableLock interface {
	// Check checks if the resource is locked, and is a middleware
	Check(http.Handler) http.Handler

	// Get returns the status of the lock
	HTTPGet(http.ResponseWriter, *http.Request)

	// Set changes the status of the lock
	HTTPSet(http.ResponseWriter, *http.Request)
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
				w.Write([]byte("Access denied\n"))
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

// NewAL returns a new axis locker
func NewAL() *AxisLocker {
	return &AxisLocker{locked: map[string]*Locker{}}
}

// AxisLocker is a Locker, but for multi-axis devices, enabling granular locks (per-axis)
type AxisLocker struct {
	// locked maps if the axes are locked (true) or not (false)
	locked map[string]*Locker
}

// Check is an HTTP middleware that implements the locker
func (al *AxisLocker) Check(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		url := r.URL.String()
		if strings.Contains(url, "lock") {
			next.ServeHTTP(w, r)
			return
		}
		if !strings.Contains(url, "axis") {
			next.ServeHTTP(w, r)
			return
		}
		// pat.Param can panic.  If it panics, the route does not exist and we should 404
		defer func() {
			if r := recover(); r != nil {
				http.Error(w, "404 page not found", http.StatusNotFound)
				return
			}
		}()
		axis := pat.Param(r, "axis")
		locked, ok := al.locked[axis]
		if !ok {
			al.locked[axis] = New()
			locked = al.locked[axis]
		}
		if locked.isLocked {
			// check if the path is protected
			protected := true
			url := r.URL.Path
			for _, str := range locked.DoNotProtect {
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
func (al *AxisLocker) HTTPSet(w http.ResponseWriter, r *http.Request) {
	b := server.BoolT{}
	err := json.NewDecoder(r.Body).Decode(&b)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	axis := pat.Param(r, "axis")
	locked, ok := al.locked[axis]
	if !ok {
		al.locked[axis] = New()
		locked = al.locked[axis]
	}
	if b.Bool {
		locked.Lock()
	} else {
		locked.Unlock()
	}
	w.WriteHeader(http.StatusOK)
}

// HTTPGet returns Locked() over HTTP as JSON
func (al *AxisLocker) HTTPGet(w http.ResponseWriter, r *http.Request) {
	axis := pat.Param(r, "axis")
	locked, ok := al.locked[axis]
	if !ok {
		al.locked[axis] = New()
		locked = al.locked[axis]
	}
	b := locked.Locked()
	hp := server.HumanPayload{T: types.Bool, Bool: b}
	hp.EncodeAndRespond(w, r)
	return
}
