// Package stats holds the small pieces of arithmetic the insights are built
// from. Every function is pure: no database, no clock, no time zones.
package stats

import (
	"math"
	"slices"
)

// Mean is the average of xs, and false when there are none.
func Mean(xs []float64) (float64, bool) {
	if len(xs) == 0 {
		return 0, false
	}
	var sum float64
	for _, x := range xs {
		sum += x
	}
	return sum / float64(len(xs)), true
}

// Median is the middle of xs (the mean of the two middle values for an even
// count), and false when there are none. xs is left as it was.
func Median(xs []float64) (float64, bool) {
	if len(xs) == 0 {
		return 0, false
	}
	s := slices.Clone(xs)
	slices.Sort(s)
	mid := len(s) / 2
	if len(s)%2 == 1 {
		return s[mid], true
	}
	return (s[mid-1] + s[mid]) / 2, true
}

// Round rounds x to the given number of decimals.
func Round(x float64, decimals int) float64 {
	p := math.Pow(10, float64(decimals))
	return math.Round(x*p) / p
}
