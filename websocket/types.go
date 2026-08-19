package websocket

import "encoding/json"

const (
	ProductionURL      = "wss://ws-feed.exchange.coinbase.com"
	SandboxURL         = "wss://ws-feed-public.sandbox.exchange.coinbase.com"
	ChannelHeartbeat   = "heartbeat"
	ChannelStatus      = "status"
	ChannelTicker      = "ticker"
	ChannelTickerBatch = "ticker_batch"
	ChannelLevel2      = "level2"
	ChannelLevel2Batch = "level2_batch"
	ChannelMatches     = "matches"
	ChannelFull        = "full"
)

// Channel identifies an Exchange WebSocket subscription.
type Channel struct {
	Name       string   `json:"name"`
	ProductIDs []string `json:"product_ids,omitempty"`
}

// SubscriptionRequest is a subscribe or unsubscribe payload.
type SubscriptionRequest struct {
	Type       string    `json:"type"`
	ProductIDs []string  `json:"product_ids,omitempty"`
	Channels   []Channel `json:"channels"`
}

// Event is one decoded WebSocket message.
type Event struct {
	Type  string
	Value any
	Raw   json.RawMessage
}

// Subscriptions acknowledges the current subscription set.
type Subscriptions struct {
	Type     string    `json:"type"`
	Channels []Channel `json:"channels"`
}

// ErrorMessage is an Exchange feed error.
type ErrorMessage struct {
	Type    string `json:"type"`
	Message string `json:"message"`
	Reason  string `json:"reason,omitempty"`
}

// Heartbeat is a heartbeat event. IDs remain raw to avoid width assumptions.
type Heartbeat struct {
	Type        string          `json:"type"`
	Sequence    json.RawMessage `json:"sequence"`
	LastTradeID json.RawMessage `json:"last_trade_id"`
	ProductID   string          `json:"product_id"`
	Time        string          `json:"time"`
}

// Status is Exchange feed metadata, kept separately from REST schemas.
type Status struct {
	Type       string           `json:"type"`
	Products   []StatusProduct  `json:"products"`
	Currencies []StatusCurrency `json:"currencies"`
}

// StatusProduct is WebSocket product metadata.
type StatusProduct struct {
	ID             string `json:"id"`
	BaseCurrency   string `json:"base_currency"`
	QuoteCurrency  string `json:"quote_currency"`
	BaseIncrement  string `json:"base_increment"`
	QuoteIncrement string `json:"quote_increment"`
	DisplayName    string `json:"display_name"`
	Status         string `json:"status"`
	StatusMessage  string `json:"status_message"`
	MinMarketFunds string `json:"min_market_funds"`
	PostOnly       bool   `json:"post_only"`
	LimitOnly      bool   `json:"limit_only"`
	CancelOnly     bool   `json:"cancel_only"`
	AuctionMode    bool   `json:"auction_mode"`
}

// StatusCurrency is WebSocket currency metadata.
type StatusCurrency struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	MinSize       string   `json:"min_size"`
	Status        string   `json:"status"`
	Message       string   `json:"message"`
	MaxPrecision  string   `json:"max_precision"`
	ConvertibleTo []string `json:"convertible_to"`
}

// Ticker is a real-time ticker event; pointers preserve optional numeric fields.
type Ticker struct {
	Type        string          `json:"type"`
	Sequence    *json.Number    `json:"sequence,omitempty"`
	ProductID   string          `json:"product_id"`
	Price       *string         `json:"price,omitempty"`
	Open24H     *string         `json:"open_24h,omitempty"`
	Volume24H   *string         `json:"volume_24h,omitempty"`
	Low24H      *string         `json:"low_24h,omitempty"`
	High24H     *string         `json:"high_24h,omitempty"`
	Volume30D   *string         `json:"volume_30d,omitempty"`
	BestBid     *string         `json:"best_bid,omitempty"`
	BestBidSize *string         `json:"best_bid_size,omitempty"`
	BestAsk     *string         `json:"best_ask,omitempty"`
	BestAskSize *string         `json:"best_ask_size,omitempty"`
	Side        *string         `json:"side,omitempty"`
	TradeID     json.RawMessage `json:"trade_id,omitempty"`
	LastSize    *string         `json:"last_size,omitempty"`
	Time        *string         `json:"time,omitempty"`
}

// Match is a public trade event. Side remains the maker side.
type Match struct {
	Type         string          `json:"type"`
	Sequence     json.RawMessage `json:"sequence"`
	TradeID      json.RawMessage `json:"trade_id"`
	MakerOrderID string          `json:"maker_order_id"`
	TakerOrderID string          `json:"taker_order_id"`
	ProductID    string          `json:"product_id"`
	Side         string          `json:"side"`
	Size         string          `json:"size"`
	Price        string          `json:"price"`
	Time         string          `json:"time"`
}

// Level2Snapshot initializes an L2 book.
type Level2Snapshot struct {
	Type      string     `json:"type"`
	ProductID string     `json:"product_id"`
	Bids      [][]string `json:"bids"`
	Asks      [][]string `json:"asks"`
}

// Level2Update changes absolute quantities at price levels.
type Level2Update struct {
	Type      string     `json:"type"`
	ProductID string     `json:"product_id"`
	Time      string     `json:"time"`
	Changes   [][]string `json:"changes"`
}

// FullEvent preserves documented full-channel lifecycle fields without building L3 state.
type FullEvent struct {
	Type          string          `json:"type"`
	ProductID     string          `json:"product_id"`
	Sequence      json.RawMessage `json:"sequence"`
	OrderID       string          `json:"order_id"`
	MakerOrderID  string          `json:"maker_order_id"`
	TakerOrderID  string          `json:"taker_order_id"`
	Side          string          `json:"side"`
	Price         string          `json:"price"`
	Size          string          `json:"size"`
	RemainingSize string          `json:"remaining_size"`
	Reason        string          `json:"reason"`
	Time          string          `json:"time"`
	OrderType     string          `json:"order_type"`
	Funds         string          `json:"funds"`
	NewSize       string          `json:"new_size"`
	OldSize       string          `json:"old_size"`
	NewFunds      string          `json:"new_funds"`
	OldFunds      string          `json:"old_funds"`
}
