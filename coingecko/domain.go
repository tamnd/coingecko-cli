package coingecko

import (
	"context"
	"strings"
	"time"

	"github.com/tamnd/any-cli/kit"
	"github.com/tamnd/any-cli/kit/errs"
)

// domain.go exposes coingecko as a kit Domain driver.
//
// A multi-domain host (ant) enables it with a single blank import:
//
//	import _ "github.com/tamnd/coingecko-cli/coingecko"
//
// The same Domain also builds the standalone coingecko binary (see cli.NewApp).
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

	// markets: list top coins by market cap
	kit.Handle(app, kit.OpMeta{
		Name:    "markets",
		Group:   "read",
		List:    true,
		Summary: "List top coins by market capitalisation",
	}, marketsOp)

	// trending: list currently trending coins
	kit.Handle(app, kit.OpMeta{
		Name:    "trending",
		Group:   "read",
		List:    true,
		Summary: "List currently trending coins (most searched in 24h)",
	}, trendingOp)

	// price: get price for one or more coins
	kit.Handle(app, kit.OpMeta{
		Name:    "price",
		Group:   "read",
		List:    false,
		Summary: "Get price for one or more coins",
	}, priceOp)

	// coin: full detail for a single coin
	kit.Handle(app, kit.OpMeta{
		Name:    "coin",
		Group:   "read",
		List:    false,
		Summary: "Get full detail for a specific coin",
	}, coinOp)

	// search: search coins by name/symbol
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

type marketsInput struct {
	Limit    int           `kit:"flag,inherit" help:"max coins to list"`
	Currency string        `kit:"flag" help:"target currency (e.g. usd, eur)"`
	Delay    time.Duration `kit:"flag,inherit" help:"minimum spacing between requests"`
	Client   *Client       `kit:"inject"`
}

type trendingInput struct {
	Client *Client `kit:"inject"`
}

type priceInput struct {
	IDs      []string `kit:"args" help:"one or more coin IDs"`
	Currency string   `kit:"flag" help:"comma-separated currencies (e.g. usd,eur)"`
	Client   *Client  `kit:"inject"`
}

type coinInput struct {
	ID     string  `kit:"arg" help:"coin ID (e.g. bitcoin)"`
	Client *Client `kit:"inject"`
}

type searchInput struct {
	Query  string  `kit:"arg" help:"search query"`
	Client *Client `kit:"inject"`
}

// --- handlers ---

func marketsOp(ctx context.Context, in marketsInput, emit func(Coin) error) error {
	limit := in.Limit
	if limit <= 0 {
		limit = 20
	}
	currency := in.Currency
	if currency == "" {
		currency = "usd"
	}
	items, err := in.Client.MarketsInCurrency(ctx, currency, limit)
	if err != nil {
		return mapErr(err)
	}
	for _, item := range items {
		if err := emit(item); err != nil {
			return err
		}
	}
	return nil
}

func trendingOp(ctx context.Context, in trendingInput, emit func(TrendingCoin) error) error {
	items, err := in.Client.Trending(ctx)
	if err != nil {
		return mapErr(err)
	}
	for _, item := range items {
		if err := emit(item); err != nil {
			return err
		}
	}
	return nil
}

func priceOp(ctx context.Context, in priceInput, emit func(Price) error) error {
	currencies := []string{"usd"}
	if in.Currency != "" {
		currencies = strings.Split(in.Currency, ",")
	}
	p, err := in.Client.Price(ctx, in.IDs, currencies)
	if err != nil {
		return mapErr(err)
	}
	return emit(p)
}

func coinOp(ctx context.Context, in coinInput, emit func(CoinDetail) error) error {
	d, err := in.Client.CoinInfo(ctx, in.ID)
	if err != nil {
		return mapErr(err)
	}
	return emit(d)
}

func searchOp(ctx context.Context, in searchInput, emit func(SearchResult) error) error {
	results, err := in.Client.Search(ctx, in.Query)
	if err != nil {
		return mapErr(err)
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
func (Domain) Classify(input string) (uriType, id string, err error) {
	if input == "" {
		return "", "", errs.Usage("empty coingecko reference")
	}
	return "coin", input, nil
}

// Locate returns the live https URL for a (type, id).
func (Domain) Locate(uriType, id string) (string, error) {
	switch uriType {
	case "coin":
		return "https://www.coingecko.com/en/coins/" + id, nil
	default:
		return "", errs.Usage("coingecko has no resource type %q", uriType)
	}
}

// mapErr converts a library error into the kit error kind.
func mapErr(err error) error {
	return err
}
