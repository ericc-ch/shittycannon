package cannon

import (
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"

	"ericc-ch/shittycannon/runoptions"
)

func TestHttpsStaysOnHTTP1WhenServerOffersHTTP2(t *testing.T) {
	var proto atomic.Value
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		proto.Store(r.Proto)
		_, _ = w.Write([]byte("ok"))
	}))
	server.EnableHTTP2 = true
	server.StartTLS()
	defer server.Close()

	roots := x509.NewCertPool()
	roots.AddCert(server.Certificate())
	original := http.DefaultTransport
	trusted := original.(*http.Transport).Clone()
	trusted.TLSClientConfig = &tls.Config{RootCAs: roots}
	http.DefaultTransport = trusted
	t.Cleanup(func() { http.DefaultTransport = original })

	parsed, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	result := Run(runoptions.RunOptions{
		URL:            parsed,
		Connections:    1,
		Stop:           runoptions.Amount{Requests: 1},
		Method:         runoptions.MethodGet,
		Body:           "",
		TimeoutSeconds: 5,
	})
	if result.Status2xx != 1 {
		t.Fatalf("status %+v proto %v", result, proto.Load())
	}
	got, _ := proto.Load().(string)
	if got != "HTTP/1.1" {
		t.Fatalf("proto %q", got)
	}
}
