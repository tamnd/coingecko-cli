// Package coingecko is the library behind the coingecko command line:
// the HTTP client, request shaping, and the typed data models for the CoinGecko
// public API (markets and trending).
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
const Host = "coingecko.com"

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

// Markets returns the top n coins by market capitalisation descending.
// Pass limit <= 0 to use the default of 20.
func (c *Client) Markets(ctx context.Context, limit int) ([]Coin, error) {
	n := limit
	if n <= 0 {
		n = 20
	}
	u := fmt.Sprintf("%s/coins/markets?vs_currency=usd&order=market_cap_desc&per_page=%d&sparkline=false",
		c.cfg.BaseURL, n)
	body, err := c.get(ctx, u)
	if err != nil {
		return nil, err
	}
	var raw []apiCoin
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("decode markets: %w", err)
	}
	coins := make([]Coin, 0, len(raw))
	for i, r := range raw {
		coins = append(coins, Coin{
			Rank:          i + 1,
			ID:            r.ID,
			Symbol:        r.Symbol,
			Name:          r.Name,
			PriceUSD:      r.CurrentPrice,
			Change24h:     r.PriceChangePercentage,
			MarketCapUSD:  r.MarketCap,
			MarketCapRank: r.MarketCapRank,
			Volume24hUSD:  r.TotalVolume,
			URL:           "https://www.coingecko.com/en/coins/" + r.ID,
		})
	}
	if limit > 0 && limit < len(coins) {
		coins = coins[:limit]
	}
	return coins, nil
}

// Markets returns the top n coins by market capitalisation in the given currency.
// If currency is empty, "usd" is used. Pass limit <= 0 to use the default of 20.
func (c *Client) MarketsInCurrency(ctx context.Context, currency string, limit int) ([]Coin, error) {
	if currency == "" {
		currency = "usd"
	}
	n := limit
	if n <= 0 {
		n = 20
	}
	u := fmt.Sprintf("%s/coins/markets?vs_currency=%s&order=market_cap_desc&per_page=%d&sparkline=false",
		c.cfg.BaseURL, currency, n)
	body, err := c.get(ctx, u)
	if err != nil {
		return nil, err
	}
	var raw []apiCoin
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("decode markets: %w", err)
	}
	coins := make([]Coin, 0, len(raw))
	for i, r := range raw {
		coins = append(coins, Coin{
			Rank:          i + 1,
			ID:            r.ID,
			Symbol:        r.Symbol,
			Name:          r.Name,
			PriceUSD:      r.CurrentPrice,
			Change24h:     r.PriceChangePercentage,
			MarketCapUSD:  r.MarketCap,
			MarketCapRank: r.MarketCapRank,
			Volume24hUSD:  r.TotalVolume,
			URL:           "https://www.coingecko.com/en/coins/" + r.ID,
		})
	}
	if limit > 0 && limit < len(coins) {
		coins = coins[:limit]
	}
	return coins, nil
}

// Price returns prices for the given coin IDs in the given currencies.
func (c *Client) Price(ctx context.Context, ids []string, currencies []string) (Price, error) {
	if len(ids) == 0 {
		return nil, fmt.Errorf("price: at least one coin ID required")
	}
	if len(currencies) == 0 {
		currencies = []string{"usd"}
	}
	u := fmt.Sprintf("%s/simple/price?ids=%s&vs_currencies=%s",
		c.cfg.BaseURL,
		strings.Join(ids, ","),
		strings.Join(currencies, ","))
	body, err := c.get(ctx, u)
	if err != nil {
		return nil, err
	}
	var p Price
	if err := json.Unmarshal(body, &p); err != nil {
		return nil, fmt.Errorf("decode price: %w", err)
	}
	return p, nil
}

// CoinInfo returns the full detail object for a single coin.
func (c *Client) CoinInfo(ctx context.Context, id string) (CoinDetail, error) {
	u := fmt.Sprintf("%s/coins/%s?localization=false&tickers=false&market_data=true&community_data=false&developer_data=false",
		c.cfg.BaseURL, id)
	body, err := c.get(ctx, u)
	if err != nil {
		return CoinDetail{}, err
	}
	var d CoinDetail
	if err := json.Unmarshal(body, &d); err != nil {
		return CoinDetail{}, fmt.Errorf("decode coin: %w", err)
	}
	return d, nil
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
	coins := make([]TrendingCoin, 0, len(resp.Coins))
	for i, w := range resp.Coins {
		coins = append(coins, TrendingCoin{
			Rank:          i + 1,
			ID:            w.Item.ID,
			Symbol:        w.Item.Symbol,
			Name:          w.Item.Name,
			MarketCapRank: w.Item.MarketCapRank,
			URL:           "https://www.coingecko.com/en/coins/" + w.Item.ID,
		})
	}
	return coins, nil
}

func (c *Client) get(ctx context.Context, url string) ([]byte, error) {
	var lastErr error
	for attempt := 0; attempt <= c.cfg.Retries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff(attempt)):
			}
		}
		body, retry, err := c.do(ctx, url)
		if err == nil {
			return body, nil
		}
		lastErr = err
		if !retry {
			return nil, err
		}
	}
	return nil, fmt.Errorf("get %s: %w", url, lastErr)
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
