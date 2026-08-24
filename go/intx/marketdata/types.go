// Package marketdata defines Coinbase International Exchange market-data values.
package marketdata

import (
	"bytes"
	"encoding/json"
	"fmt"
)

const (
	// InstrumentTypePerpetual identifies an INTX perpetual future.
	InstrumentTypePerpetual = "PERP"
	// InstrumentTypeSpot identifies an INTX spot instrument.
	InstrumentTypeSpot = "SPOT"
)

// Instrument is an INTX instrument. Decimal quantities remain strings.
type Instrument struct {
	InstrumentID          json.RawMessage  `json:"instrument_id"`
	InstrumentUUID        string           `json:"instrument_uuid"`
	Symbol                string           `json:"symbol"`
	Type                  string           `json:"type"`
	Mode                  string           `json:"mode"`
	BaseAssetName         string           `json:"base_asset_name"`
	QuoteAssetName        string           `json:"quote_asset_name"`
	BaseIncrement         Decimal          `json:"base_increment"`
	QuoteIncrement        Decimal          `json:"quote_increment"`
	MinQuantity           Decimal          `json:"min_quantity"`
	BaseAssetMultiplier   Decimal          `json:"base_asset_multiplier"`
	TradingState          string           `json:"trading_state"`
	ExecutionExchange     string           `json:"execution_exchange"`
	UnderlyingType        string           `json:"underlying_type"`
	FundingInterval       json.RawMessage  `json:"funding_interval"`
	PositionNotionalLimit Decimal          `json:"position_notional_limit"`
	OpenInterestLimit     Decimal          `json:"open_interest_notional_limit"`
	Quote                 *InstrumentQuote `json:"quote,omitempty"`
}

// InstrumentQuote is the latest quote embedded in an instrument response.
type InstrumentQuote struct {
	BestBidPrice Decimal `json:"best_bid_price"`
	BestBidSize  Decimal `json:"best_bid_size"`
	BestAskPrice Decimal `json:"best_ask_price"`
	BestAskSize  Decimal `json:"best_ask_size"`
	TradePrice   Decimal `json:"trade_price"`
	TradeQty     Decimal `json:"trade_qty"`
	IndexPrice   Decimal `json:"index_price"`
	MarkPrice    Decimal `json:"mark_price"`
	Timestamp    string  `json:"timestamp"`
}

// Decimal preserves a decimal value exactly as received, accepting JSON strings and numbers.
type Decimal string

// UnmarshalJSON decodes a quoted or unquoted decimal without float conversion.
//
// Version:
//   - 2026-08-24: Added.
func (d *Decimal) UnmarshalJSON(data []byte) error {
	if d == nil {
		return fmt.Errorf("failed to decode coinbase intx decimal: decimal=null")
	}
	if bytes.Equal(data, []byte("null")) {
		*d = ""
		return nil
	}
	if len(data) > 1 && data[0] == '"' {
		var value string
		if err := json.Unmarshal(data, &value); err != nil {
			return fmt.Errorf("failed to decode coinbase intx decimal: %w", err)
		}
		*d = Decimal(value)
		return nil
	}
	var number json.Number
	if err := json.Unmarshal(data, &number); err != nil {
		return fmt.Errorf("failed to decode coinbase intx decimal: %w", err)
	}
	*d = Decimal(number.String())
	return nil
}
