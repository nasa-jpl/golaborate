package zygo

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v2"

	"github.com/tarm/serial"

	"github.jpl.nasa.gov/HCIT/go-hcit/internal/serveraccess"
)

// wrapper around serial type to permit mocking
type mockableSerial struct {
	p    serial.Port
	real bool
}

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

// serial connection setup
func setupSerial() mockableSerial {
	n := viper.GetString("serialConn")
	b := viper.GetInt("serialBaud")
	if viper.GetBool("spoofSerial") {
		n, b = "/dev/ttyp1", 9600
	}
	conf := &serial.Config{Name: n, Baud: b}

	conn, err := serial.OpenPort(conf)
	if err != nil {
		log.Fatalf("cannot open serial port %q", err)
	}
	return mockableSerial{
		p:    *conn,
		real: true}
}

// "scanning" multipart serial read up to a termination sequence of bytes
func readToTermination(s serial.Port, term []byte) []byte {
	var out []byte
	for {
		buf := make([]byte, 128)
		n, _ := s.Read(buf)
		out = append(out, buf[:n]...)
		if bytes.HasSuffix(out, term) {
			break
		}
	}
	return out
}

// read cleanup off the request
func parseCleanup(w http.ResponseWriter, r *http.Request) bool {
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

// reply to the request by serving a file
func replyWithFile(fn string, w http.ResponseWriter) {
	f, err := os.Open(fn)
	defer f.Close()
	if err != nil {
		fstr := fmt.Sprintf("source file missing %s", fn)
		log.Println(fstr)
		http.Error(w, fstr, http.StatusNotFound)
	}

	// read some stuff to set the headers appropriately
	filename := filepath.Base(fn)
	fStat, _ := f.Stat()
	fSize := strconv.FormatInt(fStat.Size(), 10) // base 10 int
	cDistStr := fmt.Sprintf("attachment; filename=\"%s\"", filename)
	w.Header().Set("Content-Disposition", cDistStr)
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", fSize)

	// copy the file to the client and print to the log
	io.Copy(w, f)
}

func main() {
	// load cfg or use default values
	setupViper()

	// set up the serial connection
	conn := setupSerial()

	// set up the active user information
	serverState := serveraccess.ServerStatus{}

	http.HandleFunc("/notify-active", serverState.NotifyActive)
	http.HandleFunc("/release-active", serverState.ReleaseActive)
	http.HandleFunc("/check-active", serverState.CheckActive)
	// anonymous function in HandleFunc has access to the closure variable
	// conn.  This isn't the cleanest style, but this is a small program.
	http.HandleFunc("/measure", func(w http.ResponseWriter, r *http.Request) {
		// extract filename
		filename := r.URL.Query().Get("filename")
		if filename == "" {
			filename = "tmp.dat"
		}

		// extract cleanup true/false
		cleanup := parseCleanup(w, r)

		// log the inputs
		log.Printf("filename: %s\t cleanup: %t", filename, cleanup)

		// read the file
		// reciever knows termination at carriage return
		conn.p.Write([]byte(filename + "\r")) // "\x04" is what I would prefer
		if conn.real {
			// we need to wait for a reply
			buf := readToTermination(conn.p, []byte("\r"))
			log.Printf("serial response %q", buf)
		}

		fldr := viper.GetString("zygoFileFolder")
		if viper.GetBool("spoofFile") {
			filename = "test.txt"
			fldr = "."
		}
		filePath := filepath.Join(fldr, filename)

		replyWithFile(filePath, w)
		return
	})

	host := viper.GetString("host")
	port := strconv.Itoa(viper.GetInt("port")) // ports are given as ints for convenience

	// dump the config to the log
	c := viper.AllSettings()
	bs, err := yaml.Marshal(c)
	if err != nil {
		log.Fatalf("unable to marshal config to YAML: %+v", err)
	}
	fmt.Println("Server starting with configuration:")
	fmt.Print(string(bs))
	log.Fatal(http.ListenAndServe(host+":"+port, nil))

	conn.p.Close()
}
