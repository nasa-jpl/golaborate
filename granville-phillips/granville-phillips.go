package granvillephillips

import (
	"github.jpl.nasa.gov/HCIT/go-hcit/commonpressure"
)

// GP375 has a serial connection and can make commands
type GP375 struct {
	commonpressure.Sensor
}
