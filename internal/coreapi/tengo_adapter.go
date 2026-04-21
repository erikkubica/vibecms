package coreapi

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/d5/tengo/v2"
	"github.com/d5/tengo/v2/stdlib"
)

// ScriptCallbacks allows the ScriptEngine to communicate back to the CMS.
type ScriptCallbacks struct {
	OnRoute  func(method, path, scriptPath string)
	OnFilter func(name, scriptPath string, priority int)
	OnEvent  func(action, scriptPath string, priority int)
}

// BuildTengoModules creates a new ModuleMap and registers all VibeCMS modules.
func BuildTengoModules(api CoreAPI, caller CallerInfo, renderCtx interface{}, scriptsDir string, cb *ScriptCallbacks) *tengo.ModuleMap {
	modules := tengo.NewModuleMap()
	RegisterModules(modules, api, caller, renderCtx, scriptsDir, cb)
	return modules
}

// RegisterModules adds all VibeCMS core modules to the Tengo module map.
func RegisterModules(modules *tengo.ModuleMap, api CoreAPI, caller CallerInfo, renderCtx interface{}, scriptsDir string, cb *ScriptCallbacks) {
	// Use WithCaller to associate the script caller with the context
	ctx := WithCaller(context.Background(), caller)

	// Standard library
	for name, mod := range stdlib.BuiltinModules {
		modules.AddBuiltinModule(name, mod)
	}

	// VibeCMS Core modules
	modules.AddBuiltinModule("core/nodes", nodesModule(api, ctx))
	modules.AddBuiltinModule("core/menus", menusModule(api, ctx))
	modules.AddBuiltinModule("core/routes", routesModule(cb))
	modules.AddBuiltinModule("core/filters", filtersModule(cb))
	modules.AddBuiltinModule("core/http", httpFetchModule(api, ctx))
	modules.AddBuiltinModule("core/log", logModule(api, ctx))
	modules.AddBuiltinModule("core/nodetypes", nodeTypesModule(api, ctx))
	modules.AddBuiltinModule("core/taxonomies", taxonomiesModule(api, ctx))
	modules.AddBuiltinModule("core/helpers", helpersModule())
	modules.AddBuiltinModule("core/events", eventsModule(api, ctx, cb))
	modules.AddBuiltinModule("core/settings", settingsModule(api, ctx))

	if renderCtx != nil {
		modules.AddBuiltinModule("core/routing", routingModule(api, ctx, renderCtx))
	} else {
		modules.AddBuiltinModule("core/routing", routingModulePlaceholder())
	}

	// Load any source modules from the scripts directory
	loadSourceModules(modules, scriptsDir)
}

// ---------------------------------------------------------------------------
// core/nodes
// ---------------------------------------------------------------------------

