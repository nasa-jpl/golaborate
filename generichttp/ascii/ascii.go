// Package ascii contains some injectable HTTP interfaces to ASCII hardare
package ascii

import (
	"encoding/json"
	"go/types"
	"net/http"

	"github.jpl.nasa.gov/bdube/golab/server"
	"goji.io/pat"
)

// RawCommunicator has a single Raw method
type RawCommunicator interface {
	Raw(string) (string, error)
}

// RawWrapper is a wrapper around a raw communicator
type RawWrapper struct {
	Comm RawCommunicator
}

// HTTPRaw provides access to the raw function over http
func (rw *RawWrapper) HTTPRaw(w http.ResponseWriter, r *http.Request) {
	str := server.StrT{}
	err := json.NewDecoder(r.Body).Decode(&str)
	defer r.Body.Close()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	resp, err := rw.Comm.Raw(str.Str)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	hp := server.HumanPayload{T: types.String, String: resp}
	hp.EncodeAndRespond(w, r)
	return
}

// InjectRawComm injects a /raw POST route into the route table of an HTTPer
func InjectRawComm(other server.HTTPer, raw RawCommunicator) {
	wrap := RawWrapper{Comm: raw}
	rt := other.RT()
	rt[pat.Post("/raw")] = wrap.HTTPRaw
}
