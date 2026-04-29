package coreapi

import (
	"context"
	"strings"
	"testing"

	"github.com/d5/tengo/v2"

	"squilla/internal/cms"
)

// fakeThemeAPI satisfies the slice of CoreAPI we need: GetSetting and
// GetSettings. Other methods are unused by the theme_settings module.
type fakeThemeAPI struct {
	CoreAPI // nil interface — any unexpected call panics, which is
	// what we want for tests that should never hit them.
	store map[string]string
}

func (f *fakeThemeAPI) GetSetting(_ context.Context, key string) (string, error) {
	return f.store[key], nil
}

func (f *fakeThemeAPI) GetSettings(_ context.Context, prefix string) (map[string]string, error) {
	out := map[string]string{}
	for k, v := range f.store {
		if strings.HasPrefix(k, prefix) {
			out[strings.TrimPrefix(k, prefix)] = v
		}
	}
	return out, nil
}

// newRegistryWithPage builds a registry pre-populated with the "header"
// page used across most of the test cases.
func newRegistryWithPage() *cms.ThemeSettingsRegistry {
	r := cms.NewThemeSettingsRegistry()
	r.SetActive("hv", []cms.ThemeSettingsPage{
		{
			Slug: "header",
			Name: "Header",
			Icon: "panel-top",
			Fields: []cms.ThemeSettingsField{
				{Key: "tagline", Label: "Tagline", Type: "text"},
				{Key: "show", Label: "Show", Type: "toggle"},
				{Key: "count", Label: "Count", Type: "number", Default: []byte(`7`)},
			},
		},
	})
	return r
}

// runTengoOut compiles+runs the given source and returns the value of
// the script's `out` variable (or nil). Caller-info is taken from the
// supplied context — internal by default unless WithCaller has been used.
func runTengoOut(t *testing.T, ctx context.Context, src string, api CoreAPI, registry *cms.ThemeSettingsRegistry) (any, error) {
	t.Helper()
	mods := tengo.NewModuleMap()
	mods.AddBuiltinModule("core/theme_settings", themeSettingsModule(api, ctx, registry))
	s := tengo.NewScript([]byte(src))
	s.SetImports(mods)
	if err := s.Add("out", nil); err != nil {
		t.Fatalf("seed out: %v", err)
	}
	compiled, err := s.RunContext(context.Background())
	if err != nil {
		return nil, err
	}
	v := compiled.Get("out")
	if v == nil {
		return nil, nil
	}
	return v.Value(), nil
}

func internalCtx() context.Context {
	return WithCaller(context.Background(), InternalCaller())
}

func TestThemeSettings_ActiveTheme_ReturnsSlug(t *testing.T) {
	api := &fakeThemeAPI{store: map[string]string{}}
	reg := newRegistryWithPage()
	got, err := runTengoOut(t, internalCtx(), `
ts := import("core/theme_settings")
out = ts.active_theme()
`, api, reg)
	if err != nil {
		t.Fatalf("script error: %v", err)
	}
	if got != "hv" {
		t.Fatalf("active_theme: want %q, got %v", "hv", got)
	}
}

func TestThemeSettings_Pages_ReturnsArrayOfMaps(t *testing.T) {
	api := &fakeThemeAPI{store: map[string]string{}}
	reg := newRegistryWithPage()
	got, err := runTengoOut(t, internalCtx(), `
ts := import("core/theme_settings")
out = ts.pages()
`, api, reg)
	if err != nil {
		t.Fatalf("script error: %v", err)
	}
	arr, ok := got.([]any)
	if !ok {
		t.Fatalf("pages: want []any, got %T (%v)", got, got)
	}
	if len(arr) != 1 {
		t.Fatalf("pages: want 1 entry, got %d", len(arr))
	}
	m, ok := arr[0].(map[string]any)
	if !ok {
		t.Fatalf("pages[0]: want map, got %T", arr[0])
	}
	if m["slug"] != "header" || m["name"] != "Header" || m["icon"] != "panel-top" {
		t.Fatalf("pages[0] mismatch: %v", m)
	}
}

func TestThemeSettings_Get_TextField(t *testing.T) {
	api := &fakeThemeAPI{store: map[string]string{
		"theme:hv:header:tagline": "Hi",
	}}
	reg := newRegistryWithPage()
	got, err := runTengoOut(t, internalCtx(), `
ts := import("core/theme_settings")
out = ts.get("header", "tagline")
`, api, reg)
	if err != nil {
		t.Fatalf("script error: %v", err)
	}
	if got != "Hi" {
		t.Fatalf("get tagline: want %q, got %v", "Hi", got)
	}
}

func TestThemeSettings_Get_NumberCoerced(t *testing.T) {
	api := &fakeThemeAPI{store: map[string]string{
		"theme:hv:header:count": "42",
	}}
	reg := newRegistryWithPage()
	got, err := runTengoOut(t, internalCtx(), `
ts := import("core/theme_settings")
out = ts.get("header", "count")
`, api, reg)
	if err != nil {
		t.Fatalf("script error: %v", err)
	}
	// goToTengoObj routes 42.0 → tengo.Int (whole number) → int64 on read-back.
	switch v := got.(type) {
	case int64:
		if v != 42 {
			t.Fatalf("count: want 42, got %d", v)
		}
	case float64:
		if v != 42 {
			t.Fatalf("count: want 42, got %v", v)
		}
	default:
		t.Fatalf("count: unexpected type %T (%v)", got, got)
	}
}

func TestThemeSettings_Get_IncompatibleFallsBackToDefault(t *testing.T) {
	api := &fakeThemeAPI{store: map[string]string{
		"theme:hv:header:count": "abc",
	}}
	reg := newRegistryWithPage()
	got, err := runTengoOut(t, internalCtx(), `
ts := import("core/theme_settings")
out = ts.get("header", "count")
`, api, reg)
	if err != nil {
		t.Fatalf("script error: %v", err)
	}
	switch v := got.(type) {
	case int64:
		if v != 7 {
			t.Fatalf("count default: want 7, got %d", v)
		}
	case float64:
		if v != 7 {
			t.Fatalf("count default: want 7, got %v", v)
		}
	default:
		t.Fatalf("count default: unexpected type %T (%v)", got, got)
	}
}

func TestThemeSettings_Get_MissingFieldReturnsUndefined(t *testing.T) {
	api := &fakeThemeAPI{store: map[string]string{}}
	reg := newRegistryWithPage()
	got, err := runTengoOut(t, internalCtx(), `
ts := import("core/theme_settings")
out = ts.get("header", "nope")
`, api, reg)
	if err != nil {
		t.Fatalf("script error: %v", err)
	}
	if got != nil {
		t.Fatalf("missing field: want nil, got %v", got)
	}
}

func TestThemeSettings_All_ReturnsAllFields(t *testing.T) {
	api := &fakeThemeAPI{store: map[string]string{
		"theme:hv:header:tagline": "Hi",
		"theme:hv:header:show":    "true",
		"theme:hv:header:count":   "42",
	}}
	reg := newRegistryWithPage()
	got, err := runTengoOut(t, internalCtx(), `
ts := import("core/theme_settings")
out = ts.all("header")
`, api, reg)
	if err != nil {
		t.Fatalf("script error: %v", err)
	}
	m, ok := got.(map[string]any)
	if !ok {
		t.Fatalf("all: want map, got %T", got)
	}
	if m["tagline"] != "Hi" {
		t.Fatalf("tagline: want Hi, got %v", m["tagline"])
	}
	if m["show"] != true {
		t.Fatalf("show: want true, got %v", m["show"])
	}
	switch v := m["count"].(type) {
	case int64:
		if v != 42 {
			t.Fatalf("count: want 42, got %d", v)
		}
	case float64:
		if v != 42 {
			t.Fatalf("count: want 42, got %v", v)
		}
	default:
		t.Fatalf("count: unexpected type %T", m["count"])
	}
}

func TestThemeSettings_All_UnknownPageReturnsEmptyMap(t *testing.T) {
	api := &fakeThemeAPI{store: map[string]string{}}
	reg := newRegistryWithPage()
	got, err := runTengoOut(t, internalCtx(), `
ts := import("core/theme_settings")
out = ts.all("missing")
`, api, reg)
	if err != nil {
		t.Fatalf("script error: %v", err)
	}
	m, ok := got.(map[string]any)
	if !ok {
		t.Fatalf("all missing: want map, got %T", got)
	}
	if len(m) != 0 {
		t.Fatalf("all missing: want empty, got %v", m)
	}
}

func TestThemeSettings_NoActiveTheme(t *testing.T) {
	api := &fakeThemeAPI{store: map[string]string{}}
	reg := cms.NewThemeSettingsRegistry()

	// active_theme → ""
	got, err := runTengoOut(t, internalCtx(), `
ts := import("core/theme_settings")
out = ts.active_theme()
`, api, reg)
	if err != nil {
		t.Fatalf("active_theme err: %v", err)
	}
	if got != "" {
		t.Fatalf("active_theme: want \"\", got %v", got)
	}

	// pages → []
	got, err = runTengoOut(t, internalCtx(), `
ts := import("core/theme_settings")
out = ts.pages()
`, api, reg)
	if err != nil {
		t.Fatalf("pages err: %v", err)
	}
	arr, ok := got.([]any)
	if !ok {
		t.Fatalf("pages: want []any, got %T", got)
	}
	if len(arr) != 0 {
		t.Fatalf("pages: want empty, got %v", arr)
	}

	// get → undefined / nil
	got, err = runTengoOut(t, internalCtx(), `
ts := import("core/theme_settings")
out = ts.get("header", "tagline")
`, api, reg)
	if err != nil {
		t.Fatalf("get err: %v", err)
	}
	if got != nil {
		t.Fatalf("get: want nil, got %v", got)
	}

	// all → {}
	got, err = runTengoOut(t, internalCtx(), `
ts := import("core/theme_settings")
out = ts.all("header")
`, api, reg)
	if err != nil {
		t.Fatalf("all err: %v", err)
	}
	m, ok := got.(map[string]any)
	if !ok {
		t.Fatalf("all: want map, got %T", got)
	}
	if len(m) != 0 {
		t.Fatalf("all: want empty, got %v", m)
	}
}

