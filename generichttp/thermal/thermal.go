// Package thermal exposes an HTTP interface to thermal controllers
package thermal

import (
	"encoding/json"
	"go/types"
	"net/http"

	"github.com/nasa-jpl/golaborate/generichttp"
)

// Controller is an interface to a thermal controller with a single channel
type Controller interface {
	// GetTemperatureSetpoint gets the temperature setpoint in Celcius
	GetTemperatureSetpoint() (float64, error)

	// SetTemperatureSetpoint sets the temperature setpoint in Celcius
	SetTemperatureSetpoint(float64) error

	// GetTemperature gets the temperature in Celcius
	GetTemperature() (float64, error)
}

// GetTemperatureSetpoint returns the temperature as JSON over HTTP
func GetTemperatureSetpoint(c Controller) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		setpt, err := c.GetTemperatureSetpoint()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		hp := generichttp.HumanPayload{T: types.Float64, Float: setpt}
		hp.EncodeAndRespond(w, r)
		return
	}
}

// SetTemperatureSetpoint returns an HTTP handler func that sets the temperature setpoint over HTTP
func SetTemperatureSetpoint(c Controller) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		f := generichttp.FloatT{}
		err := json.NewDecoder(r.Body).Decode(&f)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer r.Body.Close()
		err = c.SetTemperatureSetpoint(f.F64)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		return
	}
}

// GetTemperature returns an HTTP handler func that returns the temperature over HTTP
func GetTemperature(c Controller) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		t, err := c.GetTemperature()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		hp := generichttp.HumanPayload{T: types.Float64, Float: t}
		hp.EncodeAndRespond(w, r)
		return
	}
}

// HTTPController binds routes to control temperature to the table
func HTTPController(c Controller, table generichttp.RouteTable) {
	table[generichttp.MethodPath{Method: http.MethodGet, Path: "/temperature"}] = GetTemperature(c)
	table[generichttp.MethodPath{Method: http.MethodGet, Path: "/temperature-setpoint"}] = GetTemperatureSetpoint(c)
	table[generichttp.MethodPath{Method: http.MethodPost, Path: "/temperature-setpoint"}] = SetTemperatureSetpoint(c)
}
