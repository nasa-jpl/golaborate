// Package daq provides a generic HTTP interface to ADC and DAC devices
//
// This is not the last word in speed, due to HTTP having reasonable latency in
// most client languages, but it is the last word in ease of use.
package daq

import (
	"encoding/json"
	"go/types"
	"net/http"

	"github.jpl.nasa.gov/bdube/golab/generichttp"
)

// DAC is a model for simple digital to analog converter
type DAC interface {
	// Output sends a voltage on a given channel
	Output(int, float64) error

	// OutputDN sends a data number on a given channel
	OutputDN16(int, uint16) error
}

type channelVoltage struct {
	Channel int `json:"channel"`

	Voltage float64 `json:"voltage"`
}

type channelDN struct {
	Channel int `json:"channel"`

	DN uint16 `json:"dn"`
}

// Output returns an HTTP handlerfunc that will write a voltage to a channel
func Output(d DAC) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input channelVoltage
		err := json.NewDecoder(r.Body).Decode(&input)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		err = d.Output(input.Channel, input.Voltage)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}

// OutputDN16 returns an HTTP handlerfunc that will write a data number to a channel
func OutputDN16(d DAC) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input channelDN
		err := json.NewDecoder(r.Body).Decode(&input)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		err = d.OutputDN16(input.Channel, input.DN)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}

// MultiChannelDAC allows multiple channels to be written
// at once
type MultiChannelDAC interface {
	DAC

	// OutputMulti writes a sequence of voltages to a sequence of channels
	OutputMulti([]int, []float64) error

	// OutputMultiDN16 outputs a sequence of data numbers to a sequence of channels
	OutputMultiDN16([]int, []uint16) error
}

type channelsVoltages struct {
	Channels []int `json:"channel"`

	Voltages []float64 `json:"voltage"`
}

type channelsDNs struct {
	Channels []int `json:"channel"`

	DNs []uint16 `json:"dn"`
}

// OutputMulti returns an HTTP handlerfunc that will write a voltage to a channel
func OutputMulti(d MultiChannelDAC) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input channelsVoltages
		err := json.NewDecoder(r.Body).Decode(&input)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		err = d.OutputMulti(input.Channels, input.Voltages)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}

// OutputMultiDN16 returns an HTTP handlerfunc that will write a data number to a channel
func OutputMultiDN16(d MultiChannelDAC) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input channelsDNs
		err := json.NewDecoder(r.Body).Decode(&input)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		err = d.OutputMultiDN16(input.Channels, input.DNs)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}

// ExtendedDAC is a larger part of the interface to the AP236 from Acromag
type ExtendedDAC interface {
	MultiChannelDAC

	// SetRange sets the output range of a DAC channel
	SetRange(int, string) error

	// GetRange returns the output range of a DAC channel
	GetRange(int) (string, error)

	// SetOutputSimultaneous configures a channel for simultaneous output
	SetOutputSimultaneous(int, bool) error

	// GetOutputSimultaneous returns true if a channel is configured for simultaneous output
	GetOutputSimultaneous(int) (bool, error)
}

type channelRange struct {
	Channel int `json:"channel"`

	Range string `json:"range"`
}

type channelSimultaneous struct {
	Channel int `json:"channel"`

	Simultaneous bool `json:"simultaneous"`
}

// SetRange configures the output range of one channel of a DAC
func SetRange(d ExtendedDAC) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input channelRange
		err := json.NewDecoder(r.Body).Decode(&input)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		err = d.SetRange(input.Channel, input.Range)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}

// GetRange configures the output range of one channel of a DAC
func GetRange(d ExtendedDAC) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input channelRange
		err := json.NewDecoder(r.Body).Decode(&input)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		err = d.SetRange(input.Channel, input.Range)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}

// SetOutputSimultaneous configures the output channel of a DAC to be simultaneous
func SetOutputSimultaneous(d ExtendedDAC) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input channelSimultaneous
		err := json.NewDecoder(r.Body).Decode(&input)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		err = d.SetOutputSimultaneous(input.Channel, input.Simultaneous)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}

// GetOutputSimultaneous retrieves if the output of a channel of a DAC is simultaneous
func GetOutputSimultaneous(d ExtendedDAC) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input channelSimultaneous
		err := json.NewDecoder(r.Body).Decode(&input)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		boolean, err := d.GetOutputSimultaneous(input.Channel)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		hp := generichttp.HumanPayload{T: types.Bool, Bool: boolean}
		hp.EncodeAndRespond(w, r)
	}
}

// WaveformDAC is a DAC which allows waveform playback
type WaveformDAC interface {
	ExtendedDAC

	SetOperatingMode(int, string) error

	GetOperatingMode(int) (string, error)

	SetTriggerMode(int, string) error

	GetTriggerMode(int) (string, error)

	PopulateWaveform(int, []float64) error

	StartWaveform() error

	StopWaveform() error
}

// Timer describes a clock
type Timer interface {
	SetTimerPeriod(uint32) error

	GetTimerPeriod() (uint32, error)
}

// SetTimerPeriod invokes the function of the same name on a timer
func SetTimerPeriod(t Timer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u := generichttp.Uint32T{}
		err := json.NewDecoder(r.Body).Decode(&u)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		err = t.SetTimerPeriod(u.Uint)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}

// GetTimerPeriod invokes the function of the same name on a timer
func GetTimerPeriod(t Timer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ns, err := t.GetTimerPeriod()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		s := struct {
			Uint uint32 `json:"uint"`
		}{ns}
		w.Header().Set("Content-Type", "application/json")

		err = json.NewEncoder(w).Encode(s)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}

// TriggerExport describes a piece of hardware which imports or exports a trigger
type TriggerExport interface {
	SetTriggerDirection(bool) error

	GetTriggerDirection() (bool, error)
}

// SetTriggerDirection causes the device to export a trigger if True, else import
func SetTriggerDirection(t TriggerExport) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		b := generichttp.BoolT{}
		err := json.NewDecoder(r.Body).Decode(&b)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		err = t.SetTriggerDirection(b.Bool)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}

// GetTriggerDirection invokes the function of the same name on a Trigger
func GetTriggerDirection(t TriggerExport) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		export, err := t.GetTriggerDirection()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		hp := generichttp.HumanPayload{T: types.Bool, Bool: export}
		hp.EncodeAndRespond(w, r)
	}
}
