// Package coingecko is the library behind the coingecko command line:
// the HTTP client, request shaping, and the typed data models for the CoinGecko
// public API.
//
// The free CoinGecko API requires no key. The client paces requests at a
// 2-second floor to stay well within the ~30 req/min free-tier limit, and
// retries transient failures (429 and 5xx) with exponential backoff.
package coingecko

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// Host is the site this client talks to.
const Host = "api.coingecko.com"

// Config holds all tunable parameters for the Client.
type Config struct {
	BaseURL   string
	UserAgent string
	Rate      time.Duration
	Timeout   time.Duration
	Retries   int
}

// DefaultConfig returns a Config with sensible defaults for the free tier.
func DefaultConfig() Config {
	return Config{
		BaseURL:   "https://api.coingecko.com/api/v3",
		UserAgent: "Mozilla/5.0 (compatible; coingecko-cli/dev; +https://github.com/tamnd/coingecko-cli)",
		Rate:      2 * time.Second,
		Timeout:   30 * time.Second,
		Retries:   3,
	}
}

// Client talks to CoinGecko over HTTP.
type Client struct {
	cfg  Config
	http *http.Client
	mu   sync.Mutex
	last time.Time
}

// NewClient returns a Client configured with cfg.
func NewClient(cfg Config) *Client {
	return &Client{
		cfg:  cfg,
		http: &http.Client{Timeout: cfg.Timeout},
	}
}

// Price returns prices for the given coin IDs in the given currencies.
// Each coin ID in the response becomes one Price struct.
func (c *Client) Price(ctx context.Context, ids string, currencies string) ([]Price, error) {
	if ids == "" {
		return nil, fmt.Errorf("price: at least one coin ID required")
	}
	if currencies == "" {
		currencies = "usd"
	}
	u := fmt.Sprintf("%s/simple/price?ids=%s&vs_currencies=%s",
		c.cfg.BaseURL,
		url.QueryEscape(ids),
		url.QueryEscape(currencies))
	body, err := c.get(ctx, u)
	if err != nil {
		return nil, err
	}
	// API returns: {"bitcoin":{"usd":64080,"eur":55394},...}
	var raw map[string]map[string]float64
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("decode price: %w", err)
	}
	// Preserve order from the input ids list.
	idList := strings.Split(ids, ",")
	out := make([]Price, 0, len(idList))
	for _, id := range idList {
		id = strings.TrimSpace(id)
		if prices, ok := raw[id]; ok {
			out = append(out, Price{CoinID: id, Prices: prices})
		}
	}
	return out, nil
}

// Markets returns coins sorted by market cap descending, optionally filtered by
// a comma-separated list of IDs. Pass limit <= 0 for API default (100).
func (c *Client) Markets(ctx context.Context, ids string, currency string, limit int) ([]CoinMarket, error) {
	if currency == "" {
		currency = "usd"
	}
	if limit <= 0 {
		limit = 10
	}
	u := fmt.Sprintf("%s/coins/markets?vs_currency=%s&order=market_cap_desc&per_page=%d&sparkline=false",
		c.cfg.BaseURL, url.QueryEscape(currency), limit)
	if ids != "" {
		u += "&ids=" + url.QueryEscape(ids)
	}
	body, err := c.get(ctx, u)
	if err != nil {
		return nil, err
	}
	var raw []apiCoin
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("decode markets: %w", err)
	}
	out := make([]CoinMarket, 0, len(raw))
	for _, r := range raw {
		out = append(out, CoinMarket{
			ID:                r.ID,
			Symbol:            r.Symbol,
			Name:              r.Name,
			CurrentPrice:      r.CurrentPrice,
			MarketCap:         r.MarketCap,
			MarketCapRank:     r.MarketCapRank,
			PriceChange24h:    r.PriceChangePercentage,
			TotalVolume:       r.TotalVolume,
			CirculatingSupply: r.CirculatingSupply,
			ATH:               r.ATH,
		})
	}
	return out, nil
}

// Coin returns the full detail object for a single coin.
func (c *Client) Coin(ctx context.Context, id string) (CoinDetail, error) {
	u := fmt.Sprintf("%s/coins/%s?localization=false&tickers=false&community_data=false&developer_data=false&sparkline=false",
		c.cfg.BaseURL, url.PathEscape(id))
	body, err := c.get(ctx, u)
	if err != nil {
		return CoinDetail{}, err
	}
	var d apiCoinDetail
	if err := json.Unmarshal(body, &d); err != nil {
		return CoinDetail{}, fmt.Errorf("decode coin: %w", err)
	}
	return CoinDetail{
		ID:           d.ID,
		Symbol:       d.Symbol,
		Name:         d.Name,
		Description:  d.Description["en"],
		GenesisDate:  d.GenesisDate,
		CurrentUSD:   d.MarketData.CurrentPrice["usd"],
		MarketCapUSD: d.MarketData.MarketCap["usd"],
		ATH_USD:      d.MarketData.ATH["usd"],
		Change24h:    d.MarketData.PriceChangePct24h,
	}, nil
}

// Trending returns the currently trending coins (most searched in the last 24h).
func (c *Client) Trending(ctx context.Context) ([]TrendingCoin, error) {
	u := c.cfg.BaseURL + "/search/trending"
	body, err := c.get(ctx, u)
	if err != nil {
		return nil, err
	}
	var resp trendingResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("decode trending: %w", err)
	}
	out := make([]TrendingCoin, 0, len(resp.Coins))
	for _, w := range resp.Coins {
		out = append(out, TrendingCoin{
			ID:            w.Item.ID,
			Symbol:        w.Item.Symbol,
			Name:          w.Item.Name,
			MarketCapRank: w.Item.MarketCapRank,
			PriceBTC:      w.Item.PriceBTC,
		})
	}
	return out, nil
}

// Search searches for coins matching the given query string.
func (c *Client) Search(ctx context.Context, query string) ([]SearchResult, error) {
	u := fmt.Sprintf("%s/search?query=%s", c.cfg.BaseURL, url.QueryEscape(query))
	body, err := c.get(ctx, u)
	if err != nil {
		return nil, err
	}
	var sr searchResponse
	if err := json.Unmarshal(body, &sr); err != nil {
		return nil, fmt.Errorf("decode search: %w", err)
	}
	return sr.Coins, nil
}

func (c *Client) get(ctx context.Context, rawURL string) ([]byte, error) {
	var lastErr error
	for attempt := 0; attempt <= c.cfg.Retries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff(attempt)):
			}
		}
		body, retry, err := c.do(ctx, rawURL)
		if err == nil {
			return body, nil
		}
		lastErr = err
		if !retry {
			return nil, err
		}
	}
	return nil, fmt.Errorf("get %s: %w", rawURL, lastErr)
}

func (c *Client) do(ctx context.Context, rawURL string) ([]byte, bool, error) {
	c.pace()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("User-Agent", c.cfg.UserAgent)
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, true, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
		return nil, true, fmt.Errorf("http %d", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("http %d", resp.StatusCode)
	}
	b, err := io.ReadAll(resp.Body)
	return b, err != nil, err
}

func (c *Client) pace() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.cfg.Rate <= 0 {
		return
	}
	if wait := c.cfg.Rate - time.Since(c.last); wait > 0 {
		time.Sleep(wait)
	}
	c.last = time.Now()
}

func backoff(attempt int) time.Duration {
	return min(time.Duration(attempt)*500*time.Millisecond, 5*time.Second)
}
