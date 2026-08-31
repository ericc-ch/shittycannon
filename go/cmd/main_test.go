package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

var cliBin string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "shittycannon")
	if err != nil {
		os.Exit(1)
	}
	cliBin = filepath.Join(dir, "shittycannon")
	build := exec.Command("go", "build", "-o", cliBin, ".")
	build.Stderr = os.Stderr
	if err := build.Run(); err != nil {
		os.RemoveAll(dir)
		os.Exit(1)
	}
	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}

func runCli(t *testing.T, args ...string) (code int, stdout, stderr string) {
	t.Helper()
	cmd := exec.Command(cliBin, args...)
	out, err := cmd.Output()
	stderrBytes := []byte{}
	if exit, ok := err.(*exec.ExitError); ok {
		stderrBytes = exit.Stderr
		return exit.ExitCode(), string(out), string(stderrBytes)
	}
	if err != nil {
		t.Fatal(err)
	}
	return 0, string(out), string(stderrBytes)
}

func TestHelpListsSubsetFlags(t *testing.T) {
	code, stdout, _ := runCli(t, "--help")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	for _, flag := range []string{
		"--connections",
		"--duration",
		"--amount",
		"--method",
		"--headers",
		"--body",
		"--input",
		"--timeout",
		"--json",
		"--latency",
	} {
		if !strings.Contains(stdout, flag) {
			t.Fatalf("help missing %s\n%s", flag, stdout)
		}
	}
}

func TestUnknownAutocannonFlagsFail(t *testing.T) {
	code, _, _ := runCli(t, "--pipelining", "1", "http://127.0.0.1/")
	if code == 0 {
		t.Fatal("expected non-zero exit")
	}
}

func TestAmountRunAgainstLocalServer(t *testing.T) {
	var hits atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	code, stdout, stderr := runCli(t, "-c", "2", "-a", "20", "-j", server.URL+"/")
	if code != 0 {
		t.Fatalf("exit %d stderr %s", code, stderr)
	}
	var report struct {
		Requests struct {
			Total int `json:"total"`
		} `json:"requests"`
		Status2xx int `json:"2xx"`
	}
	if err := json.Unmarshal([]byte(stdout), &report); err != nil {
		t.Fatal(err)
	}
	if report.Requests.Total != 20 || report.Status2xx != 20 {
		t.Fatalf("report %+v stdout %s", report, stdout)
	}
	if hits.Load() != 20 {
		t.Fatalf("hits %d", hits.Load())
	}
}

func TestAmountIgnoresInvalidDuration(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	code, stdout, stderr := runCli(t, "-c", "1", "-a", "1", "-d", "0", "-j", server.URL+"/")
	if code != 0 {
		t.Fatalf("exit %d stderr %s", code, stderr)
	}
	var result struct {
		Status2xx int `json:"2xx"`
	}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatal(err)
	}
	if result.Status2xx != 1 {
		t.Fatalf("%+v %s", result, stdout)
	}
}

func TestBodyIsSentRegardlessOfMethod(t *testing.T) {
	var mu sync.Mutex
	var got string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Error(err)
		}
		mu.Lock()
		got = string(body)
		mu.Unlock()
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	code, _, stderr := runCli(t, "-c", "1", "-a", "1", "-m", "GET", "-b", "hello", "-j", server.URL+"/")
	if code != 0 {
		t.Fatalf("exit %d stderr %s", code, stderr)
	}
	mu.Lock()
	defer mu.Unlock()
	if got != "hello" {
		t.Fatalf("body %q", got)
	}
}

func TestPostBodyFromInputFile(t *testing.T) {
	type seen struct {
		method string
		body   string
	}
	var mu sync.Mutex
	var got []seen
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Error(err)
		}
		mu.Lock()
		got = append(got, seen{method: r.Method, body: string(body)})
		mu.Unlock()
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	file := filepath.Join(t.TempDir(), "body.json")
	if err := os.WriteFile(file, []byte(`{"ok":true}`), 0o600); err != nil {
		t.Fatal(err)
	}

	code, _, stderr := runCli(t, "-c", "1", "-a", "3", "-m", "POST", "-i", file, "-H", "content-type=application/json", "-j", server.URL+"/")
	if code != 0 {
		t.Fatalf("exit %d stderr %s", code, stderr)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(got) != 3 {
		t.Fatalf("got %v", got)
	}
	for _, item := range got {
		if item.method != "POST" || item.body != `{"ok":true}` {
			t.Fatalf("got %v", got)
		}
	}
}

func TestBodyAndInputTogetherIsUserError(t *testing.T) {
	file := filepath.Join(t.TempDir(), "x")
	if err := os.WriteFile(file, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	code, stdout, stderr := runCli(t, "-b", "hello", "-i", file, "http://127.0.0.1/")
	if code == 0 {
		t.Fatal("expected non-zero exit")
	}
	if !strings.Contains(stdout+stderr, "either") {
		t.Fatalf("stdout %s stderr %s", stdout, stderr)
	}
}

func TestNon2xxResponsesAreCounted(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "nope", http.StatusInternalServerError)
	}))
	defer server.Close()

	code, stdout, stderr := runCli(t, "-c", "1", "-a", "4", "-j", server.URL+"/")
	if code != 0 {
		t.Fatalf("exit %d stderr %s", code, stderr)
	}
	var result struct {
		Status2xx int `json:"2xx"`
		Non2xx    int `json:"non2xx"`
	}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatal(err)
	}
	if result.Status2xx != 0 || result.Non2xx != 4 {
		t.Fatalf("%+v %s", result, stdout)
	}
}

func TestTimeoutsAreCountedWhenHandlerIsSlow(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(3 * time.Second)
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	code, stdout, stderr := runCli(t, "-c", "1", "-a", "1", "-t", "1", "-j", server.URL+"/")
	if code != 0 {
		t.Fatalf("exit %d stderr %s", code, stderr)
	}
	var result struct {
		Timeouts int `json:"timeouts"`
	}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatal(err)
	}
	if result.Timeouts != 1 {
		t.Fatalf("%+v %s", result, stdout)
	}
}

func TestDurationCountsTheInFlightRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(1500 * time.Millisecond)
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	code, stdout, stderr := runCli(t, "-c", "1", "-d", "1", "-t", "5", "-j", server.URL+"/")
	if code != 0 {
		t.Fatalf("exit %d stderr %s", code, stderr)
	}
	var result struct {
		Status2xx int `json:"2xx"`
		Requests  struct {
			Total int `json:"total"`
		} `json:"requests"`
	}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatal(err)
	}
	if result.Status2xx != 1 || result.Requests.Total != 1 {
		t.Fatalf("%+v %s", result, stdout)
	}
}
