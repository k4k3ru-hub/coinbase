// Package rest implements Coinbase International Exchange REST market data.
package rest

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/k4k3ru-hub/coinbase/go/intx/marketdata"
)

const (
	ProductionURL                 = "https://api.international.coinbase.com"
	SandboxURL                    = "https://api-n5e1.coinbase.com"
	defaultMaxResponseBytes int64 = 8 << 20
)

// HTTPClient executes HTTP requests.
type HTTPClient interface {
	Do(*http.Request) (*http.Response, error)
}

// ClientOption configures an INTX REST client.
type ClientOption struct {
	BaseURL          string
	HTTPClient       HTTPClient
	MaxResponseBytes int64
}

// Client is the INTX REST composition root.
type Client struct {
	baseURL          string
	httpClient       HTTPClient
	maxResponseBytes int64
	marketData       *MarketDataClient
}

// MarketDataClient exposes public INTX perpetual market-data operations.
type MarketDataClient struct{ client *Client }

// DefaultClientOption returns production REST defaults.
//
// Version:
//   - 2026-08-24: Added.
func DefaultClientOption() *ClientOption {
	return &ClientOption{BaseURL: ProductionURL, HTTPClient: http.DefaultClient, MaxResponseBytes: defaultMaxResponseBytes}
}

// NewClient creates an INTX REST client and composes its market-data API.
//
// Version:
//   - 2026-08-24: Added.
func NewClient(option *ClientOption) (*Client, error) {
	if option == nil {
		option = DefaultClientOption()
	}
	base := strings.TrimRight(option.BaseURL, "/")
	if base == "" {
		base = ProductionURL
	}
	if _, err := url.ParseRequestURI(base); err != nil {
		return nil, fmt.Errorf("failed to create coinbase intx rest client: invalid base URL: %w", err)
	}
	if option.HTTPClient == nil {
		return nil, fmt.Errorf("failed to create coinbase intx rest client: http_client=null")
	}
	limit := option.MaxResponseBytes
	if limit == 0 {
		limit = defaultMaxResponseBytes
	}
	if limit < 0 {
		return nil, fmt.Errorf("failed to create coinbase intx rest client: max_response_bytes=out_of_range")
	}
	c := &Client{baseURL: base, httpClient: option.HTTPClient, maxResponseBytes: limit}
	c.marketData = &MarketDataClient{client: c}
	return c, nil
}

// MarketData returns the composed INTX market-data client.
//
// Version:
//   - 2026-08-24: Added.
func (c *Client) MarketData() *MarketDataClient {
	if c == nil {
		return nil
	}
	return c.marketData
}

// ListPerpetualInstruments returns only PERP instruments from the public instrument list.
//
// Version:
//   - 2026-08-24: Added.
func (c *MarketDataClient) ListPerpetualInstruments(ctx context.Context) ([]marketdata.Instrument, error) {
	if c == nil || c.client == nil {
		return nil, fmt.Errorf("failed to list coinbase intx perpetual instruments: client=null")
	}
	if ctx == nil {
		return nil, fmt.Errorf("failed to list coinbase intx perpetual instruments: context=null")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.client.baseURL+"/api/v1/instruments", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list coinbase intx perpetual instruments: failed to create HTTP request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	resp, err := c.client.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to list coinbase intx perpetual instruments: %w", err)
	}
	if resp == nil {
		return nil, fmt.Errorf("failed to list coinbase intx perpetual instruments: response=null")
	}
	if resp.Body == nil {
		return nil, fmt.Errorf("failed to list coinbase intx perpetual instruments: response_body=null")
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, c.client.maxResponseBytes+1))
	if err != nil {
		return nil, fmt.Errorf("failed to list coinbase intx perpetual instruments: failed to read response body: %w", err)
	}
	if int64(len(body)) > c.client.maxResponseBytes {
		return nil, fmt.Errorf("failed to list coinbase intx perpetual instruments: response_body=too_long max_bytes=%d", c.client.maxResponseBytes)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("failed to list coinbase intx perpetual instruments: unexpected HTTP status: status_code=%d", resp.StatusCode)
	}
	var all []marketdata.Instrument
	decoder := json.NewDecoder(strings.NewReader(string(body)))
	decoder.UseNumber()
	if err := decoder.Decode(&all); err != nil {
		return nil, fmt.Errorf("failed to list coinbase intx perpetual instruments: failed to decode response: %w", err)
	}
	perpetuals := make([]marketdata.Instrument, 0, len(all))
	for _, instrument := range all {
		if instrument.Type == marketdata.InstrumentTypePerpetual {
			perpetuals = append(perpetuals, instrument)
		}
	}
	return perpetuals, nil
}
