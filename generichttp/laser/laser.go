// Package laser exposes control of laser controllers over HTTP
package laser

import (
	"encoding/json"
	"math"
	"net/http"

	"github.jpl.nasa.gov/bdube/golab/generichttp"
)

// CenterBandwidth is a struct holding the center wavelength (nm) and full bandwidth (nm) of a VARIA
type CenterBandwidth struct {
	Center    float64 `json:"center"`
	Bandwidth float64 `json:"bandwidth"`
}

// ShortLongToCB converts short, long wavelengths to a CenterBandwidth struct
func ShortLongToCB(short, long float64) CenterBandwidth {
	center := (short + long) / 2
	bw := math.Round(math.Abs(long-short)*10) / 10 // *10/10 to round to nearest tenth
	return CenterBandwidth{Center: center, Bandwidth: bw}
}

// ToShortLong converts a CenterBandwidth to (short, long)
func (cb CenterBandwidth) ToShortLong() (float64, float64) {
	hb := cb.Bandwidth / 2
	low := cb.Center - hb
	high := cb.Center + hb
	return high, low
}

// Controller is a basic interface for laser controllers
type Controller interface {
	// SetEmission turns emission on or off
	SetEmission(bool) error

	// GetEmission queries if the laser is currently outputting
	GetEmission() (bool, error)
}

// SetEmission configures the output state of the laser
func SetEmission(c Controller) http.HandlerFunc {
	return generichttp.SetBool(c.SetEmission)
}

// GetEmission queries the output state of the laser
func GetEmission(c Controller) http.HandlerFunc {
	return generichttp.GetBool(c.GetEmission)
}

// CurrentController can control its output current
type CurrentController interface {
	// SetCurrent sets the output current setpoint of the controller
	SetCurrent(float64) error

	// GetCurrent retrieves the output current setpoint of the controller
	GetCurrent() (float64, error)
}

// SetCurrent configures the output current of the laser
func SetCurrent(c CurrentController) http.HandlerFunc {
	return generichttp.SetFloat(c.SetCurrent)
}

// GetCurrent queries the output current of the laser
func GetCurrent(c CurrentController) http.HandlerFunc {
	return generichttp.GetFloat(c.GetCurrent)
}

// PowerController can control its output power
type PowerController interface {
	// SetPower sets the output power level of the the device
	SetPower(float64) error

	// GetPower retrieves the output power level of the device
	GetPower() (float64, error)
}

// SetPower configures the output power of the laser
func SetPower(c PowerController) http.HandlerFunc {
	return generichttp.SetFloat(c.SetPower)
}

// GetPower queries the output power of the laser
func GetPower(c PowerController) http.HandlerFunc {
	return generichttp.GetFloat(c.GetPower)
}

// NDController can control the strength of an ND filter
type NDController interface {
	// GetND retrieves the strength of the ND
	GetND() (float64, error)

	// SetND sets the strength of the ND
	SetND(float64) error
}

// SetND configures the output ND of the laser
func SetND(c NDController) http.HandlerFunc {
	return generichttp.SetFloat(c.SetND)
}

// GetND queries the output ND of the laser
func GetND(c NDController) http.HandlerFunc {
	return generichttp.GetFloat(c.GetND)
}

// BandwidthController can control the its output bandwidth
type BandwidthController interface {
	// GetShortWave gets the short wavelength of the controller
	// if the output band is set to 500-600 nm, this returns 500.
	GetShortWave() (float64, error)

	// SetShortWave sets the short wavelength cutoff of the controller
	SetShortWave(float64) error

	// GetShortWave gets the short wavelength of the controller
	// if the output band is set to 500-600 nm, this returns 600.
	GetLongWave() (float64, error)

	// SetShortWave sets the long wavelength cutoff of the controller
	SetLongWave(float64) error

	// GetCenterBandwidth returns the center wavelength and (full) bandwidth
	// of a controller.  To set the output to 500-600nm, Center=550, Bandwidth=100.
	GetCenterBandwidth() (CenterBandwidth, error)

	// SetCenterBandwidth sets the center wavelength and (full) bandwidth
	// of a controller. If output is 500-600nm, Center=550, Bandwidth=100.
	SetCenterBandwidth(CenterBandwidth) error
}

