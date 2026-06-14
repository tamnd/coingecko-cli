package coingecko

// Price is a coin's current price in a single currency.
type Price struct {
	ID       string  `kit:"id" json:"id"`
	Currency string  `json:"currency"`
	Price    float64 `json:"price"`
}

// Market is one entry from the /coins/markets listing.
type Market struct {
	ID        string `kit:"id" json:"id"`
	Symbol    string `json:"symbol"`
	Name      string `json:"name"`
	Rank      int    `json:"rank"`
	Price     string `json:"price"`
	MarketCap string `json:"market_cap"`
	Volume24h string `json:"volume_24h"`
	High24h   string `json:"high_24h"`
	Low24h    string `json:"low_24h"`
	Change24h string `json:"change_24h_pct"`
}

// Coin is the full coin detail from /coins/{id}.
type Coin struct {
	ID          string `kit:"id" json:"id"`
	Symbol      string `json:"symbol"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Price       string `json:"price"`
	MarketCap   string `json:"market_cap"`
	Volume24h   string `json:"volume_24h"`
	High24h     string `json:"high_24h"`
	Low24h      string `json:"low_24h"`
}

// Trending is one entry from the /search/trending response.
type Trending struct {
	ID     string `kit:"id" json:"id"`
	Name   string `json:"name"`
	Symbol string `json:"symbol"`
	Rank   int    `json:"rank"`
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
	High24h               float64 `json:"high_24h"`
	Low24h                float64 `json:"low_24h"`
	CirculatingSupply     float64 `json:"circulating_supply"`
}

type apiCoinDetail struct {
	ID          string            `json:"id"`
	Symbol      string            `json:"symbol"`
	Name        string            `json:"name"`
	Description map[string]string `json:"description"`
	MarketData  struct {
		CurrentPrice map[string]float64 `json:"current_price"`
		MarketCap    map[string]float64 `json:"market_cap"`
		TotalVolume  map[string]float64 `json:"total_volume"`
		High24h      map[string]float64 `json:"high_24h"`
		Low24h       map[string]float64 `json:"low_24h"`
	} `json:"market_data"`
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
