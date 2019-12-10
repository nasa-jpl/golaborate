package main

import (
	"log"
	"net/http"

	"github.jpl.nasa.gov/HCIT/go-hcit/aerotech"
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

// func main() {
// 	if len(os.Args) == 1 || os.Args[1] == "help" {
// 		fmt.Println(helpBlurb)
// 		return
// 	}
// 	cfg, err := envsrv.LoadYaml(arg)
// 	if err != nil {
// 		panic(err)
// 	}
// 	mux := envsrv.SetupDevices()

// 	log.Println("envsrv started bound to :8080")
// 	log.Fatal(http.ListenAndServe(":8080", mux))
// }

func main() {
	at := aerotech.NewEnsemble("192.168.100.154:8000", false)
	httper := aerotech.NewHTTPWrapper(*at)
	mux := goji.NewMux()
	httper.RT().Bind(mux)
	log.Fatal(http.ListenAndServe(":8083", mux))
}
