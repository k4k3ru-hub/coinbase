package websocket

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// DecodeEvent decodes one documented Exchange object message and preserves its raw bytes.
//
// Version:
//   - 2026-08-19: Added.
func DecodeEvent(data []byte) (Event, error) {
	d := json.NewDecoder(bytes.NewReader(data))
	d.UseNumber()
	var head struct {
		Type string `json:"type"`
	}
	if err := d.Decode(&head); err != nil {
		return Event{}, fmt.Errorf("failed to decode coinbase exchange websocket event: %w", err)
	}
	if head.Type == "" {
		return Event{}, fmt.Errorf("failed to decode coinbase exchange websocket event: type=empty")
	}
	var value any
	switch head.Type {
	case "subscriptions":
		value = &Subscriptions{}
	case "error":
		value = &ErrorMessage{}
	case "heartbeat":
		value = &Heartbeat{}
	case "status":
		value = &Status{}
	case "ticker":
		value = &Ticker{}
	case "snapshot":
		value = &Level2Snapshot{}
	case "l2update":
		value = &Level2Update{}
	case "match", "last_match":
		value = &Match{}
	case "received", "open", "done", "change", "activate":
		value = &FullEvent{}
	default:
		return Event{Type: head.Type, Raw: append(json.RawMessage(nil), data...)}, nil
	}
	if err := json.Unmarshal(data, value); err != nil {
		return Event{}, fmt.Errorf("failed to decode coinbase exchange websocket event: %w: type=%q", err, head.Type)
	}
	return Event{Type: head.Type, Value: value, Raw: append(json.RawMessage(nil), data...)}, nil
}
