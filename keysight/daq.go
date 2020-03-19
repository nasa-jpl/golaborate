package keysight

import (
	"fmt"
	"time"

	"github.jpl.nasa.gov/bdube/golab/comm"
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
	rd.Timeout = 24 * time.Hour
	return &DAQ{scpi.SCPI{RemoteDevice: &rd, Handshaking: true}}
}

// SetChannelLabel sets the label for a given channel.  This label has no meaning
// to the device and is purely for user identification
func (d *DAQ) SetChannelLabel(channel int, label string) error {
	cmd := fmt.Sprintf(":ROUTE:CHAN:LAB \"%s\", (@%d)", label, channel)
	return d.Write(cmd)
}

func (d *DAQ) GetChannelLabel(channel int)