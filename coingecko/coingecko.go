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
		UserAgent: "coingecko-cli/0.1 (tamnd87@gmail.com)",
		Rate:      2 * time.Second,
		Timeout:   15 * time.Second,
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

// Price returns prices for the given coin IDs in one or more currencies.
// One Price is emitted per (coin ID, currency) pair.
func (c *Client) Price(ctx context.Context, ids []string, currencies ...string) ([]Price, error) {
	if len(ids) == 0 {
		return nil, fmt.Errorf("price: at least one coin ID required")
	}
	if len(currencies) == 0 {
		currencies = []string{"usd"}
	}
	joined := strings.Join(ids, ",")
	joinedCurr := strings.Join(currencies, ",")
	u := fmt.Sprintf("%s/simple/price?ids=%s&vs_currencies=%s",
		c.cfg.BaseURL,
		url.QueryEscape(joined),
		url.QueryEscape(joinedCurr))
	body, err := c.get(ctx, u)
	if err != nil {
		return nil, err
	}
	// API returns: {"bitcoin":{"usd":64080,"eur":56087},"ethereum":{...}}
	var raw map[string]map[string]float64
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("decode price: %w", err)
	}
	out := make([]Price, 0, len(ids)*len(currencies))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		coinPrices, ok := raw[id]
		if !ok {
			continue
		}
		for _, curr := range currencies {
			curr = strings.TrimSpace(curr)
			if val, ok := coinPrices[curr]; ok {
				out = append(out, Price{
					ID:       id,
					Currency: curr,
					Price:    val,
				})
			}
		}
	}
	return out, nil
}

// Markets returns coins sorted by market cap descending.
// Pass limit <= 0 for the default (25). page is 1-indexed.
func (c *Client) Markets(ctx context.Context, currency string, limit, page int) ([]MarketCoin, error) {
	if currency == "" {
		currency = "usd"
	}
	if limit <= 0 {
		limit = 25
	}
	if page <= 0 {
		page = 1
	}
	u := fmt.Sprintf("%s/coins/markets?vs_currency=%s&order=market_cap_desc&per_page=%d&page=%d&sparkline=false",
		c.cfg.BaseURL, url.QueryEscape(currency), limit, page)
	body, err := c.get(ctx, u)
	if err != nil {
		return nil, err
	}
	var raw []apiCoin
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("decode markets: %w", err)
	}
	out := make([]MarketCoin, 0, len(raw))
	for _, r := range raw {
		out = append(out, MarketCoin{
			ID:             r.ID,
			Symbol:         r.Symbol,
			Name:           r.Name,
			CurrentPrice:   r.CurrentPrice,
			MarketCap:      r.MarketCap,
			MarketCapRank:  r.MarketCapRank,
			PriceChange24h: r.PriceChangePercentage,
			TotalVolume:    r.TotalVolume,
		})
	}
	return out, nil
}

// CoinDetail returns the full detail object for a single coin.
func (c *Client) CoinDetail(ctx context.Context, id string) (Coin, error) {
	u := fmt.Sprintf("%s/coins/%s?localization=false&tickers=false&market_data=true&community_data=false&developer_data=false",
		c.cfg.BaseURL, url.PathEscape(id))
	body, err := c.get(ctx, u)
	if err != nil {
		return Coin{}, err
	}
	var d apiCoinDetail
	if err := json.Unmarshal(body, &d); err != nil {
		return Coin{}, fmt.Errorf("decode coin: %w", err)
	}
	desc := d.Description["en"]
	if len(desc) > 300 {
		desc = desc[:300]
	}
	return Coin{
		ID:          d.ID,
		Symbol:      d.Symbol,
		Name:        d.Name,
		Description: desc,
		Price:       fmt.Sprintf("%.2f", d.MarketData.CurrentPrice["usd"]),
		MarketCap:   fmt.Sprintf("%.0f", d.MarketData.MarketCap["usd"]),
		Volume24h:   fmt.Sprintf("%.0f", d.MarketData.TotalVolume["usd"]),
		High24h:     fmt.Sprintf("%.2f", d.MarketData.High24h["usd"]),
		Low24h:      fmt.Sprintf("%.2f", d.MarketData.Low24h["usd"]),
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
			Name:          w.Item.Name,
			Symbol:        w.Item.Symbol,
			MarketCapRank: w.Item.MarketCapRank,
			PriceBTC:      w.Item.PriceBTC,
		})
	}
	return out, nil
}

// Coins returns the full list of coin IDs from CoinGecko.
// limit <= 0 means return all; otherwise return the first limit entries.
func (c *Client) Coins(ctx context.Context, limit int) ([]CoinInfo, error) {
	u := c.cfg.BaseURL + "/coins/list?include_platform=false"
	body, err := c.get(ctx, u)
	if err != nil {
		return nil, err
	}
	var raw []apiCoinList
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("decode coins list: %w", err)
	}
	if limit > 0 && limit < len(raw) {
		raw = raw[:limit]
	}
	out := make([]CoinInfo, 0, len(raw))
	for _, r := range raw {
		out = append(out, CoinInfo{
			ID:     r.ID,
			Symbol: r.Symbol,
			Name:   r.Name,
		})
	}
	return out, nil
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
	d := time.Duration(attempt) * 500 * time.Millisecond
	if d > 5*time.Second {
		d = 5 * time.Second
	}
	return d
}