func nodesModule(api CoreAPI, ctx context.Context) map[string]tengo.Object {
	return map[string]tengo.Object{
		"get": &tengo.UserFunction{Name: "get", Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) < 1 {
				return wrapError(fmt.Errorf("nodes.get: requires id argument")), nil
			}
			id := uint(tengoToInt(args[0]))
			n, err := api.GetNode(ctx, id)
			if err != nil {
				return wrapError(err), nil
			}
			return nodeToTengoObj(n), nil
		}},
		"query": &tengo.UserFunction{Name: "query", Value: func(args ...tengo.Object) (tengo.Object, error) {
			q := &NodeQuery{}
			if len(args) > 0 {
				if m := getTengoMap(args[0]); m != nil {
					applyNodeQueryFromMap(m, q)
				}
			}
			list, err := api.QueryNodes(ctx, *q)
			if err != nil {
				return wrapError(err), nil
			}
			nodes := make([]tengo.Object, len(list.Nodes))
			for i, n := range list.Nodes {
				nodes[i] = nodeToTengoObj(n)
			}
			return &tengo.ImmutableMap{Value: map[string]tengo.Object{
				"nodes": &tengo.ImmutableArray{Value: nodes},
				"total": &tengo.Int{Value: list.Total},
			}}, nil
		}},
		"create": &tengo.UserFunction{Name: "create", Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) < 1 {
				return wrapError(fmt.Errorf("nodes.create: requires input argument")), nil
			}
			m := getTengoMap(args[0])
			if m == nil {
				return wrapError(fmt.Errorf("nodes.create: input must be a map")), nil
			}
			n, err := api.CreateNode(ctx, nodeInputFromMap(m))
			if err != nil {
				return wrapError(err), nil
			}
			return nodeToTengoObj(n), nil
		}},
		"update": &tengo.UserFunction{Name: "update", Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) < 2 {
				return wrapError(fmt.Errorf("nodes.update: requires id and input arguments")), nil
			}
			id := uint(tengoToInt(args[0]))
			m := getTengoMap(args[1])
			if m == nil {
				return wrapError(fmt.Errorf("nodes.update: input must be a map")), nil
			}
			n, err := api.UpdateNode(ctx, id, nodeInputFromMap(m))
			if err != nil {
				return wrapError(err), nil
			}
			return nodeToTengoObj(n), nil
		}},
		"delete": &tengo.UserFunction{Name: "delete", Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) < 1 {
				return wrapError(fmt.Errorf("nodes.delete: requires id argument")), nil
			}
			id := uint(tengoToInt(args[0]))
			if err := api.DeleteNode(ctx, id); err != nil {
				return wrapError(err), nil
			}
			return tengo.UndefinedValue, nil
		}},
	}
}

// ---------------------------------------------------------------------------
// core/menus
// ---------------------------------------------------------------------------

func menusModule(api CoreAPI, ctx context.Context) map[string]tengo.Object {
	return map[string]tengo.Object{
		"get": &tengo.UserFunction{Name: "get", Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) < 1 {
				return wrapError(fmt.Errorf("menus.get: requires slug argument")), nil
			}
			slug := tengoToString(args[0])
			menu, err := api.GetMenu(ctx, slug)
			if err != nil {
				return wrapError(err), nil
			}
			return menuToTengoObj(menu), nil
		}},
		"list": &tengo.UserFunction{Name: "list", Value: func(args ...tengo.Object) (tengo.Object, error) {
			list, err := api.GetMenus(ctx)
			if err != nil {
				return wrapError(err), nil
			}
			results := make([]tengo.Object, len(list))
			for i, m := range list {
				results[i] = menuToTengoObj(m)
			}
			return &tengo.ImmutableArray{Value: results}, nil
		}},
	}
}

// ---------------------------------------------------------------------------
// core/nodetypes
// ---------------------------------------------------------------------------

func nodeTypesModule(api CoreAPI, ctx context.Context) map[string]tengo.Object {
	return map[string]tengo.Object{
		"register": &tengo.UserFunction{Name: "register", Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) < 1 {
				return wrapError(fmt.Errorf("nodetypes.register: requires input argument")), nil
			}
			m := getTengoMap(args[0])
			if m == nil {
				return wrapError(fmt.Errorf("nodetypes.register: input must be a map")), nil
			}
			input := nodeTypeInputFromMap(m)
			res, err := api.RegisterNodeType(ctx, input)
			if err != nil {
				return wrapError(err), nil
			}
			return nodeTypeToTengoObj(res), nil
		}},
		"get": &tengo.UserFunction{Name: "get", Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) < 1 {
				return wrapError(fmt.Errorf("nodetypes.get: requires slug argument")), nil
			}
			slug := tengoToString(args[0])
			res, err := api.GetNodeType(ctx, slug)
			if err != nil {
				return wrapError(err), nil
			}
			return nodeTypeToTengoObj(res), nil
		}},
		"list": &tengo.UserFunction{Name: "list", Value: func(args ...tengo.Object) (tengo.Object, error) {
			list, err := api.ListNodeTypes(ctx)
			if err != nil {
				return wrapError(err), nil
			}
			results := make([]tengo.Object, len(list))
			for i, m := range list {
				results[i] = nodeTypeToTengoObj(m)
			}
			return &tengo.ImmutableArray{Value: results}, nil
		}},
	}
}

