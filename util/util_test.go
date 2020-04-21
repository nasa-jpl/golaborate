package util_test

import (
	"fmt"
	"testing"
	"time"

	"github.jpl.nasa.gov/bdube/golab/util"
)

func ExampleArangeByte_EndOnly() {
	fmt.Println(util.ArangeByte(10))
	// Output: [0 1 2 3 4 5 6 7 8 9]
}

func ExampleArangeByte_StartEnd() {
	fmt.Println(util.ArangeByte(5, 15))
	// Output: [5 6 7 8 9 10 11 12 13 14]
}

func ExampleArangeByte_StartEndStep() {
	fmt.Println(util.ArangeByte(10, 22, 2))
	// Output: [10 12 14 16 18 20]
}

func ExampleSetBit_MSB() {
	out := util.SetBit(0, 7, true)
	fmt.Printf("%08b\n", out)
	// Output: 10000000
}

func ExampleSetBit_LSB() {
	out := util.SetBit(255, 0, false)
	fmt.Printf("%08b\n", out)
	// Output: 11111110
}

func TestArangeByteForward(t *testing.T) {
	var (
		start byte = 10
		end   byte = 20
		step  byte = 1
	)
	arangeRes := util.ArangeByte(start, end, step)
	for i := 0; i < len(arangeRes); i++ {
		expected := start + (byte(i) * step)
		if arangeRes[i] != expected {
			t.Errorf("expected %d at position %d, got %d", expected, i, arangeRes[i])
		}
	}
}

func TestUniqueString(t *testing.T) {
	inp := []string{"a", "b", "c", "a"}
	expected := []string{"a", "b", "c"}
	output := util.UniqueString(inp)
	for i := 0; i < len(output); i++ {
		if output[i] != expected[i] {
			t.Errorf("expected %s got %s", expected[i], output[i])
		}
	}
}

func TestIntSliceToCSV(t *testing.T) {
	inp := []int{1, 2, 3}
	expected := "1,2,3"
	out := util.IntSliceToCSV(inp)
	if expected != out {
		t.Errorf("expected %s got %s", expected, out)
	}
}

func TestClampHigh(t *testing.T) {
	var (
		low   = 0.
		high  = 10.
		input = 20.
	)
	clamped := util.Clamp(input, low, high)
	if clamped == input {
		t.Errorf("expected out of range value %f to be clipped to %f < x < %f, got %f", input, low, high, clamped)
	}
}

func TestClampLow(t *testing.T) {
	var (
		low   = 0.
		high  = 10.
		input = -1.
	)
	clamped := util.Clamp(input, low, high)
	if clamped == input {
		t.Errorf("expected out of range value %f to be clipped to %f < x < %f, got %f", input, low, high, clamped)
	}
}

func TestSecsToDuration(t *testing.T) {
	var dur time.Duration = 123456789
	secs := dur.Seconds()
	out := util.SecsToDuration(secs)
	if out != dur {
		t.Errorf("expected SecsToDuration to round trip, output %v != expected %v", out, dur)
	}
}
