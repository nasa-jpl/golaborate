// Package mccdaq provides A Go interface to MCC DACs and ADCs
package mccdaq

/*
#cgo CFLAGS: -I/usr/local
#cgo LDFLAGS: -L/usr/local/lib -luldaq
#include <stdlib.h>
#include <uldaq.h>

*/
import "C"
import "fmt"

// DAC is an interface to a D/A converter
type DAC struct {
	handle C.DaqDeviceHandle
}

// Open opens a new connection to a DAC.  This always opens a connection to the first DAC
// and would need to be refactored to work with others
func Open() (*DAC, error) {
	// this function is largely "copy/pasted" (transpiled to C/Go) from the example on
	// github.com/mccdaq/uldac under "Usage"
	var (
		ret     *DAC
		ary     [1]C.DaqDeviceDescriptor
		numdevs C.uint = 1
	)

	C.ulGetDaqDeviceInventory(C.ANY_IFC, &ary[0], &numdevs)
	if numdevs == 0 {
		return ret, fmt.Errorf("No DACs detected")
	}
	ret.handle = C.ulCreateDaqDevice(ary[0])
	if ret.handle == 0 {
		return ret, fmt.Errorf("Connection to DAC not opened properly")
	}

	err := C.ulConnectDaqDevice(ret.handle)
	if err != C.ERR_NO_ERROR {
		return ret, fmt.Errorf("Connection to DAC not opened properly [%v]", err)
	}

	return ret, nil
}

// Write sends a float to the DAC on a certain channel
func (d *DAC) Write(channel int, data float64) error {
	// BIP10VOLTS -> bipolar, +/- 10V
	// AOUT_FF_DEFAULT, "Scaled data is supplied and calibration factors are applied to output.""
	C.ulAOut(d.handle, C.int(channel), C.BIP10VOLTS, C.AOUT_FF_DEFAULT, C.double(data))
	return nil
}

// Close removes the connection to the DAC and releases the device
func (d *DAC) Close() error {
	C.ulDisconnectDaqDevice(d.handle)
	C.ulReleaseDaqDevice(d.handle)
	return nil
}
