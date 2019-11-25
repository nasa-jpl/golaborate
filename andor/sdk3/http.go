package sdk3

import (
	"encoding/json"
	"go/types"
	"image"
	"image/jpeg"
	"image/png"
	"net/http"
	"time"

	"github.com/astrogo/fitsio"
	"github.jpl.nasa.gov/HCIT/go-hcit/server"
	"github.jpl.nasa.gov/HCIT/go-hcit/util"
	"goji.io/pat"
)

// HTTPWrapper provides an HTTP interface to a camera
type HTTPWrapper struct {
	// Camera is the camera object being wrapped
	*Camera

	RouteTable server.RouteTable
}

// NewHTTPWrapper returns a new wrapper with the route table populated
func NewHTTPWrapper(c *Camera) HTTPWrapper {
	w := HTTPWrapper{Camera: c}
	w.RouteTable = server.RouteTable{
		// image capture
		pat.Get("/image"): w.GetFrame,

		// exposure manipulation
		pat.Get("/exposure-time"):  w.GetExposureTime,
		pat.Post("/exposure-time"): w.SetExposureTime,

		// thermals
		pat.Get("/fan"):                          w.GetFanOn,
		pat.Post("/fan"):                         w.SetFanOn,
		pat.Get("/sensor-cooling"):               w.GetCooling,
		pat.Post("/sensor-cooling"):              w.SetCooling,
		pat.Get("/temperature"):                  w.GetTemperature,
		pat.Get("/temperature-setpoint-options"): w.GetTemperatureSetpoints,
		pat.Get("/temperature-setpoint"):         w.GetTemperatureSetpoint,
		pat.Post("/temperature-setpoint"):        w.SetTemperatureSetpoint,
		pat.Get("/temperature-status"):           w.GetTemperatureStatus,

		// generic
		pat.Get("/feature"):           w.GetFeatures,
		pat.Get("/feature/:feature"):  w.GetFeature,
		pat.Post("/feature/:feature"): w.SetFeature,
	}
	return w
}

