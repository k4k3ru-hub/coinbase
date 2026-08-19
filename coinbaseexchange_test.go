package coinbaseexchange

import "testing"

func TestRESTCompositionRoot(t *testing.T) {
	t.Parallel()
	client, err := NewRESTClient(nil)
	if err != nil {
		t.Fatal(err)
	}
	if client.MarketData() == nil {
		t.Fatal("market data client was not composed")
	}
}
