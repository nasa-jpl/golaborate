// Package daq provides a generic HTTP interface to ADC and DAC devices
//
// This is not the last word in speed, due to HTTP having reasonable latency in
// most client languages, but it is the last word in ease of use.
package daq

import (
	"encoding/csv"
	"encoding/json"
	"go/types"
	"io"
	"net/http"
	"strconv"

	"github.jpl.nasa.gov/bdube/golab/generichttp"
)

// DAC is a model for simple digital to analog converter
type DAC interface {
	// Output sends a voltage on a given channel
	Output(int, float64) error

	// OutputDN sends a data number on a given channel
	OutputDN16(int, uint16) error
}

// HTTPBasicDAC adds routes for basic DAC operation to a table
func HTTPBasicDAC(iface DAC, table generichttp.RouteTable2) {
	table[generichttp.MethodPath{Method: http.MethodPost, Path: "/output"}] = Output(iface)
	table[generichttp.MethodPath{Method: http.MethodPost, Path: "/output-dn-16"}] = OutputDN16(iface)
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

// HTTPMultiChannel adds routes for multi channel output to the table
func HTTPMultiChannel(iface MultiChannelDAC, table generichttp.RouteTable2) {
	table[generichttp.MethodPath{Method: http.MethodPost, Path: "/output-multi"}] = OutputMulti(iface)
	table[generichttp.MethodPath{Method: http.MethodPost, Path: "/output-multi-dn-16"}] = OutputMultiDN16(iface)
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

// HTTPExtended adds routes for multi channel output to the table
func HTTPExtended(iface ExtendedDAC, table generichttp.RouteTable2) {
	table[generichttp.MethodPath{Method: http.MethodPost, Path: "/range"}] = SetRange(iface)
	table[generichttp.MethodPath{Method: http.MethodGet, Path: "/range"}] = GetRange(iface)

	table[generichttp.MethodPath{Method: http.MethodPost, Path: "/simultaneous"}] = SetOutputSimultaneous(iface)
	table[generichttp.MethodPath{Method: http.MethodGet, Path: "/simultaneous"}] = GetOutputSimultaneous(iface)
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
		rng, err := d.GetRange(input.Channel)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		hp := generichttp.HumanPayload{T: types.String, String: rng}
		hp.EncodeAndRespond(w, r)
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

// HTTPWaveform adds routes for multi channel output to the table
func HTTPWaveform(iface WaveformDAC, table generichttp.RouteTable2) {
	table[generichttp.MethodPath{Method: http.MethodPost, Path: "/operating-mode"}] = SetOperatingMode(iface)
	table[generichttp.MethodPath{Method: http.MethodGet, Path: "/operating-mode"}] = GetOperatingMode(iface)

	table[generichttp.MethodPath{Method: http.MethodPost, Path: "/trigger-mode"}] = SetTriggerMode(iface)
	table[generichttp.MethodPath{Method: http.MethodGet, Path: "/trigger-mode"}] = GetTriggerMode(iface)

	table[generichttp.MethodPath{Method: http.MethodPost, Path: "/playback/upload/float/csv"}] = UploadWaveformFloatCSV(iface)
	// line for upload DN

	table[generichttp.MethodPath{Method: http.MethodPost, Path: "/playback/start"}] = StartWaveform(iface)
	table[generichttp.MethodPath{Method: http.MethodPost, Path: "/playback/stop"}] = StopWaveform(iface)
}

type channelOpMode struct {
	Channel int `json:"channel"`

	OperatingMode string `json:"operatingMode"`
}

type channelTriggerMode struct {
	Channel int `json:"channel"`

	TriggerMode string `json:"triggerMode"`
}

// SetOperatingMode configures the operating mode of a DAC between "single"
// and "waveform" modes
func SetOperatingMode(d WaveformDAC) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input channelOpMode
		err := json.NewDecoder(r.Body).Decode(&input)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		err = d.SetOperatingMode(input.Channel, input.OperatingMode)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}

// GetOperatingMode retrieves whether the dac is in "single" or "waveform" mode
func GetOperatingMode(d WaveformDAC) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input channelOpMode
		err := json.NewDecoder(r.Body).Decode(&input)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		mode, err := d.GetOperatingMode(input.Channel)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		hp := generichttp.HumanPayload{T: types.String, String: mode}
		hp.EncodeAndRespond(w, r)
	}
}

