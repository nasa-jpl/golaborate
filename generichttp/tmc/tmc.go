// Package tmc provides an HTTP interface to test and measurement devices
package tmc

import (
	"encoding/json"
	"fmt"
	"go/types"
	"net/http"

	"github.jpl.nasa.gov/bdube/golab/generichttp/ascii"

	"github.jpl.nasa.gov/bdube/golab/oscilloscope"
	"github.jpl.nasa.gov/bdube/golab/server"
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

func getInt(fcn func() (int, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		i, err := fcn()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		hp := server.HumanPayload{T: types.Int, Int: i}
		hp.EncodeAndRespond(w, r)
		return
	}
}

func setInt(fcn func(int) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		f := server.IntT{}
		err := json.NewDecoder(r.Body).Decode(&f)
		defer r.Body.Close()
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		err = fcn(f.Int)
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

	// SetOutputLoad sets the output load of the generator in ohms
	SetOutputLoad(float64) error
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

	rt[pat.Post("/output-load")] = SetOutputLoad(fg)

	if rawer, ok := interface{}(fg).(ascii.RawCommunicator); ok {
		RW := ascii.RawWrapper{Comm: rawer}
		rt[pat.Post("/raw")] = RW.HTTPRaw
	}
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

// GetOutput exposes an HTTP interface to the GetOutput method
func GetOutput(fg FunctionGenerator) http.HandlerFunc {
	return getBool(fg.GetOutput)
}

// SetOutputLoad exposes an HTTP interface to the SetOutputLoad method
func SetOutputLoad(fg FunctionGenerator) http.HandlerFunc {
	return setFloat(fg.SetOutputLoad)
}

// HTTPFunctionGeneratorT holds an HTTP wrapper to a function generator
type HTTPFunctionGeneratorT struct {
	FG FunctionGenerator

	RouteTable server.RouteTable
}

// RT makes this server.httper compliant
func (h HTTPFunctionGeneratorT) RT() server.RouteTable {
	return h.RouteTable
}

// NewHTTPFunctionGenerator wraps a function generator in an HTTP interface
func NewHTTPFunctionGenerator(fg FunctionGenerator) HTTPFunctionGeneratorT {
	rt := server.RouteTable{}
	gen := HTTPFunctionGeneratorT{FG: fg, RouteTable: rt}
	HTTPFunctionGenerator(fg, rt)
	return gen
}

// SampleRateManipulator can manipulate their sampling rate
type SampleRateManipulator interface {
	// SetSampleRate configures the analog sampling rate of the scope
	SetSampleRate(int) error

	// GetSampleRate returns the analog sampling rate of the scope
	GetSampleRate() (int, error)
}

// Oscilloscope describes an interface to a digital oscilloscope
type Oscilloscope interface {
	SampleRateManipulator
	// SetScale configures the full vertical range of a channel
	SetScale(string, float64) error

	// GetScale returns the full vertical range of a channel
	GetScale(string) (float64, error)

	// SetTimebase configures the full vertical range of a channel
	SetTimebase(float64) error

	// GetTimebase returns the full vertical range of a channel
	GetTimebase() (float64, error)

	// SetBandwidthLimit turns the bandwidth limit for a channel on or off
	SetBandwidthLimit(string, bool) error

	// SetBitDepth configures the bit depth of a scope
	SetBitDepth(int) error

	// GetBitDepth retrieves the bit depth of the scope
	GetBitDepth() (int, error)

	// SetAcqLength configures the number of data points to capture
	SetAcqLength(int) error

	// GetAcqLength retrieves the number of data points to be captured
	GetAcqLength() (int, error)

	// SetAcqMode configures the acquisition mode used by the scope
	SetAcqMode(string) error

	// GetAcqMode returns the acquisition mode used by the scope
	GetAcqMode() (string, error)

	// StartAcq begins DAQ
	StartAcq() error

	// AcquireWaveform triggers a measurement on the scope and returns the data
	AcquireWaveform([]string) (oscilloscope.Waveform, error)
}

// SetTimebase exposes an HTTP interface to SetTimebase
func SetTimebase(o Oscilloscope) http.HandlerFunc {
	return setFloat(o.SetTimebase)
}

// GetTimebase exposes an HTTP interface to GetTimebase
func GetTimebase(o Oscilloscope) http.HandlerFunc {
	return getFloat(o.GetTimebase)
}

// SetBitDepth exposes an HTTP interface to SetBitDepth
func SetBitDepth(o Oscilloscope) http.HandlerFunc {
	return setInt(o.SetBitDepth)
}

// GetBitDepth exposes an HTTP interface to GetBitDepth
func GetBitDepth(o Oscilloscope) http.HandlerFunc {
	return getInt(o.GetBitDepth)
}

// SetSampleRate exposes an HTTP interface to SetSampleRate
func SetSampleRate(o Oscilloscope) http.HandlerFunc {
	return setInt(o.SetSampleRate)
}

// GetSampleRate exposes an HTTP interface to GetSampleRate
func GetSampleRate(o Oscilloscope) http.HandlerFunc {
	return getInt(o.GetSampleRate)
}

// SetAcqLength exposes an HTTP interface to SetAcqLength
func SetAcqLength(o Oscilloscope) http.HandlerFunc {
	return setInt(o.SetAcqLength)
}

// GetAcqLength exposes an HTTP interface to GetAcqLength
func GetAcqLength(o Oscilloscope) http.HandlerFunc {
	return getInt(o.GetAcqLength)
}

// SetAcqMode exposes an HTTP interface to SetAcqMode
func SetAcqMode(o Oscilloscope) http.HandlerFunc {
	return setString(o.SetAcqMode)
}

// GetAcqMode exposes an HTTP interface to GetAcqMode
func GetAcqMode(o Oscilloscope) http.HandlerFunc {
	return getString(o.GetAcqMode)
}

// now the few weird ones
type scalechan struct {
	Scale float64 `json:"scale"`

	Channel string `json:"channel"`
}

// GetScale returns the scale of a channel
func GetScale(o Oscilloscope) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sc := scalechan{}
		err := json.NewDecoder(r.Body).Decode(&sc)
		fmt.Println(sc)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		scale, err := o.GetScale(sc.Channel)
		fmt.Println(scale)
		hp := server.HumanPayload{T: types.Float64, Float: scale}
		hp.EncodeAndRespond(w, r)
	}
}

