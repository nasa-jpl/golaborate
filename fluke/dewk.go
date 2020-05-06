package fluke

import (
	"io"
	"strings"
	"time"

	"github.jpl.nasa.gov/bdube/golab/comm"
)

// DewK talks to a DewK 1620 temperature and humidity sensor
// and serves data HTTP routes and meta HTTP routes (route list)
type DewK struct {
	pool *comm.Pool
}

// NewDewK creates a new DewK instance
func NewDewK(addr string) *DewK {
	if !strings.HasSuffix(addr, ":10001") {
		addr = addr + ":10001"
	}
	maker := comm.BackingOffTCPConnMaker(addr, time.Second)
	pool := comm.NewPool(1, time.Minute, maker)
	return &DewK{pool: pool}
}

// Read polls the DewK for the current temperature and humidity, opening and closing a connection along the way
func (dk *DewK) Read() (TempHumid, error) {
	var ret TempHumid
	conn, err := dk.pool.Get()
	if err != nil {
		return ret, err
	}
	defer func() { dk.pool.ReturnWithError(conn, err) }()
	wrap := comm.NewTerminator(conn, '\n', '\n')
	_, err = io.WriteString(wrap, "read?")
	if err != nil {
		return ret, err
	}
	buf := make([]byte, 64)
	n, err := wrap.Read(buf)
	if err != nil {
		return ret, err
	}
	return ParseTHFromBuffer(buf[:n])
}
