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

// PictureTaker describes an interface to a camera which can capture images
type PictureTaker interface {
	// GetFrame triggers capture of a frame and returns the strided image data as 16-bit integers
	GetFrame() ([]uint16, error)

	// Burst takes N frames at a certain framerate and returns the contiguous strided buffer for the 3D array
	Burst(int, float64) ([]uint16, error)

	// SetExposureTime sets the exposure time
	SetExposureTime(time.Duration) error

	// GetExposureTime gets the exposure time
	GetExposureTime() (time.Duration, error)

	// SetAOI allows the AOI to be set
	SetAOI(AOI) error

	// GetAOI retrieves the current AOI
	GetAOI() (AOI, error)

	// SetBinning sets the binning option of the camera
	SetBinning(Binning) error

	// GetBinning returns the binning option of the camera
	GetBinning() (Binning, error)
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
func GetFrame(p PictureTaker, rec *imgrec.Recorder) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
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
			err = p.SetExposureTime(T)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
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

		aoi, err := p.GetAOI()
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
			im := &image.Gray{Pix: buf, Stride: aoi.Width, Rect: image.Rect(0, 0, aoi.Width, aoi.Height)}
			w.Header().Set("Content-Type", "image/jpeg")
			w.WriteHeader(http.StatusOK)
			jpeg.Encode(w, im, nil)
		case "png":
			buf := make([]byte, len(img))
			for idx := 0; idx < len(img); idx++ {
				buf[idx] = byte(img[idx] / 256) // scale 16 to 8 bits
			}
			im := &image.Gray{Pix: buf, Stride: aoi.Width, Rect: image.Rect(0, 0, aoi.Width, aoi.Height)}
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
			err = writeFits(w2, cards, img, aoi.Width, aoi.Height, 1)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
			return
		}

	}
}

// Burst takes a burst of N frames at M fps and returns it as a fits image cube
func Burst(p PictureTaker) http.HandlerFunc {
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
		aoi, err := p.GetAOI()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		cards := collectHeaderMetadata3(p)
		// mutate the header version because this is a burst
		cards[0].Value = cards[0].Value.(string) + "+burst" // inject burst modifier to header version
		cards = append(cards, fitsio.Card{Name: "fps", Value: t.FPS, Comment: "frame rate"})
		hdr := w.Header()
		hdr.Set("Content-Type", "image/fits")
		hdr.Set("Content-Disposition", "attachment; filename=image.fits")
		w.WriteHeader(http.StatusOK)
		err = writeFits(w, cards, img, aoi.Width, aoi.Height, t.Frames)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
}

// AOIManipulator describes an interface to a camera which has a configurable area of interest
type AOIManipulator interface {
}
