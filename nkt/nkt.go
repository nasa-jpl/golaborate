// Package nkt enables working with NKT SuperK VARIA supercontinuum laser sources.
package nkt

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"go/types"
	"time"

	"github.jpl.nasa.gov/HCIT/go-hcit/util"

	"github.com/tarm/serial"
)

// messages are encoded as [SOT][MESSAGE][EOT].
// SOT and EOT are declared in the const ( ... ) block below
// the message is formatted as
// [DEST] [SOURCE] [TYPE] [REGISTER] [0..240 data bytes] [CRC]

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

var (
	// StandardAddresses maps some addresses present for all modules
	StandardAddresses = map[string]byte{
		"TypeCode":         0x61,
		"Firmware Version": 0x64,
		"Serial":           0x65,
		"Status":           0x66,
		"ErrorCode":        0x67,
	}
)

// ModuleInformation is a struct holding information needed to communicate with a given module.
type ModuleInformation struct {
	Addresses  map[string]byte
	CodeBanks  map[string]map[int]string
	ValueTypes map[string]types.BasicKind
	TypeCode   byte
}

// HumanPayload is a struct containing the basic types NKT devices may work with
type HumanPayload struct {
	// S holds a string
	S string

	// U holds a uint16
	U uint16

	// B holds raw bytes
	B []byte

	// F holds a float
	F float64
}

// crcHelper computes the two-byte CRC value in a concurrent safe way and one line
func crcHelper(buf []byte) []byte {
	crcUint := crcTable.InitCrc()
	crcUint = crcTable.UpdateCrc(crcUint, buf)
	crcBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(crcBytes, crcTable.CRC16(crcUint))
	return crcBytes
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

// Module objects have an address and information struct
type Module struct {
	// Addr is the network or serial location of the module (i.e., is externally facing)
	Addr string

	// Info contains mapping data for a given module, see ModuleInformation for more docs.
	Info *ModuleInformation
}

// SendRecv writes a telegram to the NKT device and returns the response
func (m *Module) SendRecv(tele []byte) ([]byte, error) {
	conn, err := util.TCPSetup(m.Addr, 3*time.Second)
	if err != nil {
		return []byte{}, err
	}
	defer conn.Close()
	_, err = conn.Write(tele)
	if err != nil {
		return []byte{}, err
	}

	return bufio.NewReader(conn).ReadBytes(telEnd)
}

// GetValue reads a register
func (m *Module) GetValue(addrName string) (MessagePrimitive, HumanPayload, error) {
	mpSend := MessagePrimitive{
		Dest:     m.Info.Addresses["Module"],
		Src:      getSourceAddr(), // GSA returns a quasi-unique source address (up to ~154 per message interval)
		Register: m.Info.Addresses[addrName],
		Type:     "Read",
		Data:     []byte{}}

	tele, err := MakeTelegram(mpSend)
	if err != nil {
		return MessagePrimitive{}, HumanPayload{}, err
	}
	resp, err := m.SendRecv(tele)
	fmt.Printf("%X\n", resp)
	if err != nil {
		return MessagePrimitive{}, HumanPayload{}, err
	}

	mpRecv, err := DecodeTelegram(resp)

	// ValueTypes maps registers to types (e.g. uint16).
	// UnpackRegister converts this to a HumanPayload.  The map lookup returns 0
	// if the type is not defined, which will trigger a floating point conversion
	// at 10x resolution
	hp := UnpackRegister(mpRecv.Data, m.Info.ValueTypes[addrName])
	if err != nil {
		return MessagePrimitive{}, HumanPayload{}, err
	}
	return mpRecv, hp, nil
}

// UnpackRegister converts the raw data from a register into a HumanPayload
func UnpackRegister(b []byte, typ types.BasicKind) HumanPayload {
	if len(b) == 0 {
		return HumanPayload{}
	}
	switch typ {
	case types.Uint16:
		v := dataOrder.Uint16(b)
		return HumanPayload{U: v}
	case types.String:
		v := string(b)
		return HumanPayload{S: v}
	default: // default is 10x superres floating point value
		v := dataOrder.Uint16(b)
		return HumanPayload{F: float64(v) / 10.0}

	}
}
