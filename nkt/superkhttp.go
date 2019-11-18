package nkt

import (
	"encoding/json"
	"fmt"
	"go/types"
	"net/http"

	"github.jpl.nasa.gov/HCIT/go-hcit/mathx"
	"github.jpl.nasa.gov/HCIT/go-hcit/server"
	"goji.io"
	"goji.io/pat"
)

// SuperKHTTPWrapper wraps SuperK lasers in an HTTP interface
type SuperKHTTPWrapper struct {
	// Extreme is the Extreme main module
	Extreme *SuperKExtreme

	// Varia is the Varia variable filter module
	Varia *SuperKVaria

	// RouteTable holds the map of patterns and routes
	RouteTable map[goji.Pattern]http.HandlerFunc
}

// NewHTTPWrapper creates a new HTTP wrapper and populates the route table
func NewHTTPWrapper(urlStem string, extr *SuperKExtreme, varia *SuperKVaria) SuperKHTTPWrapper {
	w := SuperKHTTPWrapper{Extreme: extr, Varia: varia}
	rt := map[goji.Pattern]http.HandlerFunc{
		pat.Get(urlStem + "emission"):           w.GetEmission,
		pat.Get(urlStem + "emission/on"):        w.EmissionOn,
		pat.Get(urlStem + "emission/off"):       w.EmissionOff,
		pat.Get(urlStem + "power"):              w.GetPower,
		pat.Post(urlStem + "power"):             w.SetPower,
		pat.Get(urlStem + "main-module-status"): w.StatusMain,

		pat.Get(urlStem + "wl-short"):             w.GetShortWave,
		pat.Post(urlStem + "wl-short"):            w.SetShortWave,
		pat.Get(urlStem + "wl-long"):              w.GetLongWave,
		pat.Post(urlStem + "wl-long"):             w.GetShortWave,
		pat.Get(urlStem + "wl-center-bandwidth"):  w.GetCenterBandwidth,
		pat.Post(urlStem + "wl-center-bandwidth"): w.SetCenterBandwidth,
		pat.Get(urlStem + "nd"):                   w.GetND,
		pat.Post(urlStem + "nd"):                  w.SetND,
		pat.Get(urlStem + "varia-status"):         w.StatusVaria,
	}
	w.RouteTable = rt
	return w
}

// GetEmission gets the emission state and pipes it back as a bool json
func (h *SuperKHTTPWrapper) GetEmission(w http.ResponseWriter, r *http.Request) {
	mp, err := h.Extreme.GetValue("Emission")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	b := server.BoolT{Bool: mp.Data[0] > byte(0)}
	err = json.NewEncoder(w).Encode(b)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	return
}

// EmissionOn responds to an HTTP request by turning on the laser
func (h *SuperKHTTPWrapper) EmissionOn(w http.ResponseWriter, r *http.Request) {
	_, err := h.Extreme.SetValue("Emission", []byte{3}) // 3 turns the laser on, not 1 or 2
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	return
}

// EmissionOff responds to an HTTP request by turning off the laser
func (h *SuperKHTTPWrapper) EmissionOff(w http.ResponseWriter, r *http.Request) {
	_, err := h.Extreme.SetValue("Emission", []byte{0})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	return
}

// GetPower returns the power level (in pct, [0,100]) over HTTP
func (h *SuperKHTTPWrapper) GetPower(w http.ResponseWriter, r *http.Request) {
	httpGetFloatValue(w, r, "Power Level", h.Extreme.Module)
}

// SetPower sets the power level (in pct, [0,100]) over HTTP
func (h *SuperKHTTPWrapper) SetPower(w http.ResponseWriter, r *http.Request) {
	httpSetFloatValue(w, r, "Power Level", h.Extreme.Module)
}

// GetShortWave gets the short wavelength over HTTP
func (h *SuperKHTTPWrapper) GetShortWave(w http.ResponseWriter, r *http.Request) {
	httpGetFloatValue(w, r, "Short Wave Setpoint", h.Varia.Module)
}

// SetShortWave sets the short wavelength over HTTP
func (h *SuperKHTTPWrapper) SetShortWave(w http.ResponseWriter, r *http.Request) {
	httpSetFloatValue(w, r, "Short Wave Setpoint", h.Varia.Module)
}

// GetLongWave gets the long wavelength over HTTP
func (h *SuperKHTTPWrapper) GetLongWave(w http.ResponseWriter, r *http.Request) {
	httpGetFloatValue(w, r, "Long Wave Setpoint", h.Varia.Module)
}

// SetLongWave sets the long wavelength over HTTP
func (h *SuperKHTTPWrapper) SetLongWave(w http.ResponseWriter, r *http.Request) {
	httpSetFloatValue(w, r, "Long Wave Setpoint", h.Varia.Module)
}

