package acromag

/*
#cgo LDFLAGS: -lm
#include <stdlib.h>
#include "apcommon.h"
*/
import "C"
import "fmt"

func init() {
	errCode := C.InitAPLib()
	if errCode != C.S_OK {
		panicS := fmt.Sprintf("initializing Acromag library failed with code %d", errCode)
		panic(panicS)
	}
}
