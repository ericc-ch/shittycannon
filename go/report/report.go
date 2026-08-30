package report

import (
	"math"
	"strconv"
	"strings"

	"ericc-ch/shittycannon/stats"
)

type Report struct {
	URL         string          `json:"url"`
	Connections int             `json:"connections"`
	Duration    float64         `json:"duration"`
	Errors      int             `json:"errors"`
	Timeouts    int             `json:"timeouts"`
	Non2xx      int             `json:"non2xx"`
	Status2xx   int             `json:"2xx"`
	Latency     stats.Histogram `json:"latency"`
	Requests    struct {
		Total int `json:"total"`
	} `json:"requests"`
	Throughput struct {
		Total int `json:"total"`
	} `json:"throughput"`
}

type Totals struct {
	Latencies []float64
	Bytes     int
	Status2xx int
	Non2xx    int
	Errors    int
	Timeouts  int
}

func From(url string, connections int, durationSeconds float64, totals Totals) Report {
	var result Report
	result.URL = url
	result.Connections = connections
	result.Duration = durationSeconds
	result.Errors = totals.Errors
	result.Timeouts = totals.Timeouts
	result.Non2xx = totals.Non2xx
	result.Status2xx = totals.Status2xx
	result.Latency = stats.From(totals.Latencies)
	result.Requests.Total = totals.Status2xx + totals.Non2xx + totals.Errors + totals.Timeouts
	result.Throughput.Total = totals.Bytes
	return result
}

func fmtNum(value float64) string {
	if value == math.Trunc(value) {
		return strconv.FormatInt(int64(value), 10)
	}
	return strconv.FormatFloat(value, 'f', 2, 64)
}

func FormatText(result Report, printLatency bool) string {
	lines := []string{
		"Running " + fmtNum(result.Duration) + "s test @ " + result.URL,
		strconv.Itoa(result.Connections) + " connections",
		"",
		"Stat     Avg     p50     p90     p99     Max",
		"Latency  " + fmtNum(result.Latency.Average) + "     " +
			fmtNum(result.Latency.P50) + "     " +
			fmtNum(result.Latency.P90) + "     " +
			fmtNum(result.Latency.P99) + "     " +
			fmtNum(result.Latency.Max),
		"",
		strconv.Itoa(result.Requests.Total) + " requests in " +
			fmtNum(result.Duration) + "s, " + strconv.Itoa(result.Throughput.Total) + " bytes",
		strconv.Itoa(result.Status2xx) + " 2xx, " + strconv.Itoa(result.Non2xx) +
			" non2xx, " + strconv.Itoa(result.Errors) + " errors, " +
			strconv.Itoa(result.Timeouts) + " timeouts",
	}
	if printLatency {
		lines = append(lines, "",
			"Latency min "+fmtNum(result.Latency.Min)+
				" avg "+fmtNum(result.Latency.Average)+
				" p50 "+fmtNum(result.Latency.P50)+
				" p90 "+fmtNum(result.Latency.P90)+
				" p99 "+fmtNum(result.Latency.P99)+
				" max "+fmtNum(result.Latency.Max),
		)
	}
	return strings.Join(lines, "\n")
}
