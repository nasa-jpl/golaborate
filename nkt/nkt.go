// Package nkt enables working with NKT SuperK VARIA supercontinuum laser sources.
package nkt

import (
	"bufio"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"go/types"
	"log"
	"net/http"
	"time"

	"github.jpl.nasa.gov/HCIT/go-hcit/util"
	"golang.org/x/time/rate"

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

	// StandardTypes maps the types of addresses present for all modules
	StandardTypes = map[string]types.BasicKind{
		"TypeCode":         types.Byte,
		"Firmware Version": types.Byte,
		"Serial":           types.Byte,
		"Status":           types.Byte,
		"ErrorCode":        types.Byte,
	}

	// ModuleTypeMap maps bytes to human-readable strings for module types
	ModuleTypeMap = map[byte]string{
		0x0:  "N/A",
		0x60: "SuperK Extreme",
		0x61: "SuperK Front Panel",
		0x65: "SuperK Booster",
		0x68: "SuperK Varia",
		0x6B: "SuperK Extend UV",
		0x74: "SuperK COMPACT",
	}
)

// ModuleInformation is a struct holding information needed to communicate with a given module.
type ModuleInformation struct {
	Addresses  map[string]byte
	CodeBanks  map[string]map[int]string
	ValueTypes map[string]types.BasicKind
}

// all of the following types are followed with a capital T for homogenaeity and
// avoiding clashes with builtins

type strT struct {
	Str string `json:"str"`
}

type floatT struct {
	F64 float64 `json:"f64"`
}

type uintT struct {
	Int uint16 `json:"int"`
}

type byteT struct {
	Int byte `json:"int"` // we won't distinguish between bytes and ints for users
}

type bufferT struct {
	IntAry []byte `json:"int"`
}

type boolT struct {
	Bool bool `json:"bool"`
}

// HumanPayload is a struct containing the basic types NKT devices may work with
type HumanPayload struct {
	// Bool holds a binary value
	Bool bool

	// Buffer holds raw bytes
	Buffer []byte

	// Byte holds a single byte
	Byte byte

	// Float holds a float
	Float float64

	// String holds a string
	String string

	// Uint16 holds a uint16
	Uint16 uint16

	// T holds the type of data actually contained in the payload
	T types.BasicKind
}

// UnpackRegister converts the raw data from a register into a HumanPayload
func UnpackRegister(b []byte, typ types.BasicKind) HumanPayload {
	var hp HumanPayload
	if len(b) == 0 {
		return HumanPayload{}
	}
	switch typ {
	case types.Uint16:
		v := dataOrder.Uint16(b)
		hp = HumanPayload{Uint16: v}
	case types.Bool:
		v := uint8(b[0]) == 1
		hp = HumanPayload{Bool: v}
	case types.String:
		v := string(b)
		hp = HumanPayload{String: v}
	case types.Byte:
		v := b[0]
		hp = HumanPayload{Byte: v}
	default: // default is 10x superres floating point value
		v := dataOrder.Uint16(b)
		hp = HumanPayload{Float: float64(v) / 10.0}
	}

	hp.T = typ
	return hp
}

// EncodeAndRespond converts the humanpayload to a smaller struct with only one
// field and writes it to w as JSON.
func (hp *HumanPayload) EncodeAndRespond(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	switch hp.T {
	case types.Bool:
		obj := boolT{Bool: hp.Bool}

		// the logic from err to the closing brace is copy pasted a bunch in here
		err := json.NewEncoder(w).Encode(obj)
		if err != nil {
			fstr := fmt.Sprintf("error encoding %+v hp to JSON, %q", hp, err)
			log.Println(fstr)
			http.Error(w, fstr, http.StatusInternalServerError)
		}
	// skip bytes case, unhandled in Unpack
	case types.Byte:
		obj := byteT{Int: hp.Byte} // Byte -> int for consistency with uints

		err := json.NewEncoder(w).Encode(obj)
		if err != nil {
			fstr := fmt.Sprintf("error encoding %+v hp to JSON, %q", hp, err)
			log.Println(fstr)
			http.Error(w, fstr, http.StatusInternalServerError)
		}
	case types.Float64:
		obj := floatT{F64: hp.Float}

		err := json.NewEncoder(w).Encode(obj)
		if err != nil {
			fstr := fmt.Sprintf("error encoding %+v hp to JSON, %q", hp, err)
			log.Println(fstr)
			http.Error(w, fstr, http.StatusInternalServerError)
		}
	case types.String:
		obj := strT{Str: hp.String}

		err := json.NewEncoder(w).Encode(obj)
		if err != nil {
			fstr := fmt.Sprintf("error encoding %+v hp to JSON, %q", hp, err)
			log.Println(fstr)
			http.Error(w, fstr, http.StatusInternalServerError)
		}
	case types.Uint16:
		obj := uintT{Int: hp.Uint16}
		err := json.NewEncoder(w).Encode(obj)
		if err != nil {
			fstr := fmt.Sprintf("error encoding %+v hp to JSON, %q", hp, err)
			log.Println(fstr)
			http.Error(w, fstr, http.StatusInternalServerError)
		}

	}
}

