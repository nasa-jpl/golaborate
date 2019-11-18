package ixllightwave

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.jpl.nasa.gov/HCIT/go-hcit/server"
	"github.jpl.nasa.gov/HCIT/go-hcit/util"
	"goji.io"
	"goji.io/pat"
)

// HTTPWrapper provides HTTP bindings on top of the underlying Go interface
// BindRoutes must be called on it
type HTTPWrapper struct {
	// Sensor is the underlying sensor that is wrapped
	LDC *LDC3916

	// RouteTable maps goji patterns to http handlers
	RouteTable map[goji.Pattern]http.HandlerFunc
}

// NewHTTPWrapper returns a new HTTP wrapper with the route table pre-configured
func NewHTTPWrapper(urlStem string, ldc *LDC3916) HTTPWrapper {
	w := HTTPWrapper{LDC: ldc}
	rt := map[goji.Pattern]http.HandlerFunc{
		// channel
		pat.Get(urlStem + "chan"):  w.GetChan,
		pat.Post(urlStem + "chan"): w.SetChan,

		// tec
		pat.Get(urlStem + "temperature-control"):  w.GetTempControl,
		pat.Post(urlStem + "temperature-control"): w.SetTempControl,

		// laser output
		pat.Get(urlStem + "laser-output"):  w.GetLaserOutput,
		pat.Post(urlStem + "laser-output"): w.SetLaserOutput,

		// laser current
		pat.Get(urlStem + "laser-current"):  w.GetLaserCurrent,
		pat.Post(urlStem + "laser-current"): w.SetLaserCurrent,

		// raw
		pat.Post(urlStem + "raw"): w.Raw,
	}
	w.RouteTable = rt
	return w
}

// GetChan gets the currently active channel(s) over HTTP
func (h *HTTPWrapper) GetChan(w http.ResponseWriter, r *http.Request) {
	cmd := "chan"
	typ := "[]int"
	resp, err := h.LDC.processCommand(cmd, true, "")
	httpResponder(resp, typ, err)(w, r)
	return
}

// SetChan sets the currently active channel(s) over HTTP
func (h *HTTPWrapper) SetChan(w http.ResponseWriter, r *http.Request) {
	cmd := "chan"
	typ := "[]int"
	resp, err := h.LDC.processCommand(cmd, true, "")
	obj := struct {
		Ints []int `json:"ints"`
	}{}
	err = json.NewDecoder(r.Body).Decode(&obj)
	defer r.Body.Close()
	if err != nil {
		fstr := fmt.Sprintf("unable to decode int array from query.  Query must be a JSON request with \"ints\" field.  For a single channel, use a length-1 array.  %q", err)
		log.Println(fstr)
		http.Error(w, fstr, http.StatusBadRequest)
		return
	}
	// we got the channel from the request, now set it on the device
	resp, err = h.LDC.processCommand(cmd, false, util.IntSliceToCSV(obj.Ints))
	httpResponder(resp, typ, err)(w, r)
	return
}

// GetTempControl gets if TEC is currently active over HTTP
func (h *HTTPWrapper) GetTempControl(w http.ResponseWriter, r *http.Request) {
	cmd := "temperature-control"
	typ := "bool"
	resp, err := h.LDC.processCommand(cmd, true, "")
	httpResponder(resp, typ, err)(w, r)
	return
}

// SetTempControl sets if TEC is currently active over HTTP
func (h *HTTPWrapper) SetTempControl(w http.ResponseWriter, r *http.Request) {
	cmd := "temperature-control"
	typ := "bool"
	boo := server.BoolT{}
	err := json.NewDecoder(r.Body).Decode(&boo)
	defer r.Body.Close()
	if err != nil {
		fstr := fmt.Sprintf("unable to decode boolean from query.  Query must be a JSON request with \"bool\" field.  %q", err)
		log.Println(fstr)
		http.Error(w, fstr, http.StatusBadRequest)
		return
	}
	resp, err := h.LDC.processCommand(cmd, false, boolToString(boo.Bool))
	httpResponder(resp, typ, err)(w, r)
	return
}