// SetTriggerMode configures the operating mode of a DAC between "single"
// and "waveform" modes
func SetTriggerMode(d WaveformDAC) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input channelTriggerMode
		err := json.NewDecoder(r.Body).Decode(&input)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		err = d.SetTriggerMode(input.Channel, input.TriggerMode)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}

// GetTriggerMode retrieves whether the dac is in "single" or "waveform" mode
func GetTriggerMode(d WaveformDAC) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input channelTriggerMode
		err := json.NewDecoder(r.Body).Decode(&input)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		mode, err := d.GetTriggerMode(input.Channel)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		hp := generichttp.HumanPayload{T: types.String, String: mode}
		hp.EncodeAndRespond(w, r)
	}
}

// UploadWaveformFloatCSV is an HTTP interface to multiple
// PopulateWaveform calls from one CSV file
func UploadWaveformFloatCSV(d WaveformDAC) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data, err := csvToWaveformFloat(r.Body)
		defer r.Body.Close()
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		for i := 0; i < len(data); i++ {
			err = d.PopulateWaveform(data[i].channel, data[i].waveform)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
	}
}

// StartWaveform commences waveform playback
func StartWaveform(d WaveformDAC) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := d.StartWaveform()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}

// StopWaveform ceases waveform playback
func StopWaveform(d WaveformDAC) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := d.StartWaveform()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)

	}
}

// Timer describes a clock
type Timer interface {
	SetTimerPeriod(uint32) error

	GetTimerPeriod() (uint32, error)
}

// HTTPTimer adds routes for basic Timer operation to a table
func HTTPTimer(iface Timer, table generichttp.RouteTable2) {
	table[generichttp.MethodPath{Method: http.MethodPost, Path: "/timer-period"}] = SetTimerPeriod(iface)
	table[generichttp.MethodPath{Method: http.MethodGet, Path: "/timer-period"}] = GetTimerPeriod(iface)
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

type channelWaveformVolt struct {
	channel int

	waveform []float64
}

type channelWaveformDN struct {
	channel int

	waveform []uint16
}

func csvToWaveformFloat(r io.Reader) ([]channelWaveformVolt, error) {
	var out []channelWaveformVolt
	reader := csv.NewReader(r)
	skip := true
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return out, err
		}
		if skip {
			skip = false
			// allocate; one column per channel.  Leak to outer scope
			out = make([]channelWaveformVolt, len(record))
			for i := 0; i < len(record); i++ {
				c, err := strconv.Atoi(record[i])
				if err != nil {
					return out, err
				}
				out[i].channel = c
			}
			continue
		}
		for i := 0; i < len(record); i++ {
			f, err := strconv.ParseFloat(record[i], 64)
			if err != nil {
				return out, err
			}
			out[i].waveform = append(out[i].waveform, f)
		}
	}
	return out, nil
}

func csvToWaveformDN(r io.Reader) ([]channelWaveformDN, error) {
	// good old copy paste from f64 version, with one line (strconv.ParseFloat) changed
	var out []channelWaveformDN
	reader := csv.NewReader(r)
	skip := true
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return out, err
		}
		if skip {
			skip = false
			// allocate; one column per channel.  Leak to outer scope
			out = make([]channelWaveformDN, len(record))
			for i := 0; i < len(record); i++ {
				c, err := strconv.Atoi(record[i])
				if err != nil {
					return out, err
				}
				out[i].channel = c
			}
			continue
		}
		for i := 0; i < len(record); i++ {
			u, err := strconv.ParseUint(record[i], 10, 64)
			if err != nil {
				return out, err
			}
			out[i].waveform = append(out[i].waveform, uint16(u))
		}
	}
	return out, nil
}

// HTTPDAC is a type that allows setting up a DAC satisfying any combination
// of the interfaces in this package to an HTTP interface
type HTTPDAC struct {
	d DAC

	RouteTable generichttp.RouteTable2
}

// NewHTTPDAC sets up an HTTP interface to a DAC
func NewHTTPDAC(d DAC) HTTPDAC {
	w := HTTPDAC{d: d}
	rt := generichttp.RouteTable2{}
	HTTPBasicDAC(d, rt)
	if md, ok := (d).(MultiChannelDAC); ok {
		HTTPMultiChannel(md, rt)
	}
	if ed, ok := (d).(ExtendedDAC); ok {
		HTTPExtended(ed, rt)
	}
	if wd, ok := (d).(WaveformDAC); ok {
		HTTPWaveform(wd, rt)
	}
	if t, ok := (d).(Timer); ok {
		HTTPTimer(t, rt)
	}
	w.RouteTable = rt
	return w
}
