// Package util contains misc internal utilities.
package util

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

// IntSliceToCSV converts a slice of ints to CSV formatted data.
// e.g., []int{1,2,3,4,5} => "1,2,3,4,5"
func IntSliceToCSV(is []int) string {
	s := make([]string, len(is))
	for i, v := range is {
		s[i] = strconv.Itoa(v)
	}

	return strings.Join(s, ",")
}

// Float64SliceToCSV converts a slice of f64s to CSV formatted data
// sensible default values for fmt and prec are 'G' and 3 to print with
// 3 decimal places, and 'ordinary' notation
func Float64SliceToCSV(fs []float64, fmt byte, prec int) string {
	s := make([]string, len(fs))
	for i, v := range fs {
		s[i] = strconv.FormatFloat(v, fmt, prec, 64)
	}
	return strings.Join(s, ",")
}

// GetBit returns the value of a given bit in a byte
func GetBit(b byte, bitIndex uint) bool {
	return (b>>bitIndex)&1 == 1
}

// SetBit sets a single bit in a byte
func SetBit(in byte, bitIndex uint, high bool) byte {
	if high {
		in |= 1 << bitIndex
	} else {
		in &= ^(1 << bitIndex)
	}
	return in
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

// UniqueString reduces a slice of strings to the unique values
func UniqueString(slice []string) []string {
	var out []string
	seen := make(map[string]struct{})
	for _, v := range slice {
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}

// UintSliceContains returns true if value is in slice, otherwise false
func UintSliceContains(slice []uint, value uint) bool {
	ret := false
	for _, cmpV := range slice {
		if value == cmpV {
			ret = true
		}
	}
	return ret
}

// AllElementsNumbers tests if all elements of a string are numbers
func AllElementsNumbers(s string) bool {
	return !strings.ContainsAny(s, "0123456789.")
}

// Clamp limits min < input < max
func Clamp(input, min, max float64) float64 {
	if input < min {
		return min
	}
	if input > max {
		return max
	}
	return input
}

// Limiter represents a basic set of min,max limits
type Limiter struct {
	// Min is the minimum value
	Min float64 `json:"min"`

	// Max is the maximum value
	Max float64 `json:"max"`
}

// Clamp limits min < input < max
func (l *Limiter) Clamp(input float64) float64 {
	return Clamp(input, l.Min, l.Max)
}

// Check verifies if min < input < max, returns true if this is the case
func (l *Limiter) Check(input float64) bool {
	if input < l.Min {
		return false
	}
	if input > l.Max {
		return false
	}
	return true
}

// MergeErrors converts many errors to a single one, newline separated
func MergeErrors(errs []error) error {
	var strs []string
	for idx := 0; idx < len(errs); idx++ {
		err := errs[idx]
		if err != nil {
			strs = append(strs, err.Error())
		}
	}
	err := fmt.Errorf(strings.Join(strs, "\n"))
	if err.Error() == "" {
		return nil
	}
	return err
}

// ClosestIndex returns the index of the closest element in the slice to the given value
func ClosestIndex(values []float64, test float64) int {
	lowestIdx := 0
	lowestDiff := math.Inf(1)
	for idx := 0; idx < len(values); idx++ {
		diff := math.Abs(values[idx] - test)
		if diff < lowestDiff {
			lowestIdx = idx
			lowestDiff = diff
		}
	}
	return lowestIdx
}

// SecsToDuration converts floating point seconds to a time.Duration
func SecsToDuration(secs float64) time.Duration {
	return time.Duration(secs * float64(time.Second))
}
