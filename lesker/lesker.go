// Package lesker enables working with KJC300 pressure sensors.
package lesker

import (
	cp "github.jpl.nasa.gov/HCIT/go-hcit/commonpressure"
)

// NewSensor returns a new Sensor instance, in this case a KJC300
func NewSensor(addr, urlStem string, serial bool) *cp.Sensor {
	return cp.NewSensor(addr, urlStem, serial)
}
