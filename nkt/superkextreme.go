package nkt

// this file contains values relevant to the SuperK Varia accessory

// all values are encoded as uint16s and represent the quantity as described
// in the const block

const (
	//MainModAddr is the address of the main module
	MainModAddr = 0x15

	// MainModType is the byte that signals a module is a SuperK Extreme (S4x2...)
	MainModType = 0x60

	// InletTempAddr is the address the inlet temperature is stored in.
	// the value is reported in tenths of a degree celcius
	InletTempAddr = 0x13

	// EmissionAddr is the address the emission state is stored in.
	// 0 = off, 1 = on, 3 = force on (if interlock circuit has been reset).
	// reading this returns the same value written, except
	// in the cirucmstance where the state has recently switched
	// in which case the value cycles through several undefined values
	EmissionAddr = 0x30

	// SetupAddr is the address that holds the setup code.
	// See SetupCodes for possible values
	SetupAddr = 0x31

	// PowerLevelAddr is the address that holds the output power.
	// the value is encoded in tenths of a percent
	PowerLevelAddr = 0x37

	// CurrentLevelAddr is the address that holds the current (amperage) level.
	// the value is encoded in tenths of a percent
	CurrentLevelAddr = 0x38

	// NIMDelayAddr is the address holding the NIM output delay.
	// the value is encoded in, 1/1023 step size of 0..9.2ns.
	NIMDelayAddr = 0x39

	// StatusAddr is the address that holds the status bits.
	StatusAddr = 0x66

	// UserTextAddr is the address that holds the user text
	UserTextAddr = 0x6C

	// PanelLockAddr is the address holding the panel lock value.
	// it is encoded as 1, the panel control is locked out.  0 is unlocked.
	// 8-bit int
	PanelLockAddr = 0x3D

	// DisplayTextAddr is the address holding the display text.
	// the value is an 80 character wide ASCII byte slice.
	DisplayTextAddr = 0x72

	// ErrorFlashAddr is the address holding whether the display flashes on errors.
	// it is encoded as 1, the display will flash.  0 it will not. 8-bit int.
	ErrorFlashAddr = 0xBD

	// BoosterAddr holds the default address used for the booster module
	BoosterAddr = 0x05

	// BoosterType is the byte value that identifies a module as a booster
	BoosterType = 0x65
)

var (
	// Setups maps the status bit to a string
	Setups = map[uint16]string{
		0: "Constant current mode",
		1: "Constant power mode",
		2: "Externally modulated current mode",
		3: "Externally modulated power",
		4: "External feedback mode (Power Lock)",
	}

	// Interlocks maps the interlock

	// Statuses maps the status bits to statuses.
	Statuses = map[uint16]string{
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
	}

	// BoosterStatuses maps bits in the status byte of the booster module to strings
	BoosterStatuses = map[uint16]string{
		0: "Emission on",
		1: "Interlock signal off",
		2: "Interlock loop input low",
		3: "Interlock loop output low",
		4: "Module disabled",
		5: "Supply voltage out of range",
		6: "Board temperature out of range",
		7: "Heat sink temperature out of range",
	}
)
