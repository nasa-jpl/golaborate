// Package oscilloscope provides type and interface definitions for oscilloscopes
package oscilloscope

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
)

// Waveform describes a waveform recording from a scope
type Waveform struct {
	// DT is the temporal sample spacing in seconds
	DT float64 `json:"dt"`

	// Channels holds named data streams
	Channels map[string]Channel
}

// Channel represents a stream of data from an ADC.  To convert to physical units,
// compute (data-offset)*scale
type Channel struct {
	// Data is the actual buffer, []byte, []int16, []uint16, or similar
	Data Data

	// Scale is the vertical scale of the data or size of a single increment
	// in Data's native dtype
	Scale float64

	// Offset is the offset applied to the data
	Offset float64

	// Reference is the reference value for the given channel in DN
	Reference float64
}

// Physical computes the data scaled to real units
func (c Channel) Physical() []float64 {
	// a lot of copy paste, but this gets us around the type system
	switch v := c.Data.(type) {
	case []uint8:
		length := len(v)
		ret := make([]float64, length)
		for i := 0; i < length; i++ {
			ret[i] = ((float64(v[i]) - c.Reference) * c.Scale) + c.Offset
		}
		return ret
	case []uint16:
		length := len(v)
		ret := make([]float64, length)
		for i := 0; i < length; i++ {
			// ret[i] = ((float64(v[i]) - c.Reference) * c.Scale) + c.Offset
			ret[i] = float64(v[i])
		}
		return ret
	case []uint32:
		length := len(v)
		ret := make([]float64, length)
		for i := 0; i < length; i++ {
			ret[i] = ((float64(v[i]) - c.Reference) * c.Scale) + c.Offset
		}
		return ret
	case []uint64:
		length := len(v)
		ret := make([]float64, length)
		for i := 0; i < length; i++ {
			ret[i] = ((float64(v[i]) - c.Reference) * c.Scale) + c.Offset
		}
		return ret
	case []int8:
		length := len(v)
		ret := make([]float64, length)
		for i := 0; i < length; i++ {
			ret[i] = ((float64(v[i]) - c.Reference) * c.Scale) + c.Offset
		}
		return ret
	case []int16:
		length := len(v)
		ret := make([]float64, length)
		for i := 0; i < length; i++ {
			ret[i] = ((float64(v[i]) - c.Reference) * c.Scale) + c.Offset
		}
		return ret
	case []int32:
		length := len(v)
		ret := make([]float64, length)
		for i := 0; i < length; i++ {
			ret[i] = ((float64(v[i]) - c.Reference) * c.Scale) + c.Offset
		}
		return ret
	case []int64:
		length := len(v)
		ret := make([]float64, length)
		for i := 0; i < length; i++ {
			ret[i] = ((float64(v[i]) - c.Reference) * c.Scale) + c.Offset
		}
		return ret
	case []float32:
		length := len(v)
		ret := make([]float64, length)
		for i := 0; i < length; i++ {
			ret[i] = ((float64(v[i]) - c.Reference) * c.Scale) + c.Offset
		}
		return ret
	case []float64:
		length := len(v)
		ret := make([]float64, length)
		for i := 0; i < length; i++ {
			ret[i] = ((v[i] - c.Reference) * c.Scale) + c.Offset
		}
		return ret
	default:
		panic("attempt to convert non numerical data to physical units")
	}
}

// Data is a moniker for an empty interface, expected to be a slice of a concrete
// numerical type
type Data interface{}

// EncodeCSV converts the waveform data to physical units
// and writes it to a CSV in streaming fashion
func (wav *Waveform) EncodeCSV(w io.Writer) error {
	// first, assemble the floating point data and timestamps
	// so we have definite length to work with
	labels := make([]string, len(wav.Channels))
	i := 0
	for k := range wav.Channels {
		labels[i] = k
		i++
	}
	data := make([][]float64, i) // i is leaked from the loop above and can be reused
	for j := 0; j < i; j++ {
		data[j] = wav.Channels[labels[j]].Physical()
	}
	timestamps := make([]string, len(data[0]))
	for i := 0; i < len(data[0]); i++ {
		timestamps[i] = strconv.FormatFloat(float64(i)*wav.DT, 'G', -1, 64)
	}
	labels = append([]string{"time"}, labels...)

	// use a shitload of atomic writes through a buffer
	w2 := bufio.NewWriter(w)
	w3 := csv.NewWriter(w2)
	writer := csv.NewWriter(bufio.NewWriter(w))
	err := writer.Write(labels)
	if err != nil {
		return err
	}
	for i := 0; i < len(data[0]); i++ {
		labels[0] = timestamps[i]
		for j := 0; j < len(data); j++ {
			labels[j+1] = strconv.FormatFloat(data[j][i], 'G', -1, 64)
		}
		err := w3.Write(labels)
		if err != nil {
			return err
		}
	}
	w3.Flush()
	w2.Flush()
	return nil
}

// Recording is a sequence of data from the DAQ
type Recording struct {
	// RelTimes is the relative time of each sample
	RelTimes []float64

	// AbsTimes is the absolute time of each sample
	AbsTimes []time.Time

	// Measurement is the actual numeric data
	Measurement []float64

	// Name is the label to use for the data
	Name string
}

// EncodeCSV writes the recording to a CSV file
// if either of the time columns are empty or nil,
// the ther is used, or no time column in the event
// that both are empty or nil
func (r Recording) EncodeCSV(w io.Writer) error {
	if (len(r.AbsTimes) == 0) && (len(r.RelTimes) == 0) {
		encoded := make([]string, len(r.Measurement)+1)
		encoded[0] = r.Name
		for i := 0; i < len(r.Measurement); i++ {
			encoded[i+1] = strconv.FormatFloat(r.Measurement[i], 'G', -1, 64)
		}
		payload := []byte(strings.Join(encoded, "\n"))
		_, err := w.Write(payload)
		return err
	}
	return fmt.Errorf("timestamped writing not implemented")
}
