package main

import (
	"log"
	"net/http"

	"github.jpl.nasa.gov/HCIT/go-hcit/nkt"
	"github.jpl.nasa.gov/HCIT/go-hcit/server"
)

func main() {

	m := nkt.NewSuperKExtreme("192.168.100.40:2104", "/omc/nkt", false)
	mm := nkt.NewSuperKVaria("192.168.100.40:2104", "/omc/nkt", false)
	mf := server.Mainframe{}
	mf.Add(m)
	mf.Add(mm)
	mf.BindRoutes()

	log.Fatal(http.ListenAndServe(":8080", nil))
}
