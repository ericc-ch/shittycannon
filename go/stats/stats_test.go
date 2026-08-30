package stats

import "testing"

func TestHistogramEmpty(t *testing.T) {
	got := From(nil)
	if got != (Histogram{}) {
		t.Fatalf("got %+v, want zeros", got)
	}
}

func TestHistogramSingleSample(t *testing.T) {
	got := From([]float64{12})
	want := Histogram{Min: 12, Max: 12, Average: 12, P50: 12, P90: 12, P99: 12}
	if got != want {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestHistogramNearestRankTenSamples(t *testing.T) {
	got := From([]float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10})
	if got.Min != 1 || got.Max != 10 {
		t.Fatalf("min/max %+v", got)
	}
	if got.Average != 5.5 {
		t.Fatalf("average %v", got.Average)
	}
	if got.P50 != 5 || got.P90 != 9 || got.P99 != 10 {
		t.Fatalf("percentiles %+v", got)
	}
}
