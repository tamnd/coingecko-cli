package coingecko

// Coin is one entry from the CoinGecko markets listing.
type Coin struct {
	Rank          int     `json:"rank"`
	ID            string  `json:"id"`
	Symbol        string  `json:"symbol"`
	Name          string  `json:"name"`
	PriceUSD      float64 `json:"price_usd"`
	Change24h     float64 `json:"change_24h"` // percentage
	MarketCapUSD  float64 `json:"market_cap_usd"`
	MarketCapRank int     `json:"market_cap_rank"`
	Volume24hUSD  float64 `json:"volume_24h_usd"`
	URL           string  `json:"url"` // https://www.coingecko.com/en/coins/{id}
}

// TrendingCoin is one entry from the CoinGecko trending search result.
type TrendingCoin struct {
	Rank          int    `json:"rank"`
	ID            string `json:"id"`
	Symbol        string `json:"symbol"`
	Name          string `json:"name"`
	MarketCapRank int    `json:"market_cap_rank"`
	URL           string `json:"url"`
}

// Price maps coin ID -> currency -> price value.
type Price map[string]map[string]float64

// CoinMarketData holds the market_data sub-object from /coins/{id}.
type CoinMarketData struct {
	CurrentPrice      map[string]float64 `json:"current_price"`
	MarketCap         map[string]float64 `json:"market_cap"`
	PriceChangePct24h float64            `json:"price_change_percentage_24h"`
}

// CoinDetail is the full coin object returned by /coins/{id}.
type CoinDetail struct {
	ID            string            `json:"id"`
	Symbol        string            `json:"symbol"`
	Name          string            `json:"name"`
	MarketCapRank int               `json:"market_cap_rank"`
	MarketData    CoinMarketData    `json:"market_data"`
	Description   map[string]string `json:"description"`
	Categories    []string          `json:"categories"`
}

// SearchResult is one coin entry from the /search response.
type SearchResult struct {
	ID            string `json:"id"`
	Symbol        string `json:"symbol"`
	Name          string `json:"name"`
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
}

type trendingResponse struct {
	Coins []trendingCoinWrapper `json:"coins"`
}

type trendingCoinWrapper struct {
	Item trendingItem `json:"item"`
}

type trendingItem struct {
	ID            string `json:"id"`
	Symbol        string `json:"symbol"`
	Name          string `json:"name"`
	MarketCapRank int    `json:"market_cap_rank"`
}

type searchResponse struct {
	Coins []SearchResult `json:"coins"`
}
