package main

import (
	"log"
	"net/http"
	"time"

	"github.jpl.nasa.gov/HCIT/go-hcit/envsrv"
)

func main() {
	mon := envsrv.New(1*time.Minute, 60*24) // 60 min/hr * 24hr.
	mon.Start()
	http.HandleFunc("/history", mon.HTTPYield)
	log.Fatal(http.ListenAndServe(":8082", nil))
}
