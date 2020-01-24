// Package camera provides a generic HTTP interface to a scientific camera
package camera

import (
	"encoding/json"
	"go/types"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"net/http"
	"time"

	"github.com/astrogo/fitsio"
	"github.jpl.nasa.gov/HCIT/go-hcit/imgrec"
	"github.jpl.nasa.gov/HCIT/go-hcit/server"
	"github.jpl.nasa.gov/HCIT/go-hcit/util"
	"goji.io/pat"
)

// AOI describes an area of interest on the camera
type AOI struct {
	// Left is the left pixel index.  1-based
	Left int `json:"left"`

	// Top is the top pixel index.  1-based
	Top int `json:"top"`

	// Width is the width in pixels
	Width int `json:"width"`

	// Height is the height in pixels
	Height int `json:"height"`
}

// Right is shorthand for a.Left+a.Width
func (a AOI) Right() int {
	return a.Left + a.Width
}

// Bottom is shorthand for a.Top+a.Height
func (a AOI) Bottom() int {
	return a.Top + a.Height
}

// Binning encapsulates information about pixel addition on camera
type Binning struct {
	// H is the horizontal binning factor
	H int `json:"h"`

	// V is the vertical binning factor
	V int `json:"v"`
}

// ThermalManager describes an interface to a camera which can manage its thermal performance
type ThermalManager interface {
	// GetCooling queries if focal plane cooling is currently active
	GetCooling() (bool, error)

	// SetCooling turns focal plane cooling on or off
	SetCooling(bool) error

	// GetTemperature gets the current focal plane temperature in Celcius
	GetTemperature() (float64, error)

	// GetTemperatureSetpoints returns the valid temperature setpoints.  Could return a discrete list, or min/max
	GetTemperatureSetpoints() ([]string, error)

	// GetTemperatureSetpoint gets the temperature setpoint, as a string for andor SDK3 compatibility
	GetTemperatureSetpoint() (string, error)

	// SetTemperatureSetpoint sets the temperature setpoint, as a string for andor SDk3 compatibility
	SetTemperatureSetpoint(string) error

	// GetTemperatureStatus gets the status of the sensor cooling subsystem
	GetTemperatureStatus() (string, error)

	// GetFan queries if the fan is on or off
	GetFan() (bool, error)

	// SetFan turns the fan on or off
	SetFan(bool) error
}

// HTTPThermalManager binds routes for thermal amangement on the table
func HTTPThermalManager(t ThermalManager, table server.RouteTable) {
	table[pat.Get("/fan")] = GetFan(t)
	table[pat.Post("/fan")] = SetFan(t)
	table[pat.Get("/sensor-cooling")] = GetCooling(t)
	table[pat.Post("/sensor-cooling")] = SetCooling(t)
	table[pat.Get("/temperature")] = GetTemperature(t)
	table[pat.Get("/temperature-setpoint-options")] = GetTemperatureSetpoints(t)
	table[pat.Get("/temperature-setpoint")] = GetTemperatureSetpoint(t)
	table[pat.Post("/temperature-setpoint")] = SetTemperatureSetpoint(t)
	table[pat.Get("/temperature-status")] = GetTemperatureStatus(t)
}

// GetCooling returns an HTTP handler func that returns the cooling status of the camera
func GetCooling(t ThermalManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cool, err := t.GetCooling()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		hp := server.HumanPayload{T: types.Bool, Bool: cool}
		hp.EncodeAndRespond(w, r)
		return
	}
}

// SetCooling returns an HTTP handler func that turns the fan on or off over HTTP
func SetCooling(t ThermalManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		b := server.BoolT{}
		err := json.NewDecoder(r.Body).Decode(&b)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer r.Body.Close()
		err = t.SetCooling(b.Bool)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		return
	}
}

// GetTemperature returns an HTTP handler func that returns the temperature over HTTP
func GetTemperature(t ThermalManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		t, err := t.GetTemperature()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		hp := server.HumanPayload{T: types.Float64, Float: t}
		hp.EncodeAndRespond(w, r)
		return
	}
}

// GetTemperatureSetpoint returns an HTTP handler func that returns the temperature setpoint over HTTP
func GetTemperatureSetpoint(t ThermalManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		setpt, err := t.GetTemperatureSetpoint()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		hp := server.HumanPayload{T: types.String, String: setpt}
		hp.EncodeAndRespond(w, r)
		return
	}
}

// GetTemperatureSetpoints returns an HTTP handler func that returns the temperature setpoint over HTTP
func GetTemperatureSetpoints(t ThermalManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		opts, err := t.GetTemperatureSetpoints()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		err = json.NewEncoder(w).Encode(opts)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
}

