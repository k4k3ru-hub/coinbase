// Package rest implements Coinbase International Exchange REST market data.
package rest

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/k4k3ru-hub/coinbase/go/intx/marketdata"
)

const (
	ProductionURL                 = "https://api.international.coinbase.com"
	SandboxURL                    = "https://api-n5e1.coinbase.com"
	defaultMaxResponseBytes int64 = 8 << 20
	maxInstrumentLength           = 128
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

// FundingHistoryParams configures an INTX historical funding-rate request.
type FundingHistoryParams struct {
	Instrument   string
	ResultLimit  int
	ResultOffset int
}

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

// GetHistoricalFundingRates returns final funding rates for an INTX perpetual instrument.
//
// Parameters:
//   - ctx: request context
//   - params: instrument and pagination parameters
//
// Returns:
//   - Final funding-rate records.
//
// Version:
//   - 2026-08-27: Decoded the production pagination and results envelope.
//   - 2026-08-26: Added.
func (c *MarketDataClient) GetHistoricalFundingRates(ctx context.Context, params FundingHistoryParams) ([]marketdata.FundingRate, error) {
	if c == nil || c.client == nil {
		return nil, fmt.Errorf("failed to get coinbase intx historical funding rates: client=null")
	}
	if ctx == nil {
		return nil, fmt.Errorf("failed to get coinbase intx historical funding rates: context=null")
	}
	if params.Instrument == "" {
		return nil, fmt.Errorf("failed to get coinbase intx historical funding rates: instrument=empty")
	}
	if len(params.Instrument) > maxInstrumentLength {
		return nil, fmt.Errorf("failed to get coinbase intx historical funding rates: instrument=too_long actual_length=%d max_length=%d", len(params.Instrument), maxInstrumentLength)
	}
	if strings.TrimSpace(params.Instrument) != params.Instrument {
		return nil, fmt.Errorf("failed to get coinbase intx historical funding rates: instrument=invalid")
	}
	if params.ResultLimit < 0 || params.ResultLimit > 100 {
		return nil, fmt.Errorf("failed to get coinbase intx historical funding rates: result_limit=out_of_range min_value=0 max_value=100")
	}
	if params.ResultOffset < 0 {
		return nil, fmt.Errorf("failed to get coinbase intx historical funding rates: result_offset=out_of_range min_value=0")
	}
	query := url.Values{}
	if params.ResultLimit > 0 {
		query.Set("result_limit", strconv.Itoa(params.ResultLimit))
	}
	if params.ResultOffset > 0 {
		query.Set("result_offset", strconv.Itoa(params.ResultOffset))
	}
	requestURL := c.client.baseURL + "/api/v1/instruments/" + url.PathEscape(params.Instrument) + "/funding"
	if len(query) > 0 {
		requestURL += "?" + query.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get coinbase intx historical funding rates: failed to create HTTP request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	resp, err := c.client.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get coinbase intx historical funding rates: %w", err)
	}
	if resp == nil {
		return nil, fmt.Errorf("failed to get coinbase intx historical funding rates: response=null")
	}
	if resp.Body == nil {
		return nil, fmt.Errorf("failed to get coinbase intx historical funding rates: response_body=null")
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, c.client.maxResponseBytes+1))
	if err != nil {
		return nil, fmt.Errorf("failed to get coinbase intx historical funding rates: failed to read response body: %w", err)
	}
	if int64(len(body)) > c.client.maxResponseBytes {
		return nil, fmt.Errorf("failed to get coinbase intx historical funding rates: response_body=too_long max_bytes=%d", c.client.maxResponseBytes)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("failed to get coinbase intx historical funding rates: unexpected HTTP status: status_code=%d", resp.StatusCode)
	}
	rates, err := decodeFundingRates(body)
	if err != nil {
		return nil, fmt.Errorf("failed to get coinbase intx historical funding rates: failed to decode response: %w", err)
	}
	return rates, nil
}

func decodeFundingRates(body []byte) ([]marketdata.FundingRate, error) {
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) == 0 {
		return nil, fmt.Errorf("response_body=empty")
	}
	if trimmed[0] == '[' {
		var rates []marketdata.FundingRate
		decoder := json.NewDecoder(bytes.NewReader(trimmed))
		decoder.UseNumber()
		if err := decoder.Decode(&rates); err != nil {
			return nil, err
		}
		return rates, nil
	}
	var object map[string]json.RawMessage
	objectDecoder := json.NewDecoder(bytes.NewReader(trimmed))
	objectDecoder.UseNumber()
	if err := objectDecoder.Decode(&object); err != nil {
		return nil, err
	}
	if results, ok := object["results"]; ok {
		var rates []marketdata.FundingRate
		decoder := json.NewDecoder(bytes.NewReader(results))
		decoder.UseNumber()
		if err := decoder.Decode(&rates); err != nil {
			return nil, err
		}
		return rates, nil
	}
	var rate marketdata.FundingRate
	decoder := json.NewDecoder(bytes.NewReader(trimmed))
	decoder.UseNumber()
	if err := decoder.Decode(&rate); err != nil {
		return nil, err
	}
	return []marketdata.FundingRate{rate}, nil
}
