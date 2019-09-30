package main

import (
	"fmt"
	"io/ioutil"
	"log"

	"github.jpl.nasa.gov/HCIT/go-hcit/fluke"
	"github.jpl.nasa.gov/HCIT/go-hcit/sensor"
	"gopkg.in/yaml.v2"
)

// Config is a variable used to nicely reflect sensors.yml into structs
type Config struct {
	Sensors []sensor.Info `yaml:"sensors"`
}

func loadConfig() *Config {
	// read the data into a Config struct
	path := "./sensors.yml"
	yamlData, err := ioutil.ReadFile(path)
	if err != nil {
		fstr := fmt.Sprintf("Error reading sensors.yml, %q", err)
		log.Fatal(fstr)
	}
	cfg := &Config{}
	err = yaml.Unmarshal(yamlData, cfg)
	if err != nil {
		log.Fatal(err)
	}

	// now we need to bind methods and routes based on all of the gathered sensors
	for idx, sens := range cfg.Sensors {
		switch sens.Type {
		case "fluke":
			if sens.Conntype == "TCP" {
				sens.Func = fluke.TCPPollDewKCh1
			} else {
				sens.Func = fluke.SerPollDewKCh1
			}
		}
	}
}

func main() {
	cfg := loadConfig()
	log.Print(cfg.Sensors)
	// log.Fatal(http.ListenAndServe(":8080", nil))
}