func nodeTypeToTengoObj(nt *NodeType) tengo.Object {
	if nt == nil {
		return tengo.UndefinedValue
	}
	fields := make([]tengo.Object, len(nt.FieldSchema))
	for i, f := range nt.FieldSchema {
		opts := make([]tengo.Object, len(f.Options))
		for j, o := range f.Options {
			opts[j] = goToTengoObj(o)
		}
		fields[i] = &tengo.ImmutableMap{Value: map[string]tengo.Object{
			"name":     &tengo.String{Value: f.Name},
			"label":    &tengo.String{Value: f.Label},
			"type":     &tengo.String{Value: f.Type},
			"required": boolToTengo(f.Required),
			"options":  &tengo.ImmutableArray{Value: opts},
		}}
	}
	prefixes := make(map[string]tengo.Object, len(nt.URLPrefixes))
	for k, v := range nt.URLPrefixes {
		prefixes[k] = &tengo.String{Value: v}
	}
	taxes := make([]tengo.Object, len(nt.Taxonomies))
	for i, t := range nt.Taxonomies {
		taxes[i] = &tengo.ImmutableMap{Value: map[string]tengo.Object{
			"slug":     &tengo.String{Value: t.Slug},
			"label":    &tengo.String{Value: t.Label},
			"multiple": boolToTengo(t.Multiple),
		}}
	}
	return &tengo.ImmutableMap{Value: map[string]tengo.Object{
		"id":           &tengo.Int{Value: int64(nt.ID)},
		"slug":         &tengo.String{Value: nt.Slug},
		"label":        &tengo.String{Value: nt.Label},
		"icon":         &tengo.String{Value: nt.Icon},
		"description":  &tengo.String{Value: nt.Description},
		"taxonomies":   &tengo.ImmutableArray{Value: taxes},
		"field_schema": &tengo.ImmutableArray{Value: fields},
		"url_prefixes": &tengo.ImmutableMap{Value: prefixes},
	}}
}

func nodeTypeInputFromMap(m map[string]tengo.Object) NodeTypeInput {
	input := NodeTypeInput{}
	if v, ok := m["slug"]; ok {
		input.Slug = tengoToString(v)
	}
	if v, ok := m["label"]; ok {
		input.Label = tengoToString(v)
	}
	if v, ok := m["icon"]; ok {
		input.Icon = tengoToString(v)
	}
	if v, ok := m["description"]; ok {
		input.Description = tengoToString(v)
	}
	if v, ok := m["taxonomies"]; ok {
		if arr, ok := v.(*tengo.Array); ok {
			for _, item := range arr.Value {
				if tm := getTengoMap(item); tm != nil {
					input.Taxonomies = append(input.Taxonomies, TaxonomyDefinition{
						Slug:     tengoToString(tm["slug"]),
						Label:    tengoToString(tm["label"]),
						Multiple: tengoToBool(tm["multiple"]),
					})
				}
			}
		}
	}
	if v, ok := m["field_schema"]; ok {
		if arr, ok := v.(*tengo.Array); ok {
			for _, item := range arr.Value {
				if fm := getTengoMap(item); fm != nil {
					input.FieldSchema = append(input.FieldSchema, tengoToField(fm))
				}
			}
		}
	}
	if v, ok := m["url_prefixes"]; ok {
		if pm := getTengoMap(v); pm != nil {
			input.URLPrefixes = make(map[string]string, len(pm))
			for k, pv := range pm {
				input.URLPrefixes[k] = tengoToString(pv)
			}
		}
	}
	return input
}

// ---------------------------------------------------------------------------
// core/taxonomies
// ---------------------------------------------------------------------------

