// Package thermocube enables working with 200~400 series temperature controllers.
package thermocube

import "github.jpl.nasa.gov/HCIT/go-hcit/mathx"

func twoBytesToFloat(b []byte) float64 {
	// this algorithm has an equivalent, slower implementation that is perhaps
	// more intelligible:
	// b1 := byte(0x02)
	// b2 := byte(0xBC)
	// b := []byte{b1, b2}
	// intgr := binary.BigEndian.Uint16(b)
	// str := fmt.Sprint(intgr)
	// l := len(str)
	// front := str[:l-1]
	// decimal := str[l-1:]
	// str = fmt.Sprintf("%v.%v", front, decimal)
	// float, _ := strconv.ParseFloat(str, 64)
	// fmt.Println(float)

	// the above just works by doing:
	// <binary>
	// => 100
	// => "100"
	// => "10|0", split
	// => f64("10"."0") == 10.0

	// the algorithm below works at the binary level and is consequently much
	// faster (>100x)
	return float64(int(b[0])<<8+int(b[1])) / 10.0
}

// math.Round was introduced in Go 1.10, so this doesn't work on Go 1.9
// we use Go 1.9 so that we can compile for XP.  Leave this here for now,
// hopefully we can update to (latest) before this code is really needed
func floatToTwoBytes(f float64) []byte {
	n := int(mathx.Round(f*10.0, 1))
	b := []byte{byte(n >> 8), byte(n)}
	return b
}
