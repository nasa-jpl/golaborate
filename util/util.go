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