// SetTemperatureSetpoint returns an HTTP handler func that sets the temperature setpoint over HTTP
func SetTemperatureSetpoint(t ThermalManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		str := server.StrT{}
		err := json.NewDecoder(r.Body).Decode(&str)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer r.Body.Close()
		err = t.SetTemperatureSetpoint(str.Str)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		return
	}
}

// GetTemperatureStatus returns an HTTP handler func that returns the cooling status over HTTP
func GetTemperatureStatus(t ThermalManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		stat, err := t.GetTemperatureStatus()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		hp := server.HumanPayload{T: types.String, String: stat}
		hp.EncodeAndRespond(w, r)
		return
	}
}

// GetFan returns an HTTP handler func that returns the fan status over HTTP
func GetFan(t ThermalManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		on, err := t.GetFan()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		hp := server.HumanPayload{T: types.Bool, Bool: on}
		hp.EncodeAndRespond(w, r)
		return
	}
}

// SetFan returns an HTTP handler func that sets the fan status over hTTP
func SetFan(t ThermalManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		b := server.BoolT{}
		err := json.NewDecoder(r.Body).Decode(&b)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer r.Body.Close()
		err = t.SetFan(b.Bool)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		return
	}
}

// PictureTaker describes an interface to a camera which can capture images
type PictureTaker interface {
	// GetFrame triggers capture of a frame and returns the strided image data as 16-bit integers
	GetFrame() ([]uint16, error)

	//GetFrameSize returns the image (width, height)
	GetFrameSize() (int, int, error)

	// Burst takes N frames at a certain framerate and returns the contiguous strided buffer for the 3D array
	Burst(int, float64) ([]uint16, error)

	// SetExposureTime sets the exposure time
	SetExposureTime(time.Duration) error

	// GetExposureTime gets the exposure time
	GetExposureTime() (time.Duration, error)
}

// MetadataMaker can produce an array of FITS cards
type MetadataMaker interface {
	// CollectHeaderMetadata produces an array of FITS cards
	CollectHeaderMetadata() []fitsio.Card
}

// HTTPPicture injects HTTP methods into a route table for a picture taker
func HTTPPicture(p PictureTaker, table server.RouteTable, rec *imgrec.Recorder) {
	table[pat.Get("/exposure-time")] = GetExposureTime(p)
	table[pat.Post("/exposure-time")] = SetExposureTime(p)
	table[pat.Get("/image")] = GetFrame(p, rec)
	table[pat.Get("/burst")] = Burst(p, rec)
}

// SetExposureTime sets the exposure time on a POST request.
// it can be provided either as a query parameter exposureTime, formatted in a
// way that is parseable by golang/time.ParseDuration, or a json payload with
// key f64, holding the exposure time in seconds.
func SetExposureTime(p PictureTaker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		texp := q.Get("exposureTime")
		var d time.Duration
		var err error
		if texp == "" {
			f := server.FloatT{}
			err = json.NewDecoder(r.Body).Decode(&f)
			d = time.Duration(int(f.F64*1e9)) * time.Nanosecond // 1e9 s => ns
		} else {
			d, err = time.ParseDuration(texp)
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		err = p.SetExposureTime(d)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		return
	}
}

// GetExposureTime gets the exposure time on a GET request
func GetExposureTime(p PictureTaker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		f, err := p.GetExposureTime()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		hp := server.HumanPayload{T: types.Float64, Float: f.Seconds()}
		hp.EncodeAndRespond(w, r)
		return
	}
}

