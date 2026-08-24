// Package marketdata defines Coinbase Advanced Trade public market-data values.
package marketdata

const (
	ProductTypeFuture       = "FUTURE"
	ContractExpiryPerpetual = "PERPETUAL"
)

// Product is an Advanced Trade public product.
type Product struct {
	ProductID            string                `json:"product_id"`
	Price                string                `json:"price"`
	Volume24H            string                `json:"volume_24h"`
	BaseIncrement        string                `json:"base_increment"`
	QuoteIncrement       string                `json:"quote_increment"`
	BaseMinSize          string                `json:"base_min_size"`
	QuoteMinSize         string                `json:"quote_min_size"`
	ProductType          string                `json:"product_type"`
	ProductVenue         string                `json:"product_venue"`
	BaseCurrencyID       string                `json:"base_currency_id"`
	QuoteCurrencyID      string                `json:"quote_currency_id"`
	BaseDisplaySymbol    string                `json:"base_display_symbol"`
	QuoteDisplaySymbol   string                `json:"quote_display_symbol"`
	PriceIncrement       string                `json:"price_increment"`
	DisplayName          string                `json:"display_name"`
	Status               string                `json:"status"`
	IsDisabled           bool                  `json:"is_disabled"`
	TradingDisabled      bool                  `json:"trading_disabled"`
	FutureProductDetails *FutureProductDetails `json:"future_product_details,omitempty"`
}

// FutureProductDetails contains derivative contract metadata.
type FutureProductDetails struct {
	ContractCode       string            `json:"contract_code"`
	ContractSize       string            `json:"contract_size"`
	ContractRootUnit   string            `json:"contract_root_unit"`
	ContractExpiryType string            `json:"contract_expiry_type"`
	FundingInterval    string            `json:"funding_interval"`
	IndexPrice         string            `json:"index_price"`
	PerpetualDetails   *PerpetualDetails `json:"perpetual_details,omitempty"`
}

// PerpetualDetails contains current perpetual-specific metadata.
type PerpetualDetails struct {
	OpenInterest   string `json:"open_interest"`
	FundingRate    string `json:"funding_rate"`
	FundingTime    string `json:"funding_time"`
	MaxLeverage    string `json:"max_leverage"`
	BaseAssetUUID  string `json:"base_asset_uuid"`
	UnderlyingType string `json:"underlying_type"`
}
