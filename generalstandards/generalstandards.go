// Package generalstandards provides a Go interface to General Standards 16ao16 DACs through their C SDK.
package generalstandards

/*
#cgo CFLAGS: -I/usr/local -I/usr/src/linux/drivers/16ao16/include
#include <stdlib.h>
#include <16ao16_main.h>
*/
import "C"
import "fmt"

// DAC provides an object oriented Go interface to a 16ao16 dac.
type DAC struct {
	fd C.int

	// Vmax is the maximum voltage, Volts
	Vmax float64

	// Vmin is the minimum voltage, Volts
	Vmin float64
}

// Initialize sets up the DAC.
// This routine satisfies only the needs of HCIT, and has many values pre-programmed
// such as calibration, on-board clocks on the DAC, etc.
func (d *DAC) Initialize() error {
	// step 1, initialize the C library
	ret := int(C.ao16_init())
	if ret != 0 {
		return fmt.Errorf("error code %d", ret)
	}

	// step 2, open the connection to the device.
	// open in exclusive mode, explicitly DAC 0
	// argument order device index, shared mode, file descriptor
	ret = int(C.ao16_open(C.int(0), C.int(0), &d.fd))
	if ret != 0 {
		return fmt.Errorf("error code %d", ret)
	}

	// now set the range to +/- 10V
	ret = int(C.ao16_ioctl_dsl(d.fd, C.AO16_IOCTL_RANGE, &C.AO16_RANGE_10))
	if ret != 0 {
		return fmt.Errorf("error code %d", ret)
	}

	// put the DAC in simultaneous output mode
	ret = int(C.ao16_ioctl_dsl(d.fd, C.AO16_IOCTL_OUTPUT_MODE, &C.AO16_OUTPUT_MODE_SIM))
	if ret != 0 {
		return fmt.Errorf("error code %d", ret)
	}

	// enable burst trigger mode
	ret = int(C.ao16_ioctl_dsl(d.fd, C.AO16_IOCTL_BURST_ENABLE, &C.AO16_BURST_ENABLE_YES))
	if ret != 0 {
		return fmt.Errorf("error code %d", ret)
	}

	// enable on-board clock
	ret = int(C.ao16_ioctl_dsl(d.fd, C.AO16_IOCTL_CLOCK_ENABLE, &C.AO16_CLOCK_ENABLE_YES))
	if ret != 0 {
		return fmt.Errorf("error code %d", ret)
	}

	// enable demand access mode
	// Tuong had _DMA, manual has _DMDMA
	ret = int(C.ao16_ioctl_dsl(d.fd, C.AO16_IOCTL_TX_IO_MODE, &C.GSC_IO_MODE_DMDMA))
	if ret != 0 {
		return fmt.Errorf("error code %d", ret)
	}

	// autocalibrate
	ret = int(C.ao16_ioctl_dsl(d.fd, C.AO16_IOCTL_AUTO_CALIBRATE))
	if ret != int(C.AO16_AUTO_CAL_STS_PASS) {
		return fmt.Errorf("error code %d", ret)
	}
	return nil
}

// Output sets the output value on a given DAC channel to a floating point value
// in volts.  If the DAC is buffering, the data is actually enqueued
func (d *DAC) Output(channel int, value float64) {
	C.ao16_channel_sel(d.fd)
}

// volt_to_u16 clamps a voltage and converts it to the u16 value
// based on offset binary encoding
func volt_to_u16(val float64, vrange float64) uint16 {
	if val > vrange {
		return 0xFFFF
	} else if val < -vrange {
		return 0x0000
	}
	return uint16((val + vrange) / 20 * 0xFFFF)
}
