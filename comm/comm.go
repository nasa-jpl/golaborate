/*Package comm provides interfaces and embeddable types for communication with lab hardware.

Most usages of this package will boil down to:
	1.  embed RemoteDevice in a type that represents your hardware.
	2.  If you do not use carriage returns as terminators, pass a pointer to a
		length 2 slice of bytes in NewRemoteDevice
	3.  if you need to prepend a start of transmission, overload Send to do this
	4.  if you want to work with ASCII strings, overload to convert them to bytes
	5.  Write any methods you see fit based on this low-level communication implementation,

A minimal example is provided below for a temperature sensor that responds to
"RD?" with the current temperature, assuming the default termination values are
OK

	import "strconv"

	type Sensor struct {
		comm.RemoteDevice
	}

	func NewSensor(addr string, serial bool) Sensor {
		rd := NewRemoteDevice(addr, serial)
	}

	func (s *Sensor) ReadTemp() (float64, error) {
		cmd := []byte("RD?")
		err := s.Open()
		if err != nil {
			return 0, err
		}
		defer s.Close()
		resp, err := s.SendRecv(cmd)
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
	"sync"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/tarm/serial"
)

var (
	// ErrNoSerialConf is generated when .SerialConf is not overriden
	ErrNoSerialConf = errors.New("type does not define .SerialConf() method and instance IsSerial=true")

	// ErrNotConnected is generated when .Conn is nil and Send or Recv is called.
	ErrNotConnected = errors.New("conn is nil, not connected to remote")

	// ErrTerminatorNotFound is generated when the termination byte is not found in a response
	ErrTerminatorNotFound = errors.New("termination byte not found")

	// errCloseTooSoon is generated when one attempts to close a connection too soon after communicating
	errCloseTooSoon = errors.New("attempt to close a connection sooner than closeDelay after the last communication")
)

const (
	// DefaultTerminator is the default transmission termination byte
	DefaultTerminator = byte('\r')

	closeDelay = 5 * time.Second
)

// Sender has a Send method that passes along a byte slice with the transmission termination appended
type Sender interface {
	Send([]byte) error
}

// Recver has a Recv method that gets a byte slice and strips the termination byte
type Recver interface {
	Recv() ([]byte, error)
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

// A Communicator can Open, Send, Recv and Close.
//
// It makes no promises about concurrent behavior or stability
type Communicator interface {
	io.Closer
	Opener
	SendRecver
}

// Terminators holds Rx and Tx terminators where are each a single byte
type Terminators struct {
	Rx, Tx byte
}

/*RemoteDevice has an address and implements Communicator

All connects, disconnects, and write->read communication is done
with locks.  This makes the RemoteDevice concurrent-safe through blocking over
TCP.  This behavior is untested over serial.

note that if IsSerial is true, the serCfg must not be nil or calls to Open will
always return ErrNoSerialConf.

*/
type RemoteDevice struct {
	sync.Mutex

	// Addr is the address to connect to
	Addr string

	// IsSerial indicates if the connection type is serial or not
	IsSerial bool

	// Timeout holds the duration of time to wait for replies
	Timeout time.Duration

	// Conn holds the TCP or Serial connection
	Conn     io.ReadWriteCloser
	lastComm time.Time
	txTerm   byte
	rxTerm   byte

	serCfg *serial.Config
}

/*NewRemoteDevice creates a new RemoteDevice instance

Addr is the remote address to connect to

IsSerial is whether the connection is serial (true) or TCP (false)

terminators is a length-2 array of bytes (TxTerm, RxTerm)
*/
func NewRemoteDevice(addr string, serial bool, t *Terminators, s *serial.Config) RemoteDevice {
	var rx, tx byte
	if t == nil {
		rx = DefaultTerminator
		tx = DefaultTerminator
	} else {
		rx = t.Rx
		tx = t.Tx
	}
	return RemoteDevice{
		Addr:     addr,
		IsSerial: serial,
		Timeout:  3 * time.Second,
		txTerm:   tx,
		rxTerm:   rx,
		serCfg:   s}
}

