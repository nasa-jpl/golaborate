// Package nkt enables working with NKT SuperK VARIA supercontinuum laser sources.
package nkt

import (
	"bytes"
	"encoding/binary"
	"time"

	"github.com/tarm/serial"
)

// messages are encoded as [SOT][MESSAGE][EOT].
// SOT and EOT are declared in the const ( ... ) block below
// the message is formatted as
// [DEST] [SOURCE] [TYPE] [REGISTER] [0..240 data bytes] [CRC]

// the workflow to generate a telegram is as follows:
// 0.  Using the message and metadata (to/from where, what type, what register)
//     generate the message body
// 1.  Scan for special characters and replace them as described in the manual
//     and implemented in sanitize()
// 2.  Prepend and append [SOT] and [EOT]

// MakeSerConf makes a new serial config
func MakeSerConf(addr string) *serial.Config {
	return &serial.Config{
		Name:        addr,
		Baud:        115200,
		Size:        8,
		Parity:      serial.ParityNone,
		StopBits:    serial.Stop1,
		ReadTimeout: 1 * time.Second}
}

// here we define some constants, mostly related to communication
const (
	// telStart is the start of telegram byte
	telStart = 0x0D

	// telEnd is the end of telegram byte
	telEnd = 0x0A

	// modTypeRegister is the memory register holding the module number
	modTypeRegister = 0x61

	// modFirmwareRegister is the memory register holding the firmware version code
	modFirmwareRegister = 0x64

	// modSerialRegister is the memory register holding the module serial number
	modSerialRegister = 0x65

	// modStatusRegister is the memory register holding the module status
	modStatusRegister = 0x66

	// modErrCodeRegister is the memory register holding the module error code
	modErrCodeRegister = 0x67

	// dataLength is the maximum message length in bytes
	dataLength = 240

	// minSourceAddr is the minimum value used for a source address
	minSourceAddr = 0xA1

	// specialCharFirstReplacement is the first byte used to replace a special character
	specialCharFirstReplacement = 0x94

	// specialCharShift is the amount to special characters up.
	// special characters max out at 0x5E, so we will never overflow
	specialCharShift = 0x40
)

var (
	// dataOrder is the byte order
	dataOrder = binary.LittleEndian

	// crcOrder is the byte order used for the CRC message
	crcOrder = binary.BigEndian

	// specialChars is a byte slice of values that must be filtered out of messages
	specialChars = []byte{0x0A, 0x0D, 0x5E}

	// currentSourceAddr holds the current source address and can only be accessed
	// by a single thread at once
	currentSourceAddr = make(chan byte, 1)
)

// ModuleInformation is a struct holding information needed to communicate with a given module.
type ModuleInformation struct {
	Addresses   map[string]byte
	StatusCodes map[int]string
}

func getSourceAddr() byte {
	// read the current address from the channel, then put either
	// addr + 1 on the channel (incremement), or wrap down to minSourceAddr
	// if we will overflow a single byte
	addr := <-currentSourceAddr
	if addr <= 254 {
		currentSourceAddr <- addr + 1
	} else {
		currentSourceAddr <- minSourceAddr
	}
	return addr
}

// AddressTypePair holds an Address and a Type byte
type AddressTypePair [2]byte

// AddressScan scans the NKT device to see:
// - /where/ what modules are installed (an address)
// - /what/ modules are installed (a type code)
// and returns a slice of slices, or "list of tuples" in python-like language.
// e.g.,
// [
//  {0x52, 3},
//  {0x53, 4},
//  ...
// ]
func AddressScan() []AddressTypePair {
	// use a "50-100ms" timeout and query reg modNumRegister on each addr
	return []AddressTypePair{}
}

func sanitize(data []byte) []byte {
	out := make([]byte, 240)
	for _, b := range data {
		if bytes.Contains(specialChars, []byte{b}) {
			out = append(out, []byte{specialCharFirstReplacement, b + specialCharShift}...)
		} else {
			out = append(out, b)
		}
	}
	return out
}