// GetLaserOutput gets the laser output (on/off) over HTTP
func (h *HTTPWrapper) GetLaserOutput(w http.ResponseWriter, r *http.Request) {
	cmd := "laser-output"
	typ := "bool"
	resp, err := h.LDC.processCommand(cmd, true, "")
	httpResponder(resp, typ, err)(w, r)
	return
}

// SetLaserOutput set the laser output (on/off) over HTTP
func (h *HTTPWrapper) SetLaserOutput(w http.ResponseWriter, r *http.Request) {
	cmd := "laser-output"
	typ := "bool"
	boo := server.BoolT{}
	err := json.NewDecoder(r.Body).Decode(&boo)
	defer r.Body.Close()
	if err != nil {
		fstr := fmt.Sprintf("unable to decode boolean from query.  Query must be a JSON request with \"bool\" field.  %q", err)
		log.Println(fstr)
		http.Error(w, fstr, http.StatusBadRequest)
		return
	}
	resp, err := h.LDC.processCommand(cmd, false, boolToString(boo.Bool))
	httpResponder(resp, typ, err)(w, r)
	return
}

// GetLaserCurrent gets the laser current (mA) over HTTP
func (h *HTTPWrapper) GetLaserCurrent(w http.ResponseWriter, r *http.Request) {
	cmd := "laser-current"
	typ := "float"

	resp, err := h.LDC.processCommand(cmd, true, "")
	httpResponder(resp, typ, err)(w, r)
	return
}

// SetLaserCurrent set the laser current (mA) over HTTP
func (h *HTTPWrapper) SetLaserCurrent(w http.ResponseWriter, r *http.Request) {
	cmd := "laser-current"
	typ := "float"
	float := server.FloatT{}
	err := json.NewDecoder(r.Body).Decode(&float)
	defer r.Body.Close()
	if err != nil {
		fstr := fmt.Sprintf("unable to decode boolean from query.  Query must be a JSON request with \"bool\" field.  %q", err)
		log.Println(fstr)
		http.Error(w, fstr, http.StatusBadRequest)
		return
	}
	resp, err := h.LDC.processCommand(cmd, false, floatToString(float.F64))
	httpResponder(resp, typ, err)(w, r)
	return
}

// Raw sends a 'raw' command to the LDC and returns the raw response as a string
func (h *HTTPWrapper) Raw(w http.ResponseWriter, r *http.Request) {
	str := server.StrT{}
	json.NewDecoder(r.Body).Decode(&str)
	defer r.Body.Close()
	resp, err := h.LDC.processCommand(str.Str, true, "")
	httpResponder(resp, "string", err)(w, r)
	return
}

func httpResponder(data string, typ string, err error) http.HandlerFunc {
	// this function is fragile because we encode the type in a string instead
	// of using, say, types.BasicKind.  We do so because we need chan []int
	// and int slices are not a basic type
	if data == "" {
		return func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}
	}
	var ret interface{}
	switch typ {
	case "[]int":
		s := strings.Split(data, ",")
		ints := make([]int, len(s))
		for idx, str := range s {
			v, err := strconv.Atoi(str)
			if err != nil {
				log.Fatal(err)
			}
			ints[idx] = v
		}
		if len(ints) == 1 {
			intt := ints[0]
			ret = struct {
				Int int `json:"int"`
			}{intt}
		} else {
			ret = struct {
				Int []int `json:"int"`
			}{ints}
		}
	case "bool":
		b := stringToBool(data)
		ret = struct {
			Bool bool `json:"bool"`
		}{b}
	case "float":
		f, err := strconv.ParseFloat(data, 64)
		if err != nil {
			return func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		}
		ret = struct {
			F64 float64 `json:"f64"`
		}{f}
	default:
		ret = struct {
			Str string `json:"str"`
		}{data}
	}
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Printf("%+v\n", ret)
		err := json.NewEncoder(w).Encode(ret)
		if err != nil {
			fstr := fmt.Sprintf("error encoding data to json state %q", err)
			log.Println(fstr)
			http.Error(w, fstr, http.StatusInternalServerError)
		}
		return
	}
}
