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

	"github.jpl.nasa.gov/HCIT/go-hcit/mathx"
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
		pat.Get("/burst"): w.Burst,

		// exposure manipulation
		pat.Get("/exposure-time"):  w.GetExposureTime,
		pat.Post("/exposure-time"): w.SetExposureTime,

		// thermals
		pat.Get("/fan"):                          w.GetFan,
		pat.Post("/fan"):                         w.SetFan,
		pat.Get("/sensor-cooling"):               w.GetCooling,
		pat.Post("/sensor-cooling"):              w.SetCooling,
		pat.Get("/temperature"):                  w.GetTemperature,
		pat.Get("/temperature-setpoint-options"): w.GetTemperatureSetpoints,
		pat.Get("/temperature-setpoint"):         w.GetTemperatureSetpoint,
		pat.Post("/temperature-setpoint"):        w.SetTemperatureSetpoint,
		pat.Get("/temperature-status"):           w.GetTemperatureStatus,

		// generic
		pat.Get("/feature"):                  w.GetFeatures,
		pat.Get("/feature/:feature"):         w.GetFeature,
		pat.Get("/feature/:feature/options"): w.GetFeatureInfo,
		pat.Post("/feature/:feature"):        w.SetFeature,

		// AOI
		pat.Get("/aoi"):  w.GetAOI,
		pat.Post("/aoi"): w.SetAOI,
	}
	return w
}

// RT yields the route table and implements the server.HTTPer interface
func (h HTTPWrapper) RT() server.RouteTable {
	return h.RouteTable
}

// SetExposureTime sets the exposure time on a POST request.
// it can be provided either as a query parameter exposureTime, formatted in a
// way that is parseable by golang/time.ParseDuration, or a json payload with
// key f64, holding the exposure time in seconds.
func (h *HTTPWrapper) SetExposureTime(w http.ResponseWriter, r *http.Request) {
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
// if no exposure time is provided, it is not updated and the existing value is used.
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
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	img, err := h.Camera.GetFrame()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	format := q.Get("fmt")
	if format == "" {
		format = "jpg"
	}

	aoi, err := h.Camera.GetAOI()
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
		cards := collectHeaderMetadata3(h.Camera)
		hdr := w.Header()
		hdr.Set("Content-Type", "image/fits")
		hdr.Set("Content-Disposition", "attachment; filename=image.fits")
		w.WriteHeader(http.StatusOK)
		err = writeFits(w, cards, img, aoi.Width, aoi.Height, 1)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

}

// Burst takes a burst of N frames at M fps and returns it as a fits image cube
func (h *HTTPWrapper) Burst(w http.ResponseWriter, r *http.Request) {
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
	img, err := h.Camera.Burst(t.Frames, t.FPS)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	aoi, err := h.Camera.GetAOI()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	cards := collectHeaderMetadata3(h.Camera)
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

// GetFan gets if the fan is currently running over HTTP
func (h *HTTPWrapper) GetFan(w http.ResponseWriter, r *http.Request) {
	on, err := h.Camera.GetFan()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	hp := server.HumanPayload{T: types.Bool, Bool: on}
	hp.EncodeAndRespond(w, r)
	return
}

// SetFan sets the fan operation over HTTP
func (h *HTTPWrapper) SetFan(w http.ResponseWriter, r *http.Request) {
	b := server.BoolT{}
	err := json.NewDecoder(r.Body).Decode(&b)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()
	err = h.Camera.SetFan(b.Bool)
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
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
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
		err := h.Camera.Command(feature)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
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

// GetFeatureInfo gets a feature's type and options.
// For numerical features, it returns the min and max values.  For enum
// features, it returns the possible strings that can be used
func (h *HTTPWrapper) GetFeatureInfo(w http.ResponseWriter, r *http.Request) {
	feature := pat.Param(r, "feature")
	typ, known := Features[feature]
	if !known {
		err := ErrFeatureNotFound{Feature: feature}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	ret := map[string]interface{}{"type": typ}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	switch typ {
	case "command", "bool":
		err := json.NewEncoder(w).Encode(ret)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
		return
	case "int", "float":
		var err error
		if typ == "int" {
			var min, max int
			min, err = GetIntMin(h.Camera.Handle, feature)
			max, err = GetIntMax(h.Camera.Handle, feature)
			ret["min"] = min
			ret["max"] = max
		} else {
			var min, max float64
			min, err = GetFloatMin(h.Camera.Handle, feature)
			max, err = GetFloatMax(h.Camera.Handle, feature)
			ret["min"] = min
			ret["max"] = max
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		err = json.NewEncoder(w).Encode(ret)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

	case "enum":
		opts, err := GetEnumStrings(h.Camera.Handle, feature)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		ret["options"] = opts
		err = json.NewEncoder(w).Encode(ret)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	case "string":
		maxlen, err := GetStringMaxLength(h.Camera.Handle, feature)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		ret["maxLength"] = maxlen
		err = json.NewEncoder(w).Encode(ret)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
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
	switch feature {
	case "ExposureTime":
		f := server.FloatT{}
		err := json.NewDecoder(r.Body).Decode(&f)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		defer r.Body.Close()
		tNs := time.Duration(int(mathx.Round(f.F64*1e9, 1))) * time.Nanosecond
		err = h.Camera.SetExposureTime(tNs)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	case "AOIWidth", "AOIHeight", "AOITop", "AOILeft":
		// get the parameter from the client and create the struct
		i := server.IntT{}
		err := json.NewDecoder(r.Body).Decode(&i)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer r.Body.Close()
		aoi := AOI{}
		switch feature {
		case "AOIWidth":
			aoi.Width = i.Int
		case "AOIHeight":
			aoi.Height = i.Int
		case "AOILeft":
			aoi.Left = i.Int
		case "AOITop":
			aoi.Top = i.Int
		}
		err = h.Camera.SetAOI(aoi)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	case "AOIBinning":
		s := server.StrT{}
		err := json.NewDecoder(r.Body).Decode(&s)
		defer r.Body.Close()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		b := ParseBinning(s.Str)
		err = h.Camera.SetBinning(b)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		return
	default:
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

}

// GetAOI gets the AOI and returns it as json over the wire
func (h *HTTPWrapper) GetAOI(w http.ResponseWriter, r *http.Request) {
	aoi, err := h.Camera.GetAOI()
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

// SetAOI sets the AOI over HTTP via json
func (h *HTTPWrapper) SetAOI(w http.ResponseWriter, r *http.Request) {
	aoi := AOI{}
	err := json.NewDecoder(r.Body).Decode(&aoi)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	err = h.Camera.SetAOI(aoi)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusOK)
	return
}
