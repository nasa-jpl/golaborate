package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func gettemp() float64 {
	return 22 + rand.Float64()
}

func gethum() float64 {
	return 6 + rand.Float64()
}

// benchTemp = prometheus.NewGauge(prometheus.GaugeOpts{
// 	Name: "zygo_bench_temp_celcius",
// 	Help: "Current temperature of Zygo bench inside the DM shroud.",
// })

func main() {
	if err := prometheus.Register(prometheus.NewGaugeFunc(
		prometheus.GaugeOpts{
			Subsystem: "lab",
			Name:      "zygo_bench_temp_celcius",
			Help:      "Current temperature of Zygo bench inside the DM shroud.",
		},
		gettemp,
	)); err == nil {
		fmt.Println("GaugeFunc 'zygo_bench_temp_celcius' registered.")
	}

	if err := prometheus.Register(prometheus.NewGaugeFunc(
		prometheus.GaugeOpts{
			Subsystem: "lab",
			Name:      "zygo_bench_relative_humidity",
			Help:      "Current humidity of Zygo bench inside the DM shroud.",
		},
		gethum,
	)); err == nil {
		fmt.Println("GaugeFunc 'zygo_bench_relative_humidity' registered.")
	}

	// The Handler function provides a default handler to expose metrics
	// via an HTTP server. "/metrics" is the usual endpoint for that.
	http.Handle("/metrics", promhttp.Handler())
	log.Fatal(http.ListenAndServe(":8080", nil))
}
