package aerotech

import (
	"net"
	"strings"
	"time"

	"github.com/cenkalti/backoff"
	"github.jpl.nasa.gov/HCIT/go-hcit/comm"
)

/* this file contains some code that handles the branching logic needed to talk
to ensemble controllers.  some commands, say ENABLE X, return only [37] (a %)
while others return < some data >\n.  In the latter case, we can use the
central comm package for communication, but in the former case it will deadlock
while scanning for the \n that will never come.

Here, we implement an algorithm that expects the connection timeout to be
configured on the underlying TCP connection.  Then it takes a flag, singleByte
which determines which branch of logic it uses.  if singleByte, it polls until
there is one byte in the receipt buffer and then returns that byte.

else, it scans for \n using the underlying remoteDevice.
*/

func getAnyResponseFromEnsemble(rd *comm.RemoteDevice, singleByte bool) ([]byte, error) {
	// the double retry mechanism made this function quite unfortunately complicated
	if singleByte {
		// timeout implemented inside the remote device
		// we expect 1 byte, but allocate a buffer as large as the largest TCP
		// frame in case we get a larger packet
		tcpFrameSize := 1500
		buf := make([]byte, tcpFrameSize)
		n := 0
		var (
			err  error = nil
			err2 error
		)
		// the Ensemble controller will close the connection on its own
		// or we may timeout if the motion is very slow.  Patch that up here.
		op := func() error {
			n, err = rd.Conn.Read(buf)
			if err != nil {
				txt := err.Error()
				// we want to retry on timeout or resets
				if strings.Contains(txt, "timeout") {
					if con, ok := rd.Conn.(net.Conn); ok {
						// need to avoid client timeouts; 1s timeout
						// does a little bit of thrashing, but Go is fast.
						con.SetReadDeadline(time.Now().Add(1 * time.Second))
					}
					return err
				} else if strings.Contains(txt, "reset") {
					// the connect was closed, we need to re-open it
					err2 := rd.Open()
					if err2 != nil {
						return err2
					}
					return err
				}
				// other errors are captured in the closure to be handled without retrying
				return nil
			}
			return nil
		}
		for n == 0 && err == nil && err2 == nil {
			err2 = backoff.Retry(op, &backoff.ExponentialBackOff{
				InitialInterval:     1 * time.Second,
				RandomizationFactor: 0,
				Multiplier:          1,
				MaxInterval:         2 * time.Second, // this is bigger than it will ever be
				// wait up to to five minutes, the shit has really hit the fan if it takes longer than this
				MaxElapsedTime: 5 * time.Minute,
				Clock:          backoff.SystemClock})
		}
		if err2 != nil && err == nil {
			err = err2
		}
		return buf[:n], err // don't strip the end character because there is nothing to strip
	}
	return rd.Recv()
}
