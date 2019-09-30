package main

import (
	"log"
	"net/http"

	"github.jpl.nasa.gov/HCIT/go-hcit/fluke"
)

func main() {
	zygobench := fluke.DewK{Addr: "192.168.100.71", Conntype: "TCP", Name: "Zygo bench"}
	// zygobench := fluke.MockDewK{}

	http.HandleFunc("/sensors/zygo", zygobench.ReadAndReplyWithJSON)

	log.Fatal(http.ListenAndServe("localhost:8080", nil))
}
