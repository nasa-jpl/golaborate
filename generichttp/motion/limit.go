package motion

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/go-chi/chi"
	"github.jpl.nasa.gov/bdube/golab/generichttp"
	"github.jpl.nasa.gov/bdube/golab/util"
)

var (
	errClamped = errors.New("requested position violates software limits, aborted")
)

// LimitMiddleware is a type that can impose axis-specific limits on motion
// it returns a boolean "notOK" that indicates if the limit would be violated
// by a motion, stopping the chain of handling calls
type LimitMiddleware struct {
	// Limits contains the server imposed limits on the controller
	Limits map[string]util.Limiter

	// Mov is a reference to the mover, used to query axis positions
	Mov Mover
}

// Check verifies if a motion would violate the axis limit, if it exists,
// and if it does, responds with StatusBadRequest
// otherwise, flows control to the next handler
func (l *LimitMiddleware) Check(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.String(), "pos") || r.Method != http.MethodPost {
			next.ServeHTTP(w, r)
			return
		}
		// get the axis to move, and if the motion is relative
		axis, relative, err := popAxisRelative(r)
		// bail as early as possible if we don't have a limit for this axis
		limiter, ok := l.Limits[axis]
		if !ok {
			next.ServeHTTP(w, r)
			return
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		// get the command
		f := generichttp.FloatT{}
		// downstream functions might want the body...
		// read it all here, then "paste" it back with ioutil
		bodyContent, _ := ioutil.ReadAll(r.Body)
		defer r.Body.Close()
		r.Body = ioutil.NopCloser(bytes.NewBuffer(bodyContent))
		err = json.NewDecoder(bytes.NewReader(bodyContent)).Decode(&f)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		cmd := f.F64
		if relative {
			// in the relative case, shift the command by currPos
			currPos, err := l.Mov.GetPos(axis)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			cmd += currPos
		}
		ok = limiter.Check(cmd)
		if !ok {
			http.Error(w, errClamped.Error(), http.StatusBadRequest)
			return
		}
		// at this point, all checks have passed and we can move on
		next.ServeHTTP(w, r)
	})
}

// Inject places a /axis/{axis}/limits route on the table of the HTTPer
func (l LimitMiddleware) Inject(h generichttp.HTTPer) {
	h.RT()[generichttp.MethodPath{Method: http.MethodGet, Path: "/axis/{axis}/limits"}] = Limits(l)
}

// Limits returns an HTTP handler func that returns the limits for an axis
func Limits(l LimitMiddleware) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		axis := chi.URLParam(r, "axis")
		lim, ok := l.Limits[axis]
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		var err error
		if !ok {
			err = json.NewEncoder(w).Encode(nil)
		} else {
			err = json.NewEncoder(w).Encode(lim)
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
}
