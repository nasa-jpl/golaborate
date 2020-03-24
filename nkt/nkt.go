// Package nkt enables working with NKT SuperK supercontinuum lasers.
//
// The wrapping of the individual submodules could
// be made more ergonomic in Go, but no one
package nkt

import (
	"context"
	"encoding/binary"
	"errors"
	"go/types"
	"time"

	"github.jpl.nasa.gov/bdube/golab/comm"

	"github.jpl.nasa.gov/bdube/golab/server"
	"github.jpl.nasa.gov/bdube/golab/util"
	"golang.org/x/time/rate"

	"github.com/tarm/serial"
)

// messages are encoded as [SOT][MESSAGE][EOT].
// SOT and EOT are declared in the const ( ... ) block below
// the message is formatted as
// [DEST] [SOURCE] [TYPE] [REGISTER] [0..240 data bytes] [CRC]

// makeSerConf makes a new serial config
func makeSerConf(addr string) serial.Config {
	return serial.Config{
		Name:        addr,
		Baud:        115200,
		Size:        8,
		Parity:      serial.ParityNone,
		StopBits:    serial.Stop1,
		ReadTimeout: 1 * time.Second}
}

var (
	// ErrUnknownAddr is generated by an invalid address
	ErrUnknownAddr = errors.New("address is not a member of nkt.StandardAddresses or known to the module")

	// StandardAddresses maps some addresses present for all modules
	StandardAddresses = map[string]byte{
		"TypeCode":         0x61,
		"Firmware Version": 0x64,
		"Serial":           0x65,
		"Status":           0x66,
		"ErrorCode":        0x67,
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
	// Addresses is a map from friendly names like "Emission" to byte codes (hardware addresses)
	Addresses map[string]byte
	// CodeBanks maps friendly names like "Statuses" to bitfield maps
	CodeBanks map[string]map[int]string

	// Decoders maps friendly names like "Emission" to decoding functions that return a thinning json-serializable object
	Decoders map[string]func([]byte) server.HumanPayload
}

// UnpackRegister converts the raw data from a register into a server.HumanPayload
func UnpackRegister(b []byte, typ types.BasicKind) server.HumanPayload {
	var hp server.HumanPayload
	if len(b) == 0 {
		return server.HumanPayload{}
	}
	switch typ {
	case types.Uint16:
		v := dataOrder.Uint16(b)
		hp = server.HumanPayload{Uint16: v}
	case types.Bool:
		v := uint8(b[0]) == 1
		hp = server.HumanPayload{Bool: v}
	case types.String:
		v := string(b)
		hp = server.HumanPayload{String: v}
	case types.Byte:
		v := b[0]
		hp = server.HumanPayload{Byte: v}
	default: // default is 10x superres floating point value
		v := dataOrder.Uint16(b)
		hp = server.HumanPayload{Float: float64(v) / 10.0}
	}

	hp.T = typ
	return hp
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
	conn, err := comm.TCPSetup(addr, 3*time.Second)
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
		tele, err := mp.EncodeTelegram()
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
		resp, err := writeThenRead(conn, tele)
		if err == nil {
			if len(resp) > 0 {
				mpr, err := DecodeTelegram(resp)
				if err == nil {
					modules[a] = ModuleTypeMap[mpr.Data[0]]
				}
			}

		}

	}
	return modules, nil
}

// Module objects have an address and information struct
type Module struct {
	// AddrDev is the module address on the NKT device itself
	AddrDev byte

	// Info contains mapping data for a given module, see ModuleInformation for more docs.
	Info *ModuleInformation

	*comm.RemoteDevice
}

// SerialConf satisfies comm.SerialConfigurator and enables operation over a serial port
func (m *Module) SerialConf() serial.Config {
	return makeSerConf(m.RemoteDevice.Addr)
}

func (m *Module) getRegister(addrName string) (byte, error) {
	var register byte
	if value, ok := m.Info.Addresses[addrName]; ok {
		register = value
	} else if value, ok = StandardAddresses[addrName]; ok {
		register = value
	} else {
		return byte(0), errors.New("addrName is not a register known to the module or nkt.StandardAddresses")
	}
	return register, nil
}

// SendMP overloads RemoteDevice.Send to handle telegram encoding of message primitives
func (m *Module) SendMP(mp MessagePrimitive) error {
	tele, err := mp.EncodeTelegram()
	if err != nil {
		return err
	}
	err = m.Send(tele)
	return err
}

// RecvMP overloads RemoteDevice.Recv to handle telegram decoding into message primitives
func (m *Module) RecvMP() (MessagePrimitive, error) {
	buf, err := m.Recv()
	if err != nil {
		return MessagePrimitive{}, err
	}
	return DecodeTelegram(buf)
}

