// Package color is the library behind the color command line:
// the HTTP client, request shaping, and the typed data models for
// the Color Pizza API (api.color.pizza).
package color

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Host is the site this client talks to.
const Host = "api.color.pizza"

// Config holds all tunable parameters for the Client.
type Config struct {
	BaseURL   string
	UserAgent string
	Rate      time.Duration
	Timeout   time.Duration
	Retries   int
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		BaseURL:   "https://api.color.pizza",
		UserAgent: "color-cli/0.1.0 (github.com/tamnd/colorpizza-cli)",
		Rate:      200 * time.Millisecond,
		Timeout:   30 * time.Second,
		Retries:   3,
	}
}

// Client talks to api.color.pizza over HTTP.
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

// Names fetches color name(s) for the given hex values (without # prefix).
// Example: hexValues = []string{"ff0000", "00ff00"}
func (c *Client) Names(ctx context.Context, hexValues []string) ([]Color, error) {
	vals := strings.Join(hexValues, ",")
	u := fmt.Sprintf("%s/v1/?values=%s", c.cfg.BaseURL, vals)
	body, err := c.get(ctx, u)
	if err != nil {
		return nil, err
	}
	var resp colorResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("decode colors: %w", err)
	}
	items := make([]Color, 0, len(resp.Colors))
	for i, col := range resp.Colors {
		col.Rank = i + 1
		items = append(items, col)
	}
	return items, nil
}

// List fetches all named colors (or up to count if count > 0).
func (c *Client) List(ctx context.Context, count int) ([]Color, error) {
	u := c.cfg.BaseURL + "/v1/"
	body, err := c.get(ctx, u)
	if err != nil {
		return nil, err
	}
	var resp colorResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("decode colors: %w", err)
	}
	items := make([]Color, 0, len(resp.Colors))
	for i, col := range resp.Colors {
		col.Rank = i + 1
		items = append(items, col)
	}
	if count > 0 && count < len(items) {
		items = items[:count]
	}
	return items, nil
}

// Palette fetches full palette info for the given hex values (without # prefix).
func (c *Client) Palette(ctx context.Context, hexValues []string) ([]Color, string, error) {
	vals := strings.Join(hexValues, ",")
	u := fmt.Sprintf("%s/v1/?values=%s", c.cfg.BaseURL, vals)
	body, err := c.get(ctx, u)
	if err != nil {
		return nil, "", err
	}
	var resp colorResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, "", fmt.Errorf("decode palette: %w", err)
	}
	items := make([]Color, 0, len(resp.Colors))
	for i, col := range resp.Colors {
		col.Rank = i + 1
		items = append(items, col)
	}
	return items, resp.PaletteTitle, nil
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

// RGB holds the red, green, blue components of a color.
type RGB struct {
	R int `json:"r"`
	G int `json:"g"`
	B int `json:"b"`
}

// HSL holds the hue, saturation, lightness components of a color.
type HSL struct {
	H float64 `json:"h"`
	S float64 `json:"s"`
	L float64 `json:"l"`
}

// Color is a single named color entry from the API.
type Color struct {
	Rank         int     `json:"rank,omitempty"`
	Name         string  `json:"name"`
	Hex          string  `json:"hex"`
	RGB          RGB     `json:"rgb"`
	HSL          HSL     `json:"hsl"`
	Luminance    float64 `json:"luminance"`
	Distance     float64 `json:"distance,omitempty"`
	RequestedHex string  `json:"requestedHex,omitempty"`
}

// colorResponse is the raw JSON from the API.
type colorResponse struct {
	Colors       []Color `json:"colors"`
	PaletteTitle string  `json:"paletteTitle,omitempty"`
}
