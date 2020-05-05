package comm

import (
	"bufio"
	"bytes"
	"io"
	"time"
)

// CreationFunc is a function which returns a new "connection" to something
// a closure should be used to encapsulate the variables and functions needed
type CreationFunc func() (io.ReadWriteCloser, error)

// Pool is a communication pool which holds one or more connections to a device
// that will be closed if they are not in use, and re-opened as needed.
// it is concurrent safe.  Pools must be created with NewPool.
type Pool struct {
	maxSize   int                     // maximum number of connections, == cap(conns)
	onLease   int                     // number of connections given out, <= cap(conns)
	timeout   time.Duration           // every (timeout) an attempt is made to destroy a connection
	conns     chan io.ReadWriteCloser // the circular buffer of connections
	interrupt chan struct{}           // interrupt is used to stop the background closer
	sem       chan struct{}           // sem is the semaphore used to ensure acquisitions from the pool are atomic
	maker     func() (io.ReadWriteCloser, error)
}

// NewPool creates a new pool.  At each interval of timeout, a connection
// may be freed if it is available.  Calling Close terminates the background
// goroutine which closes idle connections and drains the pool as it is immediately
func NewPool(maxSize int, timeout time.Duration, maker CreationFunc) *Pool {
	p := &Pool{
		maxSize:   maxSize,
		timeout:   timeout,
		conns:     make(chan io.ReadWriteCloser, maxSize),
		interrupt: make(chan struct{}),
		sem:       make(chan struct{}, 1),
		maker:     maker,
	}
	p.sem <- struct{}{}
	go p.destroyTrash()
	return p
}

// Get retrieves a communicator from the channel, blocking until one is
// available if all are in use.  It is guaranteed that there is no contestion
// for the ReadWriter.  The consumer should not attempt to cast it to its
// concrete type and use it outside this interface.
//
// When done with the communicator, return it with Put(), or discard it with
// Destroy() if it has become no good (e.g., all calls error).
//
// If the error from Get is not nil, you must not return it to the pool.
func (p *Pool) Get() (io.ReadWriter, error) {
	select {
	case rw := <-p.conns:
		p.onLease++
		return rw, nil
	case <-time.After(100 * time.Microsecond): // wait a little bit
		// nothing in the pool or returned in time, check if all on lease.
		// If not, we need to acquire the semaphore,  which says we are the only
		// one modifying the size of the pool.
		if p.onLease == p.maxSize {
			// could recurse, but want to avoid design which can OOM bomb
			// the user.  May cause a deadlock by not returning, but that
			// is an error in their program, not the pool.
			// eventually, they must return one.
			rv := <-p.conns
			p.onLease++
			return rv, nil
		}

		// due to a subtle race, we need to select again
		select {
		case rw := <-p.conns:
			p.onLease++
			return rw, nil
		default:
			<-p.sem
			defer func() { p.sem <- struct{}{} }()
			rw, err := p.maker()
			if err == nil { // only increment if the conn was good
				p.onLease++
			}
			return rw, err
		}
	}
}

// Put restores a communicator to the pool.  It may be reused, or will be
// be freed eventually by the pool.  Junk communicators (ones that always error)
// should be Destroy()'d and not returned with Put.
func (p *Pool) Put(rw io.ReadWriter) {
	<-p.sem // acquire the semaphore
	p.onLease--
	p.conns <- (rw).(io.ReadWriteCloser)
	p.sem <- struct{}{}
	return
}

// Destroy immediately frees a communicator from the pool.  This should be used
// instead of Put if the communicator has gone bad.
func (p *Pool) Destroy(rw io.ReadWriter) {
	// no need for lock?
	rwc := (rw).(io.ReadWriteCloser)
	rwc.Close()
	<-p.sem
	p.onLease--
	p.sem <- struct{}{}
	return
}

// Size returns the number of connections in the pool, or given out from it
func (p *Pool) Size() int {
	return len(p.conns) + p.onLease
}

// Active returns the number of connections owned by the pool that are currently
// given out
func (p *Pool) Active() int {
	return p.onLease
}

// destroyTrash should be run in a goroutine.  It loops on the trash bin
// until the cancellation signal is given, at which time it returns
func (p *Pool) destroyTrash() {
	for {
		time.Sleep(p.timeout)
		select {
		case closer := <-p.conns:
			closer.Close()
		case <-p.interrupt:
			return
		}
	}
}
func (p *Pool) Close() {
	p.interrupt <- struct{}{} // stop the background "garbage collection"
	<-p.sem                   // acquire the semaphore and prevent new connections
	for len(p.conns) > 0 {    // not a range so that the loop will skip if the pool is empty
		c := <-p.conns
		c.Close()
	}
	p.sem <- struct{}{}
}

// Terminator is a struct holding termination sequences and read/writers
type Terminator struct {
	Wterm byte
	Rterm byte
	w     io.Writer
	r     io.Reader
}

func (t Terminator) Write(b []byte) (int, error) {
	b = append(b, t.Wterm)
	return t.w.Write(b)
}

// Read implements io.Reader.  The input is scanned up to the first encounter
// of Rterm.  Rterm is stripped from the message and the remainder returned.
// buf is double buffered for this purpose.
func (t Terminator) Read(buf []byte) (int, error) {
	b, err := bufio.NewReader(t.r).ReadBytes(t.Rterm)
	if err != nil {
		return 0, err
	}
	if bytes.HasSuffix(b, []byte{t.Rterm}) {
		idx := bytes.IndexByte(b, t.Rterm)
		b = b[:idx]
	}
	return copy(buf, b), nil
}

// NewTerminator returns a wrapper around a Read/Writer that appends and
// strips termination bytes
func NewTerminator(rw io.ReadWriter, Rx, Tx byte) Terminator {
	return Terminator{w: rw, r: rw, Wterm: Tx, Rterm: Rx}
}
