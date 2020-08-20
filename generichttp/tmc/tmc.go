// Package tmc provides an HTTP interface to test and measurement devices
package tmc

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/types"
	"net/http"
	"reflect"
	"unsafe"

	"github.jpl.nasa.gov/bdube/golab/generichttp"
	"github.jpl.nasa.gov/bdube/golab/generichttp/ascii"

	"github.jpl.nasa.gov/bdube/golab/oscilloscope"
)

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

	// SetPutput configures if the generator output is active
	SetOutput(bool) error

	// GetOutput queries if the generator output is active
	GetOutput() (bool, error)

	// SetOutputLoad sets the output load of the generator in ohms
	SetOutputLoad(float64) error

	// SetWaveform uplodas an arbitrary waveform to the function generator
	SetWaveform([]uint16) error
}

// HTTPFunctionGenerator injects an HTTP interface to a function generator into a route table
func HTTPFunctionGenerator(fg FunctionGenerator, table generichttp.RouteTable) {
	rt := table
	rt[generichttp.MethodPath{Method: http.MethodGet, Path: "/function"}] = GetFunction(fg)
	rt[generichttp.MethodPath{Method: http.MethodPost, Path: "/function"}] = SetFunction(fg)
	rt[generichttp.MethodPath{Method: http.MethodGet, Path: "/frequency"}] = GetFrequency(fg)
	rt[generichttp.MethodPath{Method: http.MethodPost, Path: "/frequency"}] = SetFrequency(fg)

	rt[generichttp.MethodPath{Method: http.MethodGet, Path: "/voltage"}] = GetVoltage(fg)
	rt[generichttp.MethodPath{Method: http.MethodPost, Path: "/voltage"}] = SetVoltage(fg)

	rt[generichttp.MethodPath{Method: http.MethodGet, Path: "/offset"}] = GetOffset(fg)
	rt[generichttp.MethodPath{Method: http.MethodPost, Path: "/offset"}] = SetOffset(fg)

	rt[generichttp.MethodPath{Method: http.MethodGet, Path: "/output"}] = GetOutput(fg)
	rt[generichttp.MethodPath{Method: http.MethodPost, Path: "/output"}] = SetOutput(fg)

	rt[generichttp.MethodPath{Method: http.MethodPost, Path: "/output-load"}] = SetOutputLoad(fg)

	rt[generichttp.MethodPath{Method: http.MethodPost, Path: "/waveform"}] = SetWaveform(fg)

	if rawer, ok := interface{}(fg).(ascii.RawCommunicator); ok {
		RW := ascii.RawWrapper{Comm: rawer}
		rt[generichttp.MethodPath{Method: http.MethodPost, Path: "/raw"}] = RW.HTTPRaw
	}
}

// SetFunction exposes an HTTP interface to the SetFunction method
func SetFunction(fg FunctionGenerator) http.HandlerFunc {
	return generichttp.SetString(fg.SetFunction)
}

// GetFunction exposes an HTTP interface to the GetFunction method
func GetFunction(fg FunctionGenerator) http.HandlerFunc {
	return generichttp.GetString(fg.GetFunction)
}

// SetFrequency exposes an HTTP interface to the SetFrequency method
func SetFrequency(fg FunctionGenerator) http.HandlerFunc {
	return generichttp.SetFloat(fg.SetFrequency)
}

// GetFrequency exposes an HTTP interface to the GetFrequency method
func GetFrequency(fg FunctionGenerator) http.HandlerFunc {
	return generichttp.GetFloat(fg.GetFrequency)
}

// SetVoltage exposes an HTTP interface to the SetVoltage method
func SetVoltage(fg FunctionGenerator) http.HandlerFunc {
	return generichttp.SetFloat(fg.SetVoltage)
}

// GetVoltage exposes an HTTP interface to the GetVoltage method
func GetVoltage(fg FunctionGenerator) http.HandlerFunc {
	return generichttp.GetFloat(fg.GetVoltage)
}

// SetOffset exposes an HTTP interface to the SetOffset method
func SetOffset(fg FunctionGenerator) http.HandlerFunc {
	return generichttp.SetFloat(fg.SetOffset)
}

// GetOffset exposes an HTTP interface to the GetOffset method
func GetOffset(fg FunctionGenerator) http.HandlerFunc {
	return generichttp.GetFloat(fg.GetOffset)
}

// SetOutput exposes an HTTP interface to the Output control methods
func SetOutput(fg FunctionGenerator) http.HandlerFunc {
	return generichttp.SetBool(fg.SetOutput)
}

