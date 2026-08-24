// Package rest implements Coinbase Advanced Trade public REST market data.
package rest

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/k4k3ru-hub/coinbase/go/advanced/marketdata"
)

const (
	ProductionURL                 = "https://api.coinbase.com"
	defaultMaxResponseBytes int64 = 8 << 20
)

// HTTPClient executes HTTP requests.
type HTTPClient interface {
	Do(*http.Request) (*http.Response, error)
}

// ClientOption configures an Advanced Trade public REST client.
type ClientOption struct {
	BaseURL          string
	HTTPClient       HTTPClient
	MaxResponseBytes int64
}

// Client is the Advanced Trade public REST composition root.
type Client struct {
	baseURL          string
	httpClient       HTTPClient
	maxResponseBytes int64
	marketData       *MarketDataClient
}

// MarketDataClient exposes Advanced Trade public perpetual market data.
type MarketDataClient struct{ client *Client }

// DefaultClientOption returns production REST defaults.
//
// Version:
//   - 2026-08-24: Added.
func DefaultClientOption() *ClientOption {
	return &ClientOption{BaseURL: ProductionURL, HTTPClient: http.DefaultClient, MaxResponseBytes: defaultMaxResponseBytes}
}

// NewClient creates an Advanced Trade public REST client.
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
	parsed, err := url.ParseRequestURI(base)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		if err == nil {
			err = fmt.Errorf("absolute URL required")
		}
		return nil, fmt.Errorf("failed to create coinbase advanced rest client: invalid base URL: %w", err)
	}
	if option.HTTPClient == nil {
		return nil, fmt.Errorf("failed to create coinbase advanced rest client: http_client=null")
	}
	limit := option.MaxResponseBytes
	if limit == 0 {
		limit = defaultMaxResponseBytes
	}
	if limit < 0 {
		return nil, fmt.Errorf("failed to create coinbase advanced rest client: max_response_bytes=out_of_range")
	}
	c := &Client{baseURL: base, httpClient: option.HTTPClient, maxResponseBytes: limit}
	c.marketData = &MarketDataClient{client: c}
	return c, nil
}

// MarketData returns the composed public market-data client.
//
// Version:
//   - 2026-08-24: Added.
func (c *Client) MarketData() *MarketDataClient {
	if c == nil {
		return nil
	}
	return c.marketData
}

// ListPerpetualProducts lists public FUTURE products with PERPETUAL expiry.
//
// Version:
//   - 2026-08-24: Added.
func (c *MarketDataClient) ListPerpetualProducts(ctx context.Context) ([]marketdata.Product, error) {
	if c == nil || c.client == nil {
		return nil, fmt.Errorf("failed to list coinbase advanced perpetual products: client=null")
	}
	if ctx == nil {
		return nil, fmt.Errorf("failed to list coinbase advanced perpetual products: context=null")
	}
	values := url.Values{}
	values.Set("product_type", marketdata.ProductTypeFuture)
	values.Set("contract_expiry_type", marketdata.ContractExpiryPerpetual)
	var response struct {
		Products []marketdata.Product `json:"products"`
	}
	if err := c.client.get(ctx, "/api/v3/brokerage/market/products", values, &response); err != nil {
		return nil, fmt.Errorf("failed to list coinbase advanced perpetual products: %w", err)
	}
	products := make([]marketdata.Product, 0, len(response.Products))
	for _, product := range response.Products {
		if product.ProductType == marketdata.ProductTypeFuture && product.FutureProductDetails != nil && product.FutureProductDetails.ContractExpiryType == marketdata.ContractExpiryPerpetual {
			products = append(products, product)
		}
	}
	return products, nil
}

// GetPerpetualProduct gets one public perpetual product and rejects other product types.
//
// Version:
//   - 2026-08-24: Added.
func (c *MarketDataClient) GetPerpetualProduct(ctx context.Context, productID string) (*marketdata.Product, error) {
	if c == nil || c.client == nil {
		return nil, fmt.Errorf("failed to get coinbase advanced perpetual product: client=null")
	}
	if ctx == nil {
		return nil, fmt.Errorf("failed to get coinbase advanced perpetual product: context=null")
	}
	if productID == "" {
		return nil, fmt.Errorf("failed to get coinbase advanced perpetual product: product_id=empty")
	}
	if !strings.HasSuffix(productID, "-PERP-INTX") {
		return nil, fmt.Errorf("failed to get coinbase advanced perpetual product: product_id=invalid")
	}
	var product marketdata.Product
	if err := c.client.get(ctx, "/api/v3/brokerage/market/products/"+url.PathEscape(productID), nil, &product); err != nil {
		return nil, fmt.Errorf("failed to get coinbase advanced perpetual product: %w: product_id=%q", err, productID)
	}
	if product.ProductType != marketdata.ProductTypeFuture || product.FutureProductDetails == nil || product.FutureProductDetails.ContractExpiryType != marketdata.ContractExpiryPerpetual {
		return nil, fmt.Errorf("failed to get coinbase advanced perpetual product: product_type=invalid product_id=%q", productID)
	}
	return &product, nil
}

func (c *Client) get(ctx context.Context, path string, query url.Values, out any) error {
	requestURL := c.baseURL + path
	if len(query) > 0 {
		requestURL += "?" + query.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	if resp == nil {
		return fmt.Errorf("response=null")
	}
	if resp.Body == nil {
		return fmt.Errorf("response_body=null")
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, c.maxResponseBytes+1))
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}
	if int64(len(body)) > c.maxResponseBytes {
		return fmt.Errorf("response_body=too_long max_bytes=%d", c.maxResponseBytes)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("unexpected HTTP status: status_code=%d", resp.StatusCode)
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}
	return nil
}