func taxonomiesModule(api CoreAPI, ctx context.Context) map[string]tengo.Object {
	return map[string]tengo.Object{
		"register": &tengo.UserFunction{Name: "register", Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) < 1 {
				return wrapError(fmt.Errorf("taxonomies.register: requires input argument")), nil
			}
			m := getTengoMap(args[0])
			if m == nil {
				return wrapError(fmt.Errorf("taxonomies.register: input must be a map")), nil
			}
			input := taxonomyInputFromMap(m)
			res, err := api.RegisterTaxonomy(ctx, input)
			if err != nil {
				return wrapError(err), nil
			}
			return taxonomyToTengoObj(res), nil
		}},
		"get": &tengo.UserFunction{Name: "get", Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) < 1 {
				return wrapError(fmt.Errorf("taxonomies.get: requires slug argument")), nil
			}
			slug := tengoToString(args[0])
			res, err := api.GetTaxonomy(ctx, slug)
			if err != nil {
				return wrapError(err), nil
			}
			return taxonomyToTengoObj(res), nil
		}},
		"list": &tengo.UserFunction{Name: "list", Value: func(args ...tengo.Object) (tengo.Object, error) {
			list, err := api.ListTaxonomies(ctx)
			if err != nil {
				return wrapError(err), nil
			}
			results := make([]tengo.Object, len(list))
			for i, t := range list {
				results[i] = taxonomyToTengoObj(t)
			}
			return &tengo.ImmutableArray{Value: results}, nil
		}},
		"update": &tengo.UserFunction{Name: "update", Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) < 2 {
				return wrapError(fmt.Errorf("taxonomies.update: requires slug and input arguments")), nil
			}
			slug := tengoToString(args[0])
			m := getTengoMap(args[1])
			if m == nil {
				return wrapError(fmt.Errorf("taxonomies.update: input must be a map")), nil
			}
			input := taxonomyInputFromMap(m)
			res, err := api.UpdateTaxonomy(ctx, slug, input)
			if err != nil {
				return wrapError(err), nil
			}
			return taxonomyToTengoObj(res), nil
		}},
		"delete": &tengo.UserFunction{Name: "delete", Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) < 1 {
				return wrapError(fmt.Errorf("taxonomies.delete: requires slug argument")), nil
			}
			slug := tengoToString(args[0])
			err := api.DeleteTaxonomy(ctx, slug)
			if err != nil {
				return wrapError(err), nil
			}
			return tengo.UndefinedValue, nil
		}},
	}
}

func taxonomyToTengoObj(t *Taxonomy) tengo.Object {
	if t == nil {
		return tengo.UndefinedValue
	}
	m := map[string]tengo.Object{
		"id":           &tengo.Int{Value: int64(t.ID)},
		"slug":         &tengo.String{Value: t.Slug},
		"label":        &tengo.String{Value: t.Label},
		"description":  &tengo.String{Value: t.Description},
		"hierarchical": boolToTengo(t.Hierarchical),
		"show_ui":      boolToTengo(t.ShowUI),
		"created_at":   &tengo.String{Value: t.CreatedAt.Format(time.RFC3339)},
		"updated_at":   &tengo.String{Value: t.UpdatedAt.Format(time.RFC3339)},
	}
	if t.NodeTypes != nil {
		ntArr := make([]tengo.Object, len(t.NodeTypes))
		for i, nt := range t.NodeTypes {
			ntArr[i] = &tengo.String{Value: nt}
		}
		m["node_types"] = &tengo.ImmutableArray{Value: ntArr}
	}
	if t.FieldSchema != nil {
		m["field_schema"] = goToTengoObj(t.FieldSchema)
	}
	return &tengo.ImmutableMap{Value: m}
}

func taxonomyInputFromMap(m map[string]tengo.Object) TaxonomyInput {
	input := TaxonomyInput{}
	if v, ok := m["slug"]; ok {
		input.Slug = tengoToString(v)
	}
	if v, ok := m["label"]; ok {
		input.Label = tengoToString(v)
	}
	if v, ok := m["description"]; ok {
		input.Description = tengoToString(v)
	}
	if v, ok := m["hierarchical"]; ok {
		b := tengoToBool(v)
		input.Hierarchical = &b
	}
	if v, ok := m["show_ui"]; ok {
		b := tengoToBool(v)
		input.ShowUI = &b
	}
	if v, ok := m["node_types"]; ok {
		if arr, ok := v.(*tengo.Array); ok {
			for _, item := range arr.Value {
				input.NodeTypes = append(input.NodeTypes, tengoToString(item))
			}
		} else if arr, ok := v.(*tengo.ImmutableArray); ok {
			for _, item := range arr.Value {
				input.NodeTypes = append(input.NodeTypes, tengoToString(item))
			}
		}
	}
	if v, ok := m["field_schema"]; ok {
		if arr, ok := v.(*tengo.Array); ok {
			for _, item := range arr.Value {
				if fm := getTengoMap(item); fm != nil {
					input.FieldSchema = append(input.FieldSchema, tengoToField(fm))
				}
			}
		} else if arr, ok := v.(*tengo.ImmutableArray); ok {
			for _, item := range arr.Value {
				if fm := getTengoMap(item); fm != nil {
					input.FieldSchema = append(input.FieldSchema, tengoToField(fm))
				}
			}
		}
	}
	return input
}

