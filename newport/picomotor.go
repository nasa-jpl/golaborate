package newport

import (
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/nasa-jpl/golaborate/comm"
)

// no consts here, TxTerm and RxTerm are the same between ESP and picomotor

type PicomotorError struct {
	Axis, Code int
}

func (pe PicomotorError) Error() string {
	if pe.Axis == 0 {
		return ESPErrorCodesWithoutAxes[pe.Code]
	}
	basicmsg := ESPErrorCodesWithAxes[pe.Code]
	return fmt.Sprintf("Axis %d - %s", pe.Axis, basicmsg)
}

func intToPicomotorError(i int) error {
	if i == 0 {
		return nil
	}
	axis := i / 100
	code := i % 100
	return PicomotorError{Axis: axis, Code: code}
}

// ESP301 represents an ESP301 motion controller.
type Picomotor struct {
	pool *comm.Pool

	// Handshaking controls if commands check for errors.  Higher throughput can
	// be achieved without error checking in exchange for reduced safety
	Handshaking bool
}

// NewESP301 makes a new ESP301 motion controller instance
func NewPicomotor(addr string, connectSerial bool) *Picomotor {
	var maker comm.CreationFunc
	if connectSerial {
		maker = comm.SerialConnMaker(makeSerConf(addr))
	} else {
		maker = comm.BackingOffTCPConnMaker(addr, 1*time.Second)
	}
	p := comm.NewPool(1, time.Minute, maker)
	return &Picomotor{pool: p, Handshaking: true}
}

func (p *Picomotor) writeOnlyCommand(cmd string) error {
	conn, err := p.pool.Get()
	if err != nil {
		return err
	}
	defer func() { p.pool.ReturnWithError(conn, err) }()
	wrap := comm.NewTerminator(conn, RxTerm, TxTerm)
	_, err = io.WriteString(wrap, cmd)
	if err != nil {
		return err
	}
	// TODO: check error
	if p.Handshaking {
		_, err = io.WriteString(wrap, "TE?")
		// error response will look like 0 1 nnnn which is six bytes, ten is enough
		buf := make([]byte, 10)
		n, err := wrap.Read(buf)
		if err != nil {
			return err
		}
		buf = buf[:n]
		// buf will be, for example, "108" or "46"
		// if the length is greater than two, it is an axis-specific error
		// else it is a non-axis specific error
		str := string(buf)
		i, err := strconv.Atoi(str)
		// string conversion error
		if err != nil {
			return err
		}
		// error from Picomotor
		err = intToPicomotorError(i)
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *Picomotor) writeReadCommand(cmd string) (string, error) {
	conn, err := p.pool.Get()
	if err != nil {
		return "", err
	}
	defer func() { p.pool.ReturnWithError(conn, err) }()
	wrap := comm.NewTerminator(conn, RxTerm, TxTerm)
	if p.Handshaking {
		cmd = cmd + ";TE?"
	}
	_, err = io.WriteString(wrap, cmd)
	// TODO: check error
	if err != nil {
		return "", err
	}
	buf := make([]byte, 256)
	n, err := wrap.Read(buf)
	if err != nil {
		return "", err
	}
	str := string(buf[:n])
	pieces := strings.SplitN(str, ";", 2)
	if len(pieces) != 2 {
		return "", errors.New("Picomotor controller was queried for error with command, but chose to ignore the error query")
	}
	cmdResp := pieces[0]
	errStr := pieces[1]
	i, err := strconv.Atoi(errStr)
	// string conversion error
	if err != nil {
		return "", err
	}
	// error from Picomotor
	err = intToPicomotorError(i)
	if err != nil {
		return "", err
	}
	return cmdResp, nil
}

func (p *Picomotor) readFloat(cmd string) (float64, error) {
	raw, err := p.writeReadCommand(cmd)
	if err != nil {
		return 0, err
	}
	return strconv.ParseFloat(raw, 64)
}

// Raw implements generichttp/ascii.RawCommunicator
func (p *Picomotor) Raw(s string) (string, error) {
	if strings.Contains(s, "?") {
		resp, err := p.writeReadCommand(s)
		return string(resp), err
	}
	err := p.writeOnlyCommand(s)
	return "", err
}

// minimum feature set to be recognized by generichttp is generichttp/motion.Mover,
// with GetPos, MoveAbs, MoveRel, Home

// GetPos returns the current position of an axis
func (p *Picomotor) GetPos(axis string) (float64, error) {
	cmd := fmt.Sprintf("%sTP?", axis)
	return p.readFloat(cmd)
}

// MoveAbs commands the controller to move an axis to an absolute position
func (p *Picomotor) MoveAbs(axis string, pos float64) error {
	cmd := fmt.Sprintf("%sPA%f", axis, pos)
	return p.writeOnlyCommand(cmd)
}

// MoveRel commands the controller to move an axis by a delta
func (p *Picomotor) MoveRel(axis string, pos float64) error {
	cmd := fmt.Sprintf("%sPR%f", axis, pos)
	return p.writeOnlyCommand(cmd)
}

// Home causes the controller to move an axis to its home position
func (p *Picomotor) Home(axis string) error {
	// per 8734-CL manual, need to do
	// OR
	// MD?
	// DH
	// in discussion with the customer (Caleb Baker) the preference is not to
	// destroy knowledge on repeatability,
	// so we just say go home and call it an honest day's work
	cmd := fmt.Sprintf("%sOR", axis)
	return p.writeOnlyCommand(cmd)
}
