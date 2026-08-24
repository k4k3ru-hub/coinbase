// Package websocket implements Coinbase Advanced Trade public market-data WebSocket.
package websocket

import "encoding/json"

const (
	ProductionURL       = "wss://advanced-trade-ws.coinbase.com"
	ChannelHeartbeats   = "heartbeats"
	ChannelLevel2       = "level2"
	ChannelLevel2Data   = "l2_data"
	ChannelMarketTrades = "market_trades"
	ChannelTicker       = "ticker"
	ChannelStatus       = "status"
)

// SubscriptionRequest is an anonymous public channel subscription.
type SubscriptionRequest struct {
	Type       string   `json:"type"`
	Channel    string   `json:"channel"`
	ProductIDs []string `json:"product_ids,omitempty"`
}

// Event is one decoded Advanced Trade WebSocket envelope.
type Event struct {
	Channel     string
	SequenceNum uint64
	Value       any
	Raw         json.RawMessage
}

// Envelope contains fields shared by public market-data messages.
type Envelope struct {
	Channel     string `json:"channel"`
	ClientID    string `json:"client_id"`
	Timestamp   string `json:"timestamp"`
	SequenceNum uint64 `json:"sequence_num"`
}

// Level2Envelope contains one or more book events.
type Level2Envelope struct {
	Envelope
	Events []Level2Event `json:"events"`
}

// Level2Event is one product snapshot or update batch.
type Level2Event struct {
	Type      string         `json:"type"`
	ProductID string         `json:"product_id"`
	Updates   []Level2Update `json:"updates"`
}

// Level2Update sets the absolute quantity at a price level.
type Level2Update struct {
	Side        string `json:"side"`
	EventTime   string `json:"event_time"`
	PriceLevel  string `json:"price_level"`
	NewQuantity string `json:"new_quantity"`
}

// MarketTradesEnvelope contains public trade batches.
type MarketTradesEnvelope struct {
	Envelope
	Events []MarketTradesEvent `json:"events"`
}

// MarketTradesEvent is a snapshot or update containing trades.
type MarketTradesEvent struct {
	Type   string  `json:"type"`
	Trades []Trade `json:"trades"`
}

// Trade is one public trade. Side is Coinbase's maker side.
type Trade struct {
	TradeID   string `json:"trade_id"`
	ProductID string `json:"product_id"`
	Price     string `json:"price"`
	Size      string `json:"size"`
	Side      string `json:"side"`
	Time      string `json:"time"`
}

// HeartbeatsEnvelope contains feed liveness counters.
type HeartbeatsEnvelope struct {
	Envelope
	Events []Heartbeat `json:"events"`
}

// Heartbeat is one public feed heartbeat.
type Heartbeat struct {
	CurrentTime      string          `json:"current_time"`
	HeartbeatCounter json.RawMessage `json:"heartbeat_counter"`
}

// TickerEnvelope contains ticker batches.
type TickerEnvelope struct {
	Envelope
	Events []TickerEvent `json:"events"`
}

// TickerEvent is a snapshot or update containing tickers.
type TickerEvent struct {
	Type    string   `json:"type"`
	Tickers []Ticker `json:"tickers"`
}

// Ticker contains last price and best bid/offer fields.
type Ticker struct {
	ProductID       string `json:"product_id"`
	Price           string `json:"price"`
	BestBid         string `json:"best_bid"`
	BestBidQuantity string `json:"best_bid_quantity"`
	BestAsk         string `json:"best_ask"`
	BestAskQuantity string `json:"best_ask_quantity"`
	Volume24H       string `json:"volume_24_h"`
}

// UnknownEnvelope preserves newly added public channels.
type UnknownEnvelope struct {
	Envelope
	Events json.RawMessage `json:"events"`
}
