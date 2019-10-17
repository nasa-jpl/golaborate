/*Package comm provides interfaces and embeddable types for communication with lab hardware.

Most usages of this package will boil down to:
	1.  embed RemoteDevice in a type that represents your hardware.
	2.  overload RxTerminator and TxTerminator to return the right value.  You
		can skip this step if the values are both carriage returns
		(this is the default provided by Package comm)
	3.  if you need to prepend a start of transmission, overload Send to do this
	4.  if you want to work with ASCII strings, overload to convert them to bytes
	5.  Write any methods you see fit based on this low-level communication implementation,

A minimal example is provided below for a temperature sensor that responds to
"RD?" with the current temperature, assuming the default termination values are
OK

	import "strconv"

	type MySensor struct {
		comm.RemoteDevice
	}

	func (ms *MySensor) ReadTemp() (float64, error) {
		cmd := []byte("RD?")
		err := ms.Open()
		if err != nil {
			return 0, err
		}
		defer ms.Close()
		resp, err := ms.SendRecv(cmd)
		if err != nil {
			return 0, err
		}
		return strconv.ParseFloat(string(resp), 64)

	}
*/
package comm

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/tarm/serial"
)

var (
	terminator = byte('\r')

	// ErrNoSerialConf is generated when .SerialConf is not overriden
	ErrNoSerialConf = errors.New("type does not define .SerialConf() method and instance IsSerial=true")

	// ErrNotConnected is generated when .Conn is nil and Send or Recv is called.
	ErrNotConnected = errors.New("conn is nil, not connected to remote")

	// ErrTerminatorNotFound is generated when the termination byte is not found in a response
	ErrTerminatorNotFound = errors.New("termination byte not found")
)

// Sender has a Send method that passes along a byte slice as well as a
// TxTerminator returning the transmission termination byte
type Sender interface {
	Send([]byte) error
	TxTerminator() byte
}

// Recver has a Recv method that gets a byte slice as well as an
// RxTerminator returning the receipt termination byte
type Recver interface {
	Recv() ([]byte, error)
	RxTerminator() byte
}

// SendRecver can send and recieve, and provides a method that sends then recieves
type SendRecver interface {
	Sender
	Recver

	SendRecv([]byte) ([]byte, error)
}

// Opener can open ("establish a connection" but in io language)
type Opener interface {
	Open() error
}

// A Communicator can Open, Send, Recv and Close
type Communicator interface {
	io.Closer
	Opener
	SendRecver
}

// SerialConfigurator has a SerialConf method that provides a serial.Conf suitable
// for passing to serial.OpenPort
type SerialConfigurator interface {
	SerialConf() serial.Config
}

/*RemoteDevice has an address and implements Communicator

note that if IsSerial is true, the embedding type must satisfy the
SerialConfigurator interface

the device is always concurrent-safe, and utilizes an internal queue to maintain
the order of commands
*/
type RemoteDevice struct {
	Addr     string
	IsSerial bool
	Conn     io.ReadWriteCloser
	queue    chan []byte
}

// NewRemoteDevice creates a new RemoteDevice instance
func NewRemoteDevice(addr string, serial bool) RemoteDevice {
	return RemoteDevice{
		Addr:     addr,
		IsSerial: serial,
		queue:    make(chan []byte)}
}

// SerialConf yields a pointer to a serial config object for use with serial.OpenPort
func (rd *RemoteDevice) SerialConf() *serial.Config {
	return nil
}

// Open the connection, setting the Conn variable
func (rd *RemoteDevice) Open() error {
	// we use an exponential backoff, the NKT sources
	// do not like being connection thrashed
	wasTimeout := false
	op := func() error {
		err := rd.open()
		if err != nil {
			errS := err.Error()
			errS = strings.ToLower(errS)
			if strings.Contains(errS, "refused") {
				return err
			}
			wasTimeout = true
			return nil
		}
		return nil
	}

	// backoff will cease on a timeout so we don't wait
	// forever, so we need to check for err != nil && !wasTimeout
	err := backoff.Retry(op, &backoff.ExponentialBackOff{
		InitialInterval:     25 * time.Millisecond,
		RandomizationFactor: 0.,
		Multiplier:          2.,
		MaxInterval:         1 * time.Second,
		MaxElapsedTime:      3 * time.Second,
		Clock:               backoff.SystemClock})
	if err == nil && !wasTimeout {
		return nil
	}
	// err != nil
	if wasTimeout {
		return fmt.Errorf("connection timeout to %s", rd.Addr)
	}
	return err
}

func (rd *RemoteDevice) open() error {
	var err error
	var conn io.ReadWriteCloser
	if rd.IsSerial {
		conf := rd.SerialConf()
		if conf == nil {
			return ErrNoSerialConf
		}
		conn, err = serial.OpenPort(conf)
	} else {
		conn, err = TCPSetup(rd.Addr, 3*time.Second)
	}
	if err != nil {
		return err
	}
	rd.Conn = conn
	return nil
}

// Close the connection, nil-ing the Conn variable
func (rd *RemoteDevice) Close() error {
	err := rd.Conn.Close()
	if err == nil {
		rd.Conn = nil
	}
	return err
}

// TxTerminator returns the transmission termination byte
func (rd *RemoteDevice) TxTerminator() byte {
	return terminator
}

// Send writes data to the remote
func (rd *RemoteDevice) Send(b []byte) error {
	if rd.Conn == nil {
		return ErrNotConnected
	}

	b = append(b, rd.TxTerminator())
	_, err := rd.Conn.Write(b)
	return err
}

// RxTerminator returns the receipt termination byte
func (rd *RemoteDevice) RxTerminator() byte {
	return terminator
}

// Recv recieves data from the remote and strips the Rx terminator
func (rd *RemoteDevice) Recv() ([]byte, error) {
	if rd.Conn == nil {
		return nil, ErrNotConnected
	}
	term := rd.RxTerminator()
	buf, err := bufio.NewReader(rd.Conn).ReadBytes(term)
	if err != nil {
		return []byte{}, err
	}
	if bytes.HasSuffix(buf, []byte{term}) {
		idx := bytes.IndexByte(buf, term)
		return buf[:idx], nil
	}
	return buf, ErrTerminatorNotFound

}

// SendRecv sends a buffer after appending the Tx terminator,
// then returns the response with the Rx terminator stripped
func (rd *RemoteDevice) SendRecv(b []byte) ([]byte, error) {
	if rd.Conn == nil {
		return []byte{}, ErrNotConnected
	}
	err := rd.Send(b)
	if err != nil {
		return []byte{}, err
	}
	return rd.Recv()
}

// TCPSetup opens a new TCP connection and sets a timeout on connect, read, and write
func TCPSetup(addr string, timeout time.Duration) (net.Conn, error) {
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return nil, err
	}
	deadline := time.Now().Add(timeout)
	conn.SetReadDeadline(deadline)
	conn.SetWriteDeadline(deadline)
	return conn, nil
}