// ---------------------------------------------------------------------------
// core/routes (placeholder -- engine handles registration)
// ---------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------
// core/filters (placeholder -- engine handles registration)
// ---------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------
// core/settings
// ---------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------
// core/events
// ---------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------
// core/http (outbound fetch)
// ---------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------
// core/log
// ---------------------------------------------------------------------------

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

	return map[string]tengo.Object{
		"info":  &tengo.UserFunction{Name: "info", Value: makeLogger("info")},
		"warn":  &tengo.UserFunction{Name: "warn", Value: makeLogger("warn")},
		"error": &tengo.UserFunction{Name: "error", Value: makeLogger("error")},
		"debug": &tengo.UserFunction{Name: "debug", Value: makeLogger("debug")},
	}
}

// ---------------------------------------------------------------------------
// core/helpers
// ---------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------
// core/routing (render-context-aware)
// ---------------------------------------------------------------------------

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

// ===========================================================================
// Helper functions
// ===========================================================================

func tengoToBool(obj tengo.Object) bool {
	if b, ok := obj.(*tengo.Bool); ok {
		return !b.IsFalsy()
	}
	return false
}

func tengoToField(fm map[string]tengo.Object) NodeTypeField {
	f := NodeTypeField{
		Name:  tengoToString(fm["name"]),
		Label: tengoToString(fm["label"]),
		Type:  tengoToString(fm["type"]),
	}
	if f.Name == "" {
		f.Name = tengoToString(fm["key"])
	}
	if rv, ok := fm["required"]; ok {
		f.Required = tengoToBool(rv)
	}
	if ov, ok := fm["options"]; ok {
		if oarr, ok := ov.(*tengo.Array); ok {
			for _, o := range oarr.Value {
				if m, ok := o.(*tengo.Map); ok {
					f.Options = append(f.Options, tengoMapToGoMap(m.Value))
				} else {
					f.Options = append(f.Options, tengoToString(o))
				}
			}
		}
	}
	if sfv, ok := fm["sub_fields"]; ok {
		if sfarr, ok := sfv.(*tengo.Array); ok {
			for _, sf := range sfarr.Value {
				if sm, ok := sf.(*tengo.Map); ok {
					f.SubFields = append(f.SubFields, tengoToField(sm.Value))
				}
			}
		}
	}
	if dv, ok := fm["default"]; ok {
		f.Default = tengoObjToGo(dv)
	}
	if hv, ok := fm["help"]; ok {
		f.Help = tengoToString(hv)
	}
	return f
}

func boolToTengo(b bool) tengo.Object {
	if b {
		return tengo.TrueValue
	}
	return tengo.FalseValue
}

// wrapError wraps a Go error into a Tengo Error object.
func wrapError(err error) tengo.Object {
	return &tengo.Error{Value: &tengo.String{Value: err.Error()}}
}

// tengoMapToGoMap converts a Tengo map (map[string]tengo.Object) to map[string]any.
func tengoMapToGoMap(m map[string]tengo.Object) map[string]any {
	result := make(map[string]any, len(m))
	for k, v := range m {
		result[k] = tengoObjToGo(v)
	}
	return result
}

