package main

import (
	"log"
	"net/http"

	"github.jpl.nasa.gov/HCIT/go-hcit/generichttp"
	"goji.io"

	"github.jpl.nasa.gov/HCIT/go-hcit/thorlabs"
)

// const (
// 	// TLVID is the ThorLabs vendor ID
// 	TLVID = 0x1313

// 	// LDC4001PID is the Product ID for the LDC4001 laser diode / TEC controller
// 	LDC4001PID = 0x804a
// )

func main() {
	ldc, err := thorlabs.NewITC4000()
	if err != nil {
		log.Fatal(err)
	}
	wrap := generichttp.NewHTTPLaserController(&ldc)
	mux := goji.NewMux()
	wrap.RT().Bind(mux)
	log.Fatal(http.ListenAndServe(":8001", mux))
}
