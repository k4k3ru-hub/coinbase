package rest

import "testing"

func TestCompositionRoot(t *testing.T) {
	t.Parallel()
	client, err := NewClient(nil)
	if err != nil {
		t.Fatal(err)
	}
	if client.MarketData() == nil {
		t.Fatal("market data client was not composed")
	}
}
