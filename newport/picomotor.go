package newport

import (
	"errors"
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/nasa-jpl/golaborate/comm"
)

func newConnectionFactory(addr string, serial bool) comm.CreationFunc {
	if !serial {
		return func() (io.ReadWriteCloser, error) {
			f := comm.BackingOffTCPConnMaker(addr, 10*time.Second)
			conn, err := f()
			if err != nil {
				return nil, err
			}
			buf := make([]byte, 256)
			_, err = conn.Read(buf)
			if err != nil {
				log.Println("picomotor gave an error trying to read telnet hello,", err)
				conn.Close()
				return nil, err
			}
			return conn, nil
		}
	}
	// this is the same as ESP301, as a guess... the manual does not say
	// what the serial parameters are!
	return comm.SerialConnMaker(makeSerConf(addr))
}

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

// Picomotor represents an picomotor controller.
type Picomotor struct {
	pool *comm.Pool

	// Handshaking controls if commands check for errors.  Higher throughput can
	// be achieved without error checking in exchange for reduced safety
	Handshaking bool

	// serial indicates whether this driver uses a serial connection or ethernet.
	// unlike most devices where this only matters to the connection logic,
	// newport's love of suffering means that the terminators are different on
	// ethernet and serial.  Particularly,
	// serial: terminator = \r
	// ethernet: terminator = \n
	// bonus pain: ethernet replies are \r\n
	serial bool
}

// NewESP301 makes a new ESP301 motion controller instance
func NewPicomotor(addr string, connectSerial bool) *Picomotor {
	p := comm.NewPool(1, time.Minute, newConnectionFactory(addr, connectSerial))
	return &Picomotor{pool: p, Handshaking: true}
}

func (p *Picomotor) terminator() byte {
	if p.serial {
		return CarriageReturn
	}
	return Newline
}

func (p *Picomotor) writeOnlyCommand(cmd string) error {
	conn, err := p.pool.Get()
	if err != nil {
		return err
	}
	defer func() { p.pool.ReturnWithError(conn, err) }()
	term := p.terminator()
	wrap := comm.NewTerminator(conn, term, term)
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
		n = len(buf) - 1
		if buf[n] == CarriageReturn {
			buf = buf[:n] // go slices are end-exclusive
		}
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
	term := p.terminator()
	wrap := comm.NewTerminator(conn, term, term)
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
	buf = buf[:n]
	n = len(buf) - 1
	if buf[n] == CarriageReturn {
		buf = buf[:n] // go slices are end-exclusive
	}
	str := string(buf)
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
