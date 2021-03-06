package nkt

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"net"

	"github.com/snksoft/crc"
)

const (
	// telStart is the start of telegram byte
	telStart = 0x0D

	// telEnd is the end of telegram byte
	telEnd = 0x0A

	// minSourceAddr is the minimum value used for a source address
	minSourceAddr = 0xA1

	// specialCharFirstReplacement is the first byte used to replace a special character
	specialCharFirstReplacement = 0x5E

	// specialCharShift is the amount to special characters up.
	// special characters max out at 0x5E, so we will never overflow
	specialCharShift = 0x40
)

var (

	// ErrRemoteCRCMismatch is generated when the NKT responds with the CRC mismatch telegram type code
	ErrRemoteCRCMismatch = errors.New("CRC mismatch at NKT")

	// dataOrder is the byte order
	dataOrder = binary.LittleEndian

	// specialChars is a byte slice of values that must be filtered out of messages
	specialChars = []byte{0x0A, 0x0D, 0x5E}

	crcTable = crc.NewTable(crc.XMODEM)

	// MessageTypesSB maps strings to the bytecode for the message type
	MessageTypesSB = map[string]byte{
		"Nack":      0,
		"CRC Error": 1,
		"Busy":      2,
		"Ack":       3,
		"Read":      4,
		"Write":     5,
		"Write SET": 6,
		"Write CLR": 7,
		"Datagram":  8,
		"Write TGL": 9,
	}

	// MessageTypesBS maps bytecodes to the type of message received
	MessageTypesBS = map[byte]string{
		0: "Nack",
		1: "CRC Error",
		2: "Busy",
		3: "Ack",
		4: "Read",
		5: "Write",
		6: "Write SET",
		7: "Write CLR",
		8: "Datagram",
		9: "Write TGL",
	}

	// currentSourceAddr holds the current source address and can only be accessed
	// by a single thread at once
	currentSourceAddr = make(chan byte, 1)
)

func init() {
	currentSourceAddr <- minSourceAddr
}

// MessagePrimitive is a struct holding the raw bytes for a message before packing, CRC, and other processing
type MessagePrimitive struct {
	Dest, Src, Register byte
	Type                string
	Data                []byte
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

func sanitize(data []byte) []byte {
	var out []byte
	for _, b := range data {
		if bytes.Contains(specialChars, []byte{b}) {
			out = append(out, specialCharFirstReplacement, b+specialCharShift)
		} else {
			out = append(out, b)
		}
	}
	return out
}

func reverseSanitize(data []byte) []byte {
	var out []byte
	subNext := false
	for _, b := range data {
		if b == specialCharFirstReplacement {
			// if we hit a substitution marker, do nothing with this byte
			// and indicate to subtract from the next one
			subNext = true
		} else {
			if subNext {
				b = b - specialCharShift
			}
			out = append(out, b)
			subNext = false
		}
	}
	return out
}

// crcHelper computes the two-byte CRC value in a concurrent safe way and one line
func crcHelper(buf []byte) []byte {
	crcUint := crcTable.InitCrc()
	crcUint = crcTable.UpdateCrc(crcUint, buf)
	crcBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(crcBytes, crcTable.CRC16(crcUint))
	return crcBytes
}

// messages are encoded as [SOT][MESSAGE][EOT].
// SOT and EOT are declared in the const ( ... ) block below
// the message is formatted as
// [DEST] [SOURCE] [TYPE] [REGISTER] [0..240 data bytes] [CRC]

// EncodeTelegram produces a telegram from the constituent pieces.
// the workflow to generate a telegram is as follows:
// 0.  Using the message and metadata (to/from where, what type, what register)
//     generate the message body
// 1.  Scan for special characters and replace them as described in the manual
//     and implemented in sanitize()
// 2.  Calculate a CRC-16 value based on CRC-CCITT XMODEM.  sanitize() it and
//     append to the message
// 2.  Prepend and append [SOT] and [EOT]
func (mp MessagePrimitive) EncodeTelegram() ([]byte, error) {
	// make a buffer holding the raw message
	if _, ok := MessageTypesSB[mp.Type]; !ok {
		return []byte{}, fmt.Errorf("message type %s is invalid", mp.Type)
	}
	typ := MessageTypesSB[mp.Type]
	buf := append([]byte{mp.Dest, mp.Src, typ, mp.Register}, mp.Data...)

	// compute its CRC, and ensure that CRC is sanitized
	crcBytes := sanitize(crcHelper(buf))
	buf = sanitize(buf)

	// assemble the telegram
	out := append([]byte{telStart}, buf...)
	out = append(out, crcBytes...)
	out = append(out, telEnd)
	return out, nil
}

// DecodeTelegram renders a raw byte stream into a MessagePrimitive
func DecodeTelegram(tele []byte) (MessagePrimitive, error) {
	// first make sure that we have a start and an end
	if !bytes.Contains(tele, []byte{telStart}) {
		fstr := fmt.Sprintf("telegram start byte %X not found", telStart)
		return MessagePrimitive{}, errors.New(fstr)
		// } else if !bytes.Contains(tele, []byte{telEnd}) {
		// fstr := fmt.Sprintf("telegram end byte %X not found", telEnd)
		// return MessagePrimitive{}, errors.New(fstr)
	}
	if !bytes.Contains(tele, []byte{telEnd}) {
		fstr := fmt.Sprintf("telegram end byte %X not found", telEnd)
		return MessagePrimitive{}, errors.New(fstr)
	}
	// remove SOT/EOT
	start := bytes.IndexByte(tele, telStart)
	end := bytes.IndexByte(tele, telEnd)
	// iEnd := bytes.IndexByte(tele, telEnd)
	tele = tele[start+1 : end]

	// now desanitize the message
	tele = reverseSanitize(tele)

	// pop the CRC bytes
	fidx := len(tele) - 2
	crcBytesRecv := tele[fidx:]
	tele = tele[:fidx]

	// compute the CRC and ensure we match
	crcBytesCompute := crcHelper(tele)
	if !bytes.Equal(crcBytesRecv, crcBytesCompute) {
		return MessagePrimitive{}, errors.New("CRC mismatch, significant data lost in transmission.  NKT device state unknown")
	}

	// we have passed all the checks;
	// 1.  We have a complete transmission
	// 2.  No data was lost (CRC match)
	// now we can break the message into its constituent pieces
	mp := MessagePrimitive{}
	typ := MessageTypesBS[tele[2]]
	mp.Dest = tele[0]
	mp.Src = tele[1]
	mp.Type = typ
	mp.Register = tele[3]
	if typ != "CRC Error" {
		mp.Data = tele[4:]
		return mp, nil
	}
	return mp, ErrRemoteCRCMismatch
}

// writeThenRead writes a telegram to a connection and then reads a response from it, returning an error if there is one
func writeThenRead(conn net.Conn, telegram []byte) ([]byte, error) {
	_, err := conn.Write(telegram)
	if err != nil {
		return []byte{}, err
	}
	return bufio.NewReader(conn).ReadBytes(telEnd)
}
