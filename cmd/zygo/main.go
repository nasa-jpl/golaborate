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
	"github.jpl.nasa.gov/HCIT/go-hcit/zygo"
	"gopkg.in/yaml.v2"
)

func main() {
	// load cfg or use default values
	zygo.SetupViper()

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
	defer conn.Close()
	reader := bufio.NewReader(conn)

	// set up the active user information
	serverState := server.Status{}

	// anonymous function in HandleFunc has access to the closure variable
	// conn.  This isn't the cleanest style, but this is a small program.
	http.HandleFunc("/measure", func(w http.ResponseWriter, r *http.Request) {
		// extract cleanup true/false
		cleanup := zygo.ParseCleanup(w, r)
		filename := zygo.ParseFilename(w, r)

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
}