/*Open the connection, setting the Conn variable

This function transparently opens either a TCP or a serial connection.

If conn is not nil, this function is a no-op and does not error.
*/
func (rd *RemoteDevice) Open() error {
	if rd.Conn != nil {
		return nil
	}
	rd.Lock()
	defer rd.Unlock()
	// we use an exponential backoff, the NKT sources
	// do not like being connection thrashed
	wasTimeout := false
	op := func() error {
		err := rd.open()
		if err != nil {
			errS := strings.ToLower(err.Error())
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
		RandomizationFactor: 0,
		Multiplier:          2,
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
		conf := rd.serCfg
		if conf == nil {
			return ErrNoSerialConf
		}
		conn, err = serial.OpenPort(conf)
	} else {
		conn, err = TCPSetup(rd.Addr, rd.Timeout)
	}
	if err != nil {
		return err
	}
	rd.Conn = conn
	return nil
}

// Close the connection, nil-ing the Conn variable
//
// A lock is acquired and released during this operation
func (rd *RemoteDevice) Close() error {
	rd.Lock()
	defer rd.Unlock()
	if rd.Conn != nil {
		err := rd.Conn.Close()
		if err == nil {
			rd.Conn = nil
			return nil
		}
		errS := strings.ToLower(err.Error())
		if strings.Contains(errS, "closed") { // errors containing the "closed" trigger phrase are benign
			err = nil
		}
		return err
	}
	return nil
}

func (rd *RemoteDevice) closeMaybe() error {
	now := time.Now()
	if now.Sub(rd.lastComm) < closeDelay {
		return errCloseTooSoon
	}
	return rd.Close()
}

/*CloseEventually will trigger an infinite number of attempts to close
the connection, spaced some time apart.  After the first successful close
or error on close, the function will return.

This function spawns a goroutine and is used to allow connection
persistence between communications.  Use Close if you wish to close immediately.
*/
func (rd *RemoteDevice) CloseEventually() {
	go rd.closeEventually()
}

func (rd *RemoteDevice) closeEventually() error {
	back := backoff.NewConstantBackOff(closeDelay)
	time.Sleep(closeDelay)
	return backoff.Retry(rd.closeMaybe, back)
}

// Send writes data to the remote
func (rd *RemoteDevice) Send(b []byte) error {
	if rd.Conn == nil {
		return ErrNotConnected
	}
	if conn, ok := rd.Conn.(net.Conn); ok {
		// update the deadline; deadlines are wall times and connection
		// may have persisted from a previous communication
		deadline := time.Now().Add(rd.Timeout)
		conn.SetDeadline(deadline)
	}

	b = append(b, rd.txTerm)
	_, err := rd.Conn.Write(b)
	rd.lastComm = time.Now()
	return err
}

// Recv recieves data from the remote and strips the Rx terminator
func (rd *RemoteDevice) Recv() ([]byte, error) {
	if rd.Conn == nil {
		return nil, ErrNotConnected
	}
	term := rd.rxTerm
	buf, err := bufio.NewReader(rd.Conn).ReadBytes(term)
	rd.lastComm = time.Now()
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
	rd.Lock()
	defer rd.Unlock()
	err := rd.Send(b)
	if err != nil {
		return []byte{}, err
	}
	return rd.Recv()
}

// OpenSendRecvClose calls Open(), defer CloseEventually(), SendRecv()
// this reduces a usage from:
//
//  err := rd.Open()
//  // error handling
//  defer rd.CloseEventually()
//  return rd.SendRecv([]byte)
//
// to:
// rd.OpenSendRecvClose([]byte)
//
// This relies on Open being a no-op for an existing connection,
// and the mutex inside RemoteDevice making this concurrent safe
func (rd *RemoteDevice) OpenSendRecvClose(b []byte) ([]byte, error) {
	err := rd.Open()
	if err != nil {
		return []byte{}, err
	}
	defer rd.CloseEventually()
	return rd.SendRecv(b)
}

// TCPSetup opens a new TCP connection and sets a timeout on connect, read, and write
func TCPSetup(addr string, timeout time.Duration) (net.Conn, error) {
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return nil, err
	}
	deadline := time.Now().Add(timeout)
	conn.SetDeadline(deadline)
	return conn, nil
}