// SendRecvMP sends a buffer after appending the Tx terminator,
// then returns the response with the Rx terminator stripped
func (m *Module) SendRecvMP(mp MessagePrimitive) (MessagePrimitive, error) {
	var (
		mpRecv MessagePrimitive
		err    error
	)
	m.Lock()
	defer m.Unlock()
	// try to send the message up to 3 times.  transient CRC errors are pretty common with the NKTs
	for idx := 0; idx < 5; idx++ {
		err = m.SendMP(mp)
		if err != nil {
			return mpRecv, err
		}
		mpRecv, err = m.RecvMP()
		if err == nil {
			break
		}
		mp.Src = getSourceAddr()
	}
	return mpRecv, err

}

// GetValue reads a register
func (m *Module) GetValue(addrName string) (MessagePrimitive, error) {
	register, err := m.getRegister(addrName)
	if err != nil {
		return MessagePrimitive{}, err
	}
	err = m.Open()
	if err != nil {
		return MessagePrimitive{}, err
	}
	defer m.CloseEventually()

	mpSend := MessagePrimitive{
		Dest:     m.AddrDev,
		Src:      getSourceAddr(), // GSA returns a quasi-unique source address (up to ~154 per message interval)
		Register: register,
		Type:     "Read"}

	return m.SendRecvMP(mpSend)
}

//SetValue writes a register
func (m *Module) SetValue(addrName string, data []byte) (MessagePrimitive, error) {
	register, err := m.getRegister(addrName)
	if err != nil {
		return MessagePrimitive{}, err
	}

	mpSend := MessagePrimitive{
		Dest:     m.AddrDev,
		Src:      getSourceAddr(),
		Register: register,
		Type:     "Write",
		Data:     data}

	err = m.Open()
	if err != nil {
		return MessagePrimitive{}, err
	}
	defer m.CloseEventually()

	return m.SendRecvMP(mpSend)
}

// GetValueMulti is equivalent to GetValue for multiple addresses
// if an error is encoutered along the way, the incomplete slice of MessagePrimitives will be returned with the error.
func (m *Module) GetValueMulti(addrNames []string) ([]MessagePrimitive, error) {
	err := m.Open()
	if err != nil {
		return []MessagePrimitive{}, err
	}
	defer m.CloseEventually()

	l := len(addrNames)
	messages := make([]MessagePrimitive, l, l)
	for idx, addr := range addrNames {
		register, err := m.getRegister(addr)
		if err != nil {
			return messages, err
		}
		mpSend := MessagePrimitive{
			Dest:     m.AddrDev,
			Src:      getSourceAddr(),
			Register: register,
			Type:     "Read",
			Data:     []byte{}}
		mpRecv, err := m.SendRecvMP(mpSend)
		messages[idx] = mpRecv
		if err != nil {
			return messages, err
		}
	}
	return messages, nil
}

// SetValueMulti is equivalent to SetValue for multiple addresses
// if an error is encoutered along the way, the incomplete slice of MessagePrimitives will be returned with the error.
func (m *Module) SetValueMulti(addrNames []string, data [][]byte) ([]MessagePrimitive, error) {
	err := m.Open()
	if err != nil {
		return []MessagePrimitive{}, err
	}
	defer m.CloseEventually()

	l := len(addrNames)
	messages := make([]MessagePrimitive, l, l)
	for idx, addr := range addrNames {
		register, err := m.getRegister(addr)
		if err != nil {
			return messages, err
		}
		d := data[idx]
		mpSend := MessagePrimitive{
			Dest:     m.AddrDev,
			Src:      getSourceAddr(),
			Register: register,
			Type:     "Write",
			Data:     d}
		mpRecv, err := m.SendRecvMP(mpSend)
		messages[idx] = mpRecv
		if err != nil {
			return messages, err
		}
	}
	return messages, nil
}

// GetStatus gets the status bitfield and converts it into a map of descriptive strings to booleans
func (m *Module) GetStatus() (map[string]bool, error) {
	// declare the response and get the response from the NKT
	resp := map[string]bool{}
	mp, err := m.GetValue("Status")
	if err != nil {
		return resp, err
	}

	// pop the bitfield and codebank for the module's status codes
	bitfield := mp.Data
	codebank := m.Info.CodeBanks["Status"]

	// loop over the number of bits in the codebank (which may be more than 1 byte in size)
	// each time we are modulo 8, we increment the offset
	nbits := uint(len(codebank))
	idx := uint(0)
	byteOffset := 0
	for (idx) < nbits { // 8 bits per byte
		if text, ok := codebank[int(idx)]; ok {
			bidx := idx - uint(byteOffset*8)
			resp[text] = util.GetBit(bitfield[byteOffset], bidx)
		}
		// increment the loop
		idx++
		if idx%8 == 0 {
			// if we are on a byte boundary, incremement the byte offset and roll down the index
			byteOffset++
		}
	}
	delete(resp, "-")
	return resp, nil
}
