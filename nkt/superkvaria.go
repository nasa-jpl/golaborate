package nkt

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"

	"github.jpl.nasa.gov/HCIT/go-hcit/comm"
	"github.jpl.nasa.gov/HCIT/go-hcit/mathx"
	"github.jpl.nasa.gov/HCIT/go-hcit/server"
)

// this file contains values relevant to the SuperK Varia accessory

const (
	variaDefaultAddr = 0x10
)

var (
	// SuperKVariaInfo describes the SuperK Varia module
	SuperKVariaInfo = &ModuleInformation{
		Addresses: map[string]byte{
			"Input":               0x13,
			"ND Setpoint":         0x32,
			"Short Wave Setpoint": 0x33,
			"Long Wave Setpoint":  0x34,
			"Status":              0x66},
		CodeBanks: map[string]map[int]string{
			"Status": map[int]string{
				0:  "-",
				1:  "Interlock off",
				2:  "Interlock loop in",
				3:  "Interlock loop out",
				4:  "-",
				5:  "Supply voltage low",
				6:  "-",
				7:  "-",
				8:  "Shutter sensor 1",
				9:  "Shutter sensor 2",
				10: "-",
				11: "-",
				12: "Filter 1 moving",
				13: "Filter 2 moving",
				14: "Filter 3 moving",
				15: "Error code present",
			}}}
)

// CenterBandwidth is a struct holding the center wavelength (nm) and full bandwidth (nm) of a VARIA
type CenterBandwidth struct {
	Center    float64 `json:"center"`
	Bandwidth float64 `json:"bandwidth"`
}

// ShortLongToCB converts short, long wavelengths to a CenterBandwidth struct
func ShortLongToCB(short, long float64) CenterBandwidth {
	center := (short + long) / 2
	bw := math.Abs(long - short)
	return CenterBandwidth{Center: center, Bandwidth: bw}
}

// ToShortLong converts a CenterBandwidth to (short, long)
func (cb CenterBandwidth) ToShortLong() (float64, float64) {
	hb := cb.Bandwidth / 2
	low := cb.Center - hb
	high := cb.Center + hb
	return high, low
}

// SuperKVaria embeds Module and has some quick usage methods
type SuperKVaria struct {
	Module
}

// HTTPShortWave gets the short wavelength on a GET request, or sets it on a POST request.
// POST should be JSON with a single f64 field.
func (sk *SuperKVaria) HTTPShortWave(w http.ResponseWriter, r *http.Request) {
	sk.httpFloatValue(w, r, "Short Wave Setpoint")
}

// HTTPLongWave gets the long wavelength on a GET request, or sets it on a POST request.
// POST should be JSON with a single f64 field.
func (sk *SuperKVaria) HTTPLongWave(w http.ResponseWriter, r *http.Request) {
	sk.httpFloatValue(w, r, "Long Wave Setpoint")
}

// HTTPCenterBandwidth gets the center wavelength and Bandwidth in nm on a GET request, or sets it on a POST request.
// POST should be JSON with two fields, "center", and "bandwidth"
func (sk *SuperKVaria) HTTPCenterBandwidth(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		mps, err := sk.GetValueMulti([]string{"Short Wave Setpoint", "Long Wave Setpoint"})
		if err != nil {
			log.Println(err)
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
			log.Println(fstr)
			http.Error(w, fstr, http.StatusInternalServerError)
		}
		log.Printf("%q got center wavelength, NKT %s, is %+v", r.RemoteAddr, sk.Addr, cbw)
		return
	case http.MethodPost:
		cbw := CenterBandwidth{}
		err := json.NewDecoder(r.Body).Decode(&cbw)
		defer r.Body.Close()
		if err != nil {
			fstr := fmt.Sprintf("error decoding json, should have fields of \"center\" and \"bandwidth\", %q", err)
			log.Println(fstr)
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
		_, err = sk.SetValueMulti(addrs, datas)
		if err != nil {
			log.Println(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		log.Printf("%q set center wavelength, NKT %s, is %+v", r.RemoteAddr, sk.Addr, cbw)
	default:
		server.BadMethod(w, r)
	}
	return
}

// HTTPND gets the ND filter strength on GET, or sets it on POST.
// POST should be JSON with single f64 field which is the ND strength in pct (100 = full blockage).
func (sk *SuperKVaria) HTTPND(w http.ResponseWriter, r *http.Request) {
	sk.httpFloatValue(w, r, "ND Setpoint")
}

// NewSuperKVaria create a new Module representing a SuperKVaria module
func NewSuperKVaria(addr, urlStem string, serial bool) *SuperKVaria {
	rd := comm.NewRemoteDevice(addr, serial, &comm.Terminators{Rx: telEnd, Tx: telEnd}, nil)
	srv := server.NewServer(urlStem)
	sk := SuperKVaria{Module{
		RemoteDevice: &rd,
		AddrDev:      variaDefaultAddr,
		Info:         SuperKVariaInfo}}
	srv.RouteTable["wl-short"] = sk.HTTPShortWave
	srv.RouteTable["wl-long"] = sk.HTTPLongWave
	srv.RouteTable["wl-center-bandwidth"] = sk.HTTPCenterBandwidth
	srv.RouteTable["nd"] = sk.HTTPND
	srv.RouteTable["varia-status"] = sk.HTTPStatus
	sk.Module.Server = srv
	return &sk
}
