// Package lesker enables working with KJC300 pressure sensors.
package lesker

import (
	cp "github.jpl.nasa.gov/HCIT/go-hcit/commonpressure"
)

// KJC300 embeds the commonpressure Sensor type
type KJC300 struct {
	cp.Sensor
}

// NewSensor returns a new Sensor instance
func NewSensor(addr, connType string) KJC300 {
	return KJC300{Sensor: cp.Sensor{
		Addr:     addr,
		ConnType: connType}}
}
