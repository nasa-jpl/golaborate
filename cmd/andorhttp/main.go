package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.jpl.nasa.gov/HCIT/go-hcit/server"

	"github.com/spf13/viper"

	"github.jpl.nasa.gov/HCIT/go-hcit/andor/sdk3"

	_ "net/http/pprof"
)

var (
	// Version is the version number.  Typically injected via ldflags with git build
	Version = "dev"
)

type kvp map[string]interface{}

type config struct {
	// Addr is the bind-to address, like ":8000" to listen to any remote on port 8000
	Addr string

	BootupArgs kvp
}

func setupviper() {
	viper.SetConfigName("andor-http")
	viper.AddConfigPath(".")
	viper.SetDefault("Addr", ":8000")
	viper.SetDefault("SerialNumber", "auto")
	viper.SetDefault("BootupArgs", kvp{
		"ElectronicShutteringMode": "Rolling",
		"SimplePreAmpGainControl":  "16-bit (low noise & high well capacity)",
		"FanSpeed":                 "Low",
		"PixelReadoutRate":         "280 Mhz",
		"PixelEncoding":            "Mono16",
		"TriggerMode":              "Internal",
		"MetaDataEnable":           false,
		"SensorCooling":            true,
		"SpuriousNoiseFilter":      false})
}
func root() {
	str := `andor-http is an application that exposes control of andor Neo cameras over HTTP.
This enables a server-client architecture, and the clients can leverage the excellent HTTP
libraries for any programming language, instead of custom socket logic.

Usage:
	andor-http <command>

Commands:
	run
	help
	mkconf`
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
	err := viper.ReadInConfig()
	if err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// pass
		} else {
			log.Fatalf("loading of config file failed %q", err)
		}
	}
	err = viper.WriteConfigAs("andor-http.yaml")
	if err != nil {
		log.Fatalf("writing of config file failed %q", err)
	}
	return
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
	log.Println("reached top of sn scan block")
	sn := viper.GetString("SerialNumber")
	var c *sdk3.Camera
	var snCam string
	for idx := 0; idx < ncam; idx++ {
		c, err := sdk3.Open(idx)
		if err != nil {
			log.Fatal(err)
		}
		snCam, err := c.GetSerialNumber()
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
	log.Println("reached bottom of sn scan block")
	log.Println(snCam)
	model, err := c.GetModel()
	log.Println(model, err)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("passed model query")
	log.Printf("connection to camera model %s S/N %s opened\n", model, snCam)

	c.Allocate()
	err = c.QueueBuffer()

	w := sdk3.NewHTTPWrapper(c)
	mux := server.BuildMux([]server.HTTPer{w}, []string{""})
	log.Fatal(http.ListenAndServe(viper.GetString("Addr"), mux))
}

func main() {
	var cmd string
	args := os.Args
	if len(args) == 1 {
		root()
		return
	}
	setupviper()
	cmd = args[1]
	cmd = strings.ToLower(cmd)
	switch cmd {
	case "help":
		help()
		return
	case "mkconf":
		mkconf()
		return
	case "run":
		run()
		return
	default:
		log.Fatal("unknown command")
	}
}
