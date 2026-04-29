package coreapi

import (
	"context"
	"fmt"

	"github.com/d5/tengo/v2"
)

// This file owns the core/nodes Tengo module — the surface scripts
// most often touch. Pulled out of tengo_adapter.go because the
// node-input mapping (nodeInputFromMap), query-param mapping
// (applyNodeQueryFromMap), and node-to-Tengo serialisation
// (nodeToTengoObj) together make up over 300 lines that aren't
// useful to read alongside the other module builders.

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
			warnNodeInputShape(api, ctx, "nodes.create", m)
			warnBlocksDataShape(api, ctx, "nodes.create", m)
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
			warnNodeInputShape(api, ctx, "nodes.update", m)
			warnBlocksDataShape(api, ctx, "nodes.update", m)
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

// tengoToField builds a NodeTypeField from a Tengo map. Lives next to
// nodes because content types feed it but it's also reused by
// taxonomies — the duplication-avoidance trumps strict file naming.
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
	if v, ok := fm["node_type_filter"]; ok {
		f.NodeTypeFilter = tengoToString(v)
	}
	if v, ok := fm["node_types"]; ok {
		if arr, ok := v.(*tengo.Array); ok {
			for _, s := range arr.Value {
				f.NodeTypes = append(f.NodeTypes, tengoToString(s))
			}
		} else if arr, ok := v.(*tengo.ImmutableArray); ok {
			for _, s := range arr.Value {
				f.NodeTypes = append(f.NodeTypes, tengoToString(s))
			}
		}
	}
	if v, ok := fm["multiple"]; ok {
		f.Multiple = tengoToBool(v)
	}
	if v, ok := fm["taxonomy"]; ok {
		f.Taxonomy = tengoToString(v)
	}
	if v, ok := fm["term_node_type"]; ok {
		f.TermNodeType = tengoToString(v)
	}
	return f
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
	// Accept either `layout_slug` (canonical) or `layout` (shorthand) so theme
	// seeds can pin a node to a specific layout. Resolved via slug so it
	// survives theme deactivate/reactivate cycles.
	if v, ok := m["layout_slug"]; ok {
		input.LayoutSlug = tengoToString(v)
	} else if v, ok := m["layout"]; ok {
		input.LayoutSlug = tengoToString(v)
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

// warnNodeInputShape catches the #1 silent data-loss bug: passing top-level
// `fields:` instead of `fields_data:` to nodes.create/update. nodeInputFromMap
// only reads `fields_data`, so `fields` is silently dropped.
func warnNodeInputShape(api CoreAPI, ctx context.Context, op string, m map[string]tengo.Object) {
	if _, hasFields := m["fields"]; hasFields {
		if _, hasFieldsData := m["fields_data"]; !hasFieldsData {
			_ = api.Log(ctx, "warn",
				op+": top-level `fields:` is ignored — did you mean `fields_data:`? (node-level uses fields_data, blocks inside blocks_data use fields)",
				nil)
		}
	}
}

// warnBlocksDataShape catches the inverse: blocks inside blocks_data must use
// `fields:`, not `fields_data:`. Author confusion is symmetric and equally
// silent — the renderer reads block["fields"], so block["fields_data"] is dropped.
func warnBlocksDataShape(api CoreAPI, ctx context.Context, op string, m map[string]tengo.Object) {
	bd, ok := m["blocks_data"]
	if !ok {
		return
	}
	arr, ok := bd.(*tengo.Array)
	if !ok {
		if iarr, ok := bd.(*tengo.ImmutableArray); ok {
			for i, b := range iarr.Value {
				checkBlockFieldsShape(api, ctx, op, i, b)
			}
		}
		return
	}
	for i, b := range arr.Value {
		checkBlockFieldsShape(api, ctx, op, i, b)
	}
}

// warnFieldSchemaShape catches schema misconfigurations at register time:
//   - term-typed fields missing `term_node_type` won't hydrate at render
//   - select-typed fields whose options contain {value,label} objects crash
//     the admin (React error #31). block.json options must be plain strings;
//     this also catches the equivalent mistake in nodetypes.register schemas.
func warnFieldSchemaShape(api CoreAPI, ctx context.Context, op string, fields []NodeTypeField) {
	for _, f := range fields {
		if f.Type == "term" && f.TermNodeType == "" {
			_ = api.Log(ctx, "warn",
				fmt.Sprintf("%s: field %q is type=term but term_node_type is empty — hydration will not match any term row", op, f.Name),
				nil)
		}
		if f.Type == "select" {
			for _, o := range f.Options {
				if _, isMap := o.(map[string]any); isMap {
					_ = api.Log(ctx, "warn",
						fmt.Sprintf("%s: field %q is type=select with object options ({value,label}) — admin requires plain string options", op, f.Name),
						nil)
					break
				}
			}
		}
		if len(f.SubFields) > 0 {
			warnFieldSchemaShape(api, ctx, op+"."+f.Name, f.SubFields)
		}
	}
}

func checkBlockFieldsShape(api CoreAPI, ctx context.Context, op string, idx int, b tengo.Object) {
	bm := getTengoMap(b)
	if bm == nil {
		return
	}
	if _, hasFD := bm["fields_data"]; hasFD {
		if _, hasF := bm["fields"]; !hasF {
			_ = api.Log(ctx, "warn",
				fmt.Sprintf("%s: blocks_data[%d] uses `fields_data:` — blocks inside blocks_data must use `fields:` (only the top-level node uses fields_data)", op, idx),
				nil)
		}
	}
}
