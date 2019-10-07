package nkt

import "go/types"

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
			"Emission":           0x13,
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
			}},
		ValueTypes: map[string]types.BasicKind{
			"Emission": types.Bool,
		}}

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

// NewSuperKExtreme create a new Module representing a SuperKExtreme's main module
func NewSuperKExtreme(addr string) Module {
	return Module{
		AddrConn: addr,
		AddrDev:  extremeDefaultAddr,
		Info:     SuperKExtremeMain}
}
