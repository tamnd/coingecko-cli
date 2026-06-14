package coingecko_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
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
  "bitcoin": {"usd": 64080},
  "ethereum": {"usd": 2430}
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
    {"item": {"id": "pepe", "symbol": "PEPE", "name": "Pepe", "market_cap_rank": 30}},
    {"item": {"id": "solana", "symbol": "SOL", "name": "Solana", "market_cap_rank": 5}}
  ],
  "nfts": [],
  "categories": []
}`

// --- price ---

func TestPriceParsesCoins(t *testing.T) {
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
	if items[0].Rank != 1 {
		t.Errorf("items[0].Rank = %d, want 1", items[0].Rank)
	}
	// Price is formatted as a string
	if items[0].Price != "64079.00" {
		t.Errorf("items[0].Price = %q, want 64079.00", items[0].Price)
	}
	// Change24h should have a percent sign
	if !strings.HasSuffix(items[0].Change24h, "%") {
		t.Errorf("items[0].Change24h = %q, expected percent sign", items[0].Change24h)
	}
	// Negative change for ethereum
	if !strings.HasPrefix(items[1].Change24h, "-") {
		t.Errorf("items[1].Change24h = %q, expected negative", items[1].Change24h)
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

// --- coin ---

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
	longDesc := strings.Repeat("x", 500)
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
}`, longDesc)

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
	if items[0].Rank != 30 {
		t.Errorf("items[0].Rank = %d, want 30", items[0].Rank)
	}
	if items[1].ID != "solana" {
		t.Errorf("items[1].ID = %q, want solana", items[1].ID)
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
