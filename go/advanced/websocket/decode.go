package websocket

import (
	"encoding/json"
	"fmt"
)

// DecodeEvent decodes one Advanced Trade public WebSocket envelope.
//
// Version:
//   - 2026-08-24: Added.
func DecodeEvent(data []byte) (Event, error) {
	if len(data) == 0 {
		return Event{}, fmt.Errorf("failed to decode coinbase advanced websocket event: payload=empty")
	}
	var head Envelope
	if err := json.Unmarshal(data, &head); err != nil {
		return Event{}, fmt.Errorf("failed to decode coinbase advanced websocket event: %w", err)
	}
	if head.Channel == "" {
		return Event{}, fmt.Errorf("failed to decode coinbase advanced websocket event: channel=empty")
	}
	var value any
	switch head.Channel {
	case ChannelLevel2, ChannelLevel2Data:
		value = &Level2Envelope{}
	case ChannelMarketTrades:
		value = &MarketTradesEnvelope{}
	case ChannelHeartbeats:
		value = &HeartbeatsEnvelope{}
	case ChannelTicker:
		value = &TickerEnvelope{}
	default:
		value = &UnknownEnvelope{}
	}
	if err := json.Unmarshal(data, value); err != nil {
		return Event{}, fmt.Errorf("failed to decode coinbase advanced websocket event: %w: channel=%q", err, head.Channel)
	}
	return Event{Channel: head.Channel, SequenceNum: head.SequenceNum, Value: value, Raw: append(json.RawMessage(nil), data...)}, nil
}
