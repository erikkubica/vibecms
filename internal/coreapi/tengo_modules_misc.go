package coreapi

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/d5/tengo/v2"
)

// This file holds the smaller core/* Tengo modules that don't justify
// their own files: settings, events, http, log, assets, helpers, and
// routing. Each module is short on its own; together they made
// tengo_adapter.go nearly 400 lines longer than needed.

// routesModule, wellKnownModule, and filtersModule are thin
// placeholders — actual route/filter wiring lives in the script
// engine. The Tengo modules just notify the engine via callbacks
// when scripts call routes.register / wellknown.register / filters.add.

func routesModule(cb *ScriptCallbacks) map[string]tengo.Object {
	return map[string]tengo.Object{
		"register": &tengo.UserFunction{Name: "register", Value: func(args ...tengo.Object) (tengo.Object, error) {
			// routes.register(method, path, script_path)
			if len(args) < 3 {
				return tengo.UndefinedValue, fmt.Errorf("routes.register: requires method, path, script_path")
			}
			method := tengoToString(args[0])
			path := tengoToString(args[1])
			scriptPath := tengoToString(args[2])
			if cb != nil && cb.OnRoute != nil {
				cb.OnRoute(method, path, scriptPath)
			}
			return tengo.UndefinedValue, nil
		}},
	}
}

// wellKnownModule exposes wellknown.register(path, script_path).
// `path` is the suffix after "/.well-known/" (e.g. "security.txt",
// "nodeinfo/2.0", "acme-challenge/*"). A trailing "*" registers a
// prefix handler. The script receives `request` and sets `response`
// using the same shape as core/http route handlers.
func wellKnownModule(cb *ScriptCallbacks) map[string]tengo.Object {
	return map[string]tengo.Object{
		"register": &tengo.UserFunction{Name: "register", Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) < 2 {
				return tengo.UndefinedValue, fmt.Errorf("wellknown.register: requires path and script_path")
			}
			path := tengoToString(args[0])
			scriptPath := tengoToString(args[1])
			if cb != nil && cb.OnWellKnown != nil {
				cb.OnWellKnown(path, scriptPath)
			}
			return tengo.UndefinedValue, nil
		}},
	}
}

func filtersModule(cb *ScriptCallbacks) map[string]tengo.Object {
	return map[string]tengo.Object{
		"add": &tengo.UserFunction{Name: "add", Value: func(args ...tengo.Object) (tengo.Object, error) {
			// filters.add(name, script_path[, priority])
			if len(args) < 2 {
				return tengo.UndefinedValue, fmt.Errorf("filters.add: requires name and script_path")
			}
			name := tengoToString(args[0])
			scriptPath := tengoToString(args[1])
			priority := 10
			if len(args) > 2 {
				if p, ok := tengo.ToInt(args[2]); ok {
					priority = p
				}
			}
			if cb != nil && cb.OnFilter != nil {
				cb.OnFilter(name, scriptPath, priority)
			}
			return tengo.UndefinedValue, nil
		}},
	}
}

func settingsModule(api CoreAPI, ctx context.Context) map[string]tengo.Object {
	return map[string]tengo.Object{
		"get": &tengo.UserFunction{Name: "get", Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) < 1 {
				return wrapError(fmt.Errorf("settings.get: requires key argument")), nil
			}
			key := tengoToString(args[0])
			val, err := api.GetSetting(ctx, key)
			if err != nil {
				return &tengo.String{Value: ""}, nil
			}
			return &tengo.String{Value: val}, nil
		}},
		"set": &tengo.UserFunction{Name: "set", Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) < 2 {
				return wrapError(fmt.Errorf("settings.set: requires key and value arguments")), nil
			}
			key := tengoToString(args[0])
			value := tengoToString(args[1])
			if err := api.SetSetting(ctx, key, value); err != nil {
				return wrapError(err), nil
			}
			return tengo.UndefinedValue, nil
		}},
		"all": &tengo.UserFunction{Name: "all", Value: func(args ...tengo.Object) (tengo.Object, error) {
			prefix := ""
			if len(args) > 0 {
				prefix = tengoToString(args[0])
			}
			settings, err := api.GetSettings(ctx, prefix)
			if err != nil {
				return wrapError(err), nil
			}
			out := make(map[string]tengo.Object, len(settings))
			for k, v := range settings {
				out[k] = &tengo.String{Value: v}
			}
			return &tengo.Map{Value: out}, nil
		}},
	}
}

