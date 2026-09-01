package stats

import (
	"math"
	"slices"
)

type Histogram struct {
	Min     float64 `json:"min"`
	Max     float64 `json:"max"`
	Average float64 `json:"average"`
	P50     float64 `json:"p50"`
	P90     float64 `json:"p90"`
	P99     float64 `json:"p99"`
}

func percentile(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	rank := int(math.Ceil((p/100)*float64(len(sorted)))) - 1
	rank = min(max(rank, 0), len(sorted)-1)
	return sorted[rank]
}

func From(samples []float64) Histogram {
	if len(samples) == 0 {
		return Histogram{}
	}
	sorted := slices.Clone(samples)
	slices.Sort(sorted)
	total := 0.0
	for _, sample := range sorted {
		total += sample
	}
	return Histogram{
		Min:     sorted[0],
		Max:     sorted[len(sorted)-1],
		Average: total / float64(len(sorted)),
		P50:     percentile(sorted, 50),
		P90:     percentile(sorted, 90),
		P99:     percentile(sorted, 99),
	}
}
