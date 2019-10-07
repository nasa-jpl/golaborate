package main

import (
	"log"
	"net/http"

	//gp "github.jpl.nasa.gov/HCIT/go-hcit/granvillephillips"
	// "github.jpl.nasa.gov/HCIT/go-hcit/omega"
	"github.jpl.nasa.gov/HCIT/go-hcit/fluke"
	lw "github.jpl.nasa.gov/HCIT/go-hcit/ixllightwave"
	"github.jpl.nasa.gov/HCIT/go-hcit/lesker"
)

func main() {
	// pressureGauge, err := gp.NewGuage("/dev/ttyUSB3")
	// pressureGauge, err := lesker.NewGauge("/dev/ttyUSB5")
	// pressureGauge, err := omega.NewMeter("/dev/ttyUSB4")
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// bind the IXL lightwave
	ldc := lw.LDC3916{Addr: "192.168.100.40:2106"}
	ldc.BindRoutes("/ldc")

	// bind the Zygo
	zygobench := fluke.DewK{Addr: "192.168.100.71", Conntype: "TCP", Name: "Zygo bench"}
	zygobench.BindRoutes("/zygo-bench")

	// bind the lesker
	pgauge := lesker.NewSensor("192.168.100.187:2113", "TCP")
	pgauge.BindRoutes("/dst")
	log.Fatal(http.ListenAndServe("localhost:8080", nil))
}
