package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.jpl.nasa.gov/HCIT/go-hcit/envsrv"
	"goji.io"
)

const (
	helpBlurb = `
Usage: envsrv [CONFIGPATH]
Example:
envsrv cfg.yaml
cat cfg.yaml
Flukes:
  - addr: "192.168.100.71"
    url: /zygo-bench

NKTs:
  - addr: "192.168.100.40:2106"
    url: /omc/nkt

IXLLightwaves:
  - addr: "192.168.100.40:2106"
    url: /omc/ixl-diode

Leskers:
  - addr: "192.168.100.187:2113"
    url: /dst/lesker

  GPConvectrons:
  - addr: "192.168.100.41:2106"
    url: /dst/convectron

envsrv cfg.yaml
`
)

func main() {
	if len(os.Args) == 1 || os.Args[1] == "help" {
		fmt.Println(helpBlurb)
		return
	}
	cfg, err := envsrv.LoadYaml(arg)
	if err != nil {
		panic(err)
	}
	mainframe := envsrv.SetupDevices(cfg)
	mux := goji.NewMux()
	mainframe.BindRoutes(mux)

	log.Println("envsrv started bound to :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
