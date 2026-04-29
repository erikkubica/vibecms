package cms

import (
	"bytes"
	"context"
	"errors"
	"html/template"
	"testing"
)

// fakeSettingsReader is a minimal in-memory implementation of the local
// settingsReader interface for render-context tests.
type fakeSettingsReader struct {
	store map[string]string
	err   error
}

func newFakeReader() *fakeSettingsReader {
	return &fakeSettingsReader{store: map[string]string{}}
}

func (f *fakeSettingsReader) GetSettings(_ context.Context, prefix string) (map[string]string, error) {
	if f.err != nil {
		return nil, f.err
	}
	out := map[string]string{}
	for k, v := range f.store {
		if len(prefix) == 0 || (len(k) >= len(prefix) && k[:len(prefix)] == prefix) {
			out[k[len(prefix):]] = v
		}
	}
	return out, nil
}

func TestBuildThemeSettingsContext_Empty(t *testing.T) {
	reg := NewThemeSettingsRegistry()
	got, err := BuildThemeSettingsContext(context.Background(), reg, newFakeReader())
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if got == nil {
		t.Fatal("got nil, want empty map")
	}
	if len(got) != 0 {
		t.Errorf("got %d pages, want 0", len(got))
	}
}

func TestBuildThemeSettingsContext_NoActiveTheme(t *testing.T) {
	reg := NewThemeSettingsRegistry()
	reg.SetActive("", []ThemeSettingsPage{{Slug: "header", Name: "H"}})
	got, err := BuildThemeSettingsContext(context.Background(), reg, newFakeReader())
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if len(got) != 0 {
		t.Errorf("got %d pages, want 0", len(got))
	}
}

func TestBuildThemeSettingsContext_PopulatesValues(t *testing.T) {
	reg := NewThemeSettingsRegistry()
	reg.SetActive("hv", []ThemeSettingsPage{{
		Slug: "header",
		Name: "Header",
		Fields: []ThemeSettingsField{
			makeField(t, "tagline", "Tag", "text", "Welcome"),
			makeField(t, "logo", "Logo", "image", nil),
		},
	}})
	api := newFakeReader()
	api.store[SettingKey("hv", "header", "tagline")] = "Hi"

	got, err := BuildThemeSettingsContext(context.Background(), reg, api)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	header, ok := got["header"]
	if !ok {
		t.Fatalf("missing header page in result %+v", got)
	}
	if header["tagline"] != "Hi" {
		t.Errorf("tagline = %v, want %q", header["tagline"], "Hi")
	}
	if header["logo"] != nil {
		t.Errorf("logo = %v, want nil", header["logo"])
	}
}

func TestBuildThemeSettingsContext_IncompatibleValueFallsBackToDefault(t *testing.T) {
	reg := NewThemeSettingsRegistry()
	reg.SetActive("hv", []ThemeSettingsPage{{
		Slug:   "page",
		Fields: []ThemeSettingsField{makeField(t, "count", "C", "number", 7)},
	}})
	api := newFakeReader()
	api.store[SettingKey("hv", "page", "count")] = "abc"

	got, err := BuildThemeSettingsContext(context.Background(), reg, api)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	v, ok := got["page"]["count"].(float64)
	if !ok {
		t.Fatalf("count = %v (%T), want float64", got["page"]["count"], got["page"]["count"])
	}
	if v != 7 {
		t.Errorf("count = %v, want 7", v)
	}
}

func TestBuildThemeSettingsContext_NilRegistrySafe(t *testing.T) {
	got, err := BuildThemeSettingsContext(context.Background(), nil, newFakeReader())
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if got == nil {
		t.Fatal("got nil, want empty map")
	}
	if len(got) != 0 {
		t.Errorf("got %d entries", len(got))
	}
}