func eventsModule(api CoreAPI, ctx context.Context, cb *ScriptCallbacks) map[string]tengo.Object {
	return map[string]tengo.Object{
		"emit": &tengo.UserFunction{Name: "emit", Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) < 1 {
				return wrapError(fmt.Errorf("events.emit: requires action argument")), nil
			}
			action := tengoToString(args[0])
			var payload map[string]any
			if len(args) > 1 {
				if m := getTengoMap(args[1]); m != nil {
					payload = tengoMapToGoMap(m)
				}
			}
			err := api.Emit(ctx, action, payload)
			if err != nil {
				return wrapError(err), nil
			}
			return tengo.UndefinedValue, nil
		}},
		"subscribe": &tengo.UserFunction{Name: "subscribe", Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) < 2 {
				return wrapError(fmt.Errorf("events.subscribe: requires action and script_path arguments")), nil
			}
			action := tengoToString(args[0])
			scriptPath := tengoToString(args[1])
			priority := 10
			if len(args) > 2 {
				if p, ok := tengo.ToInt(args[2]); ok {
					priority = p
				}
			}
			if cb != nil && cb.OnEvent != nil {
				cb.OnEvent(action, scriptPath, priority)
			}
			return tengo.UndefinedValue, nil
		}},
		"on": &tengo.UserFunction{Name: "on", Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) < 2 {
				return wrapError(fmt.Errorf("events.on: requires action and script_path arguments")), nil
			}
			action := tengoToString(args[0])
			scriptPath := tengoToString(args[1])
			priority := 50
			if len(args) > 2 {
				if p, ok := tengo.ToInt(args[2]); ok {
					priority = p
				}
			}
			if cb != nil && cb.OnEvent != nil {
				cb.OnEvent(action, scriptPath, priority)
			}
			return tengo.UndefinedValue, nil
		}},
	}
}

func httpFetchModule(api CoreAPI, ctx context.Context) map[string]tengo.Object {
	makeFetcher := func(method string) tengo.CallableFunc {
		return func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) < 1 {
				return wrapError(fmt.Errorf("http.%s: requires url argument", strings.ToLower(method))), nil
			}
			url := tengoToString(args[0])
			if url == "" {
				return wrapError(fmt.Errorf("http.%s: url cannot be empty", strings.ToLower(method))), nil
			}

			req := FetchRequest{Method: method, URL: url}
			if len(args) > 1 {
				if m := getTengoMap(args[1]); m != nil {
					if h, ok := m["headers"]; ok {
						if hm := getTengoMap(h); hm != nil {
							req.Headers = make(map[string]string, len(hm))
							for k, v := range hm {
								req.Headers[k] = tengoToString(v)
							}
						}
					}
					if b, ok := m["body"]; ok {
						req.Body = tengoToString(b)
					}
					if t, ok := m["timeout"]; ok {
						req.Timeout = tengoToInt(t)
					}
				}
			}

			resp, err := api.Fetch(ctx, req)
			if err != nil {
				return &tengo.Map{Value: map[string]tengo.Object{
					"status_code": &tengo.Int{Value: 0},
					"body":        &tengo.String{Value: ""},
					"error":       &tengo.String{Value: err.Error()},
				}}, nil
			}

			respHeaders := make(map[string]tengo.Object, len(resp.Headers))
			for k, v := range resp.Headers {
				respHeaders[k] = &tengo.String{Value: v}
			}

			return &tengo.Map{Value: map[string]tengo.Object{
				"status_code": &tengo.Int{Value: int64(resp.StatusCode)},
				"body":        &tengo.String{Value: resp.Body},
				"headers":     &tengo.ImmutableMap{Value: respHeaders},
				"error":       &tengo.String{Value: ""},
			}}, nil
		}
	}

	return map[string]tengo.Object{
		"get":    &tengo.UserFunction{Name: "get", Value: makeFetcher("GET")},
		"post":   &tengo.UserFunction{Name: "post", Value: makeFetcher("POST")},
		"put":    &tengo.UserFunction{Name: "put", Value: makeFetcher("PUT")},
		"delete": &tengo.UserFunction{Name: "delete", Value: makeFetcher("DELETE")},
		"fetch": &tengo.UserFunction{Name: "fetch", Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) < 2 {
				return wrapError(fmt.Errorf("http.fetch: requires method and url arguments")), nil
			}
			method := strings.ToUpper(tengoToString(args[0]))
			f := makeFetcher(method)
			return f(args[1:]...)
		}},
	}
}

