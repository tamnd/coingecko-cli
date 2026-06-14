package coingecko_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tamnd/coingecko-cli/coingecko"
)

// newTestClient creates a Client pointed at the given test server with rate limiting disabled.
func newTestClient(ts *httptest.Server) *coingecko.Client {
	cfg := coingecko.DefaultConfig()
	cfg.BaseURL = ts.URL
	cfg.Rate = 0
	return coingecko.NewClient(cfg)
}

// --- fixture JSON ---

const fakePriceJSON = `{
  "bitcoin": {"usd": 64080, "eur": 56087},
  "ethereum": {"usd": 2430, "eur": 2100}
}`

const fakeMarketsJSON = `[
  {
    "id": "bitcoin",
    "symbol": "btc",
    "name": "Bitcoin",
    "current_price": 64079,
    "market_cap": 1261234567890,
    "market_cap_rank": 1,
    "price_change_percentage_24h": 1.23,
    "total_volume": 18234567890,
    "high_24h": 64214,
    "low_24h": 63600,
    "circulating_supply": 19700000
  },
  {
    "id": "ethereum",
    "symbol": "eth",
    "name": "Ethereum",
    "current_price": 2430,
    "market_cap": 292000000000,
    "market_cap_rank": 2,
    "price_change_percentage_24h": -0.5,
    "total_volume": 8000000000,
    "high_24h": 2450,
    "low_24h": 2400,
    "circulating_supply": 120000000
  }
]`

const fakeCoinJSON = `{
  "id": "bitcoin",
  "symbol": "btc",
  "name": "Bitcoin",
  "description": {"en": "Bitcoin is a decentralized digital currency."},
  "market_data": {
    "current_price": {"usd": 64014},
    "market_cap": {"usd": 1261234567890},
    "total_volume": {"usd": 28000000000},
    "high_24h": {"usd": 64214},
    "low_24h": {"usd": 63600}
  }
}`

const fakeTrendingJSON = `{
  "coins": [
    {"item": {"id": "pepe", "symbol": "PEPE", "name": "Pepe", "market_cap_rank": 30, "price_btc": 0.0000001}},
    {"item": {"id": "solana", "symbol": "SOL", "name": "Solana", "market_cap_rank": 5, "price_btc": 0.0012}}
  ],
  "nfts": [],
  "categories": []
}`

const fakeCoinsListJSON = `[
  {"id": "bitcoin", "symbol": "btc", "name": "Bitcoin"},
  {"id": "ethereum", "symbol": "eth", "name": "Ethereum"},
  {"id": "solana", "symbol": "sol", "name": "Solana"}
]`

// --- price ---

func TestPriceSingleCurrency(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/simple/price" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		_, _ = fmt.Fprint(w, fakePriceJSON)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	prices, err := c.Price(context.Background(), []string{"bitcoin", "ethereum"}, "usd")
	if err != nil {
		t.Fatal(err)
	}
	if len(prices) != 2 {
		t.Fatalf("len(prices) = %d, want 2", len(prices))
	}
	if prices[0].ID != "bitcoin" {
		t.Errorf("prices[0].ID = %q, want bitcoin", prices[0].ID)
	}
	if prices[0].Currency != "usd" {
		t.Errorf("prices[0].Currency = %q, want usd", prices[0].Currency)
	}
	if prices[0].Price != 64080 {
		t.Errorf("bitcoin price = %v, want 64080", prices[0].Price)
	}
	if prices[1].ID != "ethereum" {
		t.Errorf("prices[1].ID = %q, want ethereum", prices[1].ID)
	}
	if prices[1].Price != 2430 {
		t.Errorf("ethereum price = %v, want 2430", prices[1].Price)
	}
}