// tengoObjToGo converts a Tengo object to a Go value.
func tengoObjToGo(obj tengo.Object) any {
	if obj == nil {
		return nil
	}
	switch v := obj.(type) {
	case *tengo.String:
		return v.Value
	case *tengo.Int:
		return v.Value
	case *tengo.Float:
		return v.Value
	case *tengo.Bool:
		return !v.IsFalsy()
	case *tengo.Map:
		return tengoMapToGoMap(v.Value)
	case *tengo.ImmutableMap:
		return tengoMapToGoMap(v.Value)
	case *tengo.Array:
		arr := make([]any, len(v.Value))
		for i, item := range v.Value {
			arr[i] = tengoObjToGo(item)
		}
		return arr
	case *tengo.ImmutableArray:
		arr := make([]any, len(v.Value))
		for i, item := range v.Value {
			arr[i] = tengoObjToGo(item)
		}
		return arr
	case *tengo.Undefined:
		return nil
	default:
		return obj.String()
	}
}

// goToTengoObj converts a Go value to a Tengo object by marshalling to JSON
// and then converting. This handles arbitrary Go types.
func goToTengoObj(v any) tengo.Object {
	if v == nil {
		return tengo.UndefinedValue
	}
	b, err := json.Marshal(v)
	if err != nil {
		return tengo.UndefinedValue
	}
	var raw any
	if err := json.Unmarshal(b, &raw); err != nil {
		return tengo.UndefinedValue
	}
	return rawToTengo(raw)
}

// rawToTengo recursively converts JSON-unmarshalled types to Tengo objects.
func rawToTengo(v any) tengo.Object {
	if v == nil {
		return tengo.UndefinedValue
	}
	switch val := v.(type) {
	case string:
		return &tengo.String{Value: val}
	case float64:
		// Check if this is actually an integer
		if val == float64(int64(val)) {
			return &tengo.Int{Value: int64(val)}
		}
		return &tengo.Float{Value: val}
	case bool:
		if val {
			return tengo.TrueValue
		}
		return tengo.FalseValue
	case map[string]any:
		m := make(map[string]tengo.Object, len(val))
		for k, item := range val {
			m[k] = rawToTengo(item)
		}
		return &tengo.ImmutableMap{Value: m}
	case []any:
		arr := make([]tengo.Object, len(val))
		for i, item := range val {
			arr[i] = rawToTengo(item)
		}
		return &tengo.ImmutableArray{Value: arr}
	default:
		return tengo.UndefinedValue
	}
}

// applyNodeQueryFromMap extracts query parameters from a Tengo map into a NodeQuery.
func applyNodeQueryFromMap(m map[string]tengo.Object, q *NodeQuery) {
	if v, ok := m["node_type"]; ok {
		q.NodeType = tengoToString(v)
	}
	if v, ok := m["status"]; ok {
		q.Status = tengoToString(v)
	}
	if v, ok := m["language_code"]; ok {
		q.LanguageCode = tengoToString(v)
	}
	if v, ok := m["slug"]; ok {
		q.Slug = tengoToString(v)
	}
	if v, ok := m["search"]; ok {
		q.Search = tengoToString(v)
	}
	if v, ok := m["limit"]; ok {
		limit := tengoToInt(v)
		if limit > 0 && limit <= 500 {
			q.Limit = limit
		}
	}
	if v, ok := m["offset"]; ok {
		q.Offset = tengoToInt(v)
	}
	if v, ok := m["order_by"]; ok {
		q.OrderBy = tengoToString(v)
	}
	if v, ok := m["category"]; ok {
		q.Category = tengoToString(v)
	}
	if v, ok := m["tax_query"]; ok {
		if tq := getTengoMap(v); tq != nil {
			q.TaxQuery = make(map[string][]string)
			for tax, termsObj := range tq {
				if arr, ok := termsObj.(*tengo.Array); ok {
					var terms []string
					for _, item := range arr.Value {
						terms = append(terms, tengoToString(item))
					}
					q.TaxQuery[tax] = terms
				} else if s, ok := termsObj.(*tengo.String); ok {
					q.TaxQuery[tax] = []string{s.Value}
				}
			}
		}
	}
	// Support page/per_page for backward compatibility
	if v, ok := m["page"]; ok {
		page := tengoToInt(v)
		if page > 1 {
			perPage := q.Limit
			if perPage <= 0 {
				perPage = 50
			}
			q.Offset = (page - 1) * perPage
		}
	}
	if v, ok := m["per_page"]; ok {
		pp := tengoToInt(v)
		if pp > 0 && pp <= 500 {
			q.Limit = pp
		}
	}
	if v, ok := m["parent_id"]; ok {
		pid := tengoToInt(v)
		if pid > 0 {
			u := uint(pid)
			q.ParentID = &u
		}
	}
}

