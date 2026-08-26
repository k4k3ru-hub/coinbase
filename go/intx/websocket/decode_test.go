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

// TestDecodeEventRisk verifies the complete Open Interest risk payload.
//
// Version:
//   - 2026-08-26: Added.
func TestDecodeEventRisk(t *testing.T) {
	event, err := DecodeEvent([]byte(`{"sequence":7,"product_id":"BTC-PERP","time":"2023-05-10T14:58:47.000Z","limit_up":"30226.3","limit_down":"27347.7","index_price":"28787","mark_price":"28787.1","settlement_price":"28786.9","indicative_open_price":"28787.8","open_interest":"32.5","channel":"RISK","type":"SNAPSHOT"}`))
	if err != nil {
		t.Fatalf("DecodeEvent() error = %v", err)
	}
	risk, ok := event.Value.(*Risk)
	if !ok {
		t.Fatalf("value type = %T", event.Value)
	}
	if risk.ProductID != "BTC-PERP" || risk.Time != "2023-05-10T14:58:47.000Z" || string(risk.Sequence) != "7" || risk.LimitUp != "30226.3" || risk.LimitDown != "27347.7" || risk.IndexPrice != "28787" || risk.MarkPrice != "28787.1" || risk.SettlementPrice != "28786.9" || risk.IndicativeOpenPrice != "28787.8" || risk.OpenInterest != "32.5" || risk.Type != "SNAPSHOT" {
		t.Fatalf("risk = %#v", risk)
	}
}
