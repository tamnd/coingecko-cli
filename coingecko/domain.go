package coingecko

import (
	"context"
	"strings"

	"github.com/tamnd/any-cli/kit"
	"github.com/tamnd/any-cli/kit/errs"
)

// domain.go exposes CoinGecko as a kit Domain: a driver that a multi-domain
// host (ant) enables with a single blank import,
//
//	import _ "github.com/tamnd/coingecko-cli/coingecko"
//
// exactly as a database/sql program enables a driver with `import _
// "github.com/lib/pq"`. The init below registers it; the host then dereferences
// coingecko:// URIs by routing to the operations Register installs. The same
// Domain also builds the standalone coingecko binary (see cli.NewApp), so the
// binary and a host share one source of truth.
func init() { kit.Register(Domain{}) }

// Domain is the CoinGecko driver. It carries no state; the per-run client is
// built by the factory Register hands kit.
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
			Long: `coingecko fetches real-time coin prices and market data from the public
CoinGecko API. No API key required for the free tier (30 req/min).`,
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
		Summary: "Get current price for one or more coins",
		Args:    []kit.Arg{{Name: "ids", Help: "coin IDs (e.g. bitcoin ethereum)", Variadic: true}},
	}, priceOp)

	kit.Handle(app, kit.OpMeta{
		Name:    "trending",
		Group:   "read",
		List:    true,
		Summary: "List currently trending coins (most searched in 24h)",
	}, trendingOp)

	kit.Handle(app, kit.OpMeta{
		Name:    "markets",
		Group:   "read",
		List:    true,
		Summary: "List coins ranked by market capitalisation",
	}, marketsOp)

	kit.Handle(app, kit.OpMeta{
		Name:    "coins",
		Group:   "read",
		List:    true,
		Summary: "List all coin IDs (useful for finding the right ID string)",
	}, coinsOp)

	kit.Handle(app, kit.OpMeta{
		Name:    "coin",
		Group:   "read",
		Single:  true,
		Summary: "Get full detail for a specific coin",
		Args:    []kit.Arg{{Name: "id", Help: "coin ID e.g. bitcoin"}},
	}, coinOp)
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
	IDs        []string `kit:"arg,variadic" help:"coin IDs e.g. bitcoin ethereum"`
	Currency   string   `kit:"flag" help:"vs currency (single)" default:"usd"`
	Currencies string   `kit:"flag" help:"comma-separated vs currencies e.g. usd,eur,btc"`
	Client     *Client  `kit:"inject"`
}

type trendingInput struct {
	Client *Client `kit:"inject"`
}

type marketsInput struct {
	Currency string  `kit:"flag" help:"vs currency" default:"usd"`
	Limit    int     `kit:"flag,inherit" help:"max results" default:"25"`
	Page     int     `kit:"flag" help:"page number (1-indexed)" default:"1"`
	Client   *Client `kit:"inject"`
}

type coinsInput struct {
	Limit  int     `kit:"flag,inherit" help:"max results (0 = all)" default:"100"`
	Client *Client `kit:"inject"`
}

type coinInput struct {
	ID     string  `kit:"arg" help:"coin ID e.g. bitcoin"`
	Client *Client `kit:"inject"`
}

// --- handlers ---

func priceOp(ctx context.Context, in priceInput, emit func(Price) error) error {
	var currencies []string
	if in.Currencies != "" {
		for _, c := range strings.Split(in.Currencies, ",") {
			c = strings.TrimSpace(c)
			if c != "" {
				currencies = append(currencies, c)
			}
		}
	}
	if len(currencies) == 0 {
		cur := in.Currency
		if cur == "" {
			cur = "usd"
		}
		currencies = []string{cur}
	}
	prices, err := in.Client.Price(ctx, in.IDs, currencies...)
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

func marketsOp(ctx context.Context, in marketsInput, emit func(MarketCoin) error) error {
	currency := in.Currency
	if currency == "" {
		currency = "usd"
	}
	limit := in.Limit
	if limit <= 0 {
		limit = 25
	}
	page := in.Page
	if page <= 0 {
		page = 1
	}
	items, err := in.Client.Markets(ctx, currency, limit, page)
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

func coinsOp(ctx context.Context, in coinsInput, emit func(CoinInfo) error) error {
	limit := in.Limit
	if limit < 0 {
		limit = 0
	}
	items, err := in.Client.Coins(ctx, limit)
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

func coinOp(ctx context.Context, in coinInput, emit func(Coin) error) error {
	d, err := in.Client.CoinDetail(ctx, in.ID)
	if err != nil {
		return err
	}
	return emit(d)
}

// --- Resolver: pure string functions, no network ---

// Classify turns an input into the canonical (type, id).
func (Domain) Classify(input string) (uriType, id string, err error) {
	if input == "" {
		return "", "", errs.Usage("empty coingecko reference")
	}
	if strings.Contains(input, ",") {
		return "ids", input, nil
	}
	lower := strings.ToLower(strings.TrimSpace(input))
	return "coin", lower, nil
}

// Locate returns the live https URL for a (type, id).
func (Domain) Locate(uriType, id string) (string, error) {
	switch uriType {
	case "coin":
		return "https://www.coingecko.com/en/coins/" + id, nil
	case "ids":
		first := strings.SplitN(id, ",", 2)[0]
		return "https://www.coingecko.com/en/coins/" + strings.TrimSpace(first), nil
	default:
		return "", errs.Usage("coingecko has no resource type %q", uriType)
	}
}
