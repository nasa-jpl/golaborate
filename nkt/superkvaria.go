package nkt

import (
	"math"

	"github.jpl.nasa.gov/HCIT/go-hcit/comm"
	"github.jpl.nasa.gov/HCIT/go-hcit/mathx"
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
	bw := mathx.Round(math.Abs(long-short), 0.1)
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

// NewSuperKVaria create a new Module representing a SuperKVaria module
func NewSuperKVaria(addr string, serial bool) *SuperKVaria {
	rd := comm.NewRemoteDevice(addr, serial, &comm.Terminators{Rx: telEnd, Tx: telEnd}, nil)
	return &SuperKVaria{Module{
		RemoteDevice: &rd,
		AddrDev:      variaDefaultAddr,
		Info:         SuperKVariaInfo}}
}
