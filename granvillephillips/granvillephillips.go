// Package granvillephillips enables working with GP375 pressure sensors.
package granvillephillips

import (
	cp "github.jpl.nasa.gov/HCIT/go-hcit/commonpressure"
)

// GP375 embeds the commonpressure Sensor type
type GP375 struct {
	cp.Sensor
}

// NewSensor returns a new Sensor instance
func NewSensor(addr, connType string) GP375 {
	return GP375{cp.Sensor{
		Addr:     addr,
		ConnType: connType}}
}
