package sdk3

/*
#cgo CFLAGS: -I/usr/local
#cgo LDFLAGS: -L/usr/local/lib -latcore -latutility
#include <stdlib.h>
#include <atcore.h>

*/
import "C"
import "unsafe"

// buffer is an 8-byte aligned block of memory owned by Go for passing to C
type buffer struct {
	buf      []uint64
	cptr     *C.AT_U8
	cptrsize C.int
	gptr     unsafe.Pointer
}
