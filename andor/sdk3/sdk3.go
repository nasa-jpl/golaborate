/*Package sdk3 exposes control of Andor cameras in Go via their SDK, v3.

 */
package sdk3

/*
#cgo CFLAGS: -I/usr/local
#cgo LDFLAGS: -L/usr/local/lib -latcore -latutility
#include <stdlib.h>
#include <atcore.h>

*/
import "C"
import (
	"fmt"
	"reflect"
	"time"
	"unsafe"
)

const (
	// LengthOfUndefinedBuffers is how large a buffer to allocate for a Wchar
	// string when we have no way of knowing ahead of time how big it is
	// it is measured in Wchars
	LengthOfUndefinedBuffers = 255
)

// Camera represents a camera from SDK3
type Camera struct {
	// buffer is written to by the SDK.
	// It must be 8-byte aligned.
	buffer []uint64

	// cptr is a C pointer to the first byte in buffer
	cptr *C.AT_U8

	// cptrsize is the 'size' of the c pointer
	cptrsize C.int

	// gptr is a Go pointer to the first byte in buffer
	gptr unsafe.Pointer

	// bufferOnQueue is a flag indicating if we have put a buffer onto the SDK's
	// queue yet
	bufferOnQueue bool

	// Resolution holds the image resolution (H, W)
	Resolution [2]int

	// Handle holds the int that points to a specific camera
	Handle int
}

// npx is shorthand for c.Resolution[0] * c.Resolution[1]
func (c *Camera) npx() int {
	return c.Resolution[0] * c.Resolution[1]
}

// updateRes updates the resolution to the current pixel dimensions

// Allocate creates the buffer that will be populated by the SDK
// it should be called at init, and whenever the AOI or encoding changes
func (c *Camera) Allocate() error {
	sze, err := GetInt(c.Handle, "ImageSizeBytes")
	if err != nil {
		return err
	}
	c.buffer = make([]uint64, sze/8) // uint64 forces byte alignment, 8 bytes per uint64
	c.gptr = unsafe.Pointer(&c.buffer[0])
	c.cptr = (*C.AT_U8)(c.gptr)
	c.cptrsize = C.int(sze)
	return nil
}

// Open opens a connection to the camera.  camIdx 1 is the system handle,
// start at 2 for the first camera, 3 for the second, and so forth
func Open(camIdx int) (*Camera, error) {
	c := Camera{}
	var hndle C.AT_H
	err := Error(int(C.AT_Open(C.int(camIdx), &hndle)))
	c.Handle = int(hndle)
	return &c, err
}

// Close closes a connection to the camera
func (c *Camera) Close() error {
	return Error(int(C.AT_Close(C.AT_H(c.Handle))))
}

// QueueBuffer puts the Camera's internal buffer into the write queue for the SDK
// only one buffer is supported in this wrapper, though the SDK supports
// multiple buffers
func (c *Camera) QueueBuffer() error {
	npx := c.npx()
	if len(c.buffer) == 0 || cap(c.buffer) < npx {
		return fmt.Errorf("Go buffer cannot hold entire frame, likely uninitialized, len=%d, cap=%d, npx=%d", len(c.buffer), cap(c.buffer), npx)
	}
	err := Error(int(C.AT_QueueBuffer(C.AT_H(c.Handle), c.cptr, c.cptrsize)))
	if err == nil {
		c.bufferOnQueue = true
	}
	return err
}

// WaitBuffer waits for the camera to push a frame into the buffer
// errors if Queue has not been called, on timeout, or on an SDK error
func (c *Camera) WaitBuffer(timeout time.Duration) error {
	if !c.bufferOnQueue {
		return ErrBufferNotOnQueue
	}
	tout := C.uint(timeout.Nanoseconds() / 1e6)
	var (
		size C.int
		ptr  *C.AT_U8
	)
	err := Error(int(C.AT_WaitBuffer(C.AT_H(c.Handle), &ptr, &size, tout)))
	return err
}

// ShutDown shuts down the camera and 'finalizes' the SDK.
// This maintains API compatibility with SDK2
// and is not an exact match of
func (c *Camera) ShutDown() {
	return
}

// ExtCamera is an extension of the Camera type with some helper and 'macro'
// functions
type ExtCamera struct {
	*Camera
}

