package websocket

import (
	"encoding/json"
	"fmt"
)

// DecodeEvent decodes a supported INTX WebSocket event.
//
// Version:
//   - 2026-08-24: Added.
func DecodeEvent(data []byte) (Event, error) {
	if len(data) == 0 {
		return Event{}, fmt.Errorf("failed to decode coinbase intx websocket event: payload=empty")
	}
	var envelope struct {
		Channel string `json:"channel"`
		Type    string `json:"type"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		return Event{}, fmt.Errorf("failed to decode coinbase intx websocket event: %w", err)
	}
	var value any
	switch envelope.Channel {
	case ChannelLevel1:
		value = &Level1{}
	case ChannelLevel2:
		value = &Level2{}
	case ChannelMatch:
		value = &Match{}
	case ChannelFunding:
		value = &Funding{}
	case ChannelRisk:
		value = &Risk{}
	case ChannelInstruments:
		value = &Instrument{}
	case "SUBSCRIPTIONS":
		value = &Subscriptions{}
	case "ERROR":
		value = &ErrorMessage{}
	default:
		return Event{Channel: envelope.Channel, Type: envelope.Type, Raw: append(json.RawMessage(nil), data...)}, nil
	}
	if err := json.Unmarshal(data, value); err != nil {
		return Event{}, fmt.Errorf("failed to decode coinbase intx websocket event: %w: channel=%q type=%q", err, envelope.Channel, envelope.Type)
	}
	return Event{Channel: envelope.Channel, Type: envelope.Type, Value: value, Raw: append(json.RawMessage(nil), data...)}, nil
}
