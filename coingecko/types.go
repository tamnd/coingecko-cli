package coingecko

// Price is a coin's price in one or more currencies.
type Price struct {
	CoinID string             `kit:"id" json:"coin_id"`
	Prices map[string]float64 `json:"prices"`
}

// CoinMarket is one entry from the /coins/markets listing.
type CoinMarket struct {
	ID                string  `kit:"id" json:"id"`
	Symbol            string  `json:"symbol"`
	Name              string  `json:"name"`
	CurrentPrice      float64 `json:"current_price"`
	MarketCap         float64 `json:"market_cap"`
	MarketCapRank     int     `json:"market_cap_rank"`
	PriceChange24h    float64 `json:"price_change_24h_pct"`
	TotalVolume       float64 `json:"total_volume"`
	CirculatingSupply float64 `json:"circulating_supply"`
	ATH               float64 `json:"ath"`
}

// CoinDetail is the full coin object returned by /coins/{id}.
type CoinDetail struct {
	ID           string  `kit:"id" json:"id"`
	Symbol       string  `json:"symbol"`
	Name         string  `json:"name"`
	Description  string  `json:"description"`
	GenesisDate  string  `json:"genesis_date"`
	CurrentUSD   float64 `json:"current_price_usd"`
	MarketCapUSD float64 `json:"market_cap_usd"`
	ATH_USD      float64 `json:"ath_usd"`
	Change24h    float64 `json:"price_change_24h_pct"`
}

// TrendingCoin is one entry from the /search/trending response.
type TrendingCoin struct {
	ID            string  `kit:"id" json:"id"`
	Symbol        string  `json:"symbol"`
	Name          string  `json:"name"`
	MarketCapRank int     `json:"market_cap_rank"`
	PriceBTC      float64 `json:"price_btc"`
}

// SearchResult is one coin entry from the /search response.
type SearchResult struct {
	ID            string `kit:"id" json:"id"`
	Name          string `json:"name"`
	Symbol        string `json:"symbol"`
	MarketCapRank int    `json:"market_cap_rank"`
}

// --- private decode types ---

type apiCoin struct {
	ID                    string  `json:"id"`
	Symbol                string  `json:"symbol"`
	Name                  string  `json:"name"`
	CurrentPrice          float64 `json:"current_price"`
	PriceChangePercentage float64 `json:"price_change_percentage_24h"`
	MarketCap             float64 `json:"market_cap"`
	MarketCapRank         int     `json:"market_cap_rank"`
	TotalVolume           float64 `json:"total_volume"`
	CirculatingSupply     float64 `json:"circulating_supply"`
	ATH                   float64 `json:"ath"`
}

type apiCoinDetail struct {
	ID          string            `json:"id"`
	Symbol      string            `json:"symbol"`
	Name        string            `json:"name"`
	GenesisDate string            `json:"genesis_date"`
	Description map[string]string `json:"description"`
	MarketData  struct {
		CurrentPrice      map[string]float64 `json:"current_price"`
		MarketCap         map[string]float64 `json:"market_cap"`
		ATH               map[string]float64 `json:"ath"`
		PriceChangePct24h float64            `json:"price_change_percentage_24h"`
	} `json:"market_data"`
}

type trendingResponse struct {
	Coins []trendingCoinWrapper `json:"coins"`
}

type trendingCoinWrapper struct {
	Item trendingItem `json:"item"`
}

type trendingItem struct {
	ID            string  `json:"id"`
	Symbol        string  `json:"symbol"`
	Name          string  `json:"name"`
	MarketCapRank int     `json:"market_cap_rank"`
	PriceBTC      float64 `json:"price_btc"`
}

type searchResponse struct {
	Coins []SearchResult `json:"coins"`
}
