package nkt

// this file contains values relevant to the SuperK Varia accessory

// all values are encoded as uint16s and represent the quantity as described
// in the const block

const (
	// ModType is the byte that signals a module is a SuperK Varia
	ModType = 0x68

	// MonInputAddr is the address the "monitor input" is stored in.
	// the value is reported in tenths of a percent
	MonInputAddr = 0x13

	// NDSetpointAddr is the address for the ND setpoint.
	// the value is reported in tenths of a percent
	NDSetpointAddr = 0x32

	// ShortWavePassAddr is the address for the short wave pass filter in 1/10 nm
	ShortWavePassAddr = 0x33

	// LongWavePassAddr is the address for the long wave pass filter in 1/10 nm
	LongWavePassAddr = 0x33
)

var (
	// Statuses maps the status bit to a string
	Statuses = map[uint16]string{
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
	}
)