// GetFrame takes a picture and returns it on a GET request.
//
// the image format may be specified in a query parameter; default to jpg
//
// the exposure time may be specified as a query parameter in any time-looking
// format, such as "25ms" or "10us".  Strictly speaking, it must be a valid
// input to golang time.ParseDuration.
//
// if no unit is appended, an s (seconds) is added.
//
// if no exposure time is provided, it is not updated and the existing value is used.
func GetFrame(p Camera, rec *imgrec.Recorder) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if pictureTaker, ok := interface{}(p).(PictureTaker); ok {
			texp := q.Get("exposureTime")
			if texp != "" {
				if util.AllElementsNumbers(texp) {
					texp = texp + "s"
				}
				T, err := time.ParseDuration(texp)
				if err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
				err = pictureTaker.SetExposureTime(T)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
			}
		}
		img, err := p.GetFrame()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		format := q.Get("fmt")
		if format == "" {
			format = "jpg"
		}

		width, height, err := p.GetFrameSize()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		switch format {
		case "jpg":
			buf := make([]byte, len(img))
			for idx := 0; idx < len(img); idx++ {
				buf[idx] = byte(img[idx] / 256) // scale 16 to 8 bits
			}
			im := &image.Gray{Pix: buf, Stride: width, Rect: image.Rect(0, 0, width, height)}
			w.Header().Set("Content-Type", "image/jpeg")
			w.WriteHeader(http.StatusOK)
			jpeg.Encode(w, im, nil)
		case "png":
			buf := make([]byte, len(img))
			for idx := 0; idx < len(img); idx++ {
				buf[idx] = byte(img[idx] / 256) // scale 16 to 8 bits
			}
			im := &image.Gray{Pix: buf, Stride: width, Rect: image.Rect(0, 0, width, height)}
			w.Header().Set("Content-Type", "image/png")
			w.WriteHeader(http.StatusOK)
			png.Encode(w, im)
		case "fits":
			// ^\- for picture taker::
			// there is some cross logic, where picturetaker introspects whether the type
			// exposes a recorder, to add the recorder logic into the fits write
			// it also introspects whether the type exposes a metadata generating function
			// for the metadata portion.
			// this isn't totally clean and decoupled logic, but I think it is the best that
			// can be done without breaking the implicit interface of
			// func HTTP<XYZ>(xyz XYZ, table server.RoutTable)
			//
			// since FITS write also rightfully lives a layer above the camera (application layer)
			// that is also introspected through a metadata interface
			// declare a writer to use to stream the file to
			var w2 io.Writer
			if rec != nil && rec.Enabled && rec.Root != "" {
				// if it is "", the recorder is not to be used
				w2 = io.MultiWriter(w, rec)
				defer rec.Incr()
			} else {
				w2 = w
			}
			cards := []fitsio.Card{}
			if carder, ok := interface{}(p).(MetadataMaker); ok {
				cards = carder.CollectHeaderMetadata()
			}

			hdr := w.Header()
			hdr.Set("Content-Type", "image/fits")
			hdr.Set("Content-Disposition", "attachment; filename=image.fits")
			err = writeFits(w2, cards, img, width, height, 1)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
			return
		}
	}
}

// Burst takes a burst of N frames at M fps and returns it as a fits image cube
func Burst(p PictureTaker, rec *imgrec.Recorder) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		t := struct {
			FPS    float64 `json:"fps"`
			Frames int     `json:"frames"`
		}{}
		err := json.NewDecoder(r.Body).Decode(&t)
		defer r.Body.Close()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		img, err := p.Burst(t.Frames, t.FPS)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		width, height, err := p.GetFrameSize()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		cards := []fitsio.Card{}
		if carder, ok := interface{}(p).(MetadataMaker); ok {
			cards = carder.CollectHeaderMetadata()
		}
		// mutate the header version because this is a burst.
		// Opportunity for a bug here if the first card isn't a header version tag,
		// but we wouldn't violate that design, would we?
		cards[0].Value = cards[0].Value.(string) + "+burst" // inject burst modifier to header version
		cards = append(cards, fitsio.Card{Name: "fps", Value: t.FPS, Comment: "frame rate"})

		var w2 io.Writer
		if rec != nil && rec.Enabled && rec.Root != "" {
			// if it is "", the recorder is not to be used
			w2 = io.MultiWriter(w, rec)
			defer rec.Incr()
		} else {
			w2 = w
		}
		hdr := w.Header()
		hdr.Set("Content-Type", "image/fits")
		hdr.Set("Content-Disposition", "attachment; filename=image.fits")
		w.WriteHeader(http.StatusOK)
		err = writeFits(w2, cards, img, width, height, t.Frames)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
}

// AOIManipulator is an interface to a camera's AOI manipulating factures
type AOIManipulator interface {
	// SetAOI allows the AOI to be set
	SetAOI(AOI) error

	// GetAOI retrieves the current AOI
	GetAOI() (AOI, error)

	// SetBinning sets the binning option of the camera
	SetBinning(Binning) error

	// GetBinning returns the binning option of the camera
	GetBinning() (Binning, error)
}

// HTTPAOIManipulator injects routes to manipulate the AOI of a camera
// into a route table
func HTTPAOIManipulator(a AOIManipulator, table server.RouteTable) {
	table[pat.Get("/aoi")] = GetAOI(a)
	table[pat.Post("/aoi")] = SetAOI(a)
}

// SetAOI returns an HTTP handler func that sets the AOI of the camera
func SetAOI(a AOIManipulator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		aoi := AOI{}
		err := json.NewDecoder(r.Body).Decode(&aoi)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		err = a.SetAOI(aoi)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
		return
	}

}

// GetAOI returns an HTTP handler func that gets the AOI of the camera
func GetAOI(a AOIManipulator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		aoi, err := a.GetAOI()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		err = json.NewEncoder(w).Encode(aoi)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
}

// FeatureManager describes an interface which can manage its features
type FeatureManager interface {
	// Configure adjusts several features of the camera at once
	Configure(map[string]interface{}) error
}

