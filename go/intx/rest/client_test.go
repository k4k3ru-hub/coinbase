package rest

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewClientComposesMarketData(t *testing.T) {
	client, err := NewClient(&ClientOption{BaseURL: "https://example.com", HTTPClient: http.DefaultClient})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	if client.MarketData() == nil {
		t.Fatal("MarketData() = nil")
	}
}

func TestListPerpetualInstrumentsFiltersSpotAndPreservesDecimals(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/instruments" {
			t.Errorf("path = %q", r.URL.Path)
		}
		_, _ = w.Write([]byte(`[{"instrument_id":1,"instrument_uuid":"97645486-8058-4d98-aa1e-5ab2685d09c8","symbol":"BTC-PERP","type":"PERP","base_asset_name":"BTC","quote_asset_name":"USDC","base_increment":0.000001,"quote_increment":"0.1","min_quantity":0.0001,"open_interest":100.25,"base_asset_multiplier":1,"quote":{"index_price":20001.45,"mark_price":"20000.6300","timestamp":"2023-11-07T05:31:56Z"}},{"instrument_id":2,"symbol":"BTC-USDC","type":"SPOT"}]`))
	}))
	defer server.Close()
	client, err := NewClient(&ClientOption{BaseURL: server.URL, HTTPClient: server.Client()})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	instruments, err := client.MarketData().ListPerpetualInstruments(context.Background())
	if err != nil {
		t.Fatalf("ListPerpetualInstruments() error = %v", err)
	}
	if len(instruments) != 1 {
		t.Fatalf("len = %d, want 1", len(instruments))
	}
	if instruments[0].Symbol != "BTC-PERP" || instruments[0].BaseAssetName != "BTC" || instruments[0].QuoteAssetName != "USDC" || instruments[0].BaseIncrement != "0.000001" || instruments[0].QuoteIncrement != "0.1" || instruments[0].OpenInterest != "100.25" || instruments[0].BaseAssetMultiplier != "1" || instruments[0].Quote == nil || instruments[0].Quote.IndexPrice != "20001.45" || instruments[0].Quote.MarkPrice != "20000.6300" || instruments[0].Quote.Timestamp != "2023-11-07T05:31:56Z" {
		t.Fatalf("instrument = %#v", instruments[0])
	}
}

// TestGetHistoricalFundingRatesBuildsPaginationAndPreservesDecimals verifies the production envelope and complete funding payload.
//
// Version:
//   - 2026-08-27: Used the production pagination and results envelope.
//   - 2026-08-26: Added.
func TestGetHistoricalFundingRatesBuildsPaginationAndPreservesDecimals(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/instruments/BTC-PERP/funding" {
			t.Errorf("path = %q", r.URL.Path)
		}
		if r.URL.Query().Get("result_limit") != "100" || r.URL.Query().Get("result_offset") != "50" {
			t.Errorf("query = %v", r.URL.Query())
		}
		_, _ = w.Write([]byte(`{"pagination":{"result_limit":100,"result_offset":50},"results":[{"instrument_id":"14thr7ft-1-0","funding_rate":0.0001543,"mark_price":"20000.6300","event_time":"2023-03-16T23:59:53.000Z"}]}`))
	}))
	defer server.Close()
	client, err := NewClient(&ClientOption{BaseURL: server.URL, HTTPClient: server.Client()})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	rates, err := client.MarketData().GetHistoricalFundingRates(context.Background(), FundingHistoryParams{
		Instrument:   "BTC-PERP",
		ResultLimit:  100,
		ResultOffset: 50,
	})
	if err != nil {
		t.Fatalf("GetHistoricalFundingRates() error = %v", err)
	}
	if len(rates) != 1 || string(rates[0].InstrumentID) != `"14thr7ft-1-0"` || rates[0].FundingRate != "0.0001543" || rates[0].MarkPrice != "20000.6300" || rates[0].EventTime != "2023-03-16T23:59:53.000Z" {
		t.Fatalf("rates = %#v", rates)
	}
}

// TestGetHistoricalFundingRatesAcceptsLegacyArrayResponse verifies array response compatibility.
//
// Version:
//   - 2026-08-27: Added.
func TestGetHistoricalFundingRatesAcceptsLegacyArrayResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`[{"instrument_id":"14thr7ft-1-0","funding_rate":"0.0001","mark_price":"20000.63","event_time":"2023-03-16T23:59:53Z"}]`))
	}))
	defer server.Close()
	client, err := NewClient(&ClientOption{BaseURL: server.URL, HTTPClient: server.Client()})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	rates, err := client.MarketData().GetHistoricalFundingRates(context.Background(), FundingHistoryParams{Instrument: "BTC-PERP"})
	if err != nil {
		t.Fatalf("GetHistoricalFundingRates() error = %v", err)
	}
	if len(rates) != 1 || rates[0].EventTime != "2023-03-16T23:59:53Z" {
		t.Fatalf("rates = %#v", rates)
	}
}

// TestGetHistoricalFundingRatesAcceptsDocumentedObjectResponse verifies compatibility with the documented example shape.
//
// Version:
//   - 2026-08-26: Added.
func TestGetHistoricalFundingRatesAcceptsDocumentedObjectResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"instrument_id":7149252043835013,"funding_rate":"0.0001","mark_price":20000.63,"event_time":"2023-03-16T23:59:53Z"}`))
	}))
	defer server.Close()
	client, err := NewClient(&ClientOption{BaseURL: server.URL, HTTPClient: server.Client()})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	rates, err := client.MarketData().GetHistoricalFundingRates(context.Background(), FundingHistoryParams{Instrument: "7149252043835013"})
	if err != nil {
		t.Fatalf("GetHistoricalFundingRates() error = %v", err)
	}
	if len(rates) != 1 || string(rates[0].InstrumentID) != "7149252043835013" || rates[0].FundingRate != "0.0001" || rates[0].MarkPrice != "20000.63" {
		t.Fatalf("rates = %#v", rates)
	}
}

// TestGetHistoricalFundingRatesValidatesParameters verifies funding-history parameter bounds.
//
// Version:
//   - 2026-08-26: Added.
func TestGetHistoricalFundingRatesValidatesParameters(t *testing.T) {
	client, err := NewClient(&ClientOption{BaseURL: "https://example.com", HTTPClient: http.DefaultClient})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	tests := []FundingHistoryParams{
		{},
		{Instrument: "BTC-PERP", ResultLimit: 101},
		{Instrument: "BTC-PERP", ResultOffset: -1},
	}
	for _, params := range tests {
		if _, err := client.MarketData().GetHistoricalFundingRates(context.Background(), params); err == nil {
			t.Errorf("GetHistoricalFundingRates(%+v) error = nil", params)
		}
	}
}
