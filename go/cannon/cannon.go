package cannon

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"ericc-ch/shittycannon/report"
	"ericc-ch/shittycannon/runoptions"
)

type intake struct {
	mu        sync.Mutex
	closed    bool
	remaining *int
}

func (i *intake) take() bool {
	i.mu.Lock()
	defer i.mu.Unlock()
	if i.closed {
		return false
	}
	if i.remaining != nil {
		if *i.remaining <= 0 {
			i.closed = true
			return false
		}
		*i.remaining--
	}
	return true
}

func (i *intake) close() {
	i.mu.Lock()
	i.closed = true
	i.mu.Unlock()
}

type totals struct {
	mu        sync.Mutex
	latencies []float64
	bytes     int
	status2xx int
	non2xx    int
	errors    int
	timeouts  int
}

func (t *totals) snapshot() report.Totals {
	t.mu.Lock()
	defer t.mu.Unlock()
	latencies := make([]float64, len(t.latencies))
	copy(latencies, t.latencies)
	return report.Totals{
		Latencies: latencies,
		Bytes:     t.bytes,
		Status2xx: t.status2xx,
		Non2xx:    t.non2xx,
		Errors:    t.errors,
		Timeouts:  t.timeouts,
	}
}

func (t *totals) recordLatency(elapsed time.Duration, bytes int, statusCode int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.latencies = append(t.latencies, float64(elapsed)/float64(time.Millisecond))
	t.bytes += bytes
	if statusCode >= 200 && statusCode < 300 {
		t.status2xx++
	} else {
		t.non2xx++
	}
}

func (t *totals) recordFailure(err error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if errors.Is(err, context.DeadlineExceeded) {
		t.timeouts++
	} else {
		t.errors++
	}
}

func fireOnce(client *http.Client, options runoptions.RunOptions, acc *totals) {
	var body io.Reader
	if options.Body != "" {
		body = strings.NewReader(options.Body)
	}
	req, err := http.NewRequest(string(options.Method), options.URL.String(), body)
	if err != nil {
		acc.recordFailure(err)
		return
	}
	for key, value := range options.Headers {
		req.Header.Set(key, value)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(options.TimeoutSeconds)*time.Second)
	defer cancel()
	req = req.WithContext(ctx)

	started := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		acc.recordFailure(err)
		return
	}
	buf, readErr := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if readErr != nil {
		acc.recordFailure(readErr)
		return
	}
	acc.recordLatency(time.Since(started), len(buf), resp.StatusCode)
}

func Run(options runoptions.RunOptions) report.Report {
	client := &http.Client{}
	if base, ok := http.DefaultTransport.(*http.Transport); ok {
		transport := base.Clone()
		transport.MaxIdleConnsPerHost = options.Connections
		transport.ForceAttemptHTTP2 = false
		if transport.TLSClientConfig == nil {
			transport.TLSClientConfig = &tls.Config{}
		}
		transport.TLSClientConfig.NextProtos = []string{"http/1.1"}
		client.Transport = transport
	}

	acc := &totals{}
	gate := &intake{}

	var wg sync.WaitGroup
	var waitForStop func()
	switch stop := options.Stop.(type) {
	case runoptions.Amount:
		remaining := stop.Requests
		gate.remaining = &remaining
		waitForStop = func() { wg.Wait() }
	case runoptions.Duration:
		waitForStop = func() {
			time.Sleep(time.Duration(stop.Seconds) * time.Second)
			gate.close()
			wg.Wait()
		}
	default:
		panic(fmt.Sprintf("cannon: unknown stop condition %T", options.Stop))
	}

	started := time.Now()
	for range options.Connections {
		wg.Go(func() {
			for {
				if !gate.take() {
					return
				}
				fireOnce(client, options, acc)
			}
		})
	}
	waitForStop()

	return report.From(options.URL.String(), options.Connections, time.Since(started).Seconds(), acc.snapshot())
}
