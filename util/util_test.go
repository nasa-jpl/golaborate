package util_test

import (
	"fmt"

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