package coreapi

import (
	"context"
	"fmt"
	"time"

	"github.com/d5/tengo/v2"
)

// This file owns the two "schema definition" Tengo modules —
// core/nodetypes (where themes register their custom content types)
// and core/taxonomies (term-based grouping definitions). They share
// the tengoToField helper from tengo_nodes.go and have similar
// register/get/list shapes, so they live together.

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
			warnFieldSchemaShape(api, ctx, "nodetypes.register["+input.Slug+"]", input.FieldSchema)
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
			warnFieldSchemaShape(api, ctx, "taxonomies.register["+input.Slug+"]", input.FieldSchema)
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
			warnFieldSchemaShape(api, ctx, "taxonomies.update["+slug+"]", input.FieldSchema)
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