// --- Capability tests ----------------------------------------------------

// callerCtx returns a non-internal caller context.
func callerCtx(caps map[string]bool) context.Context {
	return WithCaller(context.Background(), CallerInfo{
		Slug:         "some-extension",
		Type:         "tengo",
		Capabilities: caps,
	})
}

// runTengoCallRaw invokes the module function directly (bypassing the
// tengo VM) so we can inspect the returned *tengo.Error object without
// the VM converting "use of error value" into an unrelated runtime error.
func runTengoCallRaw(api CoreAPI, ctx context.Context, registry *cms.ThemeSettingsRegistry, fn string, args ...tengo.Object) tengo.Object {
	mod := themeSettingsModule(api, ctx, registry)
	uf, ok := mod[fn].(*tengo.UserFunction)
	if !ok {
		return nil
	}
	out, err := uf.Value(args...)
	if err != nil {
		return wrapError(err)
	}
	return out
}

func TestThemeSettings_Capability_DeniedForExternalCaller(t *testing.T) {
	api := &fakeThemeAPI{store: map[string]string{}}
	reg := newRegistryWithPage()
	ctx := callerCtx(nil)

	type call struct {
		fn   string
		args []tengo.Object
	}
	cases := []call{
		{"active_theme", nil},
		{"pages", nil},
		{"get", []tengo.Object{&tengo.String{Value: "header"}, &tengo.String{Value: "tagline"}}},
		{"all", []tengo.Object{&tengo.String{Value: "header"}}},
	}
	for _, c := range cases {
		got := runTengoCallRaw(api, ctx, reg, c.fn, c.args...)
		errObj, ok := got.(*tengo.Error)
		if !ok {
			t.Fatalf("%s: expected *tengo.Error, got %T (%v)", c.fn, got, got)
		}
		msg := ""
		if s, ok := errObj.Value.(*tengo.String); ok {
			msg = s.Value
		}
		if !strings.Contains(msg, "theme_settings:read") {
			t.Fatalf("%s: expected error mentioning theme_settings:read, got %q", c.fn, msg)
		}
	}
}

func TestThemeSettings_Capability_GrantedForExternalCallerWithCap(t *testing.T) {
	api := &fakeThemeAPI{store: map[string]string{
		"theme:hv:header:tagline": "Hi",
	}}
	reg := newRegistryWithPage()
	ctx := callerCtx(map[string]bool{"theme_settings:read": true})

	got, err := runTengoOut(t, ctx, `
ts := import("core/theme_settings")
out = ts.get("header", "tagline")
`, api, reg)
	if err != nil {
		t.Fatalf("script error: %v", err)
	}
	if got != "Hi" {
		t.Fatalf("granted caller: want Hi, got %v", got)
	}
}

func TestThemeSettings_Capability_InternalBypasses(t *testing.T) {
	api := &fakeThemeAPI{store: map[string]string{
		"theme:hv:header:tagline": "Hi",
	}}
	reg := newRegistryWithPage()
	// Internal caller — no capabilities map at all.
	ctx := WithCaller(context.Background(), InternalCaller())

	// active_theme
	got, err := runTengoOut(t, ctx, `
ts := import("core/theme_settings")
out = ts.active_theme()
`, api, reg)
	if err != nil || got != "hv" {
		t.Fatalf("internal active_theme: got %v err %v", got, err)
	}

	// get
	got, err = runTengoOut(t, ctx, `
ts := import("core/theme_settings")
out = ts.get("header", "tagline")
`, api, reg)
	if err != nil || got != "Hi" {
		t.Fatalf("internal get: got %v err %v", got, err)
	}

	// all
	got, err = runTengoOut(t, ctx, `
ts := import("core/theme_settings")
out = ts.all("header")
`, api, reg)
	if err != nil {
		t.Fatalf("internal all err: %v", err)
	}
	if _, ok := got.(map[string]any); !ok {
		t.Fatalf("internal all: want map, got %T", got)
	}

	// pages
	got, err = runTengoOut(t, ctx, `
ts := import("core/theme_settings")
out = ts.pages()
`, api, reg)
	if err != nil {
		t.Fatalf("internal pages err: %v", err)
	}
	if _, ok := got.([]any); !ok {
		t.Fatalf("internal pages: want []any, got %T", got)
	}
}