// SetScale sets the scale of a channel
func SetScale(o Oscilloscope) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sc := scalechan{}
		err := json.NewDecoder(r.Body).Decode(&sc)
		defer r.Body.Close()
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		err = o.SetScale(sc.Channel, sc.Scale)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}

// StartAcq triggers DAQ on the scope
func StartAcq(o Oscilloscope) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := o.StartAcq()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}

type channels struct {
	Chans []string `json:"channels"`
}

// AcquireWaveform transfers the data from the oscilloscope to the user
func AcquireWaveform(o Oscilloscope) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		chans := channels{}
		err := json.NewDecoder(r.Body).Decode(&chans)
		defer r.Body.Close()
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		data, err := o.AcquireWaveform(chans.Chans)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/csv")
		w.WriteHeader(http.StatusOK)
		err = data.EncodeCSV(w)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

// HTTPOscilloscope holds an HTTP wrapper to a function generator
type HTTPOscilloscope struct {
	O Oscilloscope

	RouteTable server.RouteTable
}

// RT makes this server.httper compliant
func (h HTTPOscilloscope) RT() server.RouteTable {
	return h.RouteTable
}

// NewHTTPOscilloscope wraps a function generator in an HTTP interface
func NewHTTPOscilloscope(o Oscilloscope) HTTPOscilloscope {
	rt := server.RouteTable{}
	rt[pat.Get("/scale")] = GetScale(o)
	rt[pat.Post("/scale")] = SetScale(o)

	rt[pat.Get("/timebase")] = GetTimebase(o)
	rt[pat.Post("/timebase")] = SetTimebase(o)

	rt[pat.Get("/bit-depth")] = GetBitDepth(o)
	rt[pat.Post("/bit-depth")] = SetBitDepth(o)

	rt[pat.Get("/sample-rate")] = GetSampleRate(o)
	rt[pat.Post("/sample-rate")] = SetSampleRate(o)

	rt[pat.Get("/acq-length")] = GetAcqLength(o)
	rt[pat.Post("/acq-length")] = SetAcqLength(o)

	rt[pat.Get("/acq-mode")] = GetAcqMode(o)
	rt[pat.Post("/acq-mode")] = SetAcqMode(o)

	rt[pat.Post("/acq-start")] = StartAcq(o)
	rt[pat.Get("/acq-waveform")] = AcquireWaveform(o)

	if rawer, ok := interface{}(o).(ascii.RawCommunicator); ok {
		RW := ascii.RawWrapper{Comm: rawer}
		rt[pat.Post("/raw")] = RW.HTTPRaw
	}
	scope := HTTPOscilloscope{O: o, RouteTable: rt}
	return scope
}

