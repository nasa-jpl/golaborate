// Package granvillephillips enables working with GP375 pressure sensors.
package granvillephillips

import (
	cp "github.jpl.nasa.gov/HCIT/go-hcit/commonpressure"
)

// NewSensor returns a new Sensor instance, in this case a GP375
func NewSensor(addr, urlStem string, serial bool) *cp.Sensor {
	return cp.NewSensor(addr, urlStem, serial)
}
