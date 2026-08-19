package rest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/k4k3ru-hub/coinbase-exchange/go/internal/transport"
)

const (
	ProductionURL                 = "https://api.exchange.coinbase.com"
	SandboxURL                    = "https://api-public.sandbox.exchange.coinbase.com"
	defaultMaxResponseBytes int64 = 8 << 20
)

// HTTPClient executes HTTP requests.
type HTTPClient interface {
	Do(*http.Request) (*http.Response, error)
}

// ClientOption configures a REST client.
type ClientOption struct {
	BaseURL          string
	HTTPClient       HTTPClient
	MaxResponseBytes int64
}

// Client is the Exchange public REST composition root.
type Client struct {
	baseURL          string
	httpClient       HTTPClient
	maxResponseBytes int64
	marketData       *MarketDataClient
}

// ResponseError represents a non-2xx Exchange response.
type ResponseError struct {
	StatusCode                     int
	Message, RequestID, RetryAfter string
	Before, After                  string
}

// Error returns a bounded REST error description.
//
// Version:
//   - 2026-08-19: Added.
func (e *ResponseError) Error() string {
	if e == nil {
		return "failed to execute coinbase exchange request: response_error=null"
	}
	return fmt.Sprintf("failed to execute coinbase exchange request: unexpected HTTP status: status_code=%d message=%q", e.StatusCode, e.Message)
}

// DefaultClientOption returns production REST defaults.
//
// Version:
//   - 2026-08-19: Added.
func DefaultClientOption() *ClientOption {
	return &ClientOption{BaseURL: ProductionURL, HTTPClient: http.DefaultClient, MaxResponseBytes: defaultMaxResponseBytes}
}

// NewClient creates a REST client and composes the public Market Data API.
//
// Parameters:
//   - option: client options
//
// Returns:
//   - Composed REST client.
//
// Version:
//   - 2026-08-19: Added.
func NewClient(option *ClientOption) (*Client, error) {
	if option == nil {
		option = DefaultClientOption()
	}
	base := strings.TrimRight(option.BaseURL, "/")
	if base == "" {
		base = ProductionURL
	}
	if _, err := url.ParseRequestURI(base); err != nil {
		return nil, fmt.Errorf("failed to create coinbase exchange rest client: invalid base URL: %w", err)
	}
	hc := option.HTTPClient
	if hc == nil {
		return nil, fmt.Errorf("failed to create coinbase exchange rest client: http_client=null")
	}
	limit := option.MaxResponseBytes
	if limit == 0 {
		limit = defaultMaxResponseBytes
	}
	if limit < 0 {
		return nil, fmt.Errorf("failed to create coinbase exchange rest client: max_response_bytes=out_of_range")
	}
	c := &Client{baseURL: base, httpClient: hc, maxResponseBytes: limit}
	c.marketData = &MarketDataClient{executor: c}
	return c, nil
}

// MarketData returns the composed public Market Data API client.
//
// Version:
//   - 2026-08-19: Added.
func (c *Client) MarketData() *MarketDataClient {
	if c == nil {
		return nil
	}
	return c.marketData
}

// Do executes a bounded public REST request.
//
// Version:
//   - 2026-08-19: Added.
func (c *Client) Do(ctx context.Context, r transport.Request) (*transport.Response, error) {
	if c == nil {
		return nil, fmt.Errorf("failed to execute coinbase exchange request: client=null")
	}
	if ctx == nil {
		return nil, fmt.Errorf("failed to execute coinbase exchange request: context=null")
	}
	requestURL := c.baseURL + "/" + strings.TrimLeft(r.Path, "/")
	if encodedQuery := r.Query.Encode(); encodedQuery != "" {
		requestURL += "?" + encodedQuery
	}
	req, err := http.NewRequestWithContext(ctx, r.Method, requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to execute coinbase exchange request: failed to create HTTP request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute coinbase exchange request: %w", err)
	}
	if resp == nil {
		return nil, fmt.Errorf("failed to execute coinbase exchange request: response=null")
	}
	if resp.Body == nil {
		return nil, fmt.Errorf("failed to execute coinbase exchange request: response_body=null")
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, c.maxResponseBytes+1))
	if err != nil {
		return nil, fmt.Errorf("failed to execute coinbase exchange request: failed to read response body: %w", err)
	}
	if int64(len(body)) > c.maxResponseBytes {
		return nil, fmt.Errorf("failed to execute coinbase exchange request: response_body=too_long max_bytes=%d", c.maxResponseBytes)
	}
	out := &transport.Response{StatusCode: resp.StatusCode, Header: resp.Header.Clone(), Body: body}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var wire struct {
			Message string `json:"message"`
		}
		_ = json.Unmarshal(body, &wire)
		if wire.Message == "" {
			wire.Message = http.StatusText(resp.StatusCode)
		}
		return nil, &ResponseError{StatusCode: resp.StatusCode, Message: wire.Message, RequestID: first(resp.Header, "CB-REQUEST-ID", "X-REQUEST-ID"), RetryAfter: resp.Header.Get("Retry-After"), Before: resp.Header.Get("CB-BEFORE"), After: resp.Header.Get("CB-AFTER")}
	}
	return out, nil
}
func first(h http.Header, names ...string) string {
	for _, n := range names {
		if v := h.Get(n); v != "" {
			return v
		}
	}
	return ""
}
func decode[T any](resp *transport.Response, out *T) error {
	if resp == nil {
		return errors.New("response=null")
	}
	if err := json.Unmarshal(resp.Body, out); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}
	return nil
}

var _ transport.Executor = (*Client)(nil)
var _ = time.Second
