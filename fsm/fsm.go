// Package fsm provides a system for operating a fast steering mirror control system at high speed
package fsm

import (
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"sync"
	"time"

	"github.jpl.nasa.gov/HCIT/go-hcit/mccdaq"
)

// ControlLoop is a struct which operates a control loop
type ControlLoop struct {
	DAC *mccdaq.DAC

	sync.Mutex
}

// Update turns the crank on the control loop.  An internal lock is acquired
// and the phase (refresh rate) of the loop, i.e. call frequency of update,
// is blocked by the phase lock of the control loop.
// Any errors writing to the DAC are bubbled up
func (c *ControlLoop) Update(target float64) error {
	c.Lock()
	defer c.Unlock()
	return c.DAC.Write(1, target)
}

// Disturbance is a pre-recorded sequence to be played back, paused, or stopped
type Disturbance struct {
	// data is the list of data points, assumed to be of uniform spacing in time
	data [][]float64

	// cursor is the index into data
	cursor int

	// signal is the internal channel used to manipulate the playback
	signal chan string

	// if paused is true, the loop is short circuited with a CPU burn
	paused bool

	// DT is the temporal spacing between elements of data,
	// changing it during playback is undefined behavior
	DT time.Duration

	// Callback is the function to run on each iteration of the loop
	Callback func([]float64)

	// Repeat determines if the playback signal repeats
	Repeat bool
}

// Play begins processing a stream of commands by calling callable for each
// element of its data buffer.  The playback can be paused or stopped, and
// loops to the beginning when the data stream is exhausted
func (d *Disturbance) Play() {
	d.fixchannel()
	go func() {
		// the double semicolon just does the end of loop clause
		// so this is like for (i := i; i < n; i++) but i++ is a sleep and
		// there is nothing done to initiate the loop or end it
		for ; ; time.Sleep(d.DT) {
			select {
			case action := <-d.signal:
				switch action {
				case "pause":
					d.paused = true
				case "resume":
					d.paused = false
				case "stop":
					d.paused = false
					d.cursor = 0
					return
				}
			default:
				if d.paused {
					continue
				}
				d.Callback(d.data[d.cursor])
				d.cursor++
				if d.cursor == len(d.data) {
					if !d.Repeat {
						return
					}
					d.cursor = 0
				}
			}
		}
	}()
}

func (d *Disturbance) fixchannel() {
	if d.signal == nil {
		d.signal = make(chan string)
	}
}

// Pause stops the loop where it is, but does not reset the cursor.
// Resume can pick up where Pause leaves off.
func (d *Disturbance) Pause() {
	d.fixchannel()
	d.signal <- "pause"
}

// Resume picks up where the loop was left off
func (d *Disturbance) Resume() {
	d.fixchannel()
	d.signal <- "resume"
}

// Stop ceases the loop and resets the cursor
func (d *Disturbance) Stop() {
	d.fixchannel()
	d.signal <- "stop"
}

// LoadCSV loads data from a CSV file.
// The file is assumed to have a header row, comma separation, three numeric f64-parse-able columns
func (d *Disturbance) LoadCSV(r io.Reader) error {
	d.data = [][]float64{}
	reader := csv.NewReader(r)
	skip := true
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if skip {
			skip = false
			continue
		}
		local := make([]float64, len(record))
		for i := 0; i < len(record); i++ {
			f, err := strconv.ParseFloat(record[i], 64)
			if err != nil {
				return err
			}
			if i > 2 {
				return fmt.Errorf("row contains at least %d records, must be == 3", i+1)
			}
			local[i] = f
		}
		d.data = append(d.data, local)
	}
	return nil
}