// LastFrame returns the current state of the buffer cast to int32
// This is not in the SDK3, but is something we provide to build
// better interfaces.
//
// This function does not copy and uses unsafe mechanisms internally,
// buyer beware.
func (c *ExtCamera) LastFrame() (*[]uint16, error) {
	if !c.bufferOnQueue {
		return nil, ErrBufferNotOnQueue
	}
	nbytes, err := GetInt(c.Handle, "ImageSizeBytes")
	nbytes = nbytes / 2
	aryuint16 := []uint16{}
	if err != nil {
		return &aryuint16, err
	}
	hdr := (*reflect.SliceHeader)(unsafe.Pointer(&aryuint16))
	hdr.Data = uintptr(unsafe.Pointer(&c.buffer[0]))
	hdr.Len = nbytes
	hdr.Cap = nbytes
	return &aryuint16, nil
}

// BufferCopy returns a copy of the current buffer at this moment in time
// may have undefined behavior if camera is writing while you read
//
// This is a copy, so do not do this inside of a a hot loop (e.g. running at 100fps)
func (c *ExtCamera) BufferCopy() ([]byte, error) {
	buf := []byte{}
	nbytes, err := GetInt(c.Handle, "ImageSizeBytes")
	if err != nil {
		return buf, err
	}
	hdr := (*reflect.SliceHeader)(unsafe.Pointer(&buf))
	hdr.Data = uintptr(unsafe.Pointer(&c.buffer[0]))
	hdr.Len = nbytes
	hdr.Cap = nbytes
	return buf, nil
}

// JPLNeoBootup runs the standard bootup sequence used for Andor Neo cameras in HCIT
// this should be called after Initialize
func JPLNeoBootup(c *Camera) error {
	fmt.Println("eshutter")
	err := SetEnumString(c.Handle, "ElectronicShutteringMode", "Rolling")
	if err != nil {
		return err
	}
	fmt.Println("preampgain skipped - notimplemented on simcam")
	// err = SetEnumString(c.Handle, "SimplePreAmpGainControl", "16-bit (low noise & high well capacity)")
	// if err != nil {
	// 	return err
	// }
	fmt.Println("fan speed")
	err = SetEnumString(c.Handle, "FanSpeed", "Off")
	if err != nil {
		return err
	}
	fmt.Println("pixel readout rate")
	// err = SetEnumString(c.Handle, "PixelReadoutRate", "280 MHz")
	err = SetEnumIndex(c.Handle, "PixelReadoutRate", 0)
	if err != nil {
		return err
	}

	fmt.Println("pixel encoding")
	// err = SetEnumIndex(c.Handle, "PixelEncoding", 2)
	err = SetEnumString(c.Handle, "PixelEncoding", "Mono12")
	// strs, err := GetEnumStrings(c.Handle, "PixelEncoding")
	// fmt.Println(strs, err)
	if err != nil {
		return err
	}

	fmt.Println("triggermode")
	err = SetEnumString(c.Handle, "TriggerMode", "Internal")
	if err != nil {
		return err
	}

	fmt.Println("metadata - skipped simcam")
	// err = SetBool(c.Handle, "MetadataEnable", true)
	// if err != nil {
	// 	return err
	// }

	// fmt.Println("metadatats")
	// err = SetBool(c.Handle, "MetadataTimestamp", true)
	// if err != nil {
	// 	return err
	// }

	fmt.Println("cooling")
	err = SetBool(c.Handle, "SensorCooling", false)
	if err != nil {
		return err
	}

	fmt.Println("spurriousnoise - skipped simcak")
	// err = SetBool(c.Handle, "SpuriousNoiseFilter", false)
	// if err != nil {
	// 	return err
	// }

	return nil
}

// UnpadBuffer strips padding bytes from a buffer
func (c *Camera) UnpadBuffer(buf []byte) ([]byte, error) {
	// TODO: this allocates something bigger than needed
	// can improve performance a little bit by changing this
	out := make([]byte, 0, len(buf))
	stride, err := GetInt(c.Handle, "AOIStride")
	if err != nil {
		return out, err
	}
	width, err := GetInt(c.Handle, "AOIWidth")
	if err != nil {
		return out, err
	}
	height, err := GetInt(c.Handle, "AOIHeight")
	if err != nil {
		return out, err
	}

	// TODO: generalize this to other modes besides 16-bit
	bidx := 0                    // byte index
	bpp := 2                     // bytes per pixel
	rowWidthBytes := bpp * width // width (stride) or a row in bytes
	// implicitly row major order, but seems to be from the SDK
	for row := 0; row < height; row++ {
		bytes := buf[bidx : bidx+rowWidthBytes]
		out = append(out, bytes...)
		// finally, move
		bidx += stride // stride is the padded stride
	}
	return out, nil
}
