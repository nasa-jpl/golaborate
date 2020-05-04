package comm

import (
	"io"
	"sync"
	"time"
)

// CreationFunc is a function which returns a new "connection" to something
// a closure should be used to encapsulate the variables and functions needed
type CreationFunc func() (io.ReadWriteCloser, error)

// Pool is a communication pool which holds one or more connections to a device
// that will be closed if they are not in use, and re-opened as needed.
// it is concurrent safe.  Pools must be created with NewPool.
type Pool struct {
	// can assume chan and timer are created by New in all methods
	// when stopping the timer, close the channel.  The drain for its channel
	// safely handles the zero value that comes on a closed channel.
	maxSize int                     // maximum number of connections, == cap(conns)
	onLease int                     // number of connections given out, <= cap(conns)
	timeout time.Duration           // time after len(conns) == 0 to free all connections
	conns   chan io.ReadWriteCloser // the circular buffer of connections
	timer   *time.Timer             // timer used to destroy connections in the pool after all are returned
	maker   func() (io.ReadWriteCloser, error)

	reclaiming bool // whether startReclaiming's goroutine is running
	mu         *sync.Mutex
}

func NewPool(maxSize int, timeout time.Duration, maker CreationFunc) *Pool {
	p := &Pool{
		maxSize: maxSize,
		timeout: timeout,
		conns:   make(chan io.ReadWriteCloser, maxSize),
		timer:   time.NewTimer(timeout),
		maker:   maker,
		mu:      &sync.Mutex{},
	}
	p.timer.Stop() // stop the timer since there is nothing to close initially
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
// If the error from Get is not nil, you must not return it
// to the pool, or you will cause a panic.
func (p *Pool) Get() (io.ReadWriter, error) {
	// first we must stop the timer and close the channel
	// that can fail as is documented ( https://golang.org/pkg/time/#Timer.Stop )
	// but a new connection will be generated with retry logic anyway,
	// so we can ignore that.
	p.timer.Stop()

	p.mu.Lock()
	defer p.mu.Unlock()
	// short circuit: if a connection is available, immediately return it
	if len(p.conns) > 0 {
		ret := <-p.conns
		p.onLease++
		return ret, nil
	}
	// check if they're all given out
	if p.onLease == p.maxSize {
		// wait for one to come back
		ret := <-p.conns
		p.onLease++
		return ret, nil
	}
	// now the easy cases are exhausted; we don't have a conn available
	// and they aren't all out; make one and give it out
	// no connections available, create one and return it
	// only incrememnt the lease count if we are giving out something
	// other than garbage
	c, err := p.maker()
	if err == nil {
		p.onLease++
	}
	return c, err
}

// Put restores a communicator to the pool.  It may be reused, or will be
// automatically freed after all connections are returned and the timout
// has elapsed.  Junk communicators (ones that always error) should be
// Destroy()'d and not returned with Put.
func (p *Pool) Put(rw io.ReadWriter) {
	rwc := (rw).(io.ReadWriteCloser)
	p.conns <- rwc
	p.onLease--
	if len(p.conns) == p.maxSize {
		p.startReclaim()
	}
	return
}

// Destroy immediately frees a communicator from the pool.  This should be used
// instead of Put if the communicator has gone bad.
func (p *Pool) Destroy(rw io.ReadWriter) {
	// no need for lock?
	rwc := (rw).(io.ReadWriteCloser)
	rwc.Close()
	p.onLease--
	return
}

// Active returns the number of connections in the pool, or given out from it
func (p *Pool) Size() int {
	return len(p.conns) + p.onLease
}

// Active returns the number of connections owned by the pool that are currently
// given out
func (p *Pool) Active() int {
	return p.onLease
}

// startReclaim spawns another goroutine which will be used to close all
// connections in the pool
func (p *Pool) startReclaim() {
	defer func() { p.reclaiming = true }()
	if !p.reclaiming {
		go func() {
			defer func() { p.reclaiming = false }()
			// wait until the timeout has elapsed, then close everything
			<-p.timer.C
			for closer := range p.conns {
				closer.Close()
			}
			p.timer.Reset(p.timeout)
		}()
	}
	return
}
