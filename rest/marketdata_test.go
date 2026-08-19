package rest

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/k4k3ru-hub/coinbase-exchange/go/marketdata"
)

func TestMarketDataOperations(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/time":
			w.Write([]byte(`{"iso":"2026-08-19T00:00:00.123456Z","epoch":123.5}`))
		case "/currencies":
			w.Write([]byte(`[{"id":"BTC","name":"Bitcoin","min_size":"0.1","max_precision":"0.00000001","convertible_to":["USD"],"supported_networks":[{"id":"bitcoin","name":"Bitcoin"}]}]`))
		case "/products":
			w.Write([]byte(`[{"id":"BTC-USD","base_currency":"BTC","quote_currency":"USD","base_increment":"0.00000001","quote_increment":"0.01","product_type":"SPOT"}]`))
		case "/products/A/B":
			if r.URL.EscapedPath() != "/products/A%2FB" {
				t.Fatalf("path was not escaped: %s", r.URL.EscapedPath())
			}
			w.Write([]byte(`{"id":"A/B"}`))
		case "/products/BTC-USD/book":
			if r.URL.Query().Get("level") == "3" {
				w.Write([]byte(`{"sequence":8,"bids":[["1","2","order"]],"asks":[]}`))
				return
			}
			w.Write([]byte(`{"sequence":7,"bids":[["1","2",3]],"asks":[["4","5",6]]}`))
		case "/products/BTC-USD/ticker":
			w.Write([]byte(`{"trade_id":9007199254740993,"price":"1.1","size":"2","bid":"1","ask":"2","volume":"3","time":"2026-08-19T00:00:00.123456789Z"}`))
		case "/products/BTC-USD/trades":
			w.Header().Set("CB-BEFORE", "11")
			w.Header().Set("CB-AFTER", "9")
			w.Write([]byte(`[{"trade_id":10,"price":"1","size":"2","side":"sell","time":"x"}]`))
		case "/products/BTC-USD/candles":
			w.Write([]byte(`[[1,"2","3","4","5","6"]]`))
		case "/products/BTC-USD/stats":
			w.Write([]byte(`{"open":"1","high":"2","low":"0","last":"1.5","volume":"9","volume_30day":"99"}`))
		case "/products/volume-summary":
			w.Write([]byte(`[[{"id":"BTC-USD","market_types":["spot"],"spot_volume_24hour":"1"}]]`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	c, err := NewClient(&ClientOption{BaseURL: server.URL, HTTPClient: server.Client()})
	if err != nil {
		t.Fatal(err)
	}
	api := c.MarketData()
	ctx := context.Background()
	if v, e := api.GetServerTime(ctx); e != nil || v.ISO == "" {
		t.Fatalf("time: %#v %v", v, e)
	}
	if v, e := api.GetCurrencies(ctx); e != nil || len(v) != 1 || v[0].SupportedNetworks[0].ID != "bitcoin" {
		t.Fatalf("currencies: %#v %v", v, e)
	}
	if v, e := api.GetProducts(ctx); e != nil || v[0].ProductType != "SPOT" {
		t.Fatalf("products: %#v %v", v, e)
	}
	if v, e := api.GetProduct(ctx, "A/B"); e != nil || v.ID != "A/B" {
		t.Fatalf("product: %#v %v", v, e)
	}
	for _, level := range []marketdata.BookLevel{1, 2, 3} {
		if v, e := api.GetProductBook(ctx, "BTC-USD", level); e != nil || v.Level != level {
			t.Fatalf("book %d: %#v %v", level, v, e)
		}
	}
	if _, e := api.GetProductTicker(ctx, "BTC-USD"); e != nil {
		t.Fatal(e)
	}
	if v, e := api.GetProductTrades(ctx, "BTC-USD"); e != nil || v.Pagination.Before != "11" || v.Trades[0].Side != "sell" {
		t.Fatalf("trades: %#v %v", v, e)
	}
	start := time.Unix(0, 0).UTC()
	end := start.Add(time.Minute)
	if v, e := api.GetHistoricRates(ctx, CandlesParams{ProductID: "BTC-USD", Start: &start, End: &end, Granularity: 60}); e != nil || v[0].Volume != "6" {
		t.Fatalf("candles: %#v %v", v, e)
	}
	if _, e := api.GetProductStats(ctx, "BTC-USD"); e != nil {
		t.Fatal(e)
	}
	if v, e := api.GetAllProductVolume(ctx); e != nil || len(v) != 1 {
		t.Fatalf("volume: %#v %v", v, e)
	}
}

func TestValidationAndErrors(t *testing.T) {
	t.Parallel()
	api := (&MarketDataClient{})
	if _, e := api.GetProduct(context.Background(), ""); e == nil || !strings.Contains(e.Error(), "product_id=empty") {
		t.Fatalf("empty: %v", e)
	}
	if _, e := api.GetProductBook(context.Background(), "BTC-USD", 4); e == nil {
		t.Fatal("expected invalid level")
	}
	start := time.Now()
	if _, e := api.GetHistoricRates(context.Background(), CandlesParams{ProductID: "BTC-USD", Start: &start, Granularity: 60}); e == nil {
		t.Fatal("expected paired time validation")
	}
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "2")
		http.Error(w, `{"message":"limited"}`, 429)
	}))
	defer s.Close()
	c, _ := NewClient(&ClientOption{BaseURL: s.URL, HTTPClient: s.Client(), MaxResponseBytes: 1024})
	_, e := c.MarketData().GetProducts(context.Background())
	var re *ResponseError
	if !errors.As(e, &re) || re.StatusCode != 429 || re.RetryAfter != "2" {
		t.Fatalf("response error: %#v %v", re, e)
	}
}

func TestInvalidBookTupleAndBodyLimit(t *testing.T) {
	t.Parallel()
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "book") {
			w.Write([]byte(`{"sequence":1,"bids":[["1"]],"asks":[]}`))
			return
		}
		w.Write([]byte("12345"))
	}))
	defer s.Close()
	c, _ := NewClient(&ClientOption{BaseURL: s.URL, HTTPClient: s.Client(), MaxResponseBytes: 4})
	if _, e := c.MarketData().GetProductBook(context.Background(), "BTC-USD", 1); e == nil {
		t.Fatal("expected invalid tuple")
	}
	if _, e := c.MarketData().GetProducts(context.Background()); e == nil || !strings.Contains(e.Error(), "too_long") {
		t.Fatalf("limit: %v", e)
	}
}