// GetCenterBandwidth gets the center wavelength and bandwidth over HTTP
func (h *SuperKHTTPWrapper) GetCenterBandwidth(w http.ResponseWriter, r *http.Request) {
	mps, err := h.Varia.GetValueMulti([]string{"Short Wave Setpoint", "Long Wave Setpoint"})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	low := float64(dataOrder.Uint16(mps[0].Data)) / 10
	high := float64(dataOrder.Uint16(mps[1].Data)) / 10
	cbw := ShortLongToCB(low, high)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	err = json.NewEncoder(w).Encode(cbw)
	if err != nil {
		fstr := fmt.Sprintf("Error encoding struct to json %q", err)
		http.Error(w, fstr, http.StatusInternalServerError)
	}
	return
}

// SetCenterBandwidth gets the center wavelength and bandwidth over HTTP
func (h *SuperKHTTPWrapper) SetCenterBandwidth(w http.ResponseWriter, r *http.Request) {
	cbw := CenterBandwidth{}
	err := json.NewDecoder(r.Body).Decode(&cbw)
	defer r.Body.Close()
	if err != nil {
		fstr := fmt.Sprintf("error decoding json, should have fields of \"center\" and \"bandwidth\", %q", err)
		http.Error(w, fstr, http.StatusBadRequest)
		return
	}
	low, high := cbw.ToShortLong()
	addrs := []string{"Short Wave Setpoint", "Long Wave Setpoint"}
	l := len(addrs)
	datas := make([][]byte, l, l)
	for idx, wav := range []float64{low, high} {
		f := mathx.Round(wav*10, 1)
		buf := make([]byte, 2)
		dataOrder.PutUint16(buf, uint16(f))
		datas[idx] = buf
	}
	_, err = h.Varia.SetValueMulti(addrs, datas)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// GetND gets the ND filter strength on GET
// POST should be JSON with single f64 field which is the ND strength in pct (100 = full blockage).
func (h *SuperKHTTPWrapper) GetND(w http.ResponseWriter, r *http.Request) {
	httpGetFloatValue(w, r, "ND Setpoint", h.Varia.Module)
}

// SetND sets the ND filter strength on POST.
// post payload should be JSON with single f64 field which is the ND strength in pct (100 = full blockage).
func (h *SuperKHTTPWrapper) SetND(w http.ResponseWriter, r *http.Request) {
	httpSetFloatValue(w, r, "ND Setpoint", h.Varia.Module)
}

// StatusMain gets the status bitfield of the main module over HTTP
func (h *SuperKHTTPWrapper) StatusMain(w http.ResponseWriter, r *http.Request) {
	bitmap, err := h.Extreme.Module.GetStatus()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	err = json.NewEncoder(w).Encode(bitmap)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	return
}

// StatusVaria gets the status bitfield of the Varia module over HTTP
func (h *SuperKHTTPWrapper) StatusVaria(w http.ResponseWriter, r *http.Request) {
	bitmap, err := h.Varia.Module.GetStatus()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	err = json.NewEncoder(w).Encode(bitmap)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	return
}

// httpGetFloatValue gets a float over HTTP
func httpGetFloatValue(w http.ResponseWriter, r *http.Request, value string, mod Module) {
	mp, err := mod.GetValue(value)
	if err != nil {
		fstr := fmt.Sprintf("Error getting %s, %q", value, err)
		http.Error(w, fstr, http.StatusInternalServerError)
		return
	}
	// if there is not an error, the message is well-formed and we have a Datagram
	wvl := float64(dataOrder.Uint16(mp.Data)) / 10
	hp := server.HumanPayload{Float: wvl, T: types.Float64}
	hp.EncodeAndRespond(w, r)
	return
}

// httpGetFloatValue gets a float over HTTP
func httpSetFloatValue(w http.ResponseWriter, r *http.Request, value string, mod Module) {
	vT := server.FloatT{}
	err := json.NewDecoder(r.Body).Decode(&vT)
	defer r.Body.Close()
	if err != nil {
		fstr := fmt.Sprintf("error decoding json, should have field \"f64\", %q", err)
		http.Error(w, fstr, http.StatusBadRequest)
		return
	}
	intt := uint16(mathx.Round(vT.F64*10, 1))
	buf := make([]byte, 2, 2)
	dataOrder.PutUint16(buf, intt)
	_, err = mod.SetValue(value, buf)
	if err != nil {
		fstr := fmt.Sprintf("Erorr getting %s, %q", value, err)
		http.Error(w, fstr, http.StatusInternalServerError)
	}
	return
}