// DAQ is the interface of a data acquisition device
type DAQ interface {
	SampleRateManipulator

	// SetChannelLabel sets the label used for a given channel
	SetChannelLabel(int, string) error
	// GetChannelLabel retrieves the label used by a given channel
	GetChannelLabel(int) (string, error)

	// SetRecordingLength sets the number of samples used
	// in a recording
	SetRecordingLength(int) error
	// GetRecordingLength retrieves the number of samples used
	// in a recording
	GetRecordingLength() (int, error)

	// SetRecordingChannel sets the channel used
	// to record data
	SetRecordingChannel(int) error
	// GetRecordingChannel retrieves the channel
	// used to record data
	GetRecordingChannel() (int, error)

	// Record captures data
	Record() (oscilloscope.Recording, error)
}

type labelChan struct {
	Chan int `json:"chan"`

	Label string `json:"label"`
}

// SetChannelLabel sets the channel label on a DAQ
func SetChannelLabel(d DAQ) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		lc := labelChan{}
		err := json.NewDecoder(r.Body).Decode(&lc)
		defer r.Body.Close()
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		err = d.SetChannelLabel(lc.Chan, lc.Label)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}

// GetChannelLabel retrieves the label associated with a channel over HTTP
func GetChannelLabel(d DAQ) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		i := server.IntT{}
		err := json.NewDecoder(r.Body).Decode(&i)
		defer r.Body.Close()
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		str, err := d.GetChannelLabel(i.Int)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		hp := server.HumanPayload{T: types.String, String: str}
		hp.EncodeAndRespond(w, r)
	}
}

// SetRecordingLength sets the length of a recording in samples
func SetRecordingLength(d DAQ) http.HandlerFunc {
	return setInt(d.SetRecordingLength)
}

// GetRecordingLength returns the length of a recording in samples
func GetRecordingLength(d DAQ) http.HandlerFunc {
	return getInt(d.GetRecordingLength)
}

// SetRecordingChannel sets the channel of a recording in samples
func SetRecordingChannel(d DAQ) http.HandlerFunc {
	return setInt(d.SetRecordingChannel)
}

// GetRecordingChannel returns the channel of a recording in samples
func GetRecordingChannel(d DAQ) http.HandlerFunc {
	return getInt(d.GetRecordingChannel)
}

// Record causes the DAQ to record and sends the result back as a CSV file
func Record(d DAQ) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		recording, err := d.Record()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/csv")
		w.WriteHeader(http.StatusOK)
		recording.EncodeCSV(w)
	}
}

// HTTPDAQ is an HTTP adapter to a DAQ
type HTTPDAQ struct {
	D DAQ

	RouteTable server.RouteTable
}

// RT satisfies server.HTTPer
func (h HTTPDAQ) RT() server.RouteTable {
	return h.RouteTable
}

// NewHTTPDAQ returns a newly HTTP wrapped DAQ
func NewHTTPDAQ(d DAQ) HTTPDAQ {
	rt := server.RouteTable{}
	rt[pat.Get("/channel-label")] = GetChannelLabel(d)
	rt[pat.Post("/channel-label")] = SetChannelLabel(d)

	rt[pat.Get("/recording-length")] = GetRecordingLength(d)
	rt[pat.Post("/recording-length")] = SetRecordingLength(d)

	rt[pat.Get("/recording-channel")] = GetRecordingChannel(d)
	rt[pat.Post("/recording-channel")] = SetRecordingChannel(d)

	rt[pat.Get("/record")] = Record(d)

	if rawer, ok := interface{}(d).(ascii.RawCommunicator); ok {
		RW := ascii.RawWrapper{Comm: rawer}
		rt[pat.Post("/raw")] = RW.HTTPRaw
	}

	return HTTPDAQ{D: d, RouteTable: rt}
}
