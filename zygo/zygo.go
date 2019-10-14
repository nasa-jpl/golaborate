// Package zygo contains functions for providing an server that interfaces
// over serial with metropro to trigger measurements and send the files back
// over HTTP
package zygo

import (
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/spf13/viper"
)

/*SetupViper configures viper with default values and config file locations
for the config options:
	- host
	- port
	- spoofFile
	- zygoFileFolder
	- serialConn
	- serialBaud

The config file is named zcon-cfg, can be any type supported by Viper, and
is located adjacent to the binary, at $HOME or $HOME/Desktop
*/
func SetupViper() {
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

// ParseCleanup extracts a boolean value (cleanup) from URL query parameter "cleanup"
func ParseCleanup(w http.ResponseWriter, r *http.Request) bool {
	// if true, delete the file after serving it
	cleanupStr := r.URL.Query().Get("cleanup")
	if cleanupStr == "" {
		cleanupStr = "false"
	}
	cleanup, ok := strconv.ParseBool(cleanupStr)
	if ok != nil {
		fstr := fmt.Sprintf("cleanup URL parameter error, given %s, cannot be converted to float", cleanupStr)
		log.Println(fstr)
		http.Error(w, fstr, http.StatusBadRequest)
	}

	return cleanup
}

// ParseFilename extracts a string from URL query parameter "filename"
// or defaults to "tmp.dat" if one is not given
func ParseFilename(w http.ResponseWriter, r *http.Request) string {
	filename := r.URL.Query().Get("filename")
	if filename == "" {
		filename = "tmp.dat"
	}
	return filename
}
