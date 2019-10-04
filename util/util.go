// Package util contains misc internal utilities.
package util

import (
	"net"
	"strconv"
	"strings"
	"time"
)

// IntSliceToCSV convets a slice of ints to CSV formatted data.
// e.g., []int{1,2,3,4,5} => "1,2,3,4,5"
func IntSliceToCSV(is []int) string {
	s := make([]string, len(is))
	for i, v := range is {
		s[i] = strconv.Itoa(v)
	}

	return strings.Join(s, ",")
}

// GetBit returns the value of a given bit in a byte
func GetBit(b byte, bitIndex uint) bool {
	return (b & (1<<bitIndex - 1)) != 0
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
