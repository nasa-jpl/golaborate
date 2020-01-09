package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.jpl.nasa.gov/HCIT/go-hcit/imgrec"

	"goji.io/pat"

	"goji.io"

	"github.com/knadh/koanf"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/providers/structs"
	"github.jpl.nasa.gov/HCIT/go-hcit/andor/sdk3"
	"github.jpl.nasa.gov/HCIT/go-hcit/server"

	yml "gopkg.in/yaml.v2"
)

var (
	// Version is the version number.  Typically injected via ldflags with git build
	Version = "8"

	// ConfigFileName is what it sounds like
	ConfigFileName = "andor-http.yml"
	k              = koanf.New(".")
)

type recorder struct {
	// Root is the root folder to write to
	Root string `yaml:"Root"`

	// Prefix is the filename prefix to use
	Prefix string `yaml:"Prefix"`
}
type config struct {
	Addr         string                 `yaml:"Addr"`
	Root         string                 `yaml:"Root"`
	SerialNumber string                 `yaml:"SerialNumber"`
	Recorder     recorder               `yaml:"Recorder"`
	BootupArgs   map[string]interface{} `yaml:"BootupArgs"`
}

func setupconfig() {
	k.Load(structs.Provider(config{
		Addr:         ":8000",
		Root:         "/",
		SerialNumber: "auto",
		Recorder:     recorder{},
		BootupArgs: map[string]interface{}{
			"ElectronicShutteringMode": "Rolling",
			"SimplePreAmpGainControl":  "16-bit (low noise & high well capacity)",
			"FanSpeed":                 "Low",
			"PixelReadoutRate":         "280 MHz",
			"PixelEncoding":            "Mono16",
			"TriggerMode":              "Internal",
			"MetadataEnable":           false,
			"SensorCooling":            true,
			"SpuriousNoiseFilter":      false}}, "koanf"), nil)
	if err := k.Load(file.Provider(ConfigFileName), yaml.Parser()); err != nil {
		errtxt := err.Error()
		if !strings.Contains(errtxt, "no such") { // file missing, who cares
			log.Fatalf("error loading config: %v", err)
		}
	}
}
func root() {
	str := `andor-http exposes control of andor Neo cameras over HTTP
This enables a server-client architecture,
and the clients can leverage the excellent HTTP
libraries for any programming language,
instead of custom socket logic.

Usage:
	andor-http <command>

Commands:
	run
	help
	mkconf
	conf
	version`
	fmt.Println(str)
}

func help() {
	str := `andor-http is amenable to configuration via its .yaml file.  For a primer on YAML, see
https://yaml.org/start.html

When no configuration is provided, the defaults are used.  Keys are not case-sensitive.
The command mkconf generates the configuration file with the default values.
There is no need to do this unless you want to start from the prepopulated defaults when making
a config file.

If for some reason there is an error during server bootup, it may be that a feature is not supported by the camera.
Modify the BootupArgs portion of the config to remove the offending parameters.

serialNumber 'auto' causes the server to scan the available cameras and pick the first one
which is not a software simulation camera.`
	fmt.Println(str)
}

func mkconf() {
	c := config{}
	err := k.Unmarshal("", &c)
	if err != nil {
		log.Fatal(err)
	}
	f, err := os.Create(ConfigFileName)
	defer f.Close()
	if err != nil {
		log.Fatal(err)
	}
	err = yml.NewEncoder(f).Encode(c)
	if err != nil {
		log.Fatal(err)
	}
}

func printconf() {
	c := config{}
	k.Unmarshal("", &c)
	err := yml.NewEncoder(os.Stdout).Encode(c)
	if err != nil {
		log.Fatal(err)
	}
}

func pversion() {
	fmt.Printf("andor-http version %v\n", Version)
}

func run() {
	// load the library and see how many cameras are connected
	err := sdk3.InitializeLibrary()
	if err != nil {
		log.Fatal(err)
	}
	defer sdk3.FinalizeLibrary()
	ncam, err := sdk3.DeviceCount()
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("There are %d cameras connected\n", ncam)
	swver, err := sdk3.SoftwareVersion()
	if err != nil {
		log.Fatal(err)
		return
	}
	log.Printf("SDK version is %s\n", swver)

	// now scan for the right serial number
	// c escapes into the outer scope
	sn := k.String("SerialNumber")
	var (
		c     *sdk3.Camera
		snCam string
	)
	for idx := 0; idx < ncam; idx++ {
		c, err = sdk3.Open(idx)
		if err != nil {
			log.Fatal(err)
		}
		snCam, err = c.GetSerialNumber()
		if err != nil {
			c.Close()
			log.Fatal(err)
		}
		if sn == "auto" {
			if !strings.Contains(sn, "SFT") {
				break
			} else {
				c.Close()
			}
		} else {
			if sn == snCam {
				break
			} else {
				c.Close()
			}
		}
	}
	model, err := c.GetModel()
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("connected to %s SN %s\n", model, snCam)

	err = c.Configure(k.Get("BootupArgs").(map[string]interface{}))
	if err != nil {
		log.Fatal(err)
	}
	c.Allocate()
	err = c.QueueBuffer()

	args := k.Get("Recorder").(recorder)
	r := &imgrec.Recorder{Root: args.Root, Prefix: args.Prefix}
	w := sdk3.NewHTTPWrapper(c, r)

	// clean up the submux string
	hndlrS := k.String("Root")
	hndlrS = server.SubMuxSanitize(hndlrS)
	root := goji.NewMux()
	mux := goji.SubMux()
	root.Handle(pat.New(hndlrS), mux)
	w.RT().Bind(mux)
	addr := k.String("Addr") + k.String("Root")
	log.Println("now listening for requests at ", addr)
	log.Fatal(http.ListenAndServe(k.String("Addr"), root))
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
