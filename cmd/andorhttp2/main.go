package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.jpl.nasa.gov/HCIT/go-hcit/andor/sdk2"
	"github.jpl.nasa.gov/HCIT/go-hcit/generichttp/camera"
	"github.jpl.nasa.gov/HCIT/go-hcit/imgrec"

	"goji.io/pat"

	"goji.io"

	"github.com/knadh/koanf"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/providers/structs"
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
			"VSAmplitude": "Normal",
			// "VSSpeed":             "1Hz",
			// "HSSpeed":             "TBD",
			"AcquisitionMode":     "SingleScan",
			"ReadoutMode":         "Image",
			"TemperatureSetpoint": "-15",
			"SensorCooling":       true}}, "koanf"), nil)
	if err := k.Load(file.Provider(ConfigFileName), yaml.Parser()); err != nil {
		errtxt := err.Error()
		if !strings.Contains(errtxt, "no such") { // file missing, who cares
			log.Fatalf("error loading config: %v", err)
		}
	}
}
func root() {
	str := `andor-http exposes control of andor iXon cameras over HTTP
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
which is not a software simulation camera.

If the files and folders created do not have the permissions you want on linux,
your umask is likely to blame  andor-http makes them with permission 666, but your
umask is probably the default of 0022 which knocks them down to 444.  Set your
umask to 0000 before running andor-http to solve this.
`
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
	cfg := config{}
	k.Unmarshal("", &cfg)
	fmt.Println("made conf")
	// load the library and see how many cameras are connected
	err := sdk2.Initialize("/usr/local/etc/andor")
	if err != nil {
		log.Fatal(err)
	}
	c := &sdk2.Camera{}
	fmt.Println("initialized sdk2")

	hwv, err := c.GetHardwareVersion()
	swv, err := c.GetSoftwareVersion()
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("connected to camera with hardware %+v hwv\n", hwv)
	log.Printf("software %+v\n", swv)

	err = c.Configure(cfg.BootupArgs)
	if err != nil {
		log.Fatal(err)
	}

	args := cfg.Recorder
	r := &imgrec.Recorder{Root: args.Root, Prefix: args.Prefix}
	w := camera.NewHTTPCamera(c, r)

	// clean up the submux string
	hndlrS := cfg.Root
	hndlrS = server.SubMuxSanitize(hndlrS)
	root := goji.NewMux()
	mux := goji.SubMux()
	root.Handle(pat.New(hndlrS), mux)
	w.RT().Bind(mux)
	addr := cfg.Addr + cfg.Root
	log.Println("now listening for requests at ", addr)
	log.Fatal(http.ListenAndServe(cfg.Addr, root))
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