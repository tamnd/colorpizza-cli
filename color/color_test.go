package color_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	colorpkg "github.com/tamnd/colorpizza-cli/color"
)

const fakeColorJSON = `{"colors":[{"name":"Red","hex":"#FF0000","rgb":{"r":255,"g":0,"b":0},"hsl":{"h":0,"s":100,"l":50},"luminance":0.2126,"requestedHex":"#ff0000","distance":0}],"paletteTitle":"My Palette"}`

func newTestClient(ts *httptest.Server) *colorpkg.Client {
	cfg := colorpkg.DefaultConfig()
	cfg.BaseURL = ts.URL
	cfg.Rate = 0
	return colorpkg.NewClient(cfg)
}

func TestNamesSendsUserAgent(t *testing.T) {
	var gotUA string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUA = r.Header.Get("User-Agent")
		_, _ = fmt.Fprint(w, fakeColorJSON)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	_, err := c.Names(context.Background(), []string{"ff0000"})
	if err != nil {
		t.Fatal(err)
	}
	if gotUA == "" {
		t.Error("User-Agent not sent")
	}
}

func TestNamesParsesColor(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, fakeColorJSON)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	colors, err := c.Names(context.Background(), []string{"ff0000"})
	if err != nil {
		t.Fatal(err)
	}
	if len(colors) != 1 {
		t.Fatalf("len(colors) = %d, want 1", len(colors))
	}
	if colors[0].Name != "Red" {
		t.Errorf("Name = %q, want Red", colors[0].Name)
	}
	if colors[0].RGB.R != 255 {
		t.Errorf("RGB.R = %d, want 255", colors[0].RGB.R)
	}
}

func TestListReturnsColors(t *testing.T) {
	allColors := make([]map[string]any, 10)
	for i := range allColors {
		allColors[i] = map[string]any{
			"name":      fmt.Sprintf("Color%d", i),
			"hex":       fmt.Sprintf("#%06X", i*1000),
			"rgb":       map[string]int{"r": i, "g": 0, "b": 0},
			"hsl":       map[string]float64{"h": 0, "s": 0, "l": float64(i)},
			"luminance": 0.1,
		}
	}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"colors": allColors})
	}))
	defer ts.Close()

	c := newTestClient(ts)
	colors, err := c.List(context.Background(), 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(colors) != 5 {
		t.Errorf("len(colors) = %d, want 5 (count limit)", len(colors))
	}
}

func TestListNoCountReturnsAll(t *testing.T) {
	allColors := make([]map[string]any, 3)
	for i := range allColors {
		allColors[i] = map[string]any{
			"name": fmt.Sprintf("C%d", i), "hex": "#000000",
			"rgb": map[string]int{"r": 0, "g": 0, "b": 0},
			"hsl": map[string]float64{"h": 0, "s": 0, "l": 0}, "luminance": 0.0,
		}
	}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"colors": allColors})
	}))
	defer ts.Close()

	c := newTestClient(ts)
	colors, err := c.List(context.Background(), 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(colors) != 3 {
		t.Errorf("len(colors) = %d, want 3", len(colors))
	}
}

func TestPaletteReturnsPaletteTitle(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, fakeColorJSON)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	colors, title, err := c.Palette(context.Background(), []string{"ff0000"})
	if err != nil {
		t.Fatal(err)
	}
	if len(colors) != 1 {
		t.Errorf("len(colors) = %d, want 1", len(colors))
	}
	if title != "My Palette" {
		t.Errorf("paletteTitle = %q, want My Palette", title)
	}
}

func TestNamesRetriesOn503(t *testing.T) {
	var hits int
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		if hits < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_, _ = fmt.Fprint(w, fakeColorJSON)
	}))
	defer ts.Close()

	cfg := colorpkg.DefaultConfig()
	cfg.BaseURL = ts.URL
	cfg.Rate = 0
	cfg.Retries = 3
	c := colorpkg.NewClient(cfg)

	_, err := c.Names(context.Background(), []string{"ff0000"})
	if err != nil {
		t.Fatal(err)
	}
	if hits != 3 {
		t.Errorf("server saw %d hits, want 3", hits)
	}
}
