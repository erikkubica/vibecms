package cms

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
)

// fakeCoreAPI is a minimal in-memory implementation of the local
// settingsAPI interface. It mirrors the CoreAPI semantics for the two
// methods this handler uses.
type fakeCoreAPI struct {
	store map[string]string
}

func newFakeAPI() *fakeCoreAPI { return &fakeCoreAPI{store: map[string]string{}} }

func (f *fakeCoreAPI) GetSetting(_ context.Context, key string) (string, error) {
	return f.store[key], nil
}

func (f *fakeCoreAPI) SetSetting(_ context.Context, key, value string) error {
	f.store[key] = value
	return nil
}

func (f *fakeCoreAPI) GetSettings(_ context.Context, prefix string) (map[string]string, error) {
	out := map[string]string{}
	for k, v := range f.store {
		if len(k) >= len(prefix) && k[:len(prefix)] == prefix {
			out[k[len(prefix):]] = v
		}
	}
	return out, nil
}

// Locale-aware fakes: tests don't exercise per-locale storage yet, so the
// shared map is used for both. Sufficient for the existing handler tests.
func (f *fakeCoreAPI) GetSettingsLoc(ctx context.Context, prefix, _ string) (map[string]string, error) {
	return f.GetSettings(ctx, prefix)
}

func (f *fakeCoreAPI) SetSettingLoc(ctx context.Context, key, _, value string) error {
	return f.SetSetting(ctx, key, value)
}

// newTestApp wires the handler onto a Fiber app WITHOUT the auth middleware,
// so tests can exercise route logic without faking session context. Auth
// wiring is covered by the E2E smoke test in Task 11.
func newTestApp(h *ThemeSettingsHandler) *fiber.App {
	app := fiber.New()
	app.Get("/theme-settings", h.List)
	app.Get("/theme-settings/", h.List)
	app.Get("/theme-settings/:page", h.Get)
	app.Put("/theme-settings/:page", h.Save)
	return app
}

func makeField(t *testing.T, key, label, fieldType string, dflt any) ThemeSettingsField {
	t.Helper()
	var raw json.RawMessage
	if dflt != nil {
		b, err := json.Marshal(dflt)
		if err != nil {
			t.Fatalf("marshal default: %v", err)
		}
		raw = b
	}
	return ThemeSettingsField{Key: key, Label: label, Type: fieldType, Default: raw}
}

func decodeData(t *testing.T, resp *http.Response, into any) {
	t.Helper()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	defer resp.Body.Close()
	envelope := struct {
		Data json.RawMessage `json:"data"`
	}{}
	if err := json.Unmarshal(body, &envelope); err != nil {
		t.Fatalf("unmarshal envelope: %v (body=%s)", err, string(body))
	}
	if err := json.Unmarshal(envelope.Data, into); err != nil {
		t.Fatalf("unmarshal data: %v (body=%s)", err, string(body))
	}
}

