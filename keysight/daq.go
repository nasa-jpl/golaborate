package keysight

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.jpl.nasa.gov/bdube/golab/comm"
	"github.jpl.nasa.gov/bdube/golab/oscilloscope"
	"github.jpl.nasa.gov/bdube/golab/scpi"
)

// DAQ is a remote interface to the DAQ973A and other DAQs with the same SCPI interface
type DAQ struct {
	scpi.SCPI
}

// NewDAQ creates a new scope instance
func NewDAQ(addr string) *DAQ {
	term := comm.Terminators{Tx: '\n', Rx: '\n'}
	rd := comm.NewRemoteDevice(addr, false, &term, nil)
	rd.Timeout = 1 * time.Hour
	return &DAQ{scpi.SCPI{RemoteDevice: &rd, Handshaking: true}}
}

// SetChannelLabel sets the label for a given channel.  This label has no meaning
// to the device and is purely for user identification
func (d *DAQ) SetChannelLabel(channel int, label string) error {
	cmd := fmt.Sprintf(":ROUTE:CHAN:LAB \"%s\", (@%d)", label, channel)
	return d.Write(cmd)
}

// GetChannelLabel returns the label associated with a given channel
func (d *DAQ) GetChannelLabel(channel int) (string, error) {
	cmd := fmt.Sprintf(":ROUTE:CHAN:LAB? (@%d)", channel)
	return d.ReadString(cmd)
}

// SetSampleRate configures the sampling rate on the DAQ
// if -1 is given, the DAQ is configured to be as fast as possible
func (d *DAQ) SetSampleRate(samplesPerSecond float64) error {
	var cmd string
	if samplesPerSecond != -1 { // a real number
		samplePeriodS := 1 / samplesPerSecond
		cmd = fmt.Sprintf(":TRIGGER:TIME %.9f", samplePeriodS)
	} else {
		cmd = ":TRIGGER:SOURCE IMMEDIATE"
	}
	return d.Write(cmd)
}

// GetSampleRate returns the sampling rate of the daq, in Hz.  If zero,
// the sampling rate is immediate "as fast as possible"
func (d *DAQ) GetSampleRate() (float64, error) {
	s, err := d.ReadString(":TRIGGER:SOURCE?")
	if err != nil {
		return 0, err
	}
	fmt.Println(s, err)
	if s == "IMM" {
		return 0, nil
	}
	period, err := d.ReadFloat(":TRIGGER:TIM?")
	fmt.Println(period, err)
	return 1 / period, err
}

// SetRecordingLength sets the number of samples in a recording
// if -1, an infinite recording length is used, and the asynchronous
// drain must be used
func (d *DAQ) SetRecordingLength(samples int) error {
	var cmd string
	if samples == -1 {
		cmd = ":TRIGGER:COUNT INFINITY"
	} else {
		cmd = fmt.Sprintf(":TRIGGER:COUNT %d", samples)
	}
	return d.Write(cmd)
}

// GetRecordingLength returns the number of samples in a recording
func (d *DAQ) GetRecordingLength() (int, error) {
	f, err := d.ReadFloat(":TRIGGER:COUNT?") // it's an int written like a float...
	return int(f), err
}

// SetRecordingChannel sets the channel used when recording data
func (d *DAQ) SetRecordingChannel(channel int) error {
	return d.Write(fmt.Sprintf(":ROUTE:SCAN (@%d)", channel))
}

// GetRecordingChannel retrieves the channel used to record with the DAQ
func (d *DAQ) GetRecordingChannel() (int, error) {
	s, err := d.ReadString(":ROUTE:SCAN?") // which channel is being scanned
	if err != nil {
		return 0, err
	}
	ses := strings.Split(s, "@")
	s = ses[len(ses)-1] // #16(101) -> 101)
	s = s[:len(s)-1]    // 101) -> 101
	i, err := strconv.Atoi(s)
	if err != nil {
		return 0, err
	}
	return i, nil
}

// Record captures data.  If the DAQ is no configured to only one
// channel, an error will be generated
func (d *DAQ) Record() (oscilloscope.Recording, error) {
	// the body of this feels long at a glance.  All it does is:
	// configures the device so it sends the data back in an expected way
	// pops the name of the recorded channel to use as a header
	// converts the "CSV" to an array of floats
	// wraps that in the wrapper type and returns
	// this does not at the moment include the timestamps on the wrapper
	// but we leave the format as something we can grow into later
	var ret oscilloscope.Recording
	err := d.Write(":FORMAT:READING:TIME OFF")
	if err != nil {
		return ret, err
	}
	err = d.Write(":FORMAT:READING:CHANNEL OFF")
	if err != nil {
		return ret, err
	}
	err = d.Write(":FORMAT:READING:ALARM OFF")
	if err != nil {
		return ret, err
	}
	err = d.Write(":FORMAT:READING:UNIT OFF")
	if err != nil {
		return ret, err
	}
	// it's possible the order here could be changed for multitasking
	// but it's not for certain; init and fetch could be split
	// and things like channel label queries used instead
	// untested.
	i, err := d.GetRecordingChannel()
	if err != nil {
		return ret, err
	}
	name, err := d.GetChannelLabel(i)
	if err != nil {
		return ret, err
	}
	s, err := d.ReadString(":INIT;FETCH?")
	if err != nil {
		return ret, err
	}
	pieces := strings.Split(s, ",")
	floats := make([]float64, len(pieces))
	for i := 0; i < len(pieces); i++ {
		f, err := strconv.ParseFloat(pieces[i], 64)
		if err != nil {
			return ret, err
		}
		floats[i] = f
	}
	ret.Measurement = floats
	ret.Name = name
	return ret, nil
}
