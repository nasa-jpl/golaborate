// Package camera provides a generic HTTP interface to a scientific camera
package camera

import (
	"encoding/json"
	"fmt"
	"go/types"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/astrogo/fitsio"
	"github.jpl.nasa.gov/bdube/golab/generichttp"
	"github.jpl.nasa.gov/bdube/golab/imgrec"
	"github.jpl.nasa.gov/bdube/golab/util"
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

// HxV is a shorthand for "{h}x{v}", e.g. b.H, b.V = 1,1 => "1x1" or 3,3 => "3x3"
func (b Binning) HxV() string {
	return fmt.Sprintf("%dx%d", b.H, b.V)
}

// HxVToBin converts a string like "3x3" => Binning{3,3}
func HxVToBin(hxv string) Binning {
	b := Binning{}
	chunks := strings.Split(hxv, "x")
	if len(chunks) != 2 {
		return b
	}
	// impossible for this to panic, since len must == 2
	b.H, _ = strconv.Atoi(chunks[0])
	b.V, _ = strconv.Atoi(chunks[1])
	return b
}

// ThermalManager describes an interface to a camera which can manage its thermal performance
type ThermalManager interface {
	// GetCooling queries if focal plane cooling is currently active
	GetCooling() (bool, error)

	// SetCooling turns focal plane cooling on or off
	SetCooling(bool) error

	// GetTemperature gets the current focal plane temperature in Celsius
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

// HTTPThermalManager binds routes for thermal management on the table
func HTTPThermalManager(t ThermalManager, table generichttp.RouteTable) {
	table[generichttp.MethodPath{Method: http.MethodGet, Path: "/fan"}] = GetFan(t)
	table[generichttp.MethodPath{Method: http.MethodPost, Path: "/fan"}] = SetFan(t)
	table[generichttp.MethodPath{Method: http.MethodGet, Path: "/sensor-cooling"}] = GetCooling(t)
	table[generichttp.MethodPath{Method: http.MethodPost, Path: "/sensor-cooling"}] = SetCooling(t)
	table[generichttp.MethodPath{Method: http.MethodGet, Path: "/temperature"}] = GetTemperature(t)
	table[generichttp.MethodPath{Method: http.MethodGet, Path: "/temperature-setpoint-options"}] = GetTemperatureSetpoints(t)
	table[generichttp.MethodPath{Method: http.MethodGet, Path: "/temperature-setpoint"}] = GetTemperatureSetpoint(t)
	table[generichttp.MethodPath{Method: http.MethodPost, Path: "/temperature-setpoint"}] = SetTemperatureSetpoint(t)
	table[generichttp.MethodPath{Method: http.MethodGet, Path: "/temperature-status"}] = GetTemperatureStatus(t)
}

// GetCooling returns an HTTP handler func that returns the cooling status of the camera
func GetCooling(t ThermalManager) http.HandlerFunc {
	return generichttp.GetBool(t.GetCooling)
}

// SetCooling returns an HTTP handler func that turns the fan on or off over HTTP
func SetCooling(t ThermalManager) http.HandlerFunc {
	return generichttp.SetBool(t.SetCooling)
}

// GetTemperature returns an HTTP handler func that returns the temperature over HTTP
func GetTemperature(t ThermalManager) http.HandlerFunc {
	return generichttp.GetFloat(t.GetTemperature)
}

// GetTemperatureSetpoint returns an HTTP handler func that returns the temperature setpoint over HTTP
func GetTemperatureSetpoint(t ThermalManager) http.HandlerFunc {
	return generichttp.GetString(t.GetTemperatureSetpoint)
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
	return generichttp.SetString(t.SetTemperatureSetpoint)
}

// GetTemperatureStatus returns an HTTP handler func that returns the cooling status over HTTP
func GetTemperatureStatus(t ThermalManager) http.HandlerFunc {
	return generichttp.GetString(t.GetTemperatureStatus)
}

// GetFan returns an HTTP handler func that returns the fan status over HTTP
func GetFan(t ThermalManager) http.HandlerFunc {
	return generichttp.GetBool(t.GetFan)
}

// SetFan returns an HTTP handler func that sets the fan status over hTTP
func SetFan(t ThermalManager) http.HandlerFunc {
	return generichttp.SetBool(t.SetFan)
}

// PictureTaker describes an interface to a camera which can capture images
type PictureTaker interface {
	Camera

	// SetExposureTime sets the exposure time
	SetExposureTime(time.Duration) error

	// GetExposureTime gets the exposure time
	GetExposureTime() (time.Duration, error)
}

// Burster describes an interface of a camera that may take a burst of frames
type Burster interface {
	// Burst takes N frames at a certain framerate and writes them to the provided channel
	Burst(int, float64, chan<- image.Image) error
}

// BurstWrapper is a type that holds the internal buffer for a burst of camera
// frames
type BurstWrapper struct {
	// ch is the channel of images streamed from the camera
	ch chan image.Image

	// errCh is the channel that will receive an error from Burst if it occurs
	errCh chan error

	// B is the bursty camera
	B Burster

	frames int
}

// SetupBurst returns a function which triggers the burst on the camera
func (b *BurstWrapper) SetupBurst(w http.ResponseWriter, r *http.Request) {
	t := struct {
		FPS    float64 `json:"fps"`
		Frames int     `json:"frames"`
		Spool  int     `json:"spool"`
	}{}
	err := json.NewDecoder(r.Body).Decode(&t)
	defer r.Body.Close()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if t.Spool == 0 {
		t.Spool = int(float64(t.Frames) * t.FPS)
	}
	b.ch = make(chan image.Image, t.Spool)
	go func() {
		b.errCh <- b.B.Burst(t.Frames, t.FPS, b.ch)
	}()
	w.WriteHeader(http.StatusOK)
	return
}

// ReadFrame returns one frame from the buffer, as FITS, over HTTP
func (b *BurstWrapper) ReadFrame(w http.ResponseWriter, r *http.Request) {
	select {
	case err := <-b.errCh:
		// there was an error, feedback to the burst to stop by closing the channel
		close(b.ch)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	case <-time.After(2 * time.Second): // if you're doing a burst the frames should come far faster than this
		// feedback to the burst
		close(b.ch)
		http.Error(w, "timeout waiting for frame from the camera", http.StatusInternalServerError)
		return
	case img := <-b.ch:
		hdr := w.Header()
		hdr.Set("Content-Type", "image/fits")
		hdr.Set("Content-Disposition", "attachment; filename=image.fits")
		w.WriteHeader(http.StatusOK)
		err := WriteFits(w, []fitsio.Card{}, []image.Image{img})
		if err != nil {
			fmt.Println(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

// ReadAllFrames reads all of the frames from the camera and writes them as a
// cube to a single FITS file
func (b *BurstWrapper) ReadAllFrames(w http.ResponseWriter, r *http.Request) {
	var images []image.Image
	for img := range b.ch {
		images = append(images, img)
	}
	// get the error if there is one, otherwise use nil as the
	// default
	err := <-b.errCh
	var errS string
	if err != nil {
		errS = err.Error()
	}
	hdr := w.Header()
	hdr.Set("Content-Type", "image/fits")
	hdr.Set("Content-Disposition", "attachment; filename=image.fits")
	w.WriteHeader(http.StatusOK)
	err = WriteFits(w, []fitsio.Card{
		{
			Name:    "ERR",
			Value:   errS,
			Comment: "error encountered capturing burst"}}, images)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// Inject puts burst management routes on a table
func (b *BurstWrapper) Inject(table generichttp.RouteTable) {
	table[generichttp.MethodPath{Method: http.MethodPost, Path: "/burst/setup"}] = b.SetupBurst
	table[generichttp.MethodPath{Method: http.MethodGet, Path: "/burst/frame"}] = b.ReadFrame
	table[generichttp.MethodPath{Method: http.MethodGet, Path: "/burst/all-frames"}] = b.ReadAllFrames
}

// MetadataMaker can produce an array of FITS cards
type MetadataMaker interface {
	// CollectHeaderMetadata produces an array of FITS cards
	CollectHeaderMetadata() []fitsio.Card
}

// HTTPPicture injects HTTP methods into a route table for a picture taker
func HTTPPicture(p PictureTaker, table generichttp.RouteTable, rec *imgrec.Recorder) {
	table[generichttp.MethodPath{Method: http.MethodGet, Path: "/exposure-time"}] = GetExposureTime(p)
	table[generichttp.MethodPath{Method: http.MethodPost, Path: "/exposure-time"}] = SetExposureTime(p)
	table[generichttp.MethodPath{Method: http.MethodGet, Path: "/image"}] = GetFrame(p, rec)
}

// SetExposureTime sets the exposure time on a POST request.
// it can be provided either as a query parameter exposureTime, formatted in a
// way that is parsable by golang/time.ParseDuration, or a json payload with
// key f64, holding the exposure time in seconds.
func SetExposureTime(p PictureTaker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		texp := q.Get("exposureTime")
		var d time.Duration
		var err error
		if texp == "" {
			f := generichttp.FloatT{}
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
		hp := generichttp.HumanPayload{T: types.Float64, Float: f.Seconds()}
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

		switch format {
		case "jpg":
			w.Header().Set("Content-Type", "image/jpeg")
			w.WriteHeader(http.StatusOK)
			if g16, ok := (img).(*image.Gray16); ok {
				uints := bytesToUint(g16.Pix)
				b := make([]byte, len(uints))
				l := len(uints)
				for i := 0; i < l; i++ {
					b[i] = byte(uints[i] / 255)
				}
				bound := g16.Bounds()
				out := &image.Gray{Pix: b, Stride: bound.Dx(), Rect: bound}
				img = out
			}
			jpeg.Encode(w, img, nil)
		case "png":
			w.Header().Set("Content-Type", "image/png")
			w.WriteHeader(http.StatusOK)
			if g16, ok := (img).(*image.Gray16); ok {
				uints := bytesToUint(g16.Pix)
				b := make([]byte, len(uints))
				l := len(uints)
				for i := 0; i < l; i++ {
					b[i] = byte(uints[i] / 255)
				}
				bound := g16.Bounds()
				out := &image.Gray{Pix: b, Stride: bound.Dx(), Rect: bound}
				img = out
			}
			png.Encode(w, img)
		case "fits":
			// ^\- for picture taker::
			// there is some cross logic, where picturetaker introspects whether the type
			// exposes a recorder, to add the recorder logic into the fits write
			// it also introspects whether the type exposes a metadata generating function
			// for the metadata portion.
			// this isn't totally clean and decoupled logic, but I think it is the best that
			// can be done without breaking the implicit interface of
			// func HTTP<XYZ>(xyz XYZ, table generichttp.RoutTable)
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
			var cards []fitsio.Card
			if carder, ok := interface{}(p).(MetadataMaker); ok {
				cards = carder.CollectHeaderMetadata()
			}

			hdr := w.Header()
			hdr.Set("Content-Type", "image/fits")
			hdr.Set("Content-Disposition", "attachment; filename=image.fits")
			w.WriteHeader(http.StatusOK)
			err = WriteFits(w2, cards, []image.Image{img})
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			return
		}
	}
}

// AOIManipulator is an interface to a camera's AOI manipulating functions
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
func HTTPAOIManipulator(a AOIManipulator, table generichttp.RouteTable) {
	table[generichttp.MethodPath{Method: http.MethodGet, Path: "/aoi"}] = GetAOI(a)
	table[generichttp.MethodPath{Method: http.MethodPost, Path: "/aoi"}] = SetAOI(a)
	table[generichttp.MethodPath{Method: http.MethodGet, Path: "/binning"}] = GetBinning(a)
	table[generichttp.MethodPath{Method: http.MethodPost, Path: "/binning"}] = SetBinning(a)
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

// SetBinning sets the binning over HTTP as JSON
func SetBinning(a AOIManipulator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		b := Binning{}
		err := json.NewDecoder(r.Body).Decode(&b)
		defer r.Body.Close()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		err = a.SetBinning(b)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}

// GetBinning gets the binning over HTTP as JSON
func GetBinning(a AOIManipulator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		b, err := a.GetBinning()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		err = json.NewEncoder(w).Encode(b)
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

// EMGainManager describes an interface that can manage its electron multiplying gain
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
	return generichttp.GetString(e.GetEMGainMode)
}

// SetEMGainMode sets the EM gain mode over HTTP as JSON
func SetEMGainMode(e EMGainManager) http.HandlerFunc {
	return generichttp.SetString(e.SetEMGainMode)
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
	return generichttp.GetInt(e.GetEMGain)
}

// SetEMGain sets the EM gain over HTTP as JSON
func SetEMGain(e EMGainManager) http.HandlerFunc {
	return generichttp.SetInt(e.SetEMGain)
}

// HTTPEMGainManager binds routes that control EM gain to the table
func HTTPEMGainManager(e EMGainManager, table generichttp.RouteTable) {
	table[generichttp.MethodPath{Method: http.MethodGet, Path: "/em-gain"}] = GetEMGain(e)
	table[generichttp.MethodPath{Method: http.MethodPost, Path: "/em-gain"}] = SetEMGain(e)
	table[generichttp.MethodPath{Method: http.MethodGet, Path: "/em-gain-mode"}] = GetEMGainMode(e)
	table[generichttp.MethodPath{Method: http.MethodPost, Path: "/em-gain-mode"}] = SetEMGainMode(e)
	table[generichttp.MethodPath{Method: http.MethodGet, Path: "/em-gain-range"}] = GetEMGainRange(e)
}

// ShutterController describes an interface to a camera which may manipulate its shutter
type ShutterController interface {
	// SetShutter sets the shutter to be open or closed
	SetShutter(bool) error

	// GetShutter returns if the shutter is open or closed
	GetShutter() (bool, error)

	// SetShutterAuto puts the shutter into automatic (camera controlled) or
	// manual (user controlled) mode
	SetShutterAuto(bool) error

	// GetShutterAuto returns if the shutter is automatically (camera) controlled
	// or user controlled
	GetShutterAuto() (bool, error)
}

// SetShutter opens or closes the shutter over HTTP as JSON
func SetShutter(s ShutterController) http.HandlerFunc {
	return generichttp.SetBool(s.SetShutter)
}

// GetShutter returns if the shutter is currently open over HTTP as JSON
func GetShutter(s ShutterController) http.HandlerFunc {
	return generichttp.GetBool(s.GetShutter)
}

// SetShutterAuto opens or closes the shutter over HTTP as JSON
func SetShutterAuto(s ShutterController) http.HandlerFunc {
	return generichttp.SetBool(s.SetShutterAuto)
}

// GetShutterAuto returns if the shutter is currently open over HTTP as JSON
func GetShutterAuto(s ShutterController) http.HandlerFunc {
	return generichttp.GetBool(s.GetShutterAuto)
}

// HTTPShutterController binds routes to control the shutter to a route table
func HTTPShutterController(s ShutterController, table generichttp.RouteTable) {
	table[generichttp.MethodPath{Method: http.MethodGet, Path: "/shutter"}] = GetShutter(s)
	table[generichttp.MethodPath{Method: http.MethodPost, Path: "/shutter"}] = SetShutter(s)
	table[generichttp.MethodPath{Method: http.MethodGet, Path: "/shutter-auto"}] = GetShutterAuto(s)
	table[generichttp.MethodPath{Method: http.MethodPost, Path: "/shutter-auto"}] = SetShutterAuto(s)
}

// Camera describes the most basic camera possible
type Camera interface {
	// GetFrame returns a frame from the device as a strided array
	GetFrame() (image.Image, error)
}

// HTTPCamera is a camera which exposes an HTTP interface to itself
type HTTPCamera struct {
	PictureTaker

	RouteTable generichttp.RouteTable
}

// NewHTTPCamera returns a new HTTP wrapper around a camera
func NewHTTPCamera(p PictureTaker, rec *imgrec.Recorder) HTTPCamera {
	w := HTTPCamera{PictureTaker: p}
	rt := generichttp.RouteTable{}
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
	if b, ok := interface{}(p).(Burster); ok {
		wrap := BurstWrapper{B: b}
		wrap.Inject(rt)

	}

	w.RouteTable = rt
	return w
}

// RT satisfies generichttp.HTTPer
func (h HTTPCamera) RT() generichttp.RouteTable {
	return h.RouteTable
}
