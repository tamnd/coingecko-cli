package coingecko_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tamnd/coingecko-cli/coingecko"
)

const fakeMarketsJSON = `[
  {
    "id": "bitcoin",
    "symbol": "btc",
    "name": "Bitcoin",
    "current_price": 64595.0,
    "price_change_percentage_24h": 1.04479,
    "market_cap": 1294560807468.0,
    "market_cap_rank": 1,
    "total_volume": 28000000000.0
  },
  {
    "id": "ethereum",
    "symbol": "eth",
    "name": "Ethereum",
    "current_price": 1674.7,
    "price_change_percentage_24h": -0.12044,
    "market_cap": 202108402502.0,
    "market_cap_rank": 2,
    "total_volume": 15000000000.0
  }
]`

const fakeTrendingJSON = `{
  "coins": [
    {"item": {"id": "humanity", "symbol": "H", "name": "Humanity", "market_cap_rank": 65}},
    {"item": {"id": "siren-2", "symbol": "SIREN", "name": "Siren", "market_cap_rank": 384}},
    {"item": {"id": "pudgy-penguins", "symbol": "PENGU", "name": "Pudgy Penguins", "market_cap_rank": 116}}
  ],
  "nfts": [],
  "categories": []
}`

func newTestClient(ts *httptest.Server) *coingecko.Client {
	cfg := coingecko.DefaultConfig()
	cfg.BaseURL = ts.URL
	cfg.Rate = 0
	return coingecko.NewClient(cfg)
}

func TestMarketsSendsUserAgent(t *testing.T) {
	var gotUA string
	var gotPerPage string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUA = r.Header.Get("User-Agent")
		gotPerPage = r.URL.Query().Get("per_page")
		_, _ = fmt.Fprint(w, fakeMarketsJSON)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	_, err := c.Markets(context.Background(), 5)
	if err != nil {
		t.Fatal(err)
	}
	if gotUA == "" {
		t.Error("User-Agent not sent")
	}
	if gotPerPage != "5" {
		t.Errorf("per_page = %q, want %q", gotPerPage, "5")
	}
}

func TestMarketsParsesItems(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, fakeMarketsJSON)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	items, err := c.Markets(context.Background(), 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("len(items) = %d, want 2", len(items))
	}
	if items[0].ID != "bitcoin" {
		t.Errorf("items[0].ID = %q, want %q", items[0].ID, "bitcoin")
	}
	if items[0].Symbol != "btc" {
		t.Errorf("items[0].Symbol = %q, want %q", items[0].Symbol, "btc")
	}
	if items[0].PriceUSD != 64595.0 {
		t.Errorf("items[0].PriceUSD = %f, want 64595.0", items[0].PriceUSD)
	}
	if items[0].MarketCapRank != 1 {
		t.Errorf("items[0].MarketCapRank = %d, want 1", items[0].MarketCapRank)
	}
	const wantURL = "https://www.coingecko.com/en/coins/bitcoin"
	if items[0].URL != wantURL {
		t.Errorf("items[0].URL = %q, want %q", items[0].URL, wantURL)
	}
	if items[1].ID != "ethereum" {
		t.Errorf("items[1].ID = %q, want %q", items[1].ID, "ethereum")
	}
	if items[1].Change24h >= 0 {
		t.Errorf("items[1].Change24h = %f, want negative", items[1].Change24h)
	}
}

func TestMarketsLimitRespected(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, fakeMarketsJSON)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	items, err := c.Markets(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Errorf("len(items) = %d, want 1", len(items))
	}
}

func TestMarketsRetriesOn503(t *testing.T) {
	var hits int
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		if hits < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_, _ = fmt.Fprint(w, fakeMarketsJSON)
	}))
	defer ts.Close()

	cfg := coingecko.DefaultConfig()
	cfg.BaseURL = ts.URL
	cfg.Rate = 0
	cfg.Retries = 3
	c := coingecko.NewClient(cfg)

	_, err := c.Markets(context.Background(), 0)
	if err != nil {
		t.Fatal(err)
	}
	if hits != 3 {
		t.Errorf("server saw %d hits, want 3", hits)
	}
}

const fakePriceJSON = `{
  "bitcoin": {"usd": 64293, "eur": 55574},
  "ethereum": {"usd": 3100, "eur": 2680}
}`

