package coingecko

import (
	"context"
	"strings"

	"github.com/tamnd/any-cli/kit"
	"github.com/tamnd/any-cli/kit/errs"
)

// domain.go exposes coingecko as a kit Domain driver.
//
// A multi-domain host (ant) enables it with a single blank import:
//
//	import _ "github.com/tamnd/coingecko-cli/coingecko"
func init() { kit.Register(Domain{}) }

// Domain is the coingecko driver.
type Domain struct{}

// Info describes the scheme, the hostnames a pasted link is matched against,
// and the identity reused for the binary's help and version.
func (Domain) Info() kit.DomainInfo {
	return kit.DomainInfo{
		Scheme: "coingecko",
		Hosts:  []string{"api.coingecko.com", "www.coingecko.com"},
		Identity: kit.Identity{
			Binary: "coingecko",
			Short:  "CoinGecko cryptocurrency market data",
			Long: `coingecko fetches real-time coin prices and trending data from the public
CoinGecko API. No API key required.`,
			Site: Host,
			Repo: "https://github.com/tamnd/coingecko-cli",
		},
	}
}

// Register installs the client factory and every operation onto app.
func (Domain) Register(app *kit.App) {
	app.SetClient(newClient)

	kit.Handle(app, kit.OpMeta{
		Name:    "price",
		Group:   "read",
		List:    true,
		Summary: "Get price for one or more coins",
	}, priceOp)

	kit.Handle(app, kit.OpMeta{
		Name:    "markets",
		Group:   "read",
		List:    true,
		Summary: "List coins by market capitalisation",
	}, marketsOp)

	kit.Handle(app, kit.OpMeta{
		Name:    "coin",
		Group:   "read",
		Single:  true,
		Summary: "Get full detail for a specific coin",
	}, coinOp)

	kit.Handle(app, kit.OpMeta{
		Name:    "trending",
		Group:   "read",
		List:    true,
		Summary: "List currently trending coins (most searched in 24h)",
	}, trendingOp)

	kit.Handle(app, kit.OpMeta{
		Name:    "search",
		Group:   "read",
		List:    true,
		Summary: "Search coins by name or symbol",
	}, searchOp)
}

// newClient builds the client from host-resolved config.
func newClient(_ context.Context, cfg kit.Config) (any, error) {
	c := DefaultConfig()
	if cfg.UserAgent != "" {
		c.UserAgent = cfg.UserAgent
	}
	if cfg.Rate > 0 {
		c.Rate = cfg.Rate
	}
	if cfg.Retries > 0 {
		c.Retries = cfg.Retries
	}
	if cfg.Timeout > 0 {
		c.Timeout = cfg.Timeout
	}
	return NewClient(c), nil
}

// --- inputs ---

type priceInput struct {
	IDs        string  `kit:"arg" help:"comma-separated coin IDs e.g. bitcoin,ethereum"`
	Currencies string  `kit:"flag" help:"comma-separated currencies" default:"usd"`
	Client     *Client `kit:"inject"`
}

type marketsInput struct {
	IDs      string  `kit:"arg" help:"comma-separated coin IDs (or leave empty for top coins)"`
	Currency string  `kit:"flag" help:"vs currency" default:"usd"`
	Limit    int     `kit:"flag,inherit" help:"max results" default:"10"`
	Client   *Client `kit:"inject"`
}

type coinInput struct {
	ID     string  `kit:"arg" help:"coin ID e.g. bitcoin"`
	Client *Client `kit:"inject"`
}

type trendingInput struct {
	Client *Client `kit:"inject"`
}

type searchInput struct {
	Query  string  `kit:"arg" help:"search query"`
	Client *Client `kit:"inject"`
}

// --- handlers ---

func priceOp(ctx context.Context, in priceInput, emit func(Price) error) error {
	currencies := in.Currencies
	if currencies == "" {
		currencies = "usd"
	}
	prices, err := in.Client.Price(ctx, in.IDs, currencies)
	if err != nil {
		return err
	}
	for _, p := range prices {
		if err := emit(p); err != nil {
			return err
		}
	}
	return nil
}

func marketsOp(ctx context.Context, in marketsInput, emit func(CoinMarket) error) error {
	limit := in.Limit
	if limit <= 0 {
		limit = 10
	}
	currency := in.Currency
	if currency == "" {
		currency = "usd"
	}
	items, err := in.Client.Markets(ctx, in.IDs, currency, limit)
	if err != nil {
		return err
	}
	for _, item := range items {
		if err := emit(item); err != nil {
			return err
		}
	}
	return nil
}

func coinOp(ctx context.Context, in coinInput, emit func(CoinDetail) error) error {
	d, err := in.Client.Coin(ctx, in.ID)
	if err != nil {
		return err
	}
	return emit(d)
}

func trendingOp(ctx context.Context, in trendingInput, emit func(TrendingCoin) error) error {
	items, err := in.Client.Trending(ctx)
	if err != nil {
		return err
	}
	for _, item := range items {
		if err := emit(item); err != nil {
			return err
		}
	}
	return nil
}

func searchOp(ctx context.Context, in searchInput, emit func(SearchResult) error) error {
	results, err := in.Client.Search(ctx, in.Query)
	if err != nil {
		return err
	}
	for _, r := range results {
		if err := emit(r); err != nil {
			return err
		}
	}
	return nil
}

// --- Resolver: pure string functions, no network ---

// Classify turns an input into the canonical (type, id).
// - known coin name/symbol (e.g. "bitcoin", "btc") → ("coin", input)
// - "," in string → ("ids", input)
// - otherwise → ("query", input)
func (Domain) Classify(input string) (uriType, id string, err error) {
	if input == "" {
		return "", "", errs.Usage("empty coingecko reference")
	}
	if strings.Contains(input, ",") {
		return "ids", input, nil
	}
	lower := strings.ToLower(input)
	if isKnownCoin(lower) {
		return "coin", lower, nil
	}
	return "query", input, nil
}

// isKnownCoin returns true for well-known coin IDs and symbols.
func isKnownCoin(s string) bool {
	known := map[string]bool{
		// IDs
		"bitcoin": true, "ethereum": true, "solana": true, "cardano": true,
		"ripple": true, "dogecoin": true, "polkadot": true, "avalanche-2": true,
		"chainlink": true, "litecoin": true, "uniswap": true, "stellar": true,
		"tron": true, "monero": true, "cosmos": true, "near": true,
		// Symbols
		"btc": true, "eth": true, "sol": true, "ada": true,
		"xrp": true, "doge": true, "dot": true, "avax": true,
		"link": true, "ltc": true, "uni": true, "xlm": true,
		"trx": true, "xmr": true, "atom": true,
	}
	return known[s]
}

// Locate returns the live https URL for a (type, id).
func (Domain) Locate(uriType, id string) (string, error) {
	switch uriType {
	case "coin", "ids":
		// For a comma-separated list, use the first ID for the URL.
		first := strings.SplitN(id, ",", 2)[0]
		return "https://www.coingecko.com/en/coins/" + strings.TrimSpace(first), nil
	case "query":
		return "https://www.coingecko.com/en/search_results?query=" + id, nil
	default:
		return "", errs.Usage("coingecko has no resource type %q", uriType)
	}
}
