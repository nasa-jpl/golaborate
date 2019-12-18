/*Package usbtmc implements datagram encoding and decoding for USB Test and
Measurement Class devices.  This is a 'minimum viable product' for the bulk
transfer mode on the Thorlabs LDC4001 laser diode and TEC controller.

It does not, for example, include features to support multi-packet
messaging, and thus assumes your data fits in the remote's buffer.

It also does not implement chatter / ping-pong for the case when data
does not fit in the remote buffer.

To send a message:
1.  Allocate a send buffer
2.  Write the header to it
3.  Write your data to it
4.  Ensure that the total transmission size is a multiple of 4 bytes before flushing

To receive a message:
1.  Allocate a receipt buffer
2.  Create a read header and send it on the Out endpoint
3.  Read from the In endpoint

These macros are implemented as Write() and Read() on the concrete USB type defined in this package.
*/
package usbtmc

import (
	"encoding/binary"
	"fmt"
	"sync"

	"github.com/google/gousb"
)

const (
	// reserved is the byte to insert when the
	reserved = 0x00
)

// BTagger can generate atomic bTags
type BTagger interface {
	nextbTag() byte
}

// bTagGen is a concurrent-safe bTag generator
type bTagGen struct {
	// embedded mutex for concurrent safety
	sync.Mutex

	value byte
	min   byte
}

func newBTagGen() *bTagGen {
	return &bTagGen{value: 1, min: 1}
}

func (b *bTagGen) nextbTag() byte {
	b.Lock()
	defer b.Unlock()
	b.value++
	if b.value < b.min {
		b.value = b.min
	}
	return b.value
}

// invbTag computes the bitwise inversion of a btag, per USBTMC standard table 1 offset 2
func invbTag(b byte) byte {
	// ^ is bitwise exclusive OR.  Comparing with 0xff (all 1s) is the bitwise inversion
	return b ^ 0xff
}

// BulkInResponse is the response from a bulk input read, split into header and payload
type BulkInResponse struct {
	// Header is the header bytes that are prepended to the data
	Header []byte

	// Data is the actual datagram body
	Data []byte
}

// encBulkOutHeader creates the header defined in USBTMC standard, Table 3
func encBulkOutHeader(btag BTagger, datalen int) [12]byte {
	out := [12]byte{} // this is an array, not a slice.  Fixed size, determined at compile time, will live on the stack
	/* data map by offset:
	0 MsgID, 1 byte, here hardcoded to 1; devDepMsgOut
	1 bTag, a single byte 1 < x < 255, unique and incrementing with each message
	2 bTagInverse, a single byte, the bitwise inverse of bTag.  Can be calculated with invbTag
	3 Reserved (0x00)
	4-11 command message specific
	---
	In the case of devDepMsgOut, 4-11 look like:
	4-7 transferSize,
		total number of message data bytes exclusive of the header and alignment.
		LSB first, > 0
	8 bitmap
		bits 7..1 0 (reserved)
		bit 0 EOM, if bit(0) == 1, this is the last message in the stream else not the end of stream
		boils down to 0x00 if not end of message, 0x01 if end of message
	9-11 reserved
	*/
	tag := btag.nextbTag()
	out[0] = 0x01 // DEV_DEP_MSG_OUT = 0x01, hardcode this type for now
	out[1] = tag
	out[2] = invbTag(tag)
	out[3] = reserved
	buf := out[4:8]
	binary.LittleEndian.PutUint32(buf, uint32(datalen))
	out[8] = 0x01 // hardcode end of message
	out[9] = reserved
	out[10] = reserved
	out[11] = reserved
	return out
}

