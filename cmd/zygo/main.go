package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/spf13/viper"
	"github.jpl.nasa.gov/HCIT/go-hcit/server"
	"github.jpl.nasa.gov/HCIT/go-hcit/zygocomm"
	"gopkg.in/yaml.v2"
)

// configuration setup
func setupViper() {
	// stuff that deals with file retrieval
	viper.SetDefault("host", "localhost")
	viper.SetDefault("port", 9876)
	viper.SetDefault("spoofFile", false)
	viper.SetDefault("zygoFileFolder", "/Users/bdube/Downloads") //"C:/Users/zygo")

	// stuff that deals with serial communication
	viper.SetDefault("serialConn", "COM5")
	viper.SetDefault("serialBaud", 9600)
	viper.SetDefault("spoofSerial", true)

	// stuff that deals with viper reading
	viper.SetConfigName("zcom-cfg")
	viper.AddConfigPath("$HOME/Desktop")
	viper.AddConfigPath("$HOME/.zygocomm")
	viper.AddConfigPath(".")
	err := viper.ReadInConfig()
	if err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// pass
		} else {
			log.Fatalf("loading of config file failed %q", err)
		}
	}

	if viper.GetBool("spoofSerial") {
		viper.Set("serialConn", "/dev/ttyp1")
		viper.Set("serialBaud", 9600)
	}
	return
}

func main() {
	// load cfg or use default values
	setupViper()

	// set up the serial connection
	conn := zygocomm.SetupSerial()

	// set up the active user information
	serverState := server.Status{}

	http.HandleFunc("/notify-active", serverState.NotifyActive)
	http.HandleFunc("/release-active", serverState.ReleaseActive)
	http.HandleFunc("/check-active", serverState.CheckActive)
	// anonymous function in HandleFunc has access to the closure variable
	// conn.  This isn't the cleanest style, but this is a small program.
	http.HandleFunc("/measure", func(w http.ResponseWriter, r *http.Request) {
		conn.TriggerMeasurement(w, r)
		server.ReplyWithFile(w, r)
		return
	})

	// dump the config to the log
	c := viper.AllSettings()
	bs, err := yaml.Marshal(c)
	if err != nil {
		log.Fatalf("unable to marshal config to YAML: %+v", err)
	}
	fmt.Println("Server starting with configuration:")
	fmt.Print(string(bs))

	// boot up the server
	host := viper.GetString("host")
	port := strconv.Itoa(viper.GetInt("port")) // ports are given as ints for convenience
	log.Fatal(http.ListenAndServe(host+":"+port, nil))

	conn.P.Close()
}
