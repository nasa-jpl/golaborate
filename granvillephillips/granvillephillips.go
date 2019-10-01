package granvillephillips

import (
	"github.com/tarm/serial"
	cp "github.jpl.nasa.gov/HCIT/go-hcit/commonpressure"
)

// GP375 has a serial connection and can make commands
type GP375 struct {
	cp.Sensor
}

// NewGauge returns a new Sensor instance
func NewGauge(addr string) (GP375, error) {
	cfg := cp.MakeSerConf(addr)
	conn, err := serial.OpenPort(cfg)
	tc := GP375{cp.Sensor{Conn: conn}}
	return tc, err
}
