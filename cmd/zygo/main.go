package main

import (
	"bufio"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/spf13/viper"
	"github.com/tarm/serial"
	"github.jpl.nasa.gov/HCIT/go-hcit/server"
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
	n, b := viper.GetString("serialConn"), viper.GetInt("serialBaud")
	if viper.GetBool("spoofSerial") {
		n, b = "/dev/ttyp1", 9600
	}
	conf := &serial.Config{
		Name:        n,
		Baud:        b,
		ReadTimeout: 5 * time.Second}

	conn, err := serial.OpenPort(conf)
	if err != nil {
		log.Fatalf("cannot open serial port %q", err)
	}
	reader := bufio.NewReader(conn)

	// set up the active user information
	serverState := server.Status{}

	http.HandleFunc("/notify-active", serverState.NotifyActive)
	http.HandleFunc("/release-active", serverState.ReleaseActive)
	http.HandleFunc("/check-active", serverState.CheckActive)
	// anonymous function in HandleFunc has access to the closure variable
	// conn.  This isn't the cleanest style, but this is a small program.
	http.HandleFunc("/measure", func(w http.ResponseWriter, r *http.Request) {
		// extract cleanup true/false
		cleanup := server.ParseCleanup(w, r)
		filename := server.ParseFilename(w, r)

		var fldr string
		if viper.GetBool("spoofFile") {
			filename = "test.txt"
			fldr = "."
		} else {
			fldr = viper.GetString("zygoFootFolder")
		}
		log.Printf("Request for filename: %s\t cleanup: %t", filename, cleanup)

		// if we aren't spoofing the serial portion, trigger MetroPro.
		// we send it the filename and it sends back the filename once done
		// both terminated by \r
		if !viper.GetBool("spoofSerial") {
			conn.Write([]byte(filename + "\r")) // "\x04" is what I would prefer
			// we need to wait for a reply but don't need to do anything with it
			reader.ReadBytes('\r') // this will timeout after 5 seconds if no reply
		}

		server.ReplyWithFile(w, r, filename, fldr)

		// silence the error, it would have already thrown in ReplyWithFile
		// allows panics from errors in edge case of file deleted during request
		// but the window is very short (ms)
		filePath, _ := filepath.Abs(filepath.Join(fldr, filename))

		if cleanup {
			err := os.Remove(filePath)
			if err != nil {
				log.Println(err)
				http.Error(w, fmt.Sprintf("%s", err), http.StatusInternalServerError)
			}
		}
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

	conn.Close()
}
