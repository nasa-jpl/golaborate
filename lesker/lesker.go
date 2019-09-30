package lesker

import (
	"github.jpl.nasa.gov/HCIT/go-hcit/commonpressure"
)

// KJC300 has a serial connection and can make commands
type KJC300 struct {
	commonpressure.Sensor
}
