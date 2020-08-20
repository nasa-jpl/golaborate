package sdk3

import (
	"encoding/json"
	"go/types"
	"net/http"
	"strings"

	"github.com/go-chi/chi"
	"github.jpl.nasa.gov/bdube/golab/generichttp"
	"github.jpl.nasa.gov/bdube/golab/generichttp/camera"
	"github.jpl.nasa.gov/bdube/golab/imgrec"
	"github.jpl.nasa.gov/bdube/golab/util"
)

// HTTPWrapper provides an HTTP interface to a camera
type HTTPWrapper struct {
	// Camera is the camera object being wrapped
	*Camera

	w camera.HTTPCamera

	recorder *imgrec.Recorder
}

// NewHTTPWrapper returns a new wrapper with the route table populated
func NewHTTPWrapper(c *Camera, r *imgrec.Recorder) HTTPWrapper {
	g := camera.NewHTTPCamera(c, r)
	w := HTTPWrapper{Camera: c, w: g, recorder: r}

	rt := g.RT()
	// things not part of the generic wrapper (yet?)
	rt[generichttp.MethodPath{http.MethodGet, "/feature"}] = w.GetFeatures
	rt[generichttp.MethodPath{http.MethodGet, "/feature/:feature"}] = w.GetFeature
	rt[generichttp.MethodPath{http.MethodGet, "/feature/{feature}/options"}] = w.GetFeatureInfo
	rt[generichttp.MethodPath{http.MethodPost, "/feature/{feature}"}] = w.SetFeature
	w2 := imgrec.NewHTTPWrapper(r)
	w2.Inject(w)
	return w
}

// RT yields the route table and implements the generichttp.HTTPer interface
func (h HTTPWrapper) RT() generichttp.RouteTable {
	return h.w.RT()
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
	feature := chi.URLParam(r, "feature")
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
		hp := generichttp.HumanPayload{T: types.Int, Int: i}
		hp.EncodeAndRespond(w, r)
	case "float":
		f, err := GetFloat(h.Camera.Handle, feature)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		hp := generichttp.HumanPayload{T: types.Float64, Float: f}
		hp.EncodeAndRespond(w, r)
	case "bool":
		b, err := GetBool(h.Camera.Handle, feature)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		hp := generichttp.HumanPayload{T: types.Bool, Bool: b}
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
		hp := generichttp.HumanPayload{T: types.String, String: str}
		hp.EncodeAndRespond(w, r)
	}
}

// GetFeatureInfo gets a feature's type and options.
// For numerical features, it returns the min and max values.  For enum
// features, it returns the possible strings that can be used
func (h *HTTPWrapper) GetFeatureInfo(w http.ResponseWriter, r *http.Request) {
	feature := chi.URLParam(r, "feature")
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
	feature := chi.URLParam(r, "feature")
	typ, known := Features[feature]
	if !known {
		err := ErrFeatureNotFound{Feature: feature}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	switch feature {
	case "ExposureTime":
		f := generichttp.FloatT{}
		err := json.NewDecoder(r.Body).Decode(&f)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		defer r.Body.Close()
		err = h.Camera.SetExposureTime(util.SecsToDuration(f.F64))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	default:
		switch typ {
		case "command":
			http.Error(w, "cannot set a command feature", http.StatusBadRequest)
			return
		case "int":
			i := generichttp.IntT{}
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
			f := generichttp.FloatT{}
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
			b := generichttp.BoolT{}
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
			s := generichttp.StrT{}
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

	if strings.Contains(feature, "AOI") {
		h.Camera.Allocate()
	}

}
