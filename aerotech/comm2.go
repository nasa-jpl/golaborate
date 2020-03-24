package aerotech

import (
	"github.jpl.nasa.gov/bdube/golab/comm"
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
	if singleByte {
		// timeout implemented inside the remote device
		// we expect 1 byte, but allocate a buffer as large as the largest TCP
		// frame in case we get a larger packet
		tcpFrameSize := 1500
		buf := make([]byte, tcpFrameSize)
		n := 0
		var err error = nil
		for n == 0 && err == nil {
			n, err = rd.Conn.Read(buf)
		}
		return buf[:n], err // don't strip the end character because there is nothing to strip
	}
	return rd.Recv()
}