// crcHelper computes the two-byte CRC value in a concurrent safe way and one line
func crcHelper(buf []byte) []byte {
	crcUint := crcTable.InitCrc()
	crcUint = crcTable.UpdateCrc(crcUint, buf)
	crcBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(crcBytes, crcTable.CRC16(crcUint))
	return crcBytes
}

// AddressScan scans the NKT device to see:
// - /where/ what modules are installed (an address)
// - /what/ modules are installed (a type code)
// and returns a map that connects addresses to (string) module types
func AddressScan(addr string) (map[byte]string, error) {
	// first establish a connection and dummy context for the limiter
	conn, err := util.TCPSetup(addr, 3*time.Second)
	if err != nil {
		return map[byte]string{}, err
	}
	defer conn.Close()
	c := context.Background()

	// now set up the rate limiter and basic message constructs
	limiter := rate.NewLimiter(15, 15)
	mp := MessagePrimitive{Type: "Read", Register: StandardAddresses["TypeCode"]}
	addrs := util.ArangeByte(1, 160)

	modules := map[byte]string{}

	for _, a := range addrs {
		mp.Dest = a
		mp.Src = getSourceAddr()
		tele, err := MakeTelegram(mp)
		if err != nil {
			return map[byte]string{}, err
		}
		err = limiter.Wait(c)
		if err != nil {
			return map[byte]string{}, err
		}

		deadline := time.Now().Add(75 * time.Millisecond) // the manual advises 50-100 ms here
		conn.SetReadDeadline(deadline)
		conn.SetWriteDeadline(deadline)
		resp, err := WriteThenRead(conn, tele)
		if err == nil {
			if len(resp) > 0 {
				mpr, err := DecodeTelegram(resp)
				if err == nil {
					fmt.Println(a, mpr.Data)
					modules[a] = ModuleTypeMap[mpr.Data[0]]
				}
			}

		}

	}
	return modules, nil
}

// Module objects have an address and information struct
type Module struct {
	// AddrConn is the network or serial location of the module (i.e., is externally facing)
	AddrConn string

	// AddrDev is the module address on the NKT device itself
	AddrDev byte

	// Info contains mapping data for a given module, see ModuleInformation for more docs.
	Info *ModuleInformation
}

func (m *Module) valueType(addrName string) types.BasicKind {
	if value, ok := m.Info.ValueTypes[addrName]; ok {
		return value
	} else if value, ok = StandardTypes[addrName]; ok {
		return value
	} else {
		return 0
	}

}

// SendRecv writes a telegram to the NKT device and returns the response
func (m *Module) SendRecv(tele []byte) ([]byte, error) {
	conn, err := util.TCPSetup(m.AddrConn, 3*time.Second)
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
	var register byte
	if value, ok := m.Info.Addresses[addrName]; ok {
		register = value
	} else if value, ok = StandardAddresses[addrName]; ok {
		register = value
	} else {
		return MessagePrimitive{}, HumanPayload{}, errors.New("addrName is not a register known to the module or nkt.StandardAddresses")
	}
	mpSend := MessagePrimitive{
		Dest:     m.AddrDev,
		Src:      getSourceAddr(), // GSA returns a quasi-unique source address (up to ~154 per message interval)
		Register: register,
		Type:     "Read",
		Data:     []byte{}}

	tele, err := MakeTelegram(mpSend)
	if err != nil {
		return MessagePrimitive{}, HumanPayload{}, err
	}
	resp, err := m.SendRecv(tele)
	if err != nil {
		return MessagePrimitive{}, HumanPayload{}, err
	}

	mpRecv, err := DecodeTelegram(resp)

	// ValueTypes maps registers to types (e.g. uint16).
	// UnpackRegister converts this to a HumanPayload.  The map lookup returns 0
	// if the type is not defined, which will trigger a floating point conversion
	// at 10x resolution
	hp := UnpackRegister(mpRecv.Data, m.valueType(addrName))
	if err != nil {
		return MessagePrimitive{}, HumanPayload{}, err
	}
	return mpRecv, hp, nil
}
