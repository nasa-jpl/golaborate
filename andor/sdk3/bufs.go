package sdk3

/*
#cgo CFLAGS: -I/usr/local
#cgo LDFLAGS: -L/usr/local/lib -latcore -latutility
#include <stdlib.h>
#include <atcore.h>
#include <shim.h>

*/
import "C"
import "unsafe"

// buffer is an 8-byte aligned block of memory owned by C used for image readout
type buffer struct {
	// cptrhead points to the head of the
	// alloc, which may not be 8-byte aligned
	// cptr points to the 8-byte aligned position
	// for the andor SDK's usages
	cptr      *C.AT_U8
	cptrsize  C.int
	size      int
	allocated bool
}

func (b *buffer) Alloc(nbytes int) {
	b.cptr = (*C.AT_U8)(C.aligned_malloc2(C.int(nbytes), 8))
	b.cptrsize = C.int(nbytes)
	b.size = nbytes
	b.allocated = true
	return
}

func (b *buffer) Free() {
	C.aligned_free2(unsafe.Pointer(b.cptr))
	b.allocated = false
}
