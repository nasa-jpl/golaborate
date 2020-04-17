// Package bmc provides control of BMC deformable mirrors
package bmc

/*
#cgo CFLAGS: -I"/opt/Boston Micromachines/include"
#cgo LDFLAGS: -L"/opt/Boston Micromachines/lib" -Wl,-rpath,"/opt/Boston Micromachines/lib" -lBMC
#include <stdlib.h>
#include <stdio.h>
#include <BMCApi.h>
*/
import "C"
import (
	"fmt"
	"unsafe"
)

// Error is an Error satisfying struct
type Error struct {
	code int
	txt string
}

func (err Error) Error() string {
	return fmt.Sprintf("%d - %s", err.code, err.txt)
}

// ctoGoErr converts a C error to a Go error
func ctoGoErr(i C.BMCRC) error {
	ig := int(i)
	if ig == 0 {
		return nil
	}
	cstr := C.BMCErrorString(i) // these are static and should not be freed
	gostr := C.GoString(cstr)
	return Error{code: ig, txt: gostr}
}

// Zero applies a zero voltage to the DM, putting it in a safe condition
func Zero(dm *DM) error {
	return nil
}

// DM encapsulates the
type DM struct {
	raw *C.struct_DM
}

// Open opens the connection to the DM driver
func Open(sn string) (*DM, error) {
	dm := DM{}
	// allocate the struct, as the SDK expects
	var raw C.struct_DM
	// per B. C. Mills 2020-01-08, no need to do anything to allocate.


	// convert the Go string to a C string and free it later
	cstr := C.CString(sn)
	defer C.free(unsafe.Pointer(cstr))

	// cerr is a C.int
	err := ctoGoErr(C.BMCOpen(&raw, cstr))
	dm.raw = &raw
	return &dm, err
}

// Close closes the connection to the DM driver
func (dm *DM) Close() error {
	return ctoGoErr(C.BMCClose(dm.raw))
}

// LoadMap loads an actuator map, if "", loads the default profile determined by the SDK
func (dm *DM) LoadMap(path string) error {
	if path == "" {
		return ctoGoErr(C.BMCLoadMap(dm.raw, nil, nil))
	}
	cstr := C.CString(path)
	defer C.free(unsafe.Pointer(cstr))
	return ctoGoErr(C.BMCLoadMap(dm.raw, cstr, nil)) // nil == C NULL, causes BMC SDK to internally do the allocation
}

// GetArray queries the DM driver for the last array of values sent to it
func (dm *DM) GetArray() ([]float64, error) {
	ary := make([]float64, dm.Actuators())
	ptr := (*C.double)(&ary[0])
	err := ctoGoErr(C.BMCGetArray(dm.raw, ptr, C.uint32_t(len(ary))))
	return ary, err
}

// SetArray sets the value for all actuators.  values must be in the range [0,1] or they are clamped by the BMC SDK.
func (dm *DM) SetArray(values []float64) error {
	ptr := (*C.double)(&values[0])
	return ctoGoErr(C.BMCSetArray(dm.raw, ptr, nil))
}

// SetSingle sets the voltage for a single actuator
func (dm *DM) SetSingle(actidx int, value float64) error {
	return ctoGoErr(C.BMCSetSingle(dm.raw, C.uint32_t(actidx), C.double(value)))
}

func (dm *DM) Actuators() int {
	return int(dm.raw.ActCount) // uint in C
}