const fakeCoinDetailJSON = `{
  "id": "bitcoin",
  "symbol": "btc",
  "name": "Bitcoin",
  "market_cap_rank": 1,
  "market_data": {
    "current_price": {"usd": 64293, "eur": 55574},
    "market_cap": {"usd": 1267000000000},
    "price_change_percentage_24h": 1.23
  },
  "description": {"en": "Bitcoin is a decentralized digital currency."},
  "categories": ["Cryptocurrency", "Layer 1 (L1)"]
}`

const fakeSearchJSON = `{
  "coins": [
    {"id": "solana", "symbol": "sol", "name": "Solana", "market_cap_rank": 5},
    {"id": "sol-token", "symbol": "sol", "name": "Sol Token", "market_cap_rank": 999}
  ],
  "categories": []
}`

func TestPriceParsesResponse(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/simple/price" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		_, _ = fmt.Fprint(w, fakePriceJSON)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	p, err := c.Price(context.Background(), []string{"bitcoin", "ethereum"}, []string{"usd", "eur"})
	if err != nil {
		t.Fatal(err)
	}
	if p["bitcoin"]["usd"] != 64293 {
		t.Errorf("bitcoin/usd = %v, want 64293", p["bitcoin"]["usd"])
	}
	if p["bitcoin"]["eur"] != 55574 {
		t.Errorf("bitcoin/eur = %v, want 55574", p["bitcoin"]["eur"])
	}
	if p["ethereum"]["usd"] != 3100 {
		t.Errorf("ethereum/usd = %v, want 3100", p["ethereum"]["usd"])
	}
}

func TestCoinInfoParsesResponse(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, fakeCoinDetailJSON)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	d, err := c.CoinInfo(context.Background(), "bitcoin")
	if err != nil {
		t.Fatal(err)
	}
	if d.ID != "bitcoin" {
		t.Errorf("ID = %q, want bitcoin", d.ID)
	}
	if d.MarketCapRank != 1 {
		t.Errorf("MarketCapRank = %d, want 1", d.MarketCapRank)
	}
	if d.MarketData.CurrentPrice["usd"] != 64293 {
		t.Errorf("CurrentPrice[usd] = %v, want 64293", d.MarketData.CurrentPrice["usd"])
	}
	if d.Description["en"] == "" {
		t.Error("Description[en] is empty")
	}
	if len(d.Categories) == 0 {
		t.Error("Categories is empty")
	}
}

func TestSearchParsesResponse(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/search" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("query") != "sol" {
			t.Errorf("query param = %q, want sol", r.URL.Query().Get("query"))
		}
		_, _ = fmt.Fprint(w, fakeSearchJSON)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	results, err := c.Search(context.Background(), "sol")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Fatalf("len(results) = %d, want 2", len(results))
	}
	if results[0].ID != "solana" {
		t.Errorf("results[0].ID = %q, want solana", results[0].ID)
	}
	if results[0].MarketCapRank != 5 {
		t.Errorf("results[0].MarketCapRank = %d, want 5", results[0].MarketCapRank)
	}
}

func TestMarketsInCurrencyEUR(t *testing.T) {
	var gotCurrency string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotCurrency = r.URL.Query().Get("vs_currency")
		_, _ = fmt.Fprint(w, fakeMarketsJSON)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	coins, err := c.MarketsInCurrency(context.Background(), "eur", 2)
	if err != nil {
		t.Fatal(err)
	}
	if gotCurrency != "eur" {
		t.Errorf("vs_currency = %q, want eur", gotCurrency)
	}
	if len(coins) != 2 {
		t.Errorf("len(coins) = %d, want 2", len(coins))
	}
}

func TestTrendingParsesItems(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, fakeTrendingJSON)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	items, err := c.Trending(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 3 {
		t.Fatalf("len(items) = %d, want 3", len(items))
	}
	if items[0].ID != "humanity" {
		t.Errorf("items[0].ID = %q, want %q", items[0].ID, "humanity")
	}
	if items[0].Symbol != "H" {
		t.Errorf("items[0].Symbol = %q, want %q", items[0].Symbol, "H")
	}
	if items[0].MarketCapRank != 65 {
		t.Errorf("items[0].MarketCapRank = %d, want 65", items[0].MarketCapRank)
	}
	if items[0].Rank != 1 {
		t.Errorf("items[0].Rank = %d, want 1", items[0].Rank)
	}
	const wantURL = "https://www.coingecko.com/en/coins/humanity"
	if items[0].URL != wantURL {
		t.Errorf("items[0].URL = %q, want %q", items[0].URL, wantURL)
	}
}