// GetOutput exposes an HTTP interface to the GetOutput method
func GetOutput(fg FunctionGenerator) http.HandlerFunc {
	return generichttp.GetBool(fg.GetOutput)
}

// SetOutputLoad exposes an HTTP interface to the SetOutputLoad method
func SetOutputLoad(fg FunctionGenerator) http.HandlerFunc {
	return generichttp.SetFloat(fg.SetOutputLoad)
}

func SetWaveform(fg FunctionGenerator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var (
			b   bytes.Buffer
			buf = &b
		)
		_, err := buf.ReadFrom(r.Body)
		defer r.Body.Close()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		buffer := buf.Bytes()
		header := *(*reflect.SliceHeader)(unsafe.Pointer(&buffer))

		// The length and capacity of the slice are different.
		header.Len /= 2
		header.Cap /= 2

		// Convert slice header to an []int32
		waveform := *(*[]uint16)(unsafe.Pointer(&header))
		err = fg.SetWaveform(waveform)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}

// HTTPFunctionGeneratorT holds an HTTP wrapper to a function generator
type HTTPFunctionGeneratorT struct {
	FG FunctionGenerator

	RouteTable generichttp.RouteTable
}

// RT makes this generichttp.httper compliant
func (h HTTPFunctionGeneratorT) RT() generichttp.RouteTable {
	return h.RouteTable
}

// NewHTTPFunctionGenerator wraps a function generator in an HTTP interface
func NewHTTPFunctionGenerator(fg FunctionGenerator) HTTPFunctionGeneratorT {
	rt := generichttp.RouteTable{}
	gen := HTTPFunctionGeneratorT{FG: fg, RouteTable: rt}
	HTTPFunctionGenerator(fg, rt)
	return gen
}

// SampleRateManipulator can manipulate their sampling rate
type SampleRateManipulator interface {
	// SetSampleRate configures the analog sampling rate of the scope
	SetSampleRate(float64) error

	// GetSampleRate returns the analog sampling rate of the scope
	GetSampleRate() (float64, error)
}

// SetSampleRate exposes an HTTP interface to SetSampleRate
func SetSampleRate(m SampleRateManipulator) http.HandlerFunc {
	return generichttp.SetFloat(m.SetSampleRate)
}

// GetSampleRate exposes an HTTP interface to GetSampleRate
func GetSampleRate(m SampleRateManipulator) http.HandlerFunc {
	return generichttp.GetFloat(m.GetSampleRate)
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
	return generichttp.SetFloat(o.SetTimebase)
}

// GetTimebase exposes an HTTP interface to GetTimebase
func GetTimebase(o Oscilloscope) http.HandlerFunc {
	return generichttp.GetFloat(o.GetTimebase)
}

// SetBitDepth exposes an HTTP interface to SetBitDepth
func SetBitDepth(o Oscilloscope) http.HandlerFunc {
	return generichttp.SetInt(o.SetBitDepth)
}

// GetBitDepth exposes an HTTP interface to GetBitDepth
func GetBitDepth(o Oscilloscope) http.HandlerFunc {
	return generichttp.GetInt(o.GetBitDepth)
}

// SetAcqLength exposes an HTTP interface to SetAcqLength
func SetAcqLength(o Oscilloscope) http.HandlerFunc {
	return generichttp.SetInt(o.SetAcqLength)
}

// GetAcqLength exposes an HTTP interface to GetAcqLength
func GetAcqLength(o Oscilloscope) http.HandlerFunc {
	return generichttp.GetInt(o.GetAcqLength)
}

// SetAcqMode exposes an HTTP interface to SetAcqMode
func SetAcqMode(o Oscilloscope) http.HandlerFunc {
	return generichttp.SetString(o.SetAcqMode)
}

// GetAcqMode exposes an HTTP interface to GetAcqMode
func GetAcqMode(o Oscilloscope) http.HandlerFunc {
	return generichttp.GetString(o.GetAcqMode)
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
		hp := generichttp.HumanPayload{T: types.Float64, Float: scale}
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

	RouteTable generichttp.RouteTable
}

// RT makes this generichttp.httper compliant
func (h HTTPOscilloscope) RT() generichttp.RouteTable {
	return h.RouteTable
}

