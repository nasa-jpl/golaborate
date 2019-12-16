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

	yml "github.com/go-yaml/yaml"
)

var (
	// Version is the version number.  Typically injected via ldflags with git build
	Version = "dev"

	// ConfigFileName is what it sounds like
	ConfigFileName = "multiserver.yml"
	k              = koanf.New(".")
)

func setupconfig() {
	k.Load(structs.Provider(multiserver.Config{}, "koanf"), nil)
	if err := k.Load(file.Provider(ConfigFileName), yaml.Parser()); err != nil {
		errtxt := err.Error()
		if !strings.Contains(errtxt, "no such") { // file missing, who cares
			log.Fatalf("error loading config: %v", err)
		}
	}
}
func root() {
	str := `multiserver communicates with lab hardware and exposes an HTTP interface to them
This enables a server-client architecture,
and the clients can leverage the excellent HTTP
libraries for any programming language,
instead of custom socket logic.

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

When no configuration is provided, the defaults are used.  Keys are not case-sensitive.
The command mkconf generates the configuration file with the default values.
There is no need to do this unless you want to start from the prepopulated defaults when making
a config file.`
	fmt.Println(str)
}

func mkconf() {
	c := multiserver.Config{}
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
	fmt.Printf("multiserver version %v\n", Version)
}

func run() {
	c := multiserver.Config{}
	err := k.Unmarshal("", &c)
	if err != nil {
		log.Fatal(err)
	}
	mux := c.BuildMux()
	log.Println("now listening for requests at ", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
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
