package report

import (
	"strings"
	"testing"
)

func TestFromSumsRequestsAndKeepsTotals(t *testing.T) {
	result := From("http://127.0.0.1:1/", 4, 1.5, Totals{
		Latencies: []float64{1, 2, 3},
		Bytes:     42,
		Status2xx: 7,
		Non2xx:    2,
		Errors:    1,
		Timeouts:  1,
	})
	if result.URL != "http://127.0.0.1:1/" || result.Connections != 4 || result.Duration != 1.5 {
		t.Fatalf("got %+v", result)
	}
	if result.Status2xx != 7 || result.Non2xx != 2 || result.Errors != 1 || result.Timeouts != 1 {
		t.Fatalf("got %+v", result)
	}
	if result.Requests.Total != 11 {
		t.Fatalf("requests total %d, want 11", result.Requests.Total)
	}
	if result.Throughput.Total != 42 {
		t.Fatalf("throughput total %d, want 42", result.Throughput.Total)
	}
	if result.Latency.Average != 2 || result.Latency.P50 != 2 {
		t.Fatalf("latency %+v", result.Latency)
	}
}

func TestFormatTextRendersWholeAndFractionalNumbers(t *testing.T) {
	result := From("http://127.0.0.1:1/", 1, 10, Totals{
		Latencies: []float64{1.5},
		Bytes:     3,
		Status2xx: 1,
	})
	text := FormatText(result, true)
	for _, want := range []string{
		"Running 10s test @ http://127.0.0.1:1/",
		"1 connections",
		"Latency  1.50     1.50     1.50     1.50     1.50",
		"1 requests in 10s, 3 bytes",
		"1 2xx, 0 non2xx, 0 errors, 0 timeouts",
		"Latency min 1.50 avg 1.50 p50 1.50 p90 1.50 p99 1.50 max 1.50",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %q in:\n%s", want, text)
		}
	}
}
