// Package ixllightwave contains code for operating IXL Lightwave LDC3916 laser diode controllers.
// It contains several single-value structs that are used to enable a "better"
// http interface where the return types are concrete and not strings, but
// they are buried behind a JSON field.  Each of these structs implements
// EncodeAndRespond, and the bodies of these functions are nearly copy pasted
// and can be ignored by the reader.
package ixllightwave

import (
	"strconv"

	"github.jpl.nasa.gov/bdube/golab/comm"
)

// the controller terminates with <CR> <NL> <END>
// it expects terminations of <NL> or <END> or <NL><END>
// we will use NL

const (
	// termination is the message termination used by the device
	termination = '\n'
)

var (
	cmdTable = map[string]string{
		"chan":                "chan",
		"temperature-control": "tec:out",
		"laser-output":        "las:out",
		"laser-current":       "las:ldi",
	}
)

func stringToBool(s string) bool {
	return s == "1"
}

func boolToString(b bool) string {
	if b {
		return "1"
	}
	return "0"
}

func floatToString(f float64) string {
	return strconv.FormatFloat(f, 'g', -1, 64)
}

// LDC3916 represents an LDC3916 laser diode controller
type LDC3916 struct {
	*comm.RemoteDevice
}

// NewLDC3916 creates a new LDC3916 instance, which embeds both comm.RemoteDevice and server.Server
func NewLDC3916(addr string) *LDC3916 {
	term := &comm.Terminators{Rx: '\n', Tx: '\n'}
	rd := comm.NewRemoteDevice(addr, false, term, nil)
	return &LDC3916{RemoteDevice: &rd}
}

func (ldc *LDC3916) processCommand(cmd string, read bool, data string) (string, error) {
	cmd = cmdTable[cmd]
	if read {
		cmd = cmd + "?"
	}
	if data != "" {
		cmd = cmd + " " + data
	}
	err := ldc.Open()
	if err != nil {
		return "", err
	}
	defer ldc.CloseEventually()
	ldc.Lock()
	defer ldc.Unlock()
	err = ldc.Send([]byte(cmd))
	if err != nil {
		return "", err
	}
	if read {
		r, err := ldc.Recv()
		if err != nil {
			return "", err
		}
		return string(r), nil
	}
	return "", nil
}
