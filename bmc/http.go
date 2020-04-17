package bmc

import (
	"encoding/json"
	"net/http"

	"github.jpl.nasa.gov/bdube/golab/server"
)

// HTTPWrapper wraps a DM in an HTTP control interface
type HTTPWrapper struct {
	*DM

	server.RouteTable
}

// RT satisfies server.HTTPer
func (h *HTTPWrapper) RT() server.RouteTable {
	return h.RouteTable
}

func NewHTTPWrapper(dm *DM) HTTPWrapper {
	w := HTTPWrapper{DM: dm}
	// rt :=
	return w
}

// single is used to decode single actuator commands over JSON
type single struct {
	Idx   int     `json:"idx"`
	Value float64 `json:"value"`
}

// jsonarray is used to decode array commands over JSON.
// this is very inefficient encoding and not suitable for high speed operation,
// but offers simplicity when speed is not paramount
type jsonarray struct {
	Value []float64 `json:"value"`
}

// Zero zeros all actuators of the DM
func (h HTTPWrapper) Zero(w http.ResponseWriter, r *http.Request) {
	err := Zero(h.DM)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// SetArray writes an array to the DM driver, it takes JSON for simplicity or a buffer of doubles for speed
func (h HTTPWrapper) SetArray(w http.ResponseWriter, r *http.Request) {
	var data []float64
	if r.Header.Get("Content-Type") == "application/json" {
		ja := jsonarray{}
		err := json.NewDecoder(r.Body).Decode(&ja)
		defer r.Body.Close()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		data = ja.Value
	} else {
		http.Error(w, "raw buffer not yet supported", http.StatusBadRequest)
		return
	}

	err := h.DM.SetArray(data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	return
}

// SetSingle writes a single command to the DM driver
func (h HTTPWrapper) SetSingle(w http.ResponseWriter, r *http.Request) {
	s := single{}
	err := json.NewDecoder(r.Body).Decode(&s)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	err = h.DM.SetSingle(s.Idx, s.Value)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}