func TestPriceMultipleCurrencies(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, fakePriceJSON)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	// bitcoin in usd and eur -> 2 Price records
	prices, err := c.Price(context.Background(), []string{"bitcoin"}, "usd", "eur")
	if err != nil {
		t.Fatal(err)
	}
	if len(prices) != 2 {
		t.Fatalf("len(prices) = %d, want 2 (one per currency)", len(prices))
	}
	if prices[0].Currency != "usd" {
		t.Errorf("prices[0].Currency = %q, want usd", prices[0].Currency)
	}
	if prices[1].Currency != "eur" {
		t.Errorf("prices[1].Currency = %q, want eur", prices[1].Currency)
	}
	if prices[1].Price != 56087 {
		t.Errorf("bitcoin EUR price = %v, want 56087", prices[1].Price)
	}
}

func TestPriceDefaultsCurrencyToUSD(t *testing.T) {
	var gotCurr string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotCurr = r.URL.Query().Get("vs_currencies")
		_, _ = fmt.Fprint(w, `{"bitcoin":{"usd":64080}}`)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	_, err := c.Price(context.Background(), []string{"bitcoin"})
	if err != nil {
		t.Fatal(err)
	}
	if gotCurr != "usd" {
		t.Errorf("vs_currencies = %q, want usd", gotCurr)
	}
}

func TestPriceRequiresIDs(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, "{}")
	}))
	defer ts.Close()

	c := newTestClient(ts)
	_, err := c.Price(context.Background(), nil, "usd")
	if err == nil {
		t.Error("Price with empty IDs should return error")
	}
}

// --- markets ---

func TestMarketsParsesItems(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/coins/markets" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		_, _ = fmt.Fprint(w, fakeMarketsJSON)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	items, err := c.Markets(context.Background(), "usd", 10, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("len(items) = %d, want 2", len(items))
	}
	if items[0].ID != "bitcoin" {
		t.Errorf("items[0].ID = %q, want bitcoin", items[0].ID)
	}
	if items[0].Symbol != "btc" {
		t.Errorf("items[0].Symbol = %q, want btc", items[0].Symbol)
	}
	if items[0].MarketCapRank != 1 {
		t.Errorf("items[0].MarketCapRank = %d, want 1", items[0].MarketCapRank)
	}
	if items[0].CurrentPrice != 64079 {
		t.Errorf("items[0].CurrentPrice = %v, want 64079", items[0].CurrentPrice)
	}
	if items[0].MarketCap != 1261234567890 {
		t.Errorf("items[0].MarketCap = %v, want 1261234567890", items[0].MarketCap)
	}
	if items[0].TotalVolume != 18234567890 {
		t.Errorf("items[0].TotalVolume = %v, want 18234567890", items[0].TotalVolume)
	}
	if items[0].PriceChange24h != 1.23 {
		t.Errorf("items[0].PriceChange24h = %v, want 1.23", items[0].PriceChange24h)
	}
	if items[1].PriceChange24h != -0.5 {
		t.Errorf("items[1].PriceChange24h = %v, want -0.5", items[1].PriceChange24h)
	}
}

func TestMarketsPassesCurrencyParam(t *testing.T) {
	var gotCurrency string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotCurrency = r.URL.Query().Get("vs_currency")
		_, _ = fmt.Fprint(w, fakeMarketsJSON)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	_, err := c.Markets(context.Background(), "eur", 5, 1)
	if err != nil {
		t.Fatal(err)
	}
	if gotCurrency != "eur" {
		t.Errorf("vs_currency = %q, want eur", gotCurrency)
	}
}

func TestMarketsPassesPageParam(t *testing.T) {
	var gotPage string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPage = r.URL.Query().Get("page")
		_, _ = fmt.Fprint(w, fakeMarketsJSON)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	_, err := c.Markets(context.Background(), "usd", 10, 2)
	if err != nil {
		t.Fatal(err)
	}
	if gotPage != "2" {
		t.Errorf("page = %q, want 2", gotPage)
	}
}

// --- coin detail ---

