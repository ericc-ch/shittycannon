package cannon

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net/http"
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

func fireOnce(client *http.Client, options runoptions.RunOptions, acc *totals) {
	var body io.Reader
	switch b := options.Body.(type) {
	case runoptions.TextBody:
		if runoptions.AllowsBody(options.Method) {
			body = bytes.NewReader([]byte(b.Value))
		}
	}
	req, err := http.NewRequest(string(options.Method), options.URL.String(), body)
	if err != nil {
		acc.mu.Lock()
		acc.errors++
		acc.mu.Unlock()
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
		acc.mu.Lock()
		if errors.Is(err, context.DeadlineExceeded) {
			acc.timeouts++
		} else {
			acc.errors++
		}
		acc.mu.Unlock()
		return
	}
	buf, readErr := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if readErr != nil {
		acc.mu.Lock()
		if errors.Is(readErr, context.DeadlineExceeded) {
			acc.timeouts++
		} else {
			acc.errors++
		}
		acc.mu.Unlock()
		return
	}
	elapsed := time.Since(started)
	acc.mu.Lock()
	acc.latencies = append(acc.latencies, float64(elapsed.Milliseconds()))
	acc.bytes += len(buf)
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		acc.status2xx++
	} else {
		acc.non2xx++
	}
	acc.mu.Unlock()
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
	switch stop := options.Stop.(type) {
	case runoptions.Amount:
		remaining := stop.Requests
		gate.remaining = &remaining
	}

	var wg sync.WaitGroup
	started := time.Now()
	for range options.Connections {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				if !gate.take() {
					return
				}
				fireOnce(client, options, acc)
			}
		}()
	}

	switch stop := options.Stop.(type) {
	case runoptions.Duration:
		time.Sleep(time.Duration(stop.Seconds) * time.Second)
		gate.close()
		wg.Wait()
	default:
		wg.Wait()
	}

	return report.From(options.URL.String(), options.Connections, time.Since(started).Seconds(), acc.snapshot())
}
