package lesker

import (
	"github.com/tarm/serial"
	cp "github.jpl.nasa.gov/HCIT/go-hcit/commonpressure"
)

// KJC300 has a serial connection and can make commands
type KJC300 struct {
	cp.Sensor
}

// NewGauge returns a new Sensor instance
func NewGauge(addr string) (KJC300, error) {
	cfg := cp.MakeSerConf(addr)
	conn, err := serial.OpenPort(cfg)
	tc := KJC300{cp.Sensor{Conn: conn}}
	return tc, err
}