// endBulkInHeader creates the header defined in USBTMC standard, Table 4.
// if terminator is nil, puts 0x00 in the header and sets the bit to use it to false
func encBulkInHeader(btag BTagger, bufsize int, terminator *byte) [12]byte {
	out := [12]byte{}
	/* this differs from BulkOut by bytes 8~11
	8 bitmap
		bits 8..1 0 (reserved)
		bit 0 termination character enabled,
		if 1 datagram must end on term char
		if 0 device must ignore termination char
	9 terminator byte
	10~11 reserved
	*/
	tag := btag.nextbTag()
	out[0] = 0x02 // REQUEST_DEV_DEP_MSG_IN
	out[1] = tag
	out[2] = invbTag(tag)
	out[3] = reserved
	buf := out[4:8]
	binary.LittleEndian.PutUint32(buf, uint32(bufsize))
	if terminator != nil {
		out[8] = 0x02 // end-1th bit is 1 -> hex 2
		out[9] = *terminator
	} else {
		out[8] = 0x00
		out[9] = 0x00
	}
	out[10] = reserved
	out[11] = reserved
	return out
}

// USBDevice is a struct hiding the details of USB and exposing an extended io.ReadWriteCloser interface
type USBDevice struct {
	tagger BTagger
	in     *gousb.InEndpoint
	out    *gousb.OutEndpoint
	device *gousb.Device
	iface  *gousb.Interface
	closer func()
}

// NewUSBDevice creates a new USB device from its vendor and product ID
func NewUSBDevice(vid, pid uint16) (USBDevice, error) {
	var out USBDevice
	var err error
	ctx := gousb.NewContext()
	out.device, err = ctx.OpenDeviceWithVIDPID(gousb.ID(vid), gousb.ID(pid))
	if err != nil {
		return out, err
	}
	err = out.device.SetAutoDetach(true)
	if err != nil {
		return out, err
	}
	out.iface, out.closer, err = out.device.DefaultInterface()
	if err != nil {
		out.closer()
		return out, err
	}
	out.in, err = out.iface.InEndpoint(2)
	if err != nil {
		return out, err
	}
	out.out, err = out.iface.OutEndpoint(2)
	if err != nil {
		return out, err
	}
	return out, nil
}

func (d *USBDevice) Read() (BulkInResponse, error) {
	const (
		bufSize = 1500
	)
	var out BulkInResponse
	term := byte(0x10)
	hdr := encBulkInHeader(d.tagger, bufSize, &term) // 0x10 == '\n'
	n, err := d.out.Write(hdr[:])                    // [:] fixed size array to byte slice
	if err != nil {                                  // problem in transmission
		return out, err
	}
	if n < 12 { // incomplete transmission
		nOld := n
		// attempt a second write
		hdrB := hdr[n:]
		n, err = d.out.Write(hdrB)
		total := n + nOld
		if err != nil {
			return out, err
		}
		if total != 12 {
			return out, fmt.Errorf("wrote %d bytes, not full 12 required to transmit read request", total)
		}
	}
	// if this line was reached, the entire request succeeded, now we can actually do the read
	buf := make([]byte, bufSize) // 1 TCP MTU, not even related to USB but pretty big, good enough for now
	n, err = d.in.Read(buf)
	if err != nil {
		return out, err
	}
	if n < 12 {
		return out, fmt.Errorf("only received %d bytes, need at least 12 to form header", n)
	}
	// if this line was reached, we can pop the header and return a bonafide response
	buf = buf[:n]
	out.Header = buf[:12] // first 12 bytes are the header
	out.Data = buf[12:]   // remainder are the datagram
	return out, nil
}

func (d *USBDevice) Write(b []byte) error {
	const (
		alignment = 4
	)
	hdr := encBulkOutHeader(d.tagger, len(b))
	b = append(hdr[:], b...) // [:] array => slice of underlying values

	if residual := len(b) % alignment; residual > 0 {
		paddingLen := alignment - residual
		// panguage spec that the contents of the slice is zero value
		padding := make([]byte, paddingLen)
		b = append(b, padding...)
	}
	_, err := d.out.Write(b)
	if err != nil {
		return err
	}
	return nil
}

// Close closes the device
func (d *USBDevice) Close() error {
	d.closer()
	return d.device.Close()
}
