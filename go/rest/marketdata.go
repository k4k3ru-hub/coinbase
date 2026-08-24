package rest

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/k4k3ru-hub/coinbase/go/internal/transport"
	"github.com/k4k3ru-hub/coinbase/go/marketdata"
)

const maxProductIDLength = 128

// MarketDataClient provides Coinbase Exchange public REST operations.
type MarketDataClient struct{ executor transport.Executor }

// CandlesParams configures a historic rates request.
type CandlesParams struct {
	ProductID   string
	Start, End  *time.Time
	Granularity int
}

// GetServerTime gets Exchange server time.
//
// Version:
//   - 2026-08-19: Added.
func (c *MarketDataClient) GetServerTime(ctx context.Context) (*marketdata.Time, error) {
	var out marketdata.Time
	if err := c.get(ctx, "/time", nil, &out); err != nil {
		return nil, fmt.Errorf("failed to get coinbase exchange server time: %w", err)
	}
	return &out, nil
}

// GetCurrencies gets public currency metadata.
//
// Version:
//   - 2026-08-19: Added.
func (c *MarketDataClient) GetCurrencies(ctx context.Context) ([]marketdata.Currency, error) {
	var out []marketdata.Currency
	if err := c.get(ctx, "/currencies", nil, &out); err != nil {
		return nil, fmt.Errorf("failed to get coinbase exchange currencies: %w", err)
	}
	return out, nil
}

// GetProducts gets public product metadata.
//
// Version:
//   - 2026-08-19: Added.
func (c *MarketDataClient) GetProducts(ctx context.Context) ([]marketdata.Product, error) {
	var out []marketdata.Product
	if err := c.get(ctx, "/products", nil, &out); err != nil {
		return nil, fmt.Errorf("failed to get coinbase exchange products: %w", err)
	}
	return out, nil
}

// GetProduct gets one public product.
//
// Version:
//   - 2026-08-19: Added.
func (c *MarketDataClient) GetProduct(ctx context.Context, productID string) (*marketdata.Product, error) {
	if err := validateProductID(productID); err != nil {
		return nil, fmt.Errorf("failed to get coinbase exchange product: %w", err)
	}
	var out marketdata.Product
	if err := c.get(ctx, productPath(productID), nil, &out); err != nil {
		return nil, fmt.Errorf("failed to get coinbase exchange product: %w: product_id=%q", err, productID)
	}
	return &out, nil
}

// GetProductBook gets a level-specific product order book.
//
// Version:
//   - 2026-08-19: Added.
func (c *MarketDataClient) GetProductBook(ctx context.Context, productID string, level marketdata.BookLevel) (*marketdata.ProductBook, error) {
	if err := validateProductID(productID); err != nil {
		return nil, fmt.Errorf("failed to get coinbase exchange product book: %w", err)
	}
	if level < 1 || level > 3 {
		return nil, fmt.Errorf("failed to get coinbase exchange product book: level=out_of_range")
	}
	resp, err := c.executor.Do(ctx, transport.Request{Method: http.MethodGet, Path: productPath(productID) + "/book", Query: url.Values{"level": {strconv.Itoa(int(level))}}})
	if err != nil {
		return nil, fmt.Errorf("failed to get coinbase exchange product book: %w: product_id=%q level=%d", err, productID, level)
	}
	var raw struct {
		Sequence   uint64 `json:"sequence"`
		Bids, Asks [][]json.RawMessage
	}
	if err := decode(resp, &raw); err != nil {
		return nil, fmt.Errorf("failed to get coinbase exchange product book: %w: product_id=%q level=%d", err, productID, level)
	}
	out := &marketdata.ProductBook{Sequence: raw.Sequence, Level: level}
	if out.Bids, err = decodeBookEntries(raw.Bids, level); err != nil {
		return nil, fmt.Errorf("failed to get coinbase exchange product book: invalid bids: %w: product_id=%q level=%d", err, productID, level)
	}
	if out.Asks, err = decodeBookEntries(raw.Asks, level); err != nil {
		return nil, fmt.Errorf("failed to get coinbase exchange product book: invalid asks: %w: product_id=%q level=%d", err, productID, level)
	}
	return out, nil
}

