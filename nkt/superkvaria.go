package nkt

// this file contains values relevant to the SuperK Varia accessory

const (
	variaDefaultAddr = 0x10
)

var (
	// SuperKVaria describes the SuperK Varia module
	SuperKVaria = &ModuleInformation{
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

// NewSuperKVaria create a new Module representing a SuperKExtreme's main module
func NewSuperKVaria(addr string) Module {
	return Module{
		AddrConn: addr,
		AddrDev:  variaDefaultAddr,
		Info:     SuperKExtremeMain}
}