// TestList_NoActiveTheme — empty registry → 200 with empty pages array.
func TestList_NoActiveTheme(t *testing.T) {
	reg := NewThemeSettingsRegistry()
	h := NewThemeSettingsHandler(reg, newFakeAPI(), nil, nil, nil)
	app := newTestApp(h)

	resp, err := app.Test(httptest.NewRequest("GET", "/theme-settings", nil))
	if err != nil {
		t.Fatalf("Test: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var got listResponse
	decodeData(t, resp, &got)
	if got.ActiveThemeSlug != "" {
		t.Errorf("active_theme_slug = %q, want \"\"", got.ActiveThemeSlug)
	}
	if got.Pages == nil {
		t.Errorf("pages must be empty array, not nil")
	}
	if len(got.Pages) != 0 {
		t.Errorf("expected 0 pages, got %d", len(got.Pages))
	}
}

// TestList_TwoPages — two pages declared → both returned in order.
func TestList_TwoPages(t *testing.T) {
	reg := NewThemeSettingsRegistry()
	reg.SetActive("hello-vietnam", []ThemeSettingsPage{
		{Slug: "header", Name: "Header Settings", Icon: "panel-top"},
		{Slug: "api-keys", Name: "API Keys", Icon: "key"},
	})
	h := NewThemeSettingsHandler(reg, newFakeAPI(), nil, nil, nil)
	app := newTestApp(h)

	resp, err := app.Test(httptest.NewRequest("GET", "/theme-settings", nil))
	if err != nil {
		t.Fatalf("Test: %v", err)
	}
	var got listResponse
	decodeData(t, resp, &got)
	if got.ActiveThemeSlug != "hello-vietnam" {
		t.Errorf("active_theme_slug = %q", got.ActiveThemeSlug)
	}
	if len(got.Pages) != 2 || got.Pages[0].Slug != "header" || got.Pages[1].Slug != "api-keys" {
		t.Errorf("pages = %+v", got.Pages)
	}
}

// TestGet_PageNotFound — request for a page not declared by the active theme → 404.
func TestGet_PageNotFound(t *testing.T) {
	reg := NewThemeSettingsRegistry()
	reg.SetActive("hello", []ThemeSettingsPage{{Slug: "header", Name: "H"}})
	h := NewThemeSettingsHandler(reg, newFakeAPI(), nil, nil, nil)
	app := newTestApp(h)

	resp, err := app.Test(httptest.NewRequest("GET", "/theme-settings/footer", nil))
	if err != nil {
		t.Fatalf("Test: %v", err)
	}
	if resp.StatusCode != 404 {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

// TestGet_ReturnsSchemaPlusValues — schema mirrors fields, values map covers
// every field even when no row is stored.
func TestGet_ReturnsSchemaPlusValues(t *testing.T) {
	reg := NewThemeSettingsRegistry()
	reg.SetActive("hv", []ThemeSettingsPage{{
		Slug: "header",
		Name: "Header",
		Fields: []ThemeSettingsField{
			makeField(t, "a", "A", "text", nil),
			makeField(t, "b", "B", "text", nil),
		},
	}})
	api := newFakeAPI()
	api.store[SettingKey("hv", "header", "a")] = "stored"
	h := NewThemeSettingsHandler(reg, api, nil, nil, nil)
	app := newTestApp(h)

	resp, err := app.Test(httptest.NewRequest("GET", "/theme-settings/header", nil))
	if err != nil {
		t.Fatalf("Test: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var got getResponse
	decodeData(t, resp, &got)
	if len(got.Page.Fields) != 2 {
		t.Errorf("fields len = %d", len(got.Page.Fields))
	}
	if got.Values["a"].Value != "stored" || !got.Values["a"].Compatible {
		t.Errorf("values[a] = %+v", got.Values["a"])
	}
	if got.Values["b"].Value != nil || !got.Values["b"].Compatible {
		t.Errorf("values[b] = %+v", got.Values["b"])
	}
}

// TestGet_IncompatibleValueFallsBackToDefault — number field with non-numeric
// stored raw → compatible=false, value falls back to declared default.
func TestGet_IncompatibleValueFallsBackToDefault(t *testing.T) {
	reg := NewThemeSettingsRegistry()
	reg.SetActive("hv", []ThemeSettingsPage{{
		Slug: "p",
		Name: "P",
		Fields: []ThemeSettingsField{
			makeField(t, "x", "X", "number", 7),
		},
	}})
	api := newFakeAPI()
	api.store[SettingKey("hv", "p", "x")] = "abc"
	h := NewThemeSettingsHandler(reg, api, nil, nil, nil)
	app := newTestApp(h)

	resp, err := app.Test(httptest.NewRequest("GET", "/theme-settings/p", nil))
	if err != nil {
		t.Fatalf("Test: %v", err)
	}
	var got getResponse
	decodeData(t, resp, &got)
	v := got.Values["x"]
	if v.Compatible {
		t.Errorf("expected compatible=false")
	}
	if v.Raw != "abc" {
		t.Errorf("raw = %q", v.Raw)
	}
	// JSON unmarshals numbers as float64.
	if f, ok := v.Value.(float64); !ok || f != 7 {
		t.Errorf("value = %v (%T), want 7", v.Value, v.Value)
	}
}

func putJSON(t *testing.T, app *fiber.App, path string, body any) *http.Response {
	t.Helper()
	b, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	req := httptest.NewRequest("PUT", path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Test: %v", err)
	}
	return resp
}

// TestSave_PersistsAllProvidedFields — three different field types saved →
// all three keys appear in the store with correctly encoded values; a
// follow-up Get returns them.
func TestSave_PersistsAllProvidedFields(t *testing.T) {
	reg := NewThemeSettingsRegistry()
	reg.SetActive("hv", []ThemeSettingsPage{{
		Slug: "p",
		Name: "P",
		Fields: []ThemeSettingsField{
			makeField(t, "tagline", "Tag", "text", nil),
			makeField(t, "count", "Count", "number", nil),
			makeField(t, "logo", "Logo", "image", nil),
		},
	}})
	api := newFakeAPI()
	h := NewThemeSettingsHandler(reg, api, nil, nil, nil)
	app := newTestApp(h)

	resp := putJSON(t, app, "/theme-settings/p", map[string]any{
		"values": map[string]any{
			"tagline": "Hi",
			"count":   42,
			"logo":    map[string]any{"id": 1, "url": "/x"},
		},
	})
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if got := api.store[SettingKey("hv", "p", "tagline")]; got != "Hi" {
		t.Errorf("tagline stored = %q", got)
	}
	if got := api.store[SettingKey("hv", "p", "count")]; got != "42" {
		t.Errorf("count stored = %q", got)
	}
	if got := api.store[SettingKey("hv", "p", "logo")]; got == "" {
		t.Errorf("logo not stored")
	}

	// Round-trip via GET.
	getResp, _ := app.Test(httptest.NewRequest("GET", "/theme-settings/p", nil))
	var got getResponse
	decodeData(t, getResp, &got)
	if got.Values["tagline"].Value != "Hi" {
		t.Errorf("tagline round-trip = %v", got.Values["tagline"].Value)
	}
	if f, _ := got.Values["count"].Value.(float64); f != 42 {
		t.Errorf("count round-trip = %v", got.Values["count"].Value)
	}
	if m, ok := got.Values["logo"].Value.(map[string]any); !ok || m["url"] != "/x" {
		t.Errorf("logo round-trip = %v", got.Values["logo"].Value)
	}
}

// TestSave_TextFieldStoresRawString — text payload "Hi" stored verbatim, no
// JSON quotes.
func TestSave_TextFieldStoresRawString(t *testing.T) {
	reg := NewThemeSettingsRegistry()
	reg.SetActive("hv", []ThemeSettingsPage{{
		Slug:   "p",
		Fields: []ThemeSettingsField{makeField(t, "tagline", "T", "text", nil)},
	}})
	api := newFakeAPI()
	h := NewThemeSettingsHandler(reg, api, nil, nil, nil)
	app := newTestApp(h)

	resp := putJSON(t, app, "/theme-settings/p", map[string]any{"values": map[string]any{"tagline": "Hi"}})
	if resp.StatusCode != 200 {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	if got := api.store[SettingKey("hv", "p", "tagline")]; got != "Hi" {
		t.Errorf("stored = %q, want %q", got, "Hi")
	}
}

// TestSave_NumberFieldStoresJSON — number 42 stored as the JSON literal "42".
func TestSave_NumberFieldStoresJSON(t *testing.T) {
	reg := NewThemeSettingsRegistry()
	reg.SetActive("hv", []ThemeSettingsPage{{
		Slug:   "p",
		Fields: []ThemeSettingsField{makeField(t, "count", "C", "number", nil)},
	}})
	api := newFakeAPI()
	h := NewThemeSettingsHandler(reg, api, nil, nil, nil)
	app := newTestApp(h)

	resp := putJSON(t, app, "/theme-settings/p", map[string]any{"values": map[string]any{"count": 42}})
	if resp.StatusCode != 200 {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	if got := api.store[SettingKey("hv", "p", "count")]; got != "42" {
		t.Errorf("stored = %q, want %q", got, "42")
	}
}

// TestSave_ToggleStoresJSON — boolean true stored as JSON literal "true".
func TestSave_ToggleStoresJSON(t *testing.T) {
	reg := NewThemeSettingsRegistry()
	reg.SetActive("hv", []ThemeSettingsPage{{
		Slug:   "p",
		Fields: []ThemeSettingsField{makeField(t, "show", "S", "toggle", nil)},
	}})
	api := newFakeAPI()
	h := NewThemeSettingsHandler(reg, api, nil, nil, nil)
	app := newTestApp(h)

	resp := putJSON(t, app, "/theme-settings/p", map[string]any{"values": map[string]any{"show": true}})
	if resp.StatusCode != 200 {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	if got := api.store[SettingKey("hv", "p", "show")]; got != "true" {
		t.Errorf("stored = %q, want \"true\"", got)
	}
}

// TestSave_ObjectFieldStoresJSON — object value stored as its JSON
// serialization.
func TestSave_ObjectFieldStoresJSON(t *testing.T) {
	reg := NewThemeSettingsRegistry()
	reg.SetActive("hv", []ThemeSettingsPage{{
		Slug:   "p",
		Fields: []ThemeSettingsField{makeField(t, "logo", "L", "image", nil)},
	}})
	api := newFakeAPI()
	h := NewThemeSettingsHandler(reg, api, nil, nil, nil)
	app := newTestApp(h)

	resp := putJSON(t, app, "/theme-settings/p", map[string]any{"values": map[string]any{"logo": map[string]any{"id": 1, "url": "/x"}}})
	if resp.StatusCode != 200 {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	got := api.store[SettingKey("hv", "p", "logo")]
	var back map[string]any
	if err := json.Unmarshal([]byte(got), &back); err != nil {
		t.Fatalf("stored value not JSON: %q", got)
	}
	if back["url"] != "/x" {
		t.Errorf("decoded = %+v", back)
	}
}

// TestSave_FieldNotInPagesIgnored — keys outside the page's declared schema
// are silently dropped, never written to storage.
func TestSave_FieldNotInPagesIgnored(t *testing.T) {
	reg := NewThemeSettingsRegistry()
	reg.SetActive("hv", []ThemeSettingsPage{{
		Slug:   "p",
		Fields: []ThemeSettingsField{makeField(t, "tagline", "T", "text", nil)},
	}})
	api := newFakeAPI()
	h := NewThemeSettingsHandler(reg, api, nil, nil, nil)
	app := newTestApp(h)

	resp := putJSON(t, app, "/theme-settings/p", map[string]any{"values": map[string]any{
		"tagline":   "Hi",
		"undeclared": "nope",
	}})
	if resp.StatusCode != 200 {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	if _, present := api.store[SettingKey("hv", "p", "undeclared")]; present {
		t.Errorf("undeclared key was stored")
	}
}

// TestSave_PageNotFound — PUT to a page not declared by the active theme → 404.
func TestSave_PageNotFound(t *testing.T) {
	reg := NewThemeSettingsRegistry()
	reg.SetActive("hv", []ThemeSettingsPage{{Slug: "p", Fields: []ThemeSettingsField{}}})
	h := NewThemeSettingsHandler(reg, newFakeAPI(), nil, nil, nil)
	app := newTestApp(h)

	resp := putJSON(t, app, "/theme-settings/missing", map[string]any{"values": map[string]any{}})
	if resp.StatusCode != 404 {
		t.Fatalf("status = %d", resp.StatusCode)
	}
}

// TestSave_BadBody — non-JSON body → 400.
func TestSave_BadBody(t *testing.T) {
	reg := NewThemeSettingsRegistry()
	reg.SetActive("hv", []ThemeSettingsPage{{Slug: "p", Fields: []ThemeSettingsField{}}})
	h := NewThemeSettingsHandler(reg, newFakeAPI(), nil, nil, nil)
	app := newTestApp(h)

	req := httptest.NewRequest("PUT", "/theme-settings/p", strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Test: %v", err)
	}
	if resp.StatusCode != 400 {
		t.Fatalf("status = %d", resp.StatusCode)
	}
}

// TestSave_OmittedFieldsNotTouched — partial save updates only provided
// fields and leaves preexisting values untouched.
func TestSave_OmittedFieldsNotTouched(t *testing.T) {
	reg := NewThemeSettingsRegistry()
	reg.SetActive("hv", []ThemeSettingsPage{{
		Slug: "p",
		Fields: []ThemeSettingsField{
			makeField(t, "a", "A", "text", nil),
			makeField(t, "b", "B", "text", nil),
		},
	}})
	api := newFakeAPI()
	api.store[SettingKey("hv", "p", "a")] = "preserved"
	h := NewThemeSettingsHandler(reg, api, nil, nil, nil)
	app := newTestApp(h)

	resp := putJSON(t, app, "/theme-settings/p", map[string]any{"values": map[string]any{"b": "new"}})
	if resp.StatusCode != 200 {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	if got := api.store[SettingKey("hv", "p", "a")]; got != "preserved" {
		t.Errorf("field a was modified, got %q", got)
	}
	if got := api.store[SettingKey("hv", "p", "b")]; got != "new" {
		t.Errorf("field b = %q", got)
	}
}