// NewHTTPOscilloscope wraps a function generator in an HTTP interface
func NewHTTPOscilloscope(o Oscilloscope) HTTPOscilloscope {
	rt := generichttp.RouteTable{}
	rt[generichttp.MethodPath{Method: http.MethodGet, Path: "/scale"}] = GetScale(o)
	rt[generichttp.MethodPath{Method: http.MethodPost, Path: "/scale"}] = SetScale(o)

	rt[generichttp.MethodPath{Method: http.MethodGet, Path: "/timebase"}] = GetTimebase(o)
	rt[generichttp.MethodPath{Method: http.MethodPost, Path: "/timebase"}] = SetTimebase(o)

	rt[generichttp.MethodPath{Method: http.MethodGet, Path: "/bit-depth"}] = GetBitDepth(o)
	rt[generichttp.MethodPath{Method: http.MethodPost, Path: "/bit-depth"}] = SetBitDepth(o)

	rt[generichttp.MethodPath{Method: http.MethodGet, Path: "/sample-rate"}] = GetSampleRate(o)
	rt[generichttp.MethodPath{Method: http.MethodPost, Path: "/sample-rate"}] = SetSampleRate(o)

	rt[generichttp.MethodPath{Method: http.MethodGet, Path: "/acq-length"}] = GetAcqLength(o)
	rt[generichttp.MethodPath{Method: http.MethodPost, Path: "/acq-length"}] = SetAcqLength(o)

	rt[generichttp.MethodPath{Method: http.MethodGet, Path: "/acq-mode"}] = GetAcqMode(o)
	rt[generichttp.MethodPath{Method: http.MethodPost, Path: "/acq-mode"}] = SetAcqMode(o)

	rt[generichttp.MethodPath{Method: http.MethodPost, Path: "/acq-start"}] = StartAcq(o)
	rt[generichttp.MethodPath{Method: http.MethodGet, Path: "/acq-waveform"}] = AcquireWaveform(o)

	if rawer, ok := interface{}(o).(ascii.RawCommunicator); ok {
		RW := ascii.RawWrapper{Comm: rawer}
		rt[generichttp.MethodPath{Method: http.MethodPost, Path: "/raw"}] = RW.HTTPRaw
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
	Chan int `json:"channel"`

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
		i := generichttp.IntT{}
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
		hp := generichttp.HumanPayload{T: types.String, String: str}
		hp.EncodeAndRespond(w, r)
	}
}

// SetRecordingLength sets the length of a recording in samples
func SetRecordingLength(d DAQ) http.HandlerFunc {
	return generichttp.SetInt(d.SetRecordingLength)
}

// GetRecordingLength returns the length of a recording in samples
func GetRecordingLength(d DAQ) http.HandlerFunc {
	return generichttp.GetInt(d.GetRecordingLength)
}

// SetRecordingChannel sets the channel of a recording in samples
func SetRecordingChannel(d DAQ) http.HandlerFunc {
	return generichttp.SetInt(d.SetRecordingChannel)
}

// GetRecordingChannel returns the channel of a recording in samples
func GetRecordingChannel(d DAQ) http.HandlerFunc {
	return generichttp.GetInt(d.GetRecordingChannel)
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

	RouteTable generichttp.RouteTable
}

// RT satisfies generichttp.HTTPer
func (h HTTPDAQ) RT() generichttp.RouteTable {
	return h.RouteTable
}

// NewHTTPDAQ returns a newly HTTP wrapped DAQ
func NewHTTPDAQ(d DAQ) HTTPDAQ {
	rt := generichttp.RouteTable{}
	rt[generichttp.MethodPath{Method: http.MethodGet, Path: "/channel-label"}] = GetChannelLabel(d)
	rt[generichttp.MethodPath{Method: http.MethodPost, Path: "/channel-label"}] = SetChannelLabel(d)

	rt[generichttp.MethodPath{Method: http.MethodGet, Path: "/sample-rate"}] = GetSampleRate(d)
	rt[generichttp.MethodPath{Method: http.MethodPost, Path: "/sample-rate"}] = SetSampleRate(d)

	rt[generichttp.MethodPath{Method: http.MethodGet, Path: "/recording-length"}] = GetRecordingLength(d)
	rt[generichttp.MethodPath{Method: http.MethodPost, Path: "/recording-length"}] = SetRecordingLength(d)

	rt[generichttp.MethodPath{Method: http.MethodGet, Path: "/recording-channel"}] = GetRecordingChannel(d)
	rt[generichttp.MethodPath{Method: http.MethodPost, Path: "/recording-channel"}] = SetRecordingChannel(d)

	rt[generichttp.MethodPath{Method: http.MethodGet, Path: "/record"}] = Record(d)

	if rawer, ok := interface{}(d).(ascii.RawCommunicator); ok {
		RW := ascii.RawWrapper{Comm: rawer}
		rt[generichttp.MethodPath{Method: http.MethodPost, Path: "/raw"}] = RW.HTTPRaw
	}

	return HTTPDAQ{D: d, RouteTable: rt}
}