// GetProductTicker gets the latest product trade and quote snapshot.
//
// Version:
//   - 2026-08-19: Added.
func (c *MarketDataClient) GetProductTicker(ctx context.Context, productID string) (*marketdata.Ticker, error) {
	var out marketdata.Ticker
	if err := c.productGet(ctx, productID, "/ticker", &out); err != nil {
		return nil, fmt.Errorf("failed to get coinbase exchange product ticker: %w", err)
	}
	return &out, nil
}

// GetProductTrades gets recent public trades and cursor headers.
//
// Version:
//   - 2026-08-19: Added.
func (c *MarketDataClient) GetProductTrades(ctx context.Context, productID string) (*marketdata.TradesPage, error) {
	if err := validateProductID(productID); err != nil {
		return nil, fmt.Errorf("failed to get coinbase exchange product trades: %w", err)
	}
	resp, err := c.executor.Do(ctx, transport.Request{Method: http.MethodGet, Path: productPath(productID) + "/trades"})
	if err != nil {
		return nil, fmt.Errorf("failed to get coinbase exchange product trades: %w: product_id=%q", err, productID)
	}
	var trades []marketdata.Trade
	if err := decode(resp, &trades); err != nil {
		return nil, fmt.Errorf("failed to get coinbase exchange product trades: %w: product_id=%q", err, productID)
	}
	return &marketdata.TradesPage{Trades: trades, Pagination: marketdata.Pagination{Before: resp.Header.Get("CB-BEFORE"), After: resp.Header.Get("CB-AFTER")}}, nil
}

// GetHistoricRates gets historic product candles without filling missing intervals.
//
// Version:
//   - 2026-08-19: Added.
func (c *MarketDataClient) GetHistoricRates(ctx context.Context, p CandlesParams) ([]marketdata.Candle, error) {
	if err := validateProductID(p.ProductID); err != nil {
		return nil, fmt.Errorf("failed to get coinbase exchange historic rates: %w", err)
	}
	if !validGranularity(p.Granularity) {
		return nil, fmt.Errorf("failed to get coinbase exchange historic rates: granularity=invalid")
	}
	if (p.Start == nil) != (p.End == nil) {
		return nil, fmt.Errorf("failed to get coinbase exchange historic rates: start_end=invalid")
	}
	if p.Start != nil && p.End != nil {
		if !p.Start.Before(*p.End) {
			return nil, fmt.Errorf("failed to get coinbase exchange historic rates: time_range=invalid")
		}
		if p.End.Sub(*p.Start) > time.Duration(p.Granularity)*time.Second*300 {
			return nil, fmt.Errorf("failed to get coinbase exchange historic rates: candle_count=out_of_range max_candles=300")
		}
	}
	q := url.Values{"granularity": {strconv.Itoa(p.Granularity)}}
	if p.Start != nil {
		q.Set("start", p.Start.Format(time.RFC3339Nano))
		q.Set("end", p.End.Format(time.RFC3339Nano))
	}
	resp, err := c.executor.Do(ctx, transport.Request{Method: http.MethodGet, Path: productPath(p.ProductID) + "/candles", Query: q})
	if err != nil {
		return nil, fmt.Errorf("failed to get coinbase exchange historic rates: %w: product_id=%q granularity=%d", err, p.ProductID, p.Granularity)
	}
	var rows [][]json.RawMessage
	if err := decode(resp, &rows); err != nil {
		return nil, fmt.Errorf("failed to get coinbase exchange historic rates: %w", err)
	}
	out := make([]marketdata.Candle, 0, len(rows))
	for i, row := range rows {
		if len(row) != 6 {
			return nil, fmt.Errorf("failed to get coinbase exchange historic rates: candle_tuple=invalid index=%d tuple_length=%d", i, len(row))
		}
		var v marketdata.Candle
		if err := json.Unmarshal(row[0], &v.Time); err != nil {
			return nil, fmt.Errorf("failed to get coinbase exchange historic rates: invalid candle time: %w", err)
		}
		fields := []*string{&v.Low, &v.High, &v.Open, &v.Close, &v.Volume}
		for j, d := range fields {
			if err := json.Unmarshal(row[j+1], d); err != nil {
				return nil, fmt.Errorf("failed to get coinbase exchange historic rates: invalid candle decimal: %w index=%d", err, i)
			}
		}
		out = append(out, v)
	}
	return out, nil
}