type EMGainManager interface {
	// GetEMGainMode returns how the EM gain is applied in the camera
	GetEMGainMode() (string, error)

	// SetEMGainMode changes how the EM gain is applied in the camera
	SetEMGainMode(string) error

	// GetEMGainRange returns the lower, upper limits on EM gain
	GetEMGainRange() (int, int, error)

	// GetEMGain returns the current EM gain setting
	GetEMGain() (int, error)

	// SetEMGain sets the current EM gain setting
	SetEMGain(int) error
}

// GetEMGainMode returns the EM gain mode over HTTP as JSON
func GetEMGainMode(e EMGainManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		mode, err := e.GetEMGainMode()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		hp := server.HumanPayload{T: types.String, String: mode}
		hp.EncodeAndRespond(w, r)
		return
	}
}

// SetEMGainMode sets the EM gain mode over HTTP as JSON
func SetEMGainMode(e EMGainManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		str := server.StrT{}
		err := json.NewDecoder(r.Body).Decode(&str)
		defer r.Body.Close()
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		err = e.SetEMGainMode(str.Str)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}

// GetEMGainRange returns the min/max EM gain over HTTP as JSON
func GetEMGainRange(e EMGainManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		min, max, err := e.GetEMGainRange()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		ret := struct {
			Min int `json:"min"`
			Max int `json:"max"`
		}{Min: min, Max: max}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		err = json.NewEncoder(w).Encode(ret)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

// GetEMGain gets the EM gain over HTTP as JSON
func GetEMGain(e EMGainManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		i, err := e.GetEMGain()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		hp := server.HumanPayload{T: types.Int, Int: i}
		hp.EncodeAndRespond(w, r)
		return
	}
}

// SetEMGain sets the EM gain over HTTP as JSON
func SetEMGain(e EMGainManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		iT := server.IntT{}
		err := json.NewDecoder(r.Body).Decode(&iT)
		defer r.Body.Close()
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		err = e.SetEMGain(iT.Int)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

// HTTPEMGainManager binds routes that control EM gain to the table
func HTTPEMGainManager(e EMGainManager, table server.RouteTable) {
	table[pat.Get("/em-gain")] = GetEMGain(e)
	table[pat.Post("/em-gain")] = SetEMGain(e)
	table[pat.Get("/em-gain-mode")] = GetEMGainMode(e)
	table[pat.Post("/em-gain-mode")] = SetEMGainMode(e)
	table[pat.Get("/em-gain-range")] = GetEMGainRange(e)
}

// ShutterController describes an interface to a camera which may manipulate its shutter
type ShutterController interface {
	// SetShutter sets the shutter to be open or closed
	SetShutter(bool) error

	// GetShutter returns if the shutter is open or closed
	GetShutter() (bool, error)
}

// SetShutter opens or closes the shutter over HTTP as JSON
func SetShutter(s ShutterController) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		b := server.BoolT{}
		err := json.NewDecoder(r.Body).Decode(&b)
		defer r.Body.Close()
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		err = s.SetShutter(b.Bool)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}

// GetShutter returns if the shutter is currently open over HTTP as JSON
func GetShutter(s ShutterController) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		open, err := s.GetShutter()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		hp := server.HumanPayload{T: types.Bool, Bool: open}
		hp.EncodeAndRespond(w, r)
		return
	}
}

// HTTPShutterController binds routes to control the shutter to a route table
func HTTPShutterController(s ShutterController, table server.RouteTable) {
	table[pat.Get("/shutter")] = GetShutter(s)
	table[pat.Post("/shutter")] = SetShutter(s)
}

// Camera describes the most basic camera possible
type Camera interface {
	// GetFrame returns a frame from the device as a strided array
	GetFrame() ([]uint16, error)

	//GetFrameSize gets the (W, H) of a frame
	GetFrameSize() (int, int, error)
}

// HTTPCamera is a camera which exposes an HTTP interface to itself
type HTTPCamera struct {
	PictureTaker

	RouteTable server.RouteTable
}

// NewHTTPCamera returns a new HTTP wrapper around a camera
func NewHTTPCamera(p PictureTaker, rec *imgrec.Recorder) HTTPCamera {
	w := HTTPCamera{PictureTaker: p}
	rt := server.RouteTable{}
	HTTPPicture(p, rt, rec)
	if thermal, ok := interface{}(p).(ThermalManager); ok {
		HTTPThermalManager(thermal, rt)
	}
	if aoi, ok := interface{}(p).(AOIManipulator); ok {
		HTTPAOIManipulator(aoi, rt)
	}
	if em, ok := interface{}(p).(EMGainManager); ok {
		HTTPEMGainManager(em, rt)
	}
	if sh, ok := interface{}(p).(ShutterController); ok {
		HTTPShutterController(sh, rt)
	}

	w.RouteTable = rt
	return w
}

// RT satisfies server.HTTPer
func (h HTTPCamera) RT() server.RouteTable {
	return h.RouteTable
}