// nodeInputFromMap converts a Tengo map to a NodeInput struct.
func nodeInputFromMap(m map[string]tengo.Object) NodeInput {
	input := NodeInput{}
	if v, ok := m["title"]; ok {
		input.Title = tengoToString(v)
	}
	if v, ok := m["slug"]; ok {
		input.Slug = tengoToString(v)
	}
	if v, ok := m["node_type"]; ok {
		input.NodeType = tengoToString(v)
	}
	if v, ok := m["status"]; ok {
		input.Status = tengoToString(v)
	}
	if v, ok := m["language_code"]; ok {
		input.LanguageCode = tengoToString(v)
	}
	if v, ok := m["parent_id"]; ok {
		pid := tengoToInt(v)
		if pid > 0 {
			u := uint(pid)
			input.ParentID = &u
		}
	}
	if v, ok := m["blocks_data"]; ok {
		input.BlocksData = tengoObjToGo(v)
	}
	if v, ok := m["featured_image"]; ok {
		input.FeaturedImage = tengoObjToGo(v)
	}
	if v, ok := m["excerpt"]; ok {
		input.Excerpt = tengoToString(v)
	}
	if v, ok := m["taxonomies"]; ok {
		if txMap := getTengoMap(v); txMap != nil {
			input.Taxonomies = make(map[string][]string)
			for tax, termsObj := range txMap {
				if arr, ok := termsObj.(*tengo.Array); ok {
					var terms []string
					for _, item := range arr.Value {
						terms = append(terms, tengoToString(item))
					}
					input.Taxonomies[tax] = terms
				} else if s, ok := termsObj.(*tengo.String); ok {
					input.Taxonomies[tax] = []string{s.Value}
				}
			}
		}
	}
	if v, ok := m["fields_data"]; ok {
		if fd := getTengoMap(v); fd != nil {
			input.FieldsData = tengoMapToGoMap(fd)
		}
	}
	if v, ok := m["seo_settings"]; ok {
		if sm := getTengoMap(v); sm != nil {
			input.SeoSettings = make(map[string]string, len(sm))
			for k, sv := range sm {
				input.SeoSettings[k] = tengoToString(sv)
			}
		}
	}
	return input
}

// nodeToTengoObj converts a CoreAPI Node to a Tengo ImmutableMap.
func nodeToTengoObj(n *Node) tengo.Object {
	if n == nil {
		return tengo.UndefinedValue
	}
	m := map[string]tengo.Object{
		"id":            &tengo.Int{Value: int64(n.ID)},
		"uuid":          &tengo.String{Value: n.UUID},
		"node_type":     &tengo.String{Value: n.NodeType},
		"status":        &tengo.String{Value: n.Status},
		"language_code": &tengo.String{Value: n.LanguageCode},
		"slug":          &tengo.String{Value: n.Slug},
		"full_url":      &tengo.String{Value: n.FullURL},
		"title":         &tengo.String{Value: n.Title},
		"created_at":    &tengo.String{Value: n.CreatedAt.Format("2006-01-02T15:04:05Z")},
		"updated_at":    &tengo.String{Value: n.UpdatedAt.Format("2006-01-02T15:04:05Z")},
	}

	if n.ParentID != nil {
		m["parent_id"] = &tengo.Int{Value: int64(*n.ParentID)}
	} else {
		m["parent_id"] = tengo.UndefinedValue
	}

	if n.PublishedAt != nil {
		m["published_at"] = &tengo.String{Value: n.PublishedAt.Format("2006-01-02T15:04:05Z")}
	} else {
		m["published_at"] = tengo.UndefinedValue
	}

	if n.BlocksData != nil {
		m["blocks_data"] = goToTengoObj(n.BlocksData)
	}
	if n.FeaturedImage != nil {
		m["featured_image"] = goToTengoObj(n.FeaturedImage)
	}
	if n.Excerpt != "" {
		m["excerpt"] = &tengo.String{Value: n.Excerpt}
	}
	if n.Taxonomies != nil {
		m["taxonomies"] = goToTengoObj(n.Taxonomies)
	}
	if n.FieldsData != nil {
		m["fields_data"] = goToTengoObj(n.FieldsData)
	}
	if n.SeoSettings != nil {
		m["seo_settings"] = goToTengoObj(n.SeoSettings)
	}

	return &tengo.ImmutableMap{Value: m}
}

