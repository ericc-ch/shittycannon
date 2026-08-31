package runoptions

import "net/url"

type Stop interface {
	stop()
}

type Duration struct {
	Seconds int
}

func (Duration) stop() {}

type Amount struct {
	Requests int
}

func (Amount) stop() {}

type Method string

const (
	MethodGet     Method = "GET"
	MethodPost    Method = "POST"
	MethodPut     Method = "PUT"
	MethodDelete  Method = "DELETE"
	MethodPatch   Method = "PATCH"
	MethodHead    Method = "HEAD"
	MethodOptions Method = "OPTIONS"
	MethodTrace   Method = "TRACE"
)

type RunOptions struct {
	URL            *url.URL
	Connections    int
	Stop           Stop
	Method         Method
	Headers        map[string]string
	Body           string
	TimeoutSeconds int
}
