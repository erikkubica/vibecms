package coreapi

import (
	"context"
	"fmt"

	"github.com/d5/tengo/v2"

	"squilla/internal/cms"
)

// themeSettingsModule wires the core/theme_settings Tengo module. It depends
// on the active theme's *cms.ThemeSettingsRegistry and the CoreAPI (used to
// fetch raw stored values via GetSettings under the active-theme prefix).
//
// Every function checks the theme_settings:read capability before doing work,
// matching the gating used by core/settings. Internal callers bypass.
func themeSettingsModule(
	api CoreAPI,
	ctx context.Context,
	registry *cms.ThemeSettingsRegistry,
) map[string]tengo.Object {
	requireRead := func() error {
		return checkCapability(ctx, "theme_settings:read")
	}

	return map[string]tengo.Object{
		"active_theme": &tengo.UserFunction{
			Name: "active_theme",
			Value: func(args ...tengo.Object) (tengo.Object, error) {
				if err := requireRead(); err != nil {
					return wrapError(err), nil
				}
				if registry == nil {
					return &tengo.String{Value: ""}, nil
				}
				return &tengo.String{Value: registry.ActiveSlug()}, nil
			},
		},
		"pages": &tengo.UserFunction{
			Name: "pages",
			Value: func(args ...tengo.Object) (tengo.Object, error) {
				if err := requireRead(); err != nil {
					return wrapError(err), nil
				}
				if registry == nil {
					return &tengo.Array{Value: nil}, nil
				}
				pages := registry.ActivePages()
				items := make([]tengo.Object, 0, len(pages))
				for _, p := range pages {
					items = append(items, &tengo.Map{Value: map[string]tengo.Object{
						"slug": &tengo.String{Value: p.Slug},
						"name": &tengo.String{Value: p.Name},
						"icon": &tengo.String{Value: p.Icon},
					}})
				}
				return &tengo.Array{Value: items}, nil
			},
		},
		"get": &tengo.UserFunction{
			Name: "get",
			Value: func(args ...tengo.Object) (tengo.Object, error) {
				if err := requireRead(); err != nil {
					return wrapError(err), nil
				}
				if len(args) < 2 {
					return wrapError(fmt.Errorf("theme_settings.get: requires page and key")), nil
				}
				pageSlug := tengoToString(args[0])
				fieldKey := tengoToString(args[1])
				value, err := getThemeSettingValue(ctx, api, registry, pageSlug, fieldKey)
				if err != nil {
					return wrapError(err), nil
				}
				return goToTengoObj(value), nil
			},
		},
		"all": &tengo.UserFunction{
			Name: "all",
			Value: func(args ...tengo.Object) (tengo.Object, error) {
				if err := requireRead(); err != nil {
					return wrapError(err), nil
				}
				if len(args) < 1 {
					return wrapError(fmt.Errorf("theme_settings.all: requires page")), nil
				}
				pageSlug := tengoToString(args[0])
				m, err := getThemeSettingsPage(ctx, api, registry, pageSlug)
				if err != nil {
					return wrapError(err), nil
				}
				out := make(map[string]tengo.Object, len(m))
				for k, v := range m {
					out[k] = goToTengoObj(v)
				}
				return &tengo.Map{Value: out}, nil
			},
		},
	}
}

// getThemeSettingValue returns the coerced value for a single field on a
// page of the active theme. nil + nil error when the page or field is not
// declared (Tengo gets undefined). Empty registry → undefined.
//
// The inner GetSetting call is invoked under an InternalCaller-scoped
// context so the script only needs theme_settings:read — the underlying
// settings:read gate is bypassed by design.
func getThemeSettingValue(
	ctx context.Context,
	api CoreAPI,
	registry *cms.ThemeSettingsRegistry,
	pageSlug, fieldKey string,
) (any, error) {
	if registry == nil {
		return nil, nil
	}
	slug := registry.ActiveSlug()
	if slug == "" {
		return nil, nil
	}
	page, ok := registry.ActivePage(pageSlug)
	if !ok {
		return nil, nil
	}
	var field cms.ThemeSettingsField
	found := false
	for _, f := range page.Fields {
		if f.Key == fieldKey {
			field = f
			found = true
			break
		}
	}
	if !found {
		return nil, nil
	}
	innerCtx := WithCaller(ctx, InternalCaller())
	raw, err := api.GetSetting(innerCtx, cms.SettingKey(slug, pageSlug, fieldKey))
	if err != nil {
		return nil, err
	}
	return cms.CoerceWithDefault(field, raw).Value, nil
}

// getThemeSettingsPage returns coerced values for every field on the named
// page of the active theme. Empty map (not nil) when nothing is declared.
//
// As with getThemeSettingValue, GetSettings is invoked with an
// InternalCaller-scoped context so the only gate the script faces is
// theme_settings:read.
func getThemeSettingsPage(
	ctx context.Context,
	api CoreAPI,
	registry *cms.ThemeSettingsRegistry,
	pageSlug string,
) (map[string]any, error) {
	out := map[string]any{}
	if registry == nil {
		return out, nil
	}
	slug := registry.ActiveSlug()
	if slug == "" {
		return out, nil
	}
	page, ok := registry.ActivePage(pageSlug)
	if !ok {
		return out, nil
	}
	innerCtx := WithCaller(ctx, InternalCaller())
	rawAll, err := api.GetSettings(innerCtx, cms.ThemePrefix(slug))
	if err != nil {
		return nil, err
	}
	for _, f := range page.Fields {
		out[f.Key] = cms.CoerceWithDefault(f, rawAll[pageSlug+":"+f.Key]).Value
	}
	return out, nil
}
