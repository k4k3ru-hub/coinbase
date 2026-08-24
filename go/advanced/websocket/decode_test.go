package websocket

import "testing"

func TestDecodeLevel2(t *testing.T) {
	event, err := DecodeEvent([]byte(`{"channel":"l2_data","timestamp":"2026-08-24T00:00:00Z","sequence_num":7,"events":[{"type":"snapshot","product_id":"BTC-PERP-INTX","updates":[{"side":"bid","price_level":"79000.1","new_quantity":"0.2","event_time":"2026-08-24T00:00:00Z"},{"side":"offer","price_level":"79000.2","new_quantity":"0.3","event_time":"2026-08-24T00:00:00Z"}]}]}`))
	if err != nil {
		t.Fatal(err)
	}
	envelope, ok := event.Value.(*Level2Envelope)
	if !ok {
		t.Fatalf("value type = %T", event.Value)
	}
	if event.SequenceNum != 7 || envelope.Events[0].ProductID != "BTC-PERP-INTX" {
		t.Fatalf("event = %#v", event)
	}
}

func TestDecodeMarketTrades(t *testing.T) {
	event, err := DecodeEvent([]byte(`{"channel":"market_trades","sequence_num":2,"events":[{"type":"update","trades":[{"trade_id":"1","product_id":"BTC-PERP-INTX","price":"79000","size":"0.01","side":"BUY","time":"2026-08-24T00:00:00Z"}]}]}`))
	if err != nil {
		t.Fatal(err)
	}
	envelope, ok := event.Value.(*MarketTradesEnvelope)
	if !ok {
		t.Fatalf("value type = %T", event.Value)
	}
	if envelope.Events[0].Trades[0].Size != "0.01" {
		t.Fatalf("event = %#v", event)
	}
}

func TestDecodeHeartbeatAcceptsNumericCounter(t *testing.T) {
	event, err := DecodeEvent([]byte(`{"channel":"heartbeats","sequence_num":3,"events":[{"current_time":"2026-08-24T00:00:00Z","heartbeat_counter":42}]}`))
	if err != nil {
		t.Fatal(err)
	}
	envelope, ok := event.Value.(*HeartbeatsEnvelope)
	if !ok || string(envelope.Events[0].HeartbeatCounter) != "42" {
		t.Fatalf("event = %#v", event)
	}
}
