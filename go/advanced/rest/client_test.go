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

func TestListPerpetualProductsFiltersResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("product_type"); got != "FUTURE" {
			t.Errorf("product_type = %q", got)
		}
		if got := r.URL.Query().Get("contract_expiry_type"); got != "PERPETUAL" {
			t.Errorf("contract_expiry_type = %q", got)
		}
		_, _ = w.Write([]byte(`{"products":[{"product_id":"BTC-PERP-INTX","product_type":"FUTURE","product_venue":"INTX","base_increment":"0.0001","price_increment":"0.1","future_product_details":{"contract_code":"BTC","contract_size":"1","contract_root_unit":"BTC","contract_expiry_type":"PERPETUAL","funding_interval":"36000000000","index_price":"20001.45","perpetual_details":{"open_interest":"100.25","funding_rate":"0.0001","funding_time":"2023-11-07T08:00:00Z","max_leverage":"10","base_asset_uuid":"592a8039-db3e-45ed-b752-ffd1983eead2","underlying_type":"SPOT"}}},{"product_id":"BTC-USD","product_type":"SPOT"}]}`))
	}))
	defer server.Close()
	client, err := NewClient(&ClientOption{BaseURL: server.URL, HTTPClient: server.Client()})
	if err != nil {
		t.Fatal(err)
	}
	products, err := client.MarketData().ListPerpetualProducts(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(products) != 1 || products[0].ProductID != "BTC-PERP-INTX" || products[0].FutureProductDetails.ContractRootUnit != "BTC" {
		t.Fatalf("products = %#v", products)
	}
	details := products[0].FutureProductDetails
	if details.ContractSize != "1" || details.IndexPrice != "20001.45" || details.PerpetualDetails == nil || details.PerpetualDetails.OpenInterest != "100.25" || details.PerpetualDetails.FundingRate != "0.0001" || details.PerpetualDetails.FundingTime != "2023-11-07T08:00:00Z" {
		t.Fatalf("future product details = %#v", details)
	}
}

func TestGetPerpetualProductRejectsSpotID(t *testing.T) {
	client, err := NewClient(&ClientOption{BaseURL: "https://example.com", HTTPClient: http.DefaultClient})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.MarketData().GetPerpetualProduct(context.Background(), "BTC-USD"); err == nil {
		t.Fatal("GetPerpetualProduct() error = nil")
	}
}
