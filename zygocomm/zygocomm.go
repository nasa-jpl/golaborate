package zygocomm

import (
	"bytes"
	"log"
	"net/http"

	"github.com/spf13/viper"
	"github.com/tarm/serial"
	"github.jpl.nasa.gov/HCIT/go-hcit/server"
)

// MockableSerial holds a serial Port and Real boolean
type MockableSerial struct {
	P    *serial.Port
	Real bool
}

// TriggerMeasurement triggers measurement on the Zygo interferometer.
// if ms.Real and reply by serving the file that results from the measurement.
// File is extracted from URL "filename" query parameter.
// If !ms.Real, serves tmp.dat out of the same directory as the server binary, renamed as "filename."
func (ms *MockableSerial) TriggerMeasurement(w http.ResponseWriter, r *http.Request) {

	// extract cleanup true/false
	cleanup := server.ParseCleanup(w, r)
	filename := server.ParseFilename(w, r)

	log.Printf("Request for filename: %s\t cleanup: %t", filename, cleanup)

	// read the file
	if ms.Real {
		// reciever knows termination at carriage return
		ms.P.Write([]byte(filename + "\r")) // "\x04" is what I would prefer
		// we need to wait for a reply
		buf := ReadToTermination(*ms.P, []byte("\r"))
		log.Printf("serial response %q", buf)
	}
	return
}

// SetupSerial loads serialConn, serialBaud, and spoofSerial from viper.
// If spoofSerial is false, it connects to /dev/ttyp1 but does nothing with
// this connection.
func SetupSerial() MockableSerial {
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
	return MockableSerial{
		P:    conn,
		Real: !viper.GetBool("spoofSerial")}
}

// ReadToTermination reads data from a serial.Port until a termination sequence
// is encountered, returning a slice of bytes including the sequence.
func ReadToTermination(s serial.Port, term []byte) []byte {
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
