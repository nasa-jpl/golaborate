package nkt

import (
	"encoding/json"
	"log"
	"net/http"

	"github.jpl.nasa.gov/HCIT/go-hcit/comm"
	"github.jpl.nasa.gov/HCIT/go-hcit/server"
)

// this file contains values relevant to the SuperK Extreme modules

const (
	extremeDefaultAddr        = 0x0F
	extremeFrontDefaultAddr   = 0x01
	extremeBoosterDefaultAddr = 0x65
)

var (
	// SuperKExtremeMain describes the SuperK Extreme Main module
	SuperKExtremeMain = &ModuleInformation{
		Addresses: map[string]byte{
			"Inlet Temperature":  0x11,
			"Emission":           0x30,
			"Setup":              0x31,
			"Interlock":          0x32,
			"Pulse Picker Ratio": 0x34,
			"Watchdog Interval":  0x36,
			"Power Level":        0x37,
			"Current Level":      0x38,
			"NIM Delay":          0x39,
			"Status":             0x66,
			"User Text":          0x6C},
		CodeBanks: map[string]map[int]string{
			"Setup": map[int]string{
				0: "Constant current mode",
				1: "Constant power mode",
				2: "Externally modulated current mode",
				3: "Externally modulated power",
				4: "External feedback mode (Power Lock)"},
			"Status": map[int]string{
				0:  "Emission on",
				1:  "Interlock relays off",
				2:  "Interlock supply voltage low (possible short circuit)",
				3:  "Interlock loop open",
				4:  "Output control signal low",
				5:  "Supply voltage low",
				6:  "Inlet temperature out of range",
				7:  "Clock battery low voltage",
				8:  "-",
				9:  "-",
				10: "-",
				11: "-",
				12: "-",
				13: "CRC error on startup (possible module address conflict)",
				14: "Log error code present",
				15: "System error code present",
			}}}

	// SuperKFrontDisplay describes the SuperK front display module
	SuperKFrontDisplay = ModuleInformation{
		Addresses: map[string]byte{
			"Panel Lock":  0x3D,
			"Text":        0x72,
			"Error Flash": 0xBD}}

	// SuperKBooster describes the SuperK Booster module
	SuperKBooster = ModuleInformation{
		Addresses: map[string]byte{
			"Module":           0x05,
			"Emission Runtime": 0x80,
			"Status":           0x66},
		CodeBanks: map[string]map[int]string{
			"Status": map[int]string{
				0: "Emission on",
				1: "Interlock signal off",
				2: "Interlock loop input low",
				3: "Interlock loop output low",
				4: "Module disabled",
				5: "Supply voltage out of range",
				6: "Board temperature out of range",
				7: "Heat sink temperature out of range",
			}}}
)

// SuperKExtreme embeds Module and has some quick usage methods
type SuperKExtreme struct {
	Module
}

// HTTPEmissionGet gets the emission state and pipes it back as a bool json
func (sk *SuperKExtreme) HTTPEmissionGet(w http.ResponseWriter, r *http.Request) {
	mp, err := sk.GetValue("Emission")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	b := server.BoolT{Bool: mp.Data[0] == byte(1)}
	err = json.NewEncoder(w).Encode(b)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	log.Printf("%s got emission state NKT %s, %t\n", r.RemoteAddr, sk.Addr, b.Bool)
	return
}

// HTTPEmissionOn responds to an HTTP request by turning on the laser
func (sk *SuperKExtreme) HTTPEmissionOn(w http.ResponseWriter, r *http.Request) {
	_, err := sk.SetValue("Emission", []byte{3}) // 3 turns the laser on, not 1 or 2
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	log.Printf("%s set emission state NKT %s, true\n", r.RemoteAddr, sk.Addr)
	return
}

// HTTPEmissionOff responds to an HTTP request by turning off the laser
func (sk *SuperKExtreme) HTTPEmissionOff(w http.ResponseWriter, r *http.Request) {
	_, err := sk.SetValue("Emission", []byte{0})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	log.Printf("%s set emission state NKT %s, false\n", r.RemoteAddr, sk.Addr)
	return
}

// HTTPPower responds to HTTP requests by getting or setting the power level in percent
func (sk *SuperKExtreme) HTTPPower(w http.ResponseWriter, r *http.Request) {
	sk.httpFloatValue(w, r, "Power Level")
}

// NewSuperKExtreme create a new Module representing a SuperKExtreme's main module
func NewSuperKExtreme(addr, urlStem string, serial bool) *SuperKExtreme {
	rd := comm.NewRemoteDevice(addr, serial, &comm.Terminators{Rx: telEnd, Tx: telEnd}, nil)
	srv := server.NewServer(urlStem)
	sk := SuperKExtreme{Module{
		RemoteDevice: &rd,
		AddrDev:      extremeDefaultAddr,
		Info:         SuperKExtremeMain}}
	srv.RouteTable["emission"] = sk.HTTPEmissionGet
	srv.RouteTable["emission/on"] = sk.HTTPEmissionOn
	srv.RouteTable["emission/off"] = sk.HTTPEmissionOff
	srv.RouteTable["power"] = sk.HTTPPower
	srv.RouteTable["main-module-status"] = sk.HTTPStatus
	sk.Module.Server = srv
	return &sk
}