func logModule(api CoreAPI, ctx context.Context) map[string]tengo.Object {
	makeLogger := func(level string) tengo.CallableFunc {
		return func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) < 1 {
				return tengo.UndefinedValue, nil
			}
			msg := tengoToString(args[0])
			var fields map[string]any
			if len(args) > 1 {
				if m := getTengoMap(args[1]); m != nil {
					fields = tengoMapToGoMap(m)
				}
			}
			_ = api.Log(ctx, level, msg, fields)
			return tengo.UndefinedValue, nil
		}
	}

	// `error` is reserved by the Tengo parser as a selector token, so
	// `log.error("…")` is a parse error even though the function exists.
	// We register both names: `log.err` is the reachable alias for scripts.
	// (Direct API callers — gRPC, internal Go — still use the `error` name.)
	errFn := makeLogger("error")
	return map[string]tengo.Object{
		"info":  &tengo.UserFunction{Name: "info", Value: makeLogger("info")},
		"warn":  &tengo.UserFunction{Name: "warn", Value: makeLogger("warn")},
		"error": &tengo.UserFunction{Name: "error", Value: errFn},
		"err":   &tengo.UserFunction{Name: "err", Value: errFn},
		"debug": &tengo.UserFunction{Name: "debug", Value: makeLogger("debug")},
	}
}

// assetsModule exposes read-only access to files inside the theme/extension
// root (the parent of the scripts directory). Used by themes to ship per-form
// HTML layouts, email templates, JSON fixtures, etc. as plain files instead of
// inlining them in theme.tengo.
//
// Path traversal outside the root is rejected. Empty or absolute paths are
// rejected.
func assetsModule(scriptsDir string) map[string]tengo.Object {
	root := ""
	if scriptsDir != "" {
		root = filepath.Dir(scriptsDir)
	}
	resolve := func(rel string) (string, error) {
		if root == "" {
			return "", fmt.Errorf("assets: no theme/extension root available")
		}
		if rel == "" {
			return "", fmt.Errorf("assets: path required")
		}
		if filepath.IsAbs(rel) {
			return "", fmt.Errorf("assets: absolute paths not allowed")
		}
		clean := filepath.Clean(filepath.Join(root, rel))
		rootAbs, err := filepath.Abs(root)
		if err != nil {
			return "", err
		}
		cleanAbs, err := filepath.Abs(clean)
		if err != nil {
			return "", err
		}
		if !strings.HasPrefix(cleanAbs, rootAbs+string(filepath.Separator)) && cleanAbs != rootAbs {
			return "", fmt.Errorf("assets: path escapes theme root")
		}
		return clean, nil
	}
	return map[string]tengo.Object{
		"read": &tengo.UserFunction{Name: "read", Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) < 1 {
				return wrapError(fmt.Errorf("assets.read: path required")), nil
			}
			rel := tengoToString(args[0])
			path, err := resolve(rel)
			if err != nil {
				return wrapError(err), nil
			}
			b, err := os.ReadFile(path)
			if err != nil {
				return wrapError(err), nil
			}
			return &tengo.String{Value: string(b)}, nil
		}},
		"exists": &tengo.UserFunction{Name: "exists", Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) < 1 {
				return tengo.FalseValue, nil
			}
			rel := tengoToString(args[0])
			path, err := resolve(rel)
			if err != nil {
				return tengo.FalseValue, nil
			}
			if _, err := os.Stat(path); err != nil {
				return tengo.FalseValue, nil
			}
			return tengo.TrueValue, nil
		}},
	}
}

