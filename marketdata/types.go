package marketdata

import "encoding/json"

// Decimal preserves an Exchange decimal exactly as text.
type Decimal = string

// Time is the server time response.
type Time struct {
	ISO   string          `json:"iso"`
	Epoch json.RawMessage `json:"epoch"`
}

// Currency is public currency metadata. Network and unknown evolving objects remain raw JSON.
type Currency struct {
	ID                string          `json:"id"`
	Name              string          `json:"name"`
	MinSize           Decimal         `json:"min_size"`
	Status            string          `json:"status"`
	Message           string          `json:"message"`
	MaxPrecision      Decimal         `json:"max_precision"`
	ConvertibleTo     []string        `json:"convertible_to"`
	Details           json.RawMessage `json:"details"`
	DefaultNetwork    string          `json:"default_network"`
	SupportedNetworks []Network       `json:"supported_networks"`
}

// Network is public currency network metadata.
type Network struct {
	ID                    string          `json:"id"`
	Name                  string          `json:"name"`
	Status                string          `json:"status"`
	ContractAddress       string          `json:"contract_address"`
	MinWithdrawal         Decimal         `json:"min_withdrawal_amount"`
	MaxWithdrawal         Decimal         `json:"max_withdrawal_amount"`
	NetworkConfirmations  json.RawMessage `json:"network_confirmations"`
	ProcessingTimeSeconds json.RawMessage `json:"processing_time_seconds"`
}

// Product is public Exchange product metadata. String enums deliberately accept unknown values.
type Product struct {
	ID                     string  `json:"id"`
	BaseCurrency           string  `json:"base_currency"`
	QuoteCurrency          string  `json:"quote_currency"`
	BaseIncrement          Decimal `json:"base_increment"`
	QuoteIncrement         Decimal `json:"quote_increment"`
	DisplayName            string  `json:"display_name"`
	MinMarketFunds         Decimal `json:"min_market_funds"`
	MaxMarketFunds         Decimal `json:"max_market_funds"`
	BaseMinSize            Decimal `json:"base_min_size"`
	BaseMaxSize            Decimal `json:"base_max_size"`
	Status                 string  `json:"status"`
	StatusMessage          string  `json:"status_message"`
	ProductType            string  `json:"product_type"`
	CancelOnly             bool    `json:"cancel_only"`
	LimitOnly              bool    `json:"limit_only"`
	PostOnly               bool    `json:"post_only"`
	TradingDisabled        bool    `json:"trading_disabled"`
	AuctionMode            bool    `json:"auction_mode"`
	MarginEnabled          bool    `json:"margin_enabled"`
	FXStablecoin           bool    `json:"fx_stablecoin"`
	MaxSlippagePercentage  Decimal `json:"max_slippage_percentage"`
	HighBidLimitPercentage Decimal `json:"high_bid_limit_percentage"`
}

// Ticker is the latest product trade and quote snapshot.
type Ticker struct {
	TradeID json.RawMessage `json:"trade_id"`
	Price   Decimal         `json:"price"`
	Size    Decimal         `json:"size"`
	Bid     Decimal         `json:"bid"`
	Ask     Decimal         `json:"ask"`
	Volume  Decimal         `json:"volume"`
	Time    string          `json:"time"`
}

// Trade is one public trade. Side is the maker order side as defined by Exchange.
type Trade struct {
	TradeID json.RawMessage `json:"trade_id"`
	Price   Decimal         `json:"price"`
	Size    Decimal         `json:"size"`
	Time    string          `json:"time"`
	Side    string          `json:"side"`
}

// Stats is Exchange product rolling statistics.
type Stats struct {
	Open        Decimal `json:"open"`
	High        Decimal `json:"high"`
	Low         Decimal `json:"low"`
	Last        Decimal `json:"last"`
	Volume      Decimal `json:"volume"`
	Volume30Day Decimal `json:"volume_30day"`
}

// VolumeSummary is a product's 24-hour and 30-day volume by market type.
type VolumeSummary struct {
	ID                     string   `json:"id"`
	BaseCurrency           string   `json:"base_currency"`
	QuoteCurrency          string   `json:"quote_currency"`
	DisplayName            string   `json:"display_name"`
	MarketTypes            []string `json:"market_types"`
	SpotVolume24Hour       Decimal  `json:"spot_volume_24hour"`
	SpotVolume30Day        Decimal  `json:"spot_volume_30day"`
	RFQVolume24Hour        Decimal  `json:"rfq_volume_24hour"`
	RFQVolume30Day         Decimal  `json:"rfq_volume_30day"`
	ConversionVolume24Hour Decimal  `json:"conversion_volume_24hour"`
	ConversionVolume30Day  Decimal  `json:"conversion_volume_30day"`
}

// Candle is a six-element historic rate bucket.
type Candle struct {
	Time                           int64
	Low, High, Open, Close, Volume Decimal
}

// Pagination contains Exchange cursor headers.
type Pagination struct {
	Before string
	After  string
}

// TradesPage contains trades and cursor metadata.
type TradesPage struct {
	Trades     []Trade
	Pagination Pagination
}

// BookLevel identifies a REST book aggregation level.
type BookLevel int

const (
	BookLevel1 BookLevel = 1
	BookLevel2 BookLevel = 2
	BookLevel3 BookLevel = 3
)

// BookEntry is implemented by level-specific book entries.
type BookEntry interface{ bookEntry() }

// AggregatedBookEntry represents L1/L2 [price,size,order_count].
type AggregatedBookEntry struct {
	Price      Decimal
	Size       Decimal
	OrderCount uint64
}

func (AggregatedBookEntry) bookEntry() {}

// OrderBookEntry represents L3 [price,size,order_id].
type OrderBookEntry struct {
	Price   Decimal
	Size    Decimal
	OrderID string
}

func (OrderBookEntry) bookEntry() {}

// ProductBook preserves level-specific entries without conflating L2 and L3.
type ProductBook struct {
	Sequence uint64
	Level    BookLevel
	Bids     []BookEntry
	Asks     []BookEntry
}
