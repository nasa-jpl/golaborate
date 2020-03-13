// Package oscilloscope provides type and interface definitions for oscilloscopes
package oscilloscope

import (
	"time"
)

/*Waveform describes a waveform recording from a scope
To convert to physical scales:

  1.  Cast the data buffer to the appropriate dtype, then convert to a
      floating point representation
  2.  Subtract the Offset
  3.  Multiply by Scale
  4.  The abscissa is range(len(cast_data)) * dT
*/
type Waveform struct {
	// Trigger is the moment the waveform recording began
	Trigger time.Time `json:"trigger"`

	// DT is the temporal sample spacing in seconds
	DT float64 `json:"dt"`

	// Dtype is the type of data represented by the buffers,
	// in machine native byte order
	Dtype string `json:"dtype"`

	// Data contains the buffer(s) of scope data, keyed by the channel
	// the bytes typically represent uint16s or int16s
	Data map[string][]byte `json:"data"`

	// Scale contains the voltage of each unit step in the data, per channel
	Scale map[string]float64 `json:"scale"`

	// Offset contains the offset (in Dtype steps) for each channel
	Offset map[string]float64 `json:"offset"`
}
