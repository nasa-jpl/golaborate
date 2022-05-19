package nkt

import (
	"github.jpl.nasa.gov/bdube/golab/comm"
)

// this file contains values relevant to the SuperK Extreme modules

const (
	extremeDefaultAddr        = 0x0F
	extremeFrontDefaultAddr   = 0x01
	extremeBoosterDefaultAddr = 0x65
)

var (
	// SuperKExtremeMain describes the SuperK Extreme Main module
	SuperKExtremeMainInfo = &ModuleInformation{
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
			"Setup": {
				0: "Constant current mode",
				1: "Constant power mode",
				2: "Externally modulated current mode",
				3: "Externally modulated power",
				4: "External feedback mode (Power Lock)"},
			"Status": {
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
	SuperKFrontDisplayInfo = ModuleInformation{
		Addresses: map[string]byte{
			"Panel Lock":  0x3D,
			"Text":        0x72,
			"Error Flash": 0xBD}}

	// SuperKBooster describes the SuperK Booster module
	SuperKBoosterInfo = ModuleInformation{
		Addresses: map[string]byte{
			"Module":           0x05,
			"Emission Runtime": 0x80,
			"Status":           0x66},
		CodeBanks: map[string]map[int]string{
			"Status": {
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

// NewSuperKExtreme create a new Module representing a SuperKExtreme's main module
func NewSuperKExtreme(addr string, pool *comm.Pool) *SuperKExtreme {
	return &SuperKExtreme{Module{
		pool:    pool,
		AddrDev: extremeDefaultAddr,
		Info:    SuperKExtremeMainInfo}}
}

// SetEmission turns emission (laser output) on
func (sk *SuperKExtreme) SetEmission(on bool) error {
	payload := []byte{0}
	if on {
		payload[0] = 3
	}
	_, err := sk.SetValue("Emission", payload)
	return err
}

// GetEmission queries if emission (laser output) is enabled
func (sk *SuperKExtreme) GetEmission() (bool, error) {
	resp, err := sk.GetValue("Emission")
	if err != nil {
		return false, err
	}
	return resp.Data[0] > 0, nil
}

// SetPower sets the output power level (0-100) of the laser
func (sk *SuperKExtreme) SetPower(level float64) error {
	return sk.SetFloat("Power Level", level)
}

// GetPower retrieves the power level of the laser
func (sk *SuperKExtreme) GetPower() (float64, error) {
	return sk.GetFloat("Power Level")
}

// SuperKBooster embeds Module and has an EmissionRuntime method
type SuperKBooster struct {
	Module
}

// NewSuperKBooster creates a new Module representing a SuperK booster (where the laser gain medium lives)
func NewSuperKBooster(addr string, pool *comm.Pool) *SuperKBooster {
	return &SuperKBooster{Module{
		pool:    pool,
		AddrDev: extremeDefaultAddr,
		Info:    &SuperKBoosterInfo}}
}

// GetEmissionRuntime gets the total runtime of the laser booster in seconds
func (sk *SuperKBooster) GetEmissionRuntime() (float64, error) {
	u, err := sk.GetUint32("Emission Runtime")
	return float64(u), err
}