func TestCoinParsesDetail(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/coins/bitcoin" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		_, _ = fmt.Fprint(w, fakeCoinJSON)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	d, err := c.CoinDetail(context.Background(), "bitcoin")
	if err != nil {
		t.Fatal(err)
	}
	if d.ID != "bitcoin" {
		t.Errorf("ID = %q, want bitcoin", d.ID)
	}
	if d.Symbol != "btc" {
		t.Errorf("Symbol = %q, want btc", d.Symbol)
	}
	if d.Price != "64014.00" {
		t.Errorf("Price = %q, want 64014.00", d.Price)
	}
	if d.Description == "" {
		t.Error("Description is empty")
	}
	if d.High24h != "64214.00" {
		t.Errorf("High24h = %q, want 64214.00", d.High24h)
	}
	if d.Low24h != "63600.00" {
		t.Errorf("Low24h = %q, want 63600.00", d.Low24h)
	}
}

func TestCoinDescriptionTruncated(t *testing.T) {
	longDesc := make([]byte, 500)
	for i := range longDesc {
		longDesc[i] = 'x'
	}
	coinJSON := fmt.Sprintf(`{
  "id": "testcoin", "symbol": "tc", "name": "Test",
  "description": {"en": %q},
  "market_data": {
    "current_price": {"usd": 1.0},
    "market_cap": {"usd": 1000000},
    "total_volume": {"usd": 500000},
    "high_24h": {"usd": 1.1},
    "low_24h": {"usd": 0.9}
  }
}`, string(longDesc))

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, coinJSON)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	d, err := c.CoinDetail(context.Background(), "testcoin")
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Description) > 300 {
		t.Errorf("Description length = %d, want <= 300", len(d.Description))
	}
}

// --- trending ---

func TestTrendingParsesItems(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/search/trending" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		_, _ = fmt.Fprint(w, fakeTrendingJSON)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	items, err := c.Trending(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("len(items) = %d, want 2", len(items))
	}
	if items[0].ID != "pepe" {
		t.Errorf("items[0].ID = %q, want pepe", items[0].ID)
	}
	if items[0].Symbol != "PEPE" {
		t.Errorf("items[0].Symbol = %q, want PEPE", items[0].Symbol)
	}
	if items[0].MarketCapRank != 30 {
		t.Errorf("items[0].MarketCapRank = %d, want 30", items[0].MarketCapRank)
	}
	if items[0].PriceBTC != 0.0000001 {
		t.Errorf("items[0].PriceBTC = %v, want 0.0000001", items[0].PriceBTC)
	}
	if items[1].ID != "solana" {
		t.Errorf("items[1].ID = %q, want solana", items[1].ID)
	}
}

// --- coins list ---

func TestCoinsListParsesItems(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/coins/list" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		_, _ = fmt.Fprint(w, fakeCoinsListJSON)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	items, err := c.Coins(context.Background(), 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 3 {
		t.Fatalf("len(items) = %d, want 3", len(items))
	}
	if items[0].ID != "bitcoin" {
		t.Errorf("items[0].ID = %q, want bitcoin", items[0].ID)
	}
	if items[0].Symbol != "btc" {
		t.Errorf("items[0].Symbol = %q, want btc", items[0].Symbol)
	}
	if items[0].Name != "Bitcoin" {
		t.Errorf("items[0].Name = %q, want Bitcoin", items[0].Name)
	}
}

func TestCoinsListLimit(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, fakeCoinsListJSON)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	items, err := c.Coins(context.Background(), 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Errorf("len(items) = %d, want 2 (limit applied)", len(items))
	}
}

// --- retry ---

func TestRetriesOn503(t *testing.T) {
	var hits int
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		if hits < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_, _ = fmt.Fprint(w, fakePriceJSON)
	}))
	defer ts.Close()

	cfg := coingecko.DefaultConfig()
	cfg.BaseURL = ts.URL
	cfg.Rate = 0
	cfg.Retries = 3
	c := coingecko.NewClient(cfg)

	_, err := c.Price(context.Background(), []string{"bitcoin"}, "usd")
	if err != nil {
		t.Fatal(err)
	}
	if hits != 3 {
		t.Errorf("server saw %d hits, want 3", hits)
	}
}

// --- user agent ---

func TestUserAgentIsSent(t *testing.T) {
	var gotUA string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUA = r.Header.Get("User-Agent")
		_, _ = fmt.Fprint(w, fakeTrendingJSON)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	_, err := c.Trending(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if gotUA == "" {
		t.Error("User-Agent header not sent")
	}
}
