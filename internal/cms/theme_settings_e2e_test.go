package cms

import (
	"context"
	"encoding/json"
	"io"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
)

// fakeRoundtripAPI is a settingsAPI / settingsReader implementation backed by
// an in-memory map. Used by the round-trip test to drive the full handler +
// render-context pipeline without a database or real server.
type fakeRoundtripAPI struct {
	store map[string]string
}

func (f *fakeRoundtripAPI) GetSetting(_ context.Context, key string) (string, error) {
	return f.store[key], nil
}
func (f *fakeRoundtripAPI) SetSetting(_ context.Context, key, value string) error {
	f.store[key] = value
	return nil
}
func (f *fakeRoundtripAPI) GetSettings(_ context.Context, prefix string) (map[string]string, error) {
	out := map[string]string{}
	for k, v := range f.store {
		if strings.HasPrefix(k, prefix) {
			out[strings.TrimPrefix(k, prefix)] = v
		}
	}
	return out, nil
}
func (f *fakeRoundtripAPI) GetSettingsLoc(ctx context.Context, prefix, _ string) (map[string]string, error) {
	return f.GetSettings(ctx, prefix)
}
func (f *fakeRoundtripAPI) SetSettingLoc(ctx context.Context, key, _, value string) error {
	return f.SetSetting(ctx, key, value)
}