func helpersModule() map[string]tengo.Object {
	return map[string]tengo.Object{
		"json_encode": &tengo.UserFunction{Name: "json_encode", Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) < 1 {
				return &tengo.String{Value: "null"}, nil
			}
			v := tengoObjToGo(args[0])
			b, err := json.Marshal(v)
			if err != nil {
				return &tengo.String{Value: "null"}, nil
			}
			return &tengo.String{Value: string(b)}, nil
		}},
		"json_decode": &tengo.UserFunction{Name: "json_decode", Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) < 1 {
				return tengo.UndefinedValue, nil
			}
			s := tengoToString(args[0])
			if s == "" {
				return tengo.UndefinedValue, nil
			}
			var v any
			if err := json.Unmarshal([]byte(s), &v); err != nil {
				return tengo.UndefinedValue, nil
			}
			return rawToTengo(v), nil
		}},
	}
}

func routingModule(api CoreAPI, ctx context.Context, renderCtx interface{}) map[string]tengo.Object {
	var nodeCtx map[string]interface{}
	var appCtx map[string]interface{}

	if rctx, ok := renderCtx.(map[string]interface{}); ok {
		if n, ok := rctx["node"].(map[string]interface{}); ok {
			nodeCtx = n
		}
		if a, ok := rctx["app"].(map[string]interface{}); ok {
			appCtx = a
		}
	}

	return map[string]tengo.Object{
		"is_404": &tengo.UserFunction{Name: "is_404", Value: func(args ...tengo.Object) (tengo.Object, error) {
			if nodeCtx == nil {
				return tengo.FalseValue, nil
			}
			slug, _ := nodeCtx["slug"].(string)
			if slug == "404" {
				return tengo.TrueValue, nil
			}
			return tengo.FalseValue, nil
		}},
		"is_homepage": &tengo.UserFunction{Name: "is_homepage", Value: func(args ...tengo.Object) (tengo.Object, error) {
			if nodeCtx == nil {
				return tengo.FalseValue, nil
			}
			currentURL, _ := nodeCtx["full_url"].(string)
			if currentURL == "/" || currentURL == "" {
				// Check via settings
				val, err := api.GetSetting(ctx, "homepage_node_id")
				if err != nil || val == "" {
					return tengo.FalseValue, nil
				}
				return tengo.TrueValue, nil
			}
			return tengo.FalseValue, nil
		}},
		"site_setting": &tengo.UserFunction{Name: "site_setting", Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) < 1 {
				return tengo.UndefinedValue, nil
			}
			key := tengoToString(args[0])
			// Try cached app context first
			if appCtx != nil {
				if settings, ok := appCtx["settings"].(map[string]interface{}); ok {
					if val, ok := settings[key]; ok {
						if s, ok := val.(string); ok {
							return &tengo.String{Value: s}, nil
						}
					}
				}
			}
			// Fall back to API
			val, err := api.GetSetting(ctx, key)
			if err != nil {
				return tengo.UndefinedValue, nil
			}
			return &tengo.String{Value: val}, nil
		}},
	}
}

func routingModulePlaceholder() map[string]tengo.Object {
	return map[string]tengo.Object{
		"is_404": &tengo.UserFunction{Name: "is_404", Value: func(args ...tengo.Object) (tengo.Object, error) {
			return tengo.FalseValue, nil
		}},
		"is_homepage": &tengo.UserFunction{Name: "is_homepage", Value: func(args ...tengo.Object) (tengo.Object, error) {
			return tengo.FalseValue, nil
		}},
		"site_setting": &tengo.UserFunction{Name: "site_setting", Value: func(args ...tengo.Object) (tengo.Object, error) {
			return tengo.UndefinedValue, nil
		}},
	}
}
