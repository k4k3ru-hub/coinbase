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
		_, _ = w.Write([]byte(`[{"instrument_id":1,"symbol":"BTC-PERP","type":"PERP","base_increment":0.000001,"quote_increment":"0.1","min_quantity":0.0001,"base_asset_multiplier":1},{"instrument_id":2,"symbol":"BTC-USDC","type":"SPOT"}]`))
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
	if instruments[0].Symbol != "BTC-PERP" || instruments[0].BaseIncrement != "0.000001" || instruments[0].QuoteIncrement != "0.1" {
		t.Fatalf("instrument = %#v", instruments[0])
	}
}