// SetShortWave configures the lower cutoff wavelength of the controller
func SetShortWave(c BandwidthController) http.HandlerFunc {
	return generichttp.SetFloat(c.SetShortWave)
}

// GetShortWave retrieves the lower cutoff wavelength of the controller
func GetShortWave(c BandwidthController) http.HandlerFunc {
	return generichttp.GetFloat(c.GetShortWave)
}

// SetLongWave configures the upper cutoff wavelength of the controller
func SetLongWave(c BandwidthController) http.HandlerFunc {
	return generichttp.SetFloat(c.SetLongWave)
}

// GetLongWave retrieves the lower cutoff wavelength of the controller
func GetLongWave(c BandwidthController) http.HandlerFunc {
	return generichttp.GetFloat(c.GetLongWave)
}

// GetCenterBandwidth retrieves the center/bandwidth as JSON
func GetCenterBandwidth(c BandwidthController) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cbw, err := c.GetCenterBandwidth()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		err = json.NewEncoder(w).Encode(cbw)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
}

// SetCenterBandwidth configures the center/bandwidth as JSON
func SetCenterBandwidth(c BandwidthController) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cbw := CenterBandwidth{}
		err := json.NewDecoder(r.Body).Decode(&cbw)
		defer r.Body.Close()
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		err = c.SetCenterBandwidth(cbw)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

// HTTPLaserController wraps a LaserController in an HTTP route table
type HTTPLaserController struct {
	// Ctl is the underlying laser controller
	Ctl Controller

	// RouteTable maps URLs to functions
	RouteTable generichttp.RouteTable
}

// NewHTTPLaserController returns a new HTTP wrapper around an existing laser controller
func NewHTTPLaserController(ctl Controller) HTTPLaserController {
	h := HTTPLaserController{Ctl: ctl}
	rt := generichttp.RouteTable{
		generichttp.MethodPath{Method: http.MethodGet, Path: "/emission"}:  GetEmission(ctl),
		generichttp.MethodPath{Method: http.MethodPost, Path: "/emission"}: SetEmission(ctl),
	}
	if currentctl, ok := interface{}(ctl).(CurrentController); ok {
		rt[generichttp.MethodPath{Method: http.MethodGet, Path: "/current"}] = GetCurrent(currentctl)
		rt[generichttp.MethodPath{Method: http.MethodPost, Path: "/current"}] = SetCurrent(currentctl)
	}
	if powerctl, ok := interface{}(ctl).(PowerController); ok {
		rt[generichttp.MethodPath{Method: http.MethodGet, Path: "/power"}] = GetPower(powerctl)
		rt[generichttp.MethodPath{Method: http.MethodPost, Path: "/power"}] = SetPower(powerctl)
	}
	if ndctl, ok := interface{}(ctl).(NDController); ok {
		rt[generichttp.MethodPath{Method: http.MethodGet, Path: "/nd"}] = GetND(ndctl)
		rt[generichttp.MethodPath{Method: http.MethodPost, Path: "/nd"}] = SetND(ndctl)
	}
	if bwctl, ok := interface{}(ctl).(BandwidthController); ok {
		rt[generichttp.MethodPath{Method: http.MethodGet, Path: "/wvl/short"}] = GetShortWave(bwctl)
		rt[generichttp.MethodPath{Method: http.MethodPost, Path: "/wvl/short"}] = SetShortWave(bwctl)

		rt[generichttp.MethodPath{Method: http.MethodGet, Path: "/wvl/long"}] = GetLongWave(bwctl)
		rt[generichttp.MethodPath{Method: http.MethodPost, Path: "/wvl/long"}] = SetLongWave(bwctl)

		rt[generichttp.MethodPath{Method: http.MethodGet, Path: "/wvl/center-bandwidth"}] = GetCenterBandwidth(bwctl)
		rt[generichttp.MethodPath{Method: http.MethodPost, Path: "/wvl/center-bandwidth"}] = SetCenterBandwidth(bwctl)
	}
	h.RouteTable = rt
	return h
}

// RT safisfies the generichttp.HTTPer interface
func (h HTTPLaserController) RT() generichttp.RouteTable {
	return h.RouteTable
}
