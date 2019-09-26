package fluke

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/tarm/serial"
)

// TempHumid holds Temperature and Humidity data, T in C and H in % RH
type TempHumid struct {
	T float64 `json:"temp"`
	H float64 `json:"rh"`
}

// ParseTHFromBuffer converts a raw buffer looking like 21.4,6.5,0,0\r to a TempHumid object for channel 1
func ParseTHFromBuffer(buf []byte) (TempHumid, error) {
	// convert it to a string, then split on ",", and cast to float64
	str := string(buf)
	pieces := strings.SplitN(str, ",", 3)[:2] // 3 pieces potentially leaves the trailing trash, [:2] drops it
	numeric := make([]float64, 2)
	for i, v := range pieces {
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return TempHumid{}, err
		}
		numeric[i] = f
	}
	return TempHumid{T: numeric[0], H: numeric[1]}, nil
}

// TCPPollDewKCh1 reads temperature and humidity from a Fluke 1620a Thermo-Hygrometer over TCP/IP.
func TCPPollDewKCh1(ip string) (TempHumid, error) {
	// these meters communicate and port 10001.  They talk in raw TCP.
	// sending read? spits back data looking like 21.4,6.5,0,0\r
	// commas separate values.  Channels are all concat'd
	port := "10001"
	cmd := "read?\n"

	// open a tcp connection to the meter and send it our command
	conn, err := net.Dial("tcp", ip+":"+port)
	defer conn.Close()
	if err != nil {
		return TempHumid{}, err
	}
	fmt.Fprintf(conn, cmd)

	// make a new buffer reader and read up to \r
	reader := bufio.NewReader(conn)
	resp, err := reader.ReadBytes('\r')
	if err != nil {
		return TempHumid{}, err
	}

	return ParseTHFromBuffer(resp)
}

// SerPollDewKCh1 reads temperature and humidity from a Fluke 1620a Thermo-Hygrometer over serial.
func SerPollDewKCh1(addr string) (TempHumid, error) {
	conf := &serial.Config{
		Name:        addr,
		Baud:        9600,
		ReadTimeout: 1 * time.Second}

	conn, err := serial.OpenPort(conf)
	if err != nil {
		log.Printf("cannot open serial port %q", err)
		return TempHumid{}, err
	}
	reader := bufio.NewReader(conn)
	buf, err := reader.ReadBytes('\r')
	if err != nil {
		log.Printf("failed to read bytes from meter, %q", err)
		return TempHumid{}, err
	}
	return ParseTHFromBuffer(buf)
}
