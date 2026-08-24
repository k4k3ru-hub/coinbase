package websocket

import "testing"

func TestBookManagerSnapshotAndUpdate(t *testing.T) {
	manager := NewBookManager()
	snapshot := Event{Value: &Level2Envelope{Events: []Level2Event{{
		Type: "snapshot", ProductID: "BTC-PERP-INTX",
		Updates: []Level2Update{{Side: "bid", PriceLevel: "100", NewQuantity: "2"}, {Side: "offer", PriceLevel: "101", NewQuantity: "3"}},
	}}}}
	if err := manager.Apply(snapshot); err != nil {
		t.Fatal(err)
	}
	bbo, ok := manager.BestBidOffer("BTC-PERP-INTX")
	if !ok || bbo.BidPrice != "100" || bbo.AskPrice != "101" {
		t.Fatalf("bbo = %#v, %v", bbo, ok)
	}
	update := Event{Value: &Level2Envelope{Events: []Level2Event{{
		Type: "update", ProductID: "BTC-PERP-INTX",
		Updates: []Level2Update{{Side: "bid", PriceLevel: "100.5", NewQuantity: "4"}, {Side: "offer", PriceLevel: "101", NewQuantity: "0"}, {Side: "offer", PriceLevel: "102", NewQuantity: "5"}},
	}}}}
	if err := manager.Apply(update); err != nil {
		t.Fatal(err)
	}
	bbo, ok = manager.BestBidOffer("BTC-PERP-INTX")
	if !ok || bbo.BidPrice != "100.5" || bbo.BidSize != "4" || bbo.AskPrice != "102" {
		t.Fatalf("bbo = %#v, %v", bbo, ok)
	}
}

func TestBookManagerRejectsUpdateBeforeSnapshot(t *testing.T) {
	manager := NewBookManager()
	err := manager.Apply(Event{Value: &Level2Envelope{Events: []Level2Event{{Type: "update", ProductID: "BTC-PERP-INTX"}}}})
	if err == nil {
		t.Fatal("Apply() error = nil")
	}
}