// menuToTengoObj converts a CoreAPI Menu to a Tengo ImmutableMap.
func menuToTengoObj(menu *Menu) tengo.Object {
	if menu == nil {
		return tengo.UndefinedValue
	}
	m := map[string]tengo.Object{
		"id":   &tengo.Int{Value: int64(menu.ID)},
		"slug": &tengo.String{Value: menu.Slug},
		"name": &tengo.String{Value: menu.Name},
	}
	m["items"] = menuItemsToTengoObj(menu.Items)
	return &tengo.ImmutableMap{Value: m}
}

// menuItemsToTengoObj converts CoreAPI MenuItems to a Tengo array.
func menuItemsToTengoObj(items []MenuItem) tengo.Object {
	arr := make([]tengo.Object, len(items))
	for i, item := range items {
		im := map[string]tengo.Object{
			"id":       &tengo.Int{Value: int64(item.ID)},
			"label":    &tengo.String{Value: item.Label},
			"url":      &tengo.String{Value: item.URL},
			"target":   &tengo.String{Value: item.Target},
			"position": &tengo.Int{Value: int64(item.Position)},
			"children": menuItemsToTengoObj(item.Children),
		}
		if item.ParentID != nil {
			im["parent_id"] = &tengo.Int{Value: int64(*item.ParentID)}
		} else {
			im["parent_id"] = tengo.UndefinedValue
		}
		arr[i] = &tengo.ImmutableMap{Value: im}
	}
	return &tengo.ImmutableArray{Value: arr}
}

// ---------------------------------------------------------------------------
// Tengo value extraction helpers
// ---------------------------------------------------------------------------

// tengoToString extracts a string from a Tengo object.
func tengoToString(obj tengo.Object) string {
	if s, ok := obj.(*tengo.String); ok {
		return s.Value
	}
	return ""
}

// tengoToInt extracts an int from a Tengo object.
func tengoToInt(obj tengo.Object) int {
	switch v := obj.(type) {
	case *tengo.Int:
		return int(v.Value)
	case *tengo.Float:
		return int(v.Value)
	default:
		return 0
	}
}

// getTengoMap extracts the underlying map from a Tengo Map or ImmutableMap.
func getTengoMap(obj tengo.Object) map[string]tengo.Object {
	switch v := obj.(type) {
	case *tengo.Map:
		return v.Value
	case *tengo.ImmutableMap:
		return v.Value
	default:
		return nil
	}
}

// ---------------------------------------------------------------------------
// Source module loader
// ---------------------------------------------------------------------------

// loadSourceModules recursively scans the scripts directory and adds all .tengo
// files as source modules with "./" prefix paths.
func loadSourceModules(modules *tengo.ModuleMap, scriptsDir string) {
	if scriptsDir == "" {
		return
	}

	filepath.Walk(scriptsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".tengo") {
			return nil
		}

		// Skip entry scripts
		base := filepath.Base(path)
		if (base == "theme.tengo" || base == "extension.tengo") && filepath.Dir(path) == scriptsDir {
			return nil
		}

		relPath, err := filepath.Rel(scriptsDir, path)
		if err != nil {
			return nil
		}

		moduleName := "./" + strings.TrimSuffix(relPath, ".tengo")
		moduleName = strings.ReplaceAll(moduleName, string(filepath.Separator), "/")

		src, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		modules.AddSourceModule(moduleName, src)
		return nil
	})
}
