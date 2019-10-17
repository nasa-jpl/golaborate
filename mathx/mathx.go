// Package mathx provides the Round function from go1.10+ for go1.9.  It is not exactly the same, but is fine for our uses
package mathx

// Round rounds a float to the nearest "unit" (0.1 for tenth, 0.01 for hundredth, and so on).
func Round(x, unit float64) float64 {
	return float64(int64(x/unit+0.5)) * unit
}
