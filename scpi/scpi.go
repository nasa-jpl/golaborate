// Package scpi provides primitives for working with devices that
// have SCPI interfaces
package scpi

import (
	"bytes"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.jpl.nasa.gov/bdube/golab/comm"
)

const (
	timeout = 5 * time.Second

	tcpFrameSize = 1500
)

// SCPI is a type for encapsulating SCPI communication
type SCPI struct {
	Pool *comm.Pool

	// Handshaking indicates if the communication shall use handshaking,
	// where an error query is sent with every message
	// to ensure the device accepted the input
	Handshaking bool
}

// Write sends a command to the device.  if f.Handshaking == true,
// it also requests an error response and checks that it is OK
// it is assumed this is used for set operations and not get.
func (s *SCPI) Write(cmds ...string) error {
	conn, err := s.Pool.Get()
	if err != nil {
		return err
	}
	defer func() { s.Pool.ReturnWithError(conn, err) }()
	var wrap io.ReadWriter
	wrap = comm.NewTerminator(conn, '\n', '\n')
	wrap, err = comm.NewTimeout(wrap, timeout)
	if err != nil {
		return err
	}
	if s.Handshaking {
		cmds = append([]string{"*CLS;"}, cmds...)
		cmds = append(cmds, ";:SYSTem:ERRor?")
	}
	str := strings.Join(cmds, " ")
	_, err = io.WriteString(wrap, str)
	if err != nil {
		return err
	}
	if s.Handshaking {
		buf := make([]byte, tcpFrameSize)
		n, err := wrap.Read(buf)
		if err != nil {
			return err
		}
		str := string(buf[:n])
		if str[0:2] != "+0" {
			return fmt.Errorf(str)
		}
		return nil
	}
	return nil
}

// WriteRead is write, but with a read call after.  It is assumed that "get"
// calls use this underlying mechanism
func (s *SCPI) WriteRead(cmds ...string) ([]byte, error) {
	var resp []byte
	conn, err := s.Pool.Get()
	if err != nil {
		return resp, err
	}
	defer func() { s.Pool.ReturnWithError(conn, err) }()
	var wrap io.ReadWriter
	wrap = comm.NewTerminator(conn, '\n', '\n')
	wrap, err = comm.NewTimeout(wrap, timeout)
	if err != nil {
		return resp, err
	}
	if s.Handshaking {
		cmds = append([]string{"*CLS;"}, cmds...)
		cmds = append(cmds, ";:SYSTem:ERRor?")
	}
	str := strings.Join(cmds, " ")
	_, err = io.WriteString(wrap, str)
	if err != nil {
		return resp, err
	}
	buf := make([]byte, tcpFrameSize)
	n, err := wrap.Read(buf)
	if err != nil {
		return resp, err
	}
	resp = buf[:n]
	if s.Handshaking {
		pieces := bytes.Split(resp, []byte{';'})
		errS := string(pieces[len(pieces)-1])
		if errS[:2] != "+0" {
			return resp, fmt.Errorf(errS)
		}
		return bytes.Join(pieces[:len(pieces)-1], []byte{}), nil
	}
	return resp, err
}

// ReadString sends a command to the device, the reads the response
// and returns it as a decoded ASCII or UTF-8 string
func (s *SCPI) ReadString(cmds ...string) (string, error) {
	resp, err := s.WriteRead(cmds...)
	if err == nil {
		if resp[len(resp)-1] == '\n' {
			resp = resp[:len(resp)-1]
		}
		if resp[len(resp)-1] == '\r' {
			resp = resp[:len(resp)-1]
		}
	}
	return string(resp), err
}

// ReadFloat sends a command to the device, then reads the
// response and parses it as a floating point value
func (s *SCPI) ReadFloat(cmds ...string) (float64, error) {
	resp, err := s.ReadString(cmds...)
	if err != nil {
		return 0, err
	}
	return strconv.ParseFloat(resp, 64)
}

// ReadBool sends a command to the device, then reads the
// response and parses it as a boolean
func (s *SCPI) ReadBool(cmds ...string) (bool, error) {
	resp, err := s.ReadString(cmds...)
	if err != nil {
		return false, err
	}
	return strconv.ParseBool(resp)
}

// ReadInt sends a command to the device, then reads the
// response and parses it as an integer
func (s *SCPI) ReadInt(cmds ...string) (int, error) {
	resp, err := s.ReadString(cmds...)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(resp)
}

// Raw sends a command to the scope and returns a response if it was a query,
// else a blank string
func (s *SCPI) Raw(str string) (string, error) {
	prev := s.Handshaking
	s.Handshaking = false
	defer func() { s.Handshaking = prev }()
	if strings.Contains(str, "?") {
		return s.ReadString(str)
	}
	return "", s.Write(str)
}

// PopError gets a single error from the queue on the device
func (s *SCPI) PopError() error {
	// SYST: ERR?
	str, err := s.ReadString("SYSTem:ERRor?") // unclear why the case needs to be this way
	if err != nil {
		return err
	}
	if str[0:2] == "+0" {
		return nil
	}
	return fmt.Errorf(str)
}

// AllErrors returns all errors from the device as a list
func (s *SCPI) AllErrors() []error {
	var errs []error
	var err error
	for {
		err = s.PopError()
		if err == nil {
			break
		}
		errs = append(errs, err)
	}
	return errs
}

// AllErrorsString is equivalent to AllErrors, but joining by newline
// if there were no errors, the error return value is nil, otherwise
// it is the first error in the list and has no particular meaning
func (s *SCPI) AllErrorsString() (string, error) {
	errs := s.AllErrors()
	if len(errs) == 0 {
		return "", nil
	}
	strs := make([]string, len(errs))
	for i := 0; i < len(errs); i++ {
		strs[i] = errs[i].Error()
	}
	return strings.Join(strs, "\n"), errs[0]
}
