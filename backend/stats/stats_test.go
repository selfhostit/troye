package stats_test

import (
	"testing"

	"github.com/selfhostit/troye/stats"
)

func TestMean(t *testing.T) {
	if m, ok := stats.Mean([]float64{1, 2, 6}); !ok || m != 3 {
		t.Errorf("mean %v %v", m, ok)
	}
	if _, ok := stats.Mean(nil); ok {
		t.Error("a mean of nothing")
	}
}

func TestMedian(t *testing.T) {
	xs := []float64{9, 1, 5}
	if m, ok := stats.Median(xs); !ok || m != 5 {
		t.Errorf("odd median %v", m)
	}
	if xs[0] != 9 {
		t.Error("the input was sorted in place")
	}
	if m, _ := stats.Median([]float64{4, 1, 3, 2}); m != 2.5 {
		t.Errorf("even median %v", m)
	}
	if _, ok := stats.Median(nil); ok {
		t.Error("a median of nothing")
	}
}

func TestRound(t *testing.T) {
	if r := stats.Round(82.349, 1); r != 82.3 {
		t.Errorf("round %v", r)
	}
}
