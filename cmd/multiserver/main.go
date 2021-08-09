package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/knadh/koanf"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/providers/structs"

	yml "gopkg.in/yaml.v2"
)

var (
	// Version is the version number.  Typically injected via ldflags with git build
	Version = "6"

	// ConfigFileName is what it sounds like
	ConfigFileName = "multiserver.yml"
	k              = koanf.New(".")
)

func setupconfig() {
	k.Load(structs.Provider(Config{
		Addr:  ":8000",
		Nodes: []ObjSetup{}}, "koanf"), nil)
	if err := k.Load(file.Provider(ConfigFileName), yaml.Parser()); err != nil {
		errtxt := err.Error()
		if !strings.Contains(errtxt, "no such") { // file missing, who cares
			log.Fatalf("error loading config: %v", err)
		}
	}
}

func root() {
	str := `multiserver communicates with lab hardware and exposes an HTTP interface to them
This enables a server-client architecture, and the clients can leverage the
excellent HTTP libraries for any programming language.

Usage:
	multiserver <command>

Commands:
	run
	help
	mkconf
	conf
	version`
	fmt.Println(str)
}

func help() {
	str := `multiserver is amenable to configuration via its .yaml file.  For a primer on YAML, see
https://yaml.org/start.html

Without a configuration, the server will close immediately and display an error
that there are no endpoints.

No two endpoints can have the same URL.

URLs may look like any variation between "omc/nkt" or "/omc/nkt/*", the leading
and trailing slashes, as well as the *, are added by the server if missing.

All hardware are supported on all common platforms (Windows, Linux, OSX).

Hardware and matching "type" fields, case insensitive, alphabetical by vendor:
- Aerotech:
	> Ensemble "aerotech", "ensemble"
- Cryocon:
	> model 12, 14, 18i "cryocon"
- Fluke
	> DewK 1620a "fluke", "dewk"
- Granville-Phillips
	> GP375 Convectron "gp", "convectron", "gpconvectron"
- IXL Lightwave
	> LDC3916, "lightwave", "ldc3916", "ixl"
- Lesker
	> KJC pressure sensor, "kjc", "lesker"
- Newport
	> ESP300 / ESP301 "esp", "esp300", "esp301"
	> XPS "xps"
- NKT
	> SuperK Extreme / SuperK Varia "nkt", "superk"
- Thermocube
	> 200, 300, 400 series "cube" fluid temperature controllers, "thermocube"
- Thorlabs
	> ITC 4000 series "itc4000", "tl-laser-diode"`
	/*
	   - Omega
	   	> DPF700 (flow meter) "dpf700" [ not actually working yet ]
	   	>
	*/
	fmt.Println(str)
}

func mkconf() {
	c := Config{}
	err := k.Unmarshal("", &c)
	if err != nil {
		log.Fatal(err)
	}
	f, err := os.Create(ConfigFileName)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	err = yml.NewEncoder(f).Encode(c)
	if err != nil {
		log.Fatal(err)
	}
}

func printconf() {
	c := Config{}
	k.Unmarshal("", &c)
	err := yml.NewEncoder(os.Stdout).Encode(c)
	if err != nil {
		log.Fatal(err)
	}
}

func pversion() {
	fmt.Printf("multiserver version %v\n", Version)
}

func run() {
	c := Config{}
	err := k.Unmarshal("", &c)
	if err != nil {
		log.Fatal(err)
	}
	mux := BuildMux(c)
	log.Println("now listening for requests at ", c.Addr)
	log.Fatal(http.ListenAndServe(c.Addr, mux))
}

func main() {
	var cmd string
	args := os.Args
	if len(args) == 1 {
		root()
		return
	}
	setupconfig()
	cmd = args[1]
	cmd = strings.ToLower(cmd)
	switch cmd {
	case "help":
		help()
		return
	case "mkconf":
		mkconf()
		return
	case "conf":
		printconf()
		return
	case "run":
		run()
		return
	case "version":
		pversion()
		return
	default:
		log.Fatal("unknown command")
	}
}
