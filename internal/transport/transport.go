package transport

import (
	"context"
	"net/http"
	"net/url"
)

// Request describes an immutable public REST request.
type Request struct {
	Method string
	Path   string
	Query  url.Values
}

// Response contains a bounded response body and selected diagnostic headers.
type Response struct {
	StatusCode int
	Header     http.Header
	Body       []byte
}

// Executor executes public REST requests.
type Executor interface {
	Do(ctx context.Context, request Request) (*Response, error)
}
