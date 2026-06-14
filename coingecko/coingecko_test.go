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
  "bitcoin": {"usd": 64080, "eur": 55394},
  "ethereum": {"usd": 1664, "eur": 1438}
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
    "circulating_supply": 19700000,
    "ath": 73750.07
  },
  {
    "id": "ethereum",
    "symbol": "eth",
    "name": "Ethereum",
    "current_price": 1664,
    "market_cap": 202000000000,
    "market_cap_rank": 2,
    "price_change_percentage_24h": -0.5,
    "total_volume": 8000000000,
    "circulating_supply": 120000000,
    "ath": 4891.70
  }
]`

const fakeCoinJSON = `{
  "id": "bitcoin",
  "symbol": "btc",
  "name": "Bitcoin",
  "genesis_date": "2009-01-03",
  "description": {"en": "Bitcoin is a decentralized digital currency."},
  "market_data": {
    "current_price": {"usd": 64014, "eur": 55300},
    "market_cap": {"usd": 1261234567890},
    "ath": {"usd": 73750.07},
    "price_change_percentage_24h": 1.23
  }
}`

const fakeTrendingJSON = `{
  "coins": [
    {"item": {"id": "pepe", "symbol": "PEPE", "name": "Pepe", "market_cap_rank": 30, "price_btc": 1.23e-7}},
    {"item": {"id": "solana", "symbol": "SOL", "name": "Solana", "market_cap_rank": 5, "price_btc": 0.00025}}
  ],
  "nfts": [],
  "categories": []
}`

const fakeSearchJSON = `{
  "coins": [
    {"id": "bitcoin", "name": "Bitcoin", "symbol": "BTC", "market_cap_rank": 1},
    {"id": "bitcoin-cash", "name": "Bitcoin Cash", "symbol": "BCH", "market_cap_rank": 18}
  ],
  "exchanges": []
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
	prices, err := c.Price(context.Background(), "bitcoin,ethereum", "usd,eur")
	if err != nil {
		t.Fatal(err)
	}
	if len(prices) != 2 {
		t.Fatalf("len(prices) = %d, want 2", len(prices))
	}
	if prices[0].CoinID != "bitcoin" {
		t.Errorf("prices[0].CoinID = %q, want bitcoin", prices[0].CoinID)
	}
	if prices[0].Prices["usd"] != 64080 {
		t.Errorf("bitcoin usd = %v, want 64080", prices[0].Prices["usd"])
	}
	if prices[0].Prices["eur"] != 55394 {
		t.Errorf("bitcoin eur = %v, want 55394", prices[0].Prices["eur"])
	}
	if prices[1].CoinID != "ethereum" {
		t.Errorf("prices[1].CoinID = %q, want ethereum", prices[1].CoinID)
	}
	if prices[1].Prices["usd"] != 1664 {
		t.Errorf("ethereum usd = %v, want 1664", prices[1].Prices["usd"])
	}
}

func TestPriceRequiresIDs(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, "{}")
	}))
	defer ts.Close()

	c := newTestClient(ts)
	_, err := c.Price(context.Background(), "", "usd")
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
	items, err := c.Markets(context.Background(), "", "usd", 10)
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
	if items[0].CurrentPrice != 64079 {
		t.Errorf("items[0].CurrentPrice = %v, want 64079", items[0].CurrentPrice)
	}
	if items[0].MarketCapRank != 1 {
		t.Errorf("items[0].MarketCapRank = %d, want 1", items[0].MarketCapRank)
	}
	if items[0].ATH != 73750.07 {
		t.Errorf("items[0].ATH = %v, want 73750.07", items[0].ATH)
	}
	if items[1].PriceChange24h >= 0 {
		t.Errorf("items[1].PriceChange24h = %v, want negative", items[1].PriceChange24h)
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
	_, err := c.Markets(context.Background(), "", "eur", 5)
	if err != nil {
		t.Fatal(err)
	}
	if gotCurrency != "eur" {
		t.Errorf("vs_currency = %q, want eur", gotCurrency)
	}
}

func TestMarketsPassesIDsParam(t *testing.T) {
	var gotIDs string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotIDs = r.URL.Query().Get("ids")
		_, _ = fmt.Fprint(w, fakeMarketsJSON)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	_, err := c.Markets(context.Background(), "bitcoin,ethereum", "usd", 10)
	if err != nil {
		t.Fatal(err)
	}
	if gotIDs != "bitcoin,ethereum" {
		t.Errorf("ids = %q, want bitcoin,ethereum", gotIDs)
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
	d, err := c.Coin(context.Background(), "bitcoin")
	if err != nil {
		t.Fatal(err)
	}
	if d.ID != "bitcoin" {
		t.Errorf("ID = %q, want bitcoin", d.ID)
	}
	if d.Symbol != "btc" {
		t.Errorf("Symbol = %q, want btc", d.Symbol)
	}
	if d.CurrentUSD != 64014 {
		t.Errorf("CurrentUSD = %v, want 64014", d.CurrentUSD)
	}
	if d.MarketCapUSD != 1261234567890 {
		t.Errorf("MarketCapUSD = %v, want 1261234567890", d.MarketCapUSD)
	}
	if d.ATH_USD != 73750.07 {
		t.Errorf("ATH_USD = %v, want 73750.07", d.ATH_USD)
	}
	if d.Change24h != 1.23 {
		t.Errorf("Change24h = %v, want 1.23", d.Change24h)
	}
	if d.Description == "" {
		t.Error("Description is empty")
	}
	if d.GenesisDate != "2009-01-03" {
		t.Errorf("GenesisDate = %q, want 2009-01-03", d.GenesisDate)
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
	if items[0].PriceBTC != 1.23e-7 {
		t.Errorf("items[0].PriceBTC = %v, want 1.23e-7", items[0].PriceBTC)
	}
}

// --- search ---

func TestSearchParsesResults(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/search" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		if r.URL.Query().Get("query") != "bitcoin" {
			t.Errorf("query = %q, want bitcoin", r.URL.Query().Get("query"))
		}
		_, _ = fmt.Fprint(w, fakeSearchJSON)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	results, err := c.Search(context.Background(), "bitcoin")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Fatalf("len(results) = %d, want 2", len(results))
	}
	if results[0].ID != "bitcoin" {
		t.Errorf("results[0].ID = %q, want bitcoin", results[0].ID)
	}
	if results[0].Symbol != "BTC" {
		t.Errorf("results[0].Symbol = %q, want BTC", results[0].Symbol)
	}
	if results[0].MarketCapRank != 1 {
		t.Errorf("results[0].MarketCapRank = %d, want 1", results[0].MarketCapRank)
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

	_, err := c.Price(context.Background(), "bitcoin", "usd")
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
