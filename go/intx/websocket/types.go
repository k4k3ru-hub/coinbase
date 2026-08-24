// Package websocket implements the Coinbase International Exchange market-data feed.
package websocket

import "encoding/json"

const (
	ProductionURL      = "wss://ws-md.international.coinbase.com"
	SandboxURL         = "wss://ws-md.n5e2.coinbase.com"
	ChannelInstruments = "INSTRUMENTS"
	ChannelMatch       = "MATCH"
	ChannelFunding     = "FUNDING"
	ChannelRisk        = "RISK"
	ChannelLevel1      = "LEVEL1"
	ChannelLevel2      = "LEVEL2"
)

// SubscriptionRequest is an INTX subscription payload.
type SubscriptionRequest struct {
	Type       string   `json:"type"`
	ProductIDs []string `json:"product_ids,omitempty"`
	Channels   []string `json:"channels"`
	Time       string   `json:"time,omitempty"`
	Key        string   `json:"key,omitempty"`
	Passphrase string   `json:"passphrase,omitempty"`
	Signature  string   `json:"signature,omitempty"`
}

// Event is one decoded INTX WebSocket message.
type Event struct {
	Channel string
	Type    string
	Value   any
	Raw     json.RawMessage
}

// Envelope contains fields shared by market-data messages.
type Envelope struct {
	Sequence  json.RawMessage `json:"sequence"`
	ProductID string          `json:"product_id"`
	Time      string          `json:"time"`
	Channel   string          `json:"channel"`
	Type      string          `json:"type"`
}

// Level1 is an INTX best-bid-offer snapshot or update.
type Level1 struct {
	Envelope
	BidPrice string `json:"bid_price"`
	BidQty   string `json:"bid_qty"`
	AskPrice string `json:"ask_price"`
	AskQty   string `json:"ask_qty"`
}

// Match is an INTX public trade update. TradeQty is reported in instrument quantity units.
type Match struct {
	Envelope
	MatchID       json.RawMessage `json:"match_id"`
	TradeQty      string          `json:"trade_qty"`
	AggressorSide string          `json:"aggressor_side"`
	TradePrice    string          `json:"trade_price"`
}

// Level2 is an INTX top-20 order-book snapshot or absolute-size update.
type Level2 struct {
	Envelope
	Bids    [][]string `json:"bids,omitempty"`
	Asks    [][]string `json:"asks,omitempty"`
	Changes [][]string `json:"changes,omitempty"`
}

// Funding is an INTX funding-rate snapshot or update.
type Funding struct {
	Envelope
	FundingRate string `json:"funding_rate"`
	IsFinal     bool   `json:"is_final"`
}

// Risk is an INTX perpetual risk snapshot or update.
type Risk struct {
	Envelope
	LimitUp         string `json:"limit_up"`
	LimitDown       string `json:"limit_down"`
	IndexPrice      string `json:"index_price"`
	MarkPrice       string `json:"mark_price"`
	SettlementPrice string `json:"settlement_price"`
	OpenInterest    string `json:"open_interest"`
}

// Instrument is an INTX WebSocket instrument snapshot or update.
type Instrument struct {
	Envelope
	InstrumentType      string          `json:"instrument_type"`
	InstrumentMode      string          `json:"instrument_mode"`
	BaseAssetName       string          `json:"base_asset_name"`
	QuoteAssetName      string          `json:"quote_asset_name"`
	BaseIncrement       string          `json:"base_increment"`
	QuoteIncrement      string          `json:"quote_increment"`
	MinQuantity         string          `json:"min_quantity"`
	BaseAssetMultiplier string          `json:"base_asset_multiplier"`
	FundingInterval     json.RawMessage `json:"funding_interval"`
	TradingState        string          `json:"trading_state"`
	UnderlyingType      string          `json:"underlying_type"`
}

// Subscriptions acknowledges the active subscription set.
type Subscriptions struct {
	Channel       string `json:"channel"`
	Type          string `json:"type"`
	Authenticated bool   `json:"authenticated"`
	Channels      []struct {
		Name       string   `json:"name"`
		ProductIDs []string `json:"product_ids"`
	} `json:"channels"`
}

// ErrorMessage is an INTX feed error.
type ErrorMessage struct {
	Channel string `json:"channel"`
	Type    string `json:"type"`
	Message string `json:"message"`
	Reason  string `json:"reason"`
}
