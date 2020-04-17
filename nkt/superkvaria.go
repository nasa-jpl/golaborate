package nkt

import (
	"math"

	"github.jpl.nasa.gov/bdube/golab/comm"
	"github.jpl.nasa.gov/bdube/golab/generichttp/laser"
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
			"Status": {
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
	bw := math.Round(math.Abs(long-short)*10) / 10 // *10/10 to round to nearest tenth
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

// GetShortWave retrieves the short wavelength setpoint of the Varia
func (sk *SuperKVaria) GetShortWave() (float64, error) {
	return sk.GetFloat("Short Wave Setpoint")
}

// SetShortWave retrieves the short wavelength setpoint of the Varia
func (sk *SuperKVaria) SetShortWave(nanometers float64) error {
	return sk.SetFloat("Short Wave Setpoint", nanometers)
}

// GetND retrieves the ND filter setpoint of the Varia
func (sk *SuperKVaria) GetND() (float64, error) {
	return sk.GetFloat("ND Setpoint")
}

// SetND retrieves the ND filter setpoint of the Varia
func (sk *SuperKVaria) SetND(nanometers float64) error {
	return sk.SetFloat("ND Setpoint", nanometers)
}

// GetLongWave retrieves the long wavelength setpoint of the Varia
func (sk *SuperKVaria) GetLongWave() (float64, error) {
	return sk.GetFloat("Long Wave Setpoint")
}

// SetLongWave retrieves the long wavelength setpoint of the Varia
func (sk *SuperKVaria) SetLongWave(nanometers float64) error {
	return sk.SetFloat("Long Wave Setpoint", nanometers)
}

// GetCenterBandwidth retrieves the center wavelength and bandwidth of the varia
func (sk *SuperKVaria) GetCenterBandwidth() (laser.CenterBandwidth, error) {
	var ret laser.CenterBandwidth
	short, err := sk.GetShortWave()
	if err != nil {
		return ret, err
	}
	long, err := sk.GetLongWave()
	if err != nil {
		return ret, err
	}
	ret = laser.ShortLongToCB(short, long)
	return ret, err
}

// SetCenterBandwidth sets the center wavelength and bandwidth of the laser
func (sk *SuperKVaria) SetCenterBandwidth(cbw laser.CenterBandwidth) error {
	short, long := cbw.ToShortLong()
	err := sk.SetShortWave(short)
	if err != nil {
		return err
	}
	err = sk.SetLongWave(long)
	if err != nil {
		return err
	}
	return nil
}