// TestThemeSettings_LoadSaveRender_RoundTrip wires loader → registry → admin
// HTTP → render-context together to prove the layers integrate. No DB, no
// real server, no real theme files — just a temp dir.
func TestThemeSettings_LoadSaveRender_RoundTrip(t *testing.T) {
	// 1. Lay out a fake theme directory.
	themeDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(themeDir, "settings"), 0o755); err != nil {
		t.Fatal(err)
	}
	pageJSON := `{
      "name": "General",
      "fields": [
        {"key": "tagline", "label": "Tagline", "type": "text", "default": "Welcome"},
        {"key": "footer_columns", "label": "Cols", "type": "number", "default": 3},
        {"key": "show_attribution", "label": "Attribution", "type": "toggle", "default": true}
      ]
    }`
	if err := os.WriteFile(filepath.Join(themeDir, "settings", "general.json"), []byte(pageJSON), 0o644); err != nil {
		t.Fatal(err)
	}
	manifest := ThemeManifest{
		SettingsPages: []ThemeSettingsPageDef{
			{Slug: "general", Name: "General", File: "settings/general.json"},
		},
	}

	// 2. Drive the loader and registry as ThemeLoader.LoadTheme would.
	pages := LoadSettingsPages(themeDir, manifest)
	if len(pages) != 1 || len(pages[0].Fields) != 3 {
		t.Fatalf("loader: unexpected pages: %#v", pages)
	}
	registry := NewThemeSettingsRegistry()
	registry.SetActive("default", pages)

	api := &fakeRoundtripAPI{store: map[string]string{}}

	// 3. Build a Fiber app with just the handler routes (no auth in tests).
	h := NewThemeSettingsHandler(registry, api, nil, nil, nil)
	app := fiber.New()
	app.Get("/theme-settings", h.List)
	app.Get("/theme-settings/:page", h.Get)
	app.Put("/theme-settings/:page", h.Save)

	// 4. GET list → expect one page.
	res, err := app.Test(httptest.NewRequest("GET", "/theme-settings", nil))
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != 200 {
		t.Fatalf("list status %d", res.StatusCode)
	}

	// 5. GET page → expect 3 fields, all values returning defaults.
	res, err = app.Test(httptest.NewRequest("GET", "/theme-settings/general", nil))
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(res.Body)
	var initial struct {
		Data struct {
			Page   pageDTO             `json:"page"`
			Values map[string]valueDTO `json:"values"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &initial); err != nil {
		t.Fatal(err)
	}
	if len(initial.Data.Page.Fields) != 3 {
		t.Fatalf("expected 3 fields, got %d", len(initial.Data.Page.Fields))
	}
	// No values stored yet — coercion of empty raw returns nil + compatible.
	if initial.Data.Values["tagline"].Value != nil {
		t.Fatalf("expected nil tagline before save, got %#v", initial.Data.Values["tagline"].Value)
	}

	// 6. PUT new values.
	putBody := `{"values":{"tagline":"Hello","footer_columns":4,"show_attribution":false}}`
	req := httptest.NewRequest("PUT", "/theme-settings/general", strings.NewReader(putBody))
	req.Header.Set("Content-Type", "application/json")
	res, err = app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != 200 {
		t.Fatalf("save status %d", res.StatusCode)
	}

	// 7. Confirm storage layout — keys are theme:default:general:<field>.
	if api.store["theme:default:general:tagline"] != "Hello" {
		t.Fatalf("tagline stored as %q", api.store["theme:default:general:tagline"])
	}
	if api.store["theme:default:general:footer_columns"] != "4" {
		t.Fatalf("number stored as %q", api.store["theme:default:general:footer_columns"])
	}
	if api.store["theme:default:general:show_attribution"] != "false" {
		t.Fatalf("toggle stored as %q", api.store["theme:default:general:show_attribution"])
	}

	// 8. GET page again — values now match what we saved, all compatible.
	res, _ = app.Test(httptest.NewRequest("GET", "/theme-settings/general", nil))
	body, _ = io.ReadAll(res.Body)
	var post struct {
		Data struct {
			Values map[string]valueDTO `json:"values"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &post); err != nil {
		t.Fatal(err)
	}
	if post.Data.Values["tagline"].Value != "Hello" {
		t.Fatalf("tagline=%v", post.Data.Values["tagline"].Value)
	}
	if v, _ := post.Data.Values["footer_columns"].Value.(float64); v != 4 {
		t.Fatalf("footer_columns=%v", post.Data.Values["footer_columns"].Value)
	}
	if post.Data.Values["show_attribution"].Value != false {
		t.Fatalf("show_attribution=%v", post.Data.Values["show_attribution"].Value)
	}
	for k, v := range post.Data.Values {
		if !v.Compatible {
			t.Fatalf("post-save value for %q should be compatible, got compatible=false", k)
		}
	}

	// 9. BuildThemeSettingsContext returns the same coerced values.
	ts, err := BuildThemeSettingsContext(context.Background(), registry, api)
	if err != nil {
		t.Fatal(err)
	}
	page := ts["general"]
	if page == nil {
		t.Fatal("expected general page in render context")
	}
	if page["tagline"] != "Hello" {
		t.Fatalf("render tagline=%v", page["tagline"])
	}
	if v, _ := page["footer_columns"].(float64); v != 4 {
		t.Fatalf("render footer_columns=%v", page["footer_columns"])
	}
	if page["show_attribution"] != false {
		t.Fatalf("render show_attribution=%v", page["show_attribution"])
	}

	// 10. Schema-change scenario: swap footer_columns into a toggle. The
	// previously stored "4" is not a valid toggle representation, so we
	// expect Compatible=false, the original raw preserved, and Value
	// falling back to the declared default (true).
	schemaChange := []ThemeSettingsPage{
		{
			Slug: "general",
			Name: "General",
			Fields: []ThemeSettingsField{
				{Key: "footer_columns", Label: "Cols", Type: "toggle", Default: json.RawMessage(`true`)},
			},
		},
	}
	registry.SetActive("default", schemaChange)

	res, _ = app.Test(httptest.NewRequest("GET", "/theme-settings/general", nil))
	body, _ = io.ReadAll(res.Body)
	var changed struct {
		Data struct {
			Values map[string]valueDTO `json:"values"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &changed); err != nil {
		t.Fatal(err)
	}
	fc := changed.Data.Values["footer_columns"]
	if fc.Compatible {
		t.Fatalf("expected incompatible after schema change, got compatible=true raw=%q", fc.Raw)
	}
	if fc.Raw != "4" {
		t.Fatalf("expected raw to retain old stored value '4', got %q", fc.Raw)
	}
	// Default value comes through as bool(true).
	if fc.Value != true {
		t.Fatalf("expected default-fallback true, got %#v", fc.Value)
	}
}
