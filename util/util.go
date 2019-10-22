// Package util contains misc internal utilities.
package util

import (
	"strconv"
	"strings"
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

/*ArangeByte replicates np.arange for byte slices

if startEnd is the only argument, it is the end value and start = 0, step = 1

if two arguments are given, they are start, end and step is 1.

if three arguments are given, they are start, end, step

*/
func ArangeByte(startEnd byte, endStep ...byte) []byte {
	// default values for start and step
	var start, end, step byte
	if len(endStep) == 0 {
		start = byte(0)
		step = byte(1)
		end = startEnd
	} else if len(endStep) == 1 {
		start = startEnd
		end = endStep[0]
		step = 1
	} else {
		start = startEnd
		end = endStep[0]
		step = endStep[1]
	}
	if step <= 0 || end < start {
		return []byte{}
	}
	s := make([]byte, 0, 1+(end-start)/step)
	for start < end {
		s = append(s, start)
		start += step
	}
	return s
}