// SetExposureTime sets the exposure time on a POST request
func (h *HTTPWrapper) SetExposureTime(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	texp := q.Get("exposureTime")
	d, err := time.ParseDuration(texp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	err = h.Camera.SetExposureTime(d)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	return
}

// GetExposureTime gets the exposure time on a GET request
func (h *HTTPWrapper) GetExposureTime(w http.ResponseWriter, r *http.Request) {
	f, err := h.Camera.GetExposureTime()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	hp := server.HumanPayload{T: types.Float64, Float: f.Seconds()}
	hp.EncodeAndRespond(w, r)
	return
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
// if no exposure time is provided, 100 ms is used
func (h *HTTPWrapper) GetFrame(w http.ResponseWriter, r *http.Request) {
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
		err = h.Camera.SetExposureTime(T)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}
	img, err := h.Camera.GetFrame()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	fmt := q.Get("fmt")
	if fmt == "" {
		fmt = "jpg"
	}
	switch fmt {
	case "jpg":
		buf := make([]byte, len(img))
		for idx := 0; idx < len(img); idx++ {
			buf[idx] = byte(img[idx] / 256) // scale 16 to 8 bits
		}
		im := &image.Gray{Pix: buf, Stride: 2560, Rect: image.Rect(0, 0, 2560, 2160)}
		w.Header().Set("Content-Type", "image/jpeg")
		jpeg.Encode(w, im, nil)
	case "png":
		buf := make([]byte, len(img))
		for idx := 0; idx < len(img); idx++ {
			buf[idx] = byte(img[idx] / 256) // scale 16 to 8 bits
		}
		im := &image.Gray{Pix: buf, Stride: 2560, Rect: image.Rect(0, 0, 2560, 2160)}
		w.Header().Set("Content-Type", "image/png")
		png.Encode(w, im)
	case "fits":
		fits, err := fitsio.Create(w)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer fits.Close()
		im := fitsio.NewImage(16, []int{2560, 2160})
		defer im.Close()
		err = im.Header().Append(
			fitsio.Card{Name: "BZERO", Value: 32768},
			fitsio.Card{Name: "BSCALE", Value: 1.0},
		)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		hdr := w.Header()
		hdr.Set("Content-Type", "image/fits")
		hdr.Set("Content-Disposition", "attachment; filename=image.fits")
		buf := make([]int16, len(img))
		for idx := 0; idx < len(img); idx++ {
			// scale uint16 to int16.  Underflow on uint16 produces the appropriate wrapping for the FITS standard
			buf[idx] = int16(img[idx] - 32768)
		}
		err = im.Write(buf)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		err = fits.Write(im)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

}

// GetCooling gets the cooling status and sends it back as a bool encoded in JSON
func (h *HTTPWrapper) GetCooling(w http.ResponseWriter, r *http.Request) {
	cool, err := h.Camera.GetCooling()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	hp := server.HumanPayload{T: types.Bool, Bool: cool}
	hp.EncodeAndRespond(w, r)
	return
}

// SetCooling sets the cooling status over HTTP
func (h *HTTPWrapper) SetCooling(w http.ResponseWriter, r *http.Request) {
	b := server.BoolT{}
	err := json.NewDecoder(r.Body).Decode(&b)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()
	err = h.Camera.SetCooling(b.Bool)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	return
}

// GetTemperature gets the temperature and sends it over HTTP
func (h *HTTPWrapper) GetTemperature(w http.ResponseWriter, r *http.Request) {
	t, err := h.Camera.GetTemperature()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	hp := server.HumanPayload{T: types.Float64, Float: t}
	hp.EncodeAndRespond(w, r)
	return
}

// GetTemperatureSetpoints gets the current temperature Setpoints
func (h *HTTPWrapper) GetTemperatureSetpoints(w http.ResponseWriter, r *http.Request) {
	opts, err := h.Camera.GetTemperatureSetpoints()
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

// GetTemperatureSetpoint gets the temp setpoint and returns it as JSON
func (h *HTTPWrapper) GetTemperatureSetpoint(w http.ResponseWriter, r *http.Request) {
	setpt, err := h.Camera.GetTemperatureSetpoint()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	hp := server.HumanPayload{T: types.String, String: setpt}
	hp.EncodeAndRespond(w, r)
	return
}

// SetTemperatureSetpoint sets the temp setpoint from JSON
func (h *HTTPWrapper) SetTemperatureSetpoint(w http.ResponseWriter, r *http.Request) {
	str := server.StrT{}
	err := json.NewDecoder(r.Body).Decode(&str)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()
	err = h.Camera.SetTemperatureSetpoint(str.Str)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	return
}

// GetTemperatureStatus gets the current temperature status as a string and returns as JSON
func (h *HTTPWrapper) GetTemperatureStatus(w http.ResponseWriter, r *http.Request) {
	stat, err := h.Camera.GetTemperatureStatus()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	hp := server.HumanPayload{T: types.String, String: stat}
	hp.EncodeAndRespond(w, r)
	return
}

// GetFanOn gets if the fan is currently running over HTTP
func (h *HTTPWrapper) GetFanOn(w http.ResponseWriter, r *http.Request) {
	on, err := h.Camera.GetFanOn()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	hp := server.HumanPayload{T: types.Bool, Bool: on}
	hp.EncodeAndRespond(w, r)
	return
}

// SetFanOn sets the fan operation over HTTP
func (h *HTTPWrapper) SetFanOn(w http.ResponseWriter, r *http.Request) {
	b := server.BoolT{}
	err := json.NewDecoder(r.Body).Decode(&b)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()
	err = h.Camera.SetFanOn(b.Bool)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	return
}

// GetFeatures gets all of the possible features, mapped by their
// type
func (h *HTTPWrapper) GetFeatures(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(Features)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// GetFeature gets a feature, the type of which is determined by the server
func (h *HTTPWrapper) GetFeature(w http.ResponseWriter, r *http.Request) {
	feature := pat.Param(r, "feature")
	typ, known := Features[feature]
	if !known {
		err := ErrFeatureNotFound{Feature: feature}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	switch typ {
	case "command":
		http.Error(w, "cannot get a command feature", http.StatusBadRequest)
		return
	case "int":
		i, err := GetInt(h.Camera.Handle, feature)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		hp := server.HumanPayload{T: types.Int, Int: i}
		hp.EncodeAndRespond(w, r)
	case "float":
		f, err := GetFloat(h.Camera.Handle, feature)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		hp := server.HumanPayload{T: types.Float64, Float: f}
		hp.EncodeAndRespond(w, r)
	case "bool":
		b, err := GetBool(h.Camera.Handle, feature)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		hp := server.HumanPayload{T: types.Bool, Bool: b}
		hp.EncodeAndRespond(w, r)
	case "enum", "string":
		var (
			str string
			err error
		)
		if typ == "enum" {
			str, err = GetEnumString(h.Camera.Handle, feature)
		} else {
			str, err = GetString(h.Camera.Handle, feature)
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		hp := server.HumanPayload{T: types.String, String: str}
		hp.EncodeAndRespond(w, r)
	}
}

// SetFeature sets a feature, the type of which is determined by the setup
func (h *HTTPWrapper) SetFeature(w http.ResponseWriter, r *http.Request) {
	// the contents of this is basically identical to GetFeature
	// but with json unmarshalling logic injected
	feature := pat.Param(r, "feature")
	typ, known := Features[feature]
	if !known {
		err := ErrFeatureNotFound{Feature: feature}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	switch typ {
	case "command":
		http.Error(w, "cannot set a command feature", http.StatusBadRequest)
		return
	case "int":
		i := server.IntT{}
		err := json.NewDecoder(r.Body).Decode(&i)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		defer r.Body.Close()
		err = SetInt(h.Camera.Handle, feature, int64(i.Int))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
		return
	case "float":
		f := server.FloatT{}
		err := json.NewDecoder(r.Body).Decode(&f)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		defer r.Body.Close()
		err = SetFloat(h.Camera.Handle, feature, f.F64)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
		return
	case "bool":
		b := server.BoolT{}
		err := json.NewDecoder(r.Body).Decode(&b)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		defer r.Body.Close()
		err = SetBool(h.Camera.Handle, feature, b.Bool)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
		return
	case "enum", "string":
		s := server.StrT{}
		err := json.NewDecoder(r.Body).Decode(&s)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		defer r.Body.Close()
		if typ == "enum" {
			err = SetEnumString(h.Camera.Handle, feature, s.Str)
		} else {
			err = SetString(h.Camera.Handle, feature, s.Str)
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
		return
	}
}
