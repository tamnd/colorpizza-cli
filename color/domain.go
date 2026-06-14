package color

import (
	"context"
	"strings"

	"github.com/tamnd/any-cli/kit"
	"github.com/tamnd/any-cli/kit/errs"
)

// domain.go exposes color pizza as a kit Domain driver.
//
// A multi-domain host (ant) enables it with a single blank import:
//
//	import _ "github.com/tamnd/colorpizza-cli/color"
func init() { kit.Register(Domain{}) }

// Domain is the colorpizza driver.
type Domain struct{}

// Info describes the scheme, the hostnames a pasted link is matched against,
// and the identity reused for the binary's help and version.
func (Domain) Info() kit.DomainInfo {
	return kit.DomainInfo{
		Scheme: "color",
		Hosts:  []string{Host},
		Identity: kit.Identity{
			Binary: "color",
			Short:  "Color name lookup and palette info from api.color.pizza",
			Long: `color fetches color names and palette information from the public api.color.pizza API.
No login or API key required.`,
			Site: Host,
			Repo: "https://github.com/tamnd/colorpizza-cli",
		},
	}
}

// Register installs the client factory and every operation onto app.
func (Domain) Register(app *kit.App) {
	app.SetClient(newClient)

	// name: get color name(s) for hex value(s)
	kit.Handle(app, kit.OpMeta{
		Name:    "name",
		Group:   "read",
		List:    true,
		Summary: "Get color names for hex values (e.g. ff0000,00ff00)",
	}, nameOp)

	// list: list all named colors
	kit.Handle(app, kit.OpMeta{
		Name:    "list",
		Group:   "read",
		List:    true,
		Summary: "List all named colors",
	}, listOp)

	// palette: get full palette info for hex values
	kit.Handle(app, kit.OpMeta{
		Name:    "palette",
		Group:   "read",
		List:    true,
		Summary: "Get full palette info for hex values",
	}, paletteOp)
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

type nameInput struct {
	Hex    string  `kit:"flag" help:"comma-separated hex values without # (e.g. ff0000,00ff00)"`
	Client *Client `kit:"inject"`
}

type listInput struct {
	Count  int     `kit:"flag,inherit" help:"max number of colors to return"`
	Client *Client `kit:"inject"`
}

type paletteInput struct {
	Hex    string  `kit:"flag" help:"comma-separated hex values without # (e.g. ff0000,00ff00,0000ff)"`
	Client *Client `kit:"inject"`
}

// --- handlers ---

func nameOp(ctx context.Context, in nameInput, emit func(Color) error) error {
	hexValues := parseHex(in.Hex)
	if len(hexValues) == 0 {
		return errs.Usage("--hex is required (e.g. --hex ff0000,00ff00)")
	}
	colors, err := in.Client.Names(ctx, hexValues)
	if err != nil {
		return err
	}
	for _, col := range colors {
		if err := emit(col); err != nil {
			return err
		}
	}
	return nil
}

func listOp(ctx context.Context, in listInput, emit func(Color) error) error {
	colors, err := in.Client.List(ctx, in.Count)
	if err != nil {
		return err
	}
	for _, col := range colors {
		if err := emit(col); err != nil {
			return err
		}
	}
	return nil
}

func paletteOp(ctx context.Context, in paletteInput, emit func(Color) error) error {
	hexValues := parseHex(in.Hex)
	if len(hexValues) == 0 {
		return errs.Usage("--hex is required (e.g. --hex ff0000,00ff00,0000ff)")
	}
	colors, _, err := in.Client.Palette(ctx, hexValues)
	if err != nil {
		return err
	}
	for _, col := range colors {
		if err := emit(col); err != nil {
			return err
		}
	}
	return nil
}

// Classify turns an input into the canonical (type, id).
func (Domain) Classify(input string) (uriType, id string, err error) {
	if input == "" {
		return "", "", errs.Usage("empty color reference")
	}
	// strip leading # if present
	id = strings.TrimPrefix(input, "#")
	return "color", id, nil
}

// Locate returns the live URL for a (type, id).
func (Domain) Locate(uriType, id string) (string, error) {
	switch uriType {
	case "color":
		return "https://api.color.pizza/v1/?values=" + id, nil
	default:
		return "", errs.Usage("colorpizza has no resource type %q", uriType)
	}
}

// parseHex splits a comma-separated hex string into individual values,
// stripping any # prefixes.
func parseHex(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		p = strings.TrimPrefix(p, "#")
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
