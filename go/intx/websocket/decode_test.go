package websocket

import "testing"

func TestDecodeEventLevel1(t *testing.T) {
	event, err := DecodeEvent([]byte(`{"sequence":1,"product_id":"BTC-PERP","bid_price":"100","bid_qty":"2","ask_price":"101","ask_qty":"3","channel":"LEVEL1","type":"UPDATE"}`))
	if err != nil {
		t.Fatalf("DecodeEvent() error = %v", err)
	}
	level, ok := event.Value.(*Level1)
	if !ok {
		t.Fatalf("value type = %T", event.Value)
	}
	if level.ProductID != "BTC-PERP" || level.BidQty != "2" || level.AskQty != "3" {
		t.Fatalf("level = %#v", level)
	}
}

func TestDecodeEventMatch(t *testing.T) {
	event, err := DecodeEvent([]byte(`{"sequence":2,"product_id":"ETH-PERP","match_id":"9","trade_qty":"0.25","aggressor_side":"BUY","trade_price":"2500","channel":"MATCH","type":"UPDATE"}`))
	if err != nil {
		t.Fatalf("DecodeEvent() error = %v", err)
	}
	match, ok := event.Value.(*Match)
	if !ok {
		t.Fatalf("value type = %T", event.Value)
	}
	if match.TradeQty != "0.25" || match.AggressorSide != "BUY" {
		t.Fatalf("match = %#v", match)
	}
}