// GetProductStats gets rolling product statistics.
//
// Version:
//   - 2026-08-19: Added.
func (c *MarketDataClient) GetProductStats(ctx context.Context, productID string) (*marketdata.Stats, error) {
	var out marketdata.Stats
	if err := c.productGet(ctx, productID, "/stats", &out); err != nil {
		return nil, fmt.Errorf("failed to get coinbase exchange product stats: %w", err)
	}
	return &out, nil
}

// GetAllProductVolume gets the Exchange product volume summary.
//
// Version:
//   - 2026-08-19: Added.
func (c *MarketDataClient) GetAllProductVolume(ctx context.Context) ([]marketdata.VolumeSummary, error) {
	var groups [][]marketdata.VolumeSummary
	if err := c.get(ctx, "/products/volume-summary", nil, &groups); err != nil {
		return nil, fmt.Errorf("failed to get coinbase exchange product volume summary: %w", err)
	}
	var out []marketdata.VolumeSummary
	for _, group := range groups {
		out = append(out, group...)
	}
	return out, nil
}

func (c *MarketDataClient) get(ctx context.Context, path string, q url.Values, out any) error {
	if c == nil || c.executor == nil {
		return fmt.Errorf("executor=null")
	}
	resp, err := c.executor.Do(ctx, transport.Request{Method: http.MethodGet, Path: path, Query: q})
	if err != nil {
		return err
	}
	if err := json.Unmarshal(resp.Body, out); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}
	return nil
}
func (c *MarketDataClient) productGet(ctx context.Context, id, suffix string, out any) error {
	if err := validateProductID(id); err != nil {
		return err
	}
	if err := c.get(ctx, productPath(id)+suffix, nil, out); err != nil {
		return fmt.Errorf("%w: product_id=%q", err, id)
	}
	return nil
}
func productPath(id string) string { return "/products/" + url.PathEscape(id) }
func validateProductID(id string) error {
	if id == "" {
		return fmt.Errorf("product_id=empty")
	}
	if len(id) > maxProductIDLength {
		return fmt.Errorf("product_id=too_long actual_length=%d max_length=%d", len(id), maxProductIDLength)
	}
	if strings.TrimSpace(id) != id {
		return fmt.Errorf("product_id=invalid")
	}
	return nil
}
func validGranularity(v int) bool {
	for _, n := range []int{60, 300, 900, 3600, 21600, 86400} {
		if v == n {
			return true
		}
	}
	return false
}
func decodeBookEntries(rows [][]json.RawMessage, level marketdata.BookLevel) ([]marketdata.BookEntry, error) {
	out := make([]marketdata.BookEntry, 0, len(rows))
	for i, row := range rows {
		if len(row) != 3 {
			return nil, fmt.Errorf("tuple_length=invalid index=%d actual_length=%d expected_length=3", i, len(row))
		}
		var price, size string
		if err := json.Unmarshal(row[0], &price); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(row[1], &size); err != nil {
			return nil, err
		}
		if level == marketdata.BookLevel3 {
			var id string
			if err := json.Unmarshal(row[2], &id); err != nil {
				return nil, err
			}
			out = append(out, marketdata.OrderBookEntry{Price: price, Size: size, OrderID: id})
		} else {
			var count uint64
			if err := json.Unmarshal(row[2], &count); err != nil {
				return nil, err
			}
			out = append(out, marketdata.AggregatedBookEntry{Price: price, Size: size, OrderCount: count})
		}
	}
	return out, nil
}