func TestBuildThemeSettingsContext_GetSettingsErrorBubblesUp(t *testing.T) {
	reg := NewThemeSettingsRegistry()
	reg.SetActive("hv", []ThemeSettingsPage{{
		Slug:   "p",
		Fields: []ThemeSettingsField{makeField(t, "k", "K", "text", nil)},
	}})
	api := newFakeReader()
	api.err = errors.New("boom")
	got, err := BuildThemeSettingsContext(context.Background(), reg, api)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if got != nil {
		t.Errorf("got = %v, want nil on error", got)
	}
}

func TestThemeSettingsFuncs_Get(t *testing.T) {
	ts := map[string]map[string]any{
		"header": {"logo": "/logo.png"},
	}
	fns := themeSettingsFuncs(ts)
	get := fns["themeSetting"].(func(string, string) any)
	if got := get("header", "logo"); got != "/logo.png" {
		t.Errorf("get(header, logo) = %v", got)
	}
}

func TestThemeSettingsFuncs_MissingPageReturnsNil(t *testing.T) {
	ts := map[string]map[string]any{"header": {"logo": "/x"}}
	fns := themeSettingsFuncs(ts)
	get := fns["themeSetting"].(func(string, string) any)
	if got := get("footer", "logo"); got != nil {
		t.Errorf("missing page = %v, want nil", got)
	}
}

func TestThemeSettingsFuncs_MissingKeyReturnsNil(t *testing.T) {
	ts := map[string]map[string]any{"header": {"logo": "/x"}}
	fns := themeSettingsFuncs(ts)
	get := fns["themeSetting"].(func(string, string) any)
	if got := get("header", "tagline"); got != nil {
		t.Errorf("missing key = %v, want nil", got)
	}
}

func TestThemeSettingsFuncs_PageMap(t *testing.T) {
	ts := map[string]map[string]any{"header": {"logo": "/x", "tagline": "Hi"}}
	fns := themeSettingsFuncs(ts)
	page := fns["themeSettingsPage"].(func(string) map[string]any)
	got := page("header")
	if got["logo"] != "/x" || got["tagline"] != "Hi" {
		t.Errorf("page map = %+v", got)
	}
	if page("missing") != nil {
		t.Errorf("missing page should return nil")
	}
}

func TestThemeSettingsFuncs_NilTemplateSettingsSafe(t *testing.T) {
	fns := themeSettingsFuncs(nil)
	get := fns["themeSetting"].(func(string, string) any)
	page := fns["themeSettingsPage"].(func(string) map[string]any)
	if v := get("header", "logo"); v != nil {
		t.Errorf("nil ts get = %v", v)
	}
	if v := page("header"); v != nil {
		t.Errorf("nil ts page = %v", v)
	}
}

func TestThemeSettingsFuncs_RenderInTemplate(t *testing.T) {
	ts := map[string]map[string]any{"header": {"logo": "/x.png"}}
	tmpl, err := template.New("t").Funcs(themeSettingsFuncs(ts)).Parse(
		`Logo: {{ themeSetting "header" "logo" }}`)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, nil); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if got := buf.String(); got != "Logo: /x.png" {
		t.Errorf("rendered = %q", got)
	}
}

func TestToMap_IncludesThemeSettings(t *testing.T) {
	td := TemplateData{
		ThemeSettings: map[string]map[string]any{"header": {"logo": "/x"}},
	}
	got := td.ToMap()
	ts, ok := got["theme_settings"].(map[string]map[string]any)
	if !ok {
		t.Fatalf("theme_settings missing or wrong type: %T", got["theme_settings"])
	}
	if ts["header"]["logo"] != "/x" {
		t.Errorf("theme_settings = %+v", ts)
	}
}

func TestToMap_NilThemeSettingsBecomesEmptyMap(t *testing.T) {
	td := TemplateData{}
	got := td.ToMap()
	ts, ok := got["theme_settings"].(map[string]map[string]any)
	if !ok {
		t.Fatalf("theme_settings missing: %T", got["theme_settings"])
	}
	if ts == nil {
		t.Error("theme_settings is nil; want empty map")
	}
}
