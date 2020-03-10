// Package tmc provides an HTTP interface to test and measurement devices
package tmc

import (
	"encoding/json"
	"go/types"
	"net/http"

	"github.jpl.nasa.gov/HCIT/go-hcit/server"
	"goji.io/pat"
)

func getFloat(fcn func() (float64, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		f, err := fcn()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		hp := server.HumanPayload{T: types.Float64, Float: f}
		hp.EncodeAndRespond(w, r)
		return
	}
}

func setFloat(fcn func(float64) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		f := server.FloatT{}
		err := json.NewDecoder(r.Body).Decode(&f)
		defer r.Body.Close()
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		err = fcn(f.F64)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}

func getString(fcn func() (string, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s, err := fcn()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		hp := server.HumanPayload{T: types.String, String: s}
		hp.EncodeAndRespond(w, r)
		return
	}
}

func setString(fcn func(string) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s := server.StrT{}
		err := json.NewDecoder(r.Body).Decode(&s)
		defer r.Body.Close()
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		err = fcn(s.Str)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}

func getBool(fcn func() (bool, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		b, err := fcn()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		hp := server.HumanPayload{T: types.Bool, Bool: b}
		hp.EncodeAndRespond(w, r)
		return
	}
}

func setBool(enable, disable func() error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		b := server.BoolT{}
		err := json.NewDecoder(r.Body).Decode(&b)
		defer r.Body.Close()
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if b.Bool {
			err = enable()
		} else {
			err = disable()
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}

// FunctionGenerator describes an interface to a function generator
type FunctionGenerator interface {
	// SetFunctions sets the function
	SetFunction(string) error

	// GetFunction returns the current function type used
	GetFunction() (string, error)

	// SetFrequency configures the frequency of the output waveform
	SetFrequency(float64) error

	// GetFrequency gets the frequency of the output waveform
	GetFrequency() (float64, error)

	// SetVoltage configures the voltage of the output waveform
	SetVoltage(float64) error

	// GetVoltage retrieves the voltage of the output waveform
	GetVoltage() (float64, error)

	// SetOffset configures the offset of the output waveform
	SetOffset(float64) error

	// GetOffset retrieves the offset of the output waveform
	GetOffset() (float64, error)

	// EnableOutput begins outputting the signal on the output connector
	EnableOutput() error

	// DisableOutput ceases output on the output connector
	DisableOutput() error

	// GetOutput queries if the generator output is active
	GetOutput() (bool, error)
}

// HTTPFunctionGenerator injects an HTTP interface to a function generator into a route table
func HTTPFunctionGenerator(fg FunctionGenerator, table server.RouteTable) {
	rt := table
	rt[pat.Get("/function")] = GetFunction(fg)
	rt[pat.Post("/function")] = SetFunction(fg)
	rt[pat.Get("/frequency")] = GetFrequency(fg)
	rt[pat.Post("/frequency")] = SetFrequency(fg)

	rt[pat.Get("/voltage")] = GetVoltage(fg)
	rt[pat.Post("/voltage")] = SetVoltage(fg)

	rt[pat.Get("/offset")] = GetOffset(fg)
	rt[pat.Post("/offset")] = SetOffset(fg)

	rt[pat.Get("/output")] = GetOutput(fg)
	rt[pat.Post("/output")] = SetOutput(fg)
}

// SetFunction exposes an HTTP interface to the SetFunction method
func SetFunction(fg FunctionGenerator) http.HandlerFunc {
	return setString(fg.SetFunction)
}

// GetFunction exposes an HTTP interface to the GetFunction method
func GetFunction(fg FunctionGenerator) http.HandlerFunc {
	return getString(fg.GetFunction)
}

// SetFrequency exposes an HTTP interface to the SetFrequency method
func SetFrequency(fg FunctionGenerator) http.HandlerFunc {
	return setFloat(fg.SetFrequency)
}

// GetFrequency exposes an HTTP interface to the GetFrequency method
func GetFrequency(fg FunctionGenerator) http.HandlerFunc {
	return getFloat(fg.GetFrequency)
}

// SetVoltage exposes an HTTP interface to the SetVoltage method
func SetVoltage(fg FunctionGenerator) http.HandlerFunc {
	return setFloat(fg.SetVoltage)
}

// GetVoltage exposes an HTTP interface to the GetVoltage method
func GetVoltage(fg FunctionGenerator) http.HandlerFunc {
	return getFloat(fg.GetVoltage)
}

// SetOffset exposes an HTTP interface to the SetOffset method
func SetOffset(fg FunctionGenerator) http.HandlerFunc {
	return setFloat(fg.SetOffset)
}

// GetOffset exposes an HTTP interface to the GetOffset method
func GetOffset(fg FunctionGenerator) http.HandlerFunc {
	return getFloat(fg.GetOffset)
}

// SetOutput exposes an HTTP interface to the Output control methods
func SetOutput(fg FunctionGenerator) http.HandlerFunc {
	return setBool(fg.EnableOutput, fg.DisableOutput)
}

// GetOutput exposes an HTTP interface to the GetOutput
func GetOutput(fg FunctionGenerator) http.HandlerFunc {
	return getBool(fg.GetOutput)
}
