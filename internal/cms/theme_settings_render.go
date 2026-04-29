package cms

import (
	"context"
	"html/template"
	"log"
)

// logThemeSettingsLoadError logs a render-time settings load failure. Split
// out so tests can replace the logger if needed.
var logThemeSettingsLoadError = func(err error) {
	log.Printf("WARN: build theme settings context: %v", err)
}

// settingsReader is the slice of CoreAPI we need for render-time loads.
// Defined locally to avoid the cms ↔ coreapi import cycle (coreapi/impl.go
// already imports cms). The full coreapi.CoreAPI satisfies this interface
// implicitly.
type settingsReader interface {
	GetSettings(ctx context.Context, prefix string) (map[string]string, error)
	GetSettingsLoc(ctx context.Context, prefix, locale string) (map[string]string, error)
}

// BuildThemeSettingsContext returns a nested map[pageSlug]map[fieldKey]any
// for the active theme. Every declared field gets exactly one entry, with
// values coerced to their typed runtime form and incompatible values
// substituted with the field's declared default. Empty map (never nil) when
// no theme is active or the active theme declares no settings pages.
//
// One GetSettings call per invocation — caller is expected to memoize per
// request scope.
func BuildThemeSettingsContext(
	ctx context.Context,
	registry *ThemeSettingsRegistry,
	api settingsReader,
) (map[string]map[string]any, error) {
	return BuildThemeSettingsContextForLocale(ctx, registry, api, "")
}

// BuildThemeSettingsContextForLocale is the locale-aware variant. Translatable
// fields resolve to (key, locale) with fallback to (key, ""); non-translatable
// fields always resolve at the shared row. An empty locale returns the shared
// row directly for every field.
func BuildThemeSettingsContextForLocale(
	ctx context.Context,
	registry *ThemeSettingsRegistry,
	api settingsReader,
	locale string,
) (map[string]map[string]any, error) {
	out := map[string]map[string]any{}
	if registry == nil {
		return out, nil
	}
	slug := registry.ActiveSlug()
	pages := registry.ActivePages()
	if slug == "" || len(pages) == 0 {
		return out, nil
	}

	prefix := ThemePrefix(slug)
	raw, err := api.GetSettingsLoc(ctx, prefix, locale)
	if err != nil {
		return nil, err
	}

	for _, p := range pages {
		page := make(map[string]any, len(p.Fields))
		for _, f := range p.Fields {
			stored := raw[p.Slug+":"+f.Key]
			page[f.Key] = CoerceWithDefault(f, stored).Value
		}
		out[p.Slug] = page
	}
	return out, nil
}

// loadThemeSettingsForRender wraps BuildThemeSettingsContext with soft-fail
// semantics: errors are logged and an empty (non-nil) map is returned, so a
// settings read failure never aborts page rendering. Returns an empty map
// when the registry/API are nil (e.g. preview/test paths).
//
// locale should be the request's language code (e.g. "en", "vi"). Translatable
// fields resolve to that locale; non-translatable fields share a single value.
func (h *PublicHandler) loadThemeSettingsForRender(ctx context.Context, locale string) map[string]map[string]any {
	if h.themeSettingsRegistry == nil || h.themeSettingsAPI == nil {
		return map[string]map[string]any{}
	}
	ts, err := BuildThemeSettingsContextForLocale(ctx, h.themeSettingsRegistry, h.themeSettingsAPI, locale)
	if err != nil {
		// Soft-fail: log and continue with an empty map. Templates already
		// need to handle missing values defensively (with/if), so an empty
		// map keeps render working without surfacing a 500 to the visitor.
		logThemeSettingsLoadError(err)
		return map[string]map[string]any{}
	}
	return ts
}

// themeSettingsFuncs exposes themeSetting / themeSettingsPage to block
// templates. ts may be nil — both funcs treat nil as "no values" and return
// nil rather than panic.
func themeSettingsFuncs(ts map[string]map[string]any) template.FuncMap {
	return template.FuncMap{
		"themeSetting": func(page, key string) any {
			if ts == nil {
				return nil
			}
			p, ok := ts[page]
			if !ok {
				return nil
			}
			return p[key]
		},
		"themeSettingsPage": func(page string) map[string]any {
			if ts == nil {
				return nil
			}
			return ts[page]
		},
	}
}

// mergeFuncMaps returns a new FuncMap containing all entries from each input.
// Later maps override earlier ones on key collision.
func mergeFuncMaps(maps ...template.FuncMap) template.FuncMap {
	out := template.FuncMap{}
	for _, m := range maps {
		for k, v := range m {
			out[k] = v
		}
	}
	return out
}
