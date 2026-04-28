package coreapi

import (
	"context"
	"fmt"
	"time"

	"github.com/d5/tengo/v2"
)

// core/terms — Tengo bindings for taxonomy term CRUD. Themes and extensions
// use this to seed actual terms (e.g. the "Foodie", "Adventure" rows under
// the trip_tag taxonomy) from a script. The taxonomy DEFINITION is created
// via core/taxonomies; the terms inside it live here.

func termsModule(api CoreAPI, ctx context.Context) map[string]tengo.Object {
	return map[string]tengo.Object{
		"list": &tengo.UserFunction{Name: "list", Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) < 2 {
				return wrapError(fmt.Errorf("terms.list: requires node_type and taxonomy arguments")), nil
			}
			nodeType := tengoToString(args[0])
			taxonomy := tengoToString(args[1])
			list, err := api.ListTerms(ctx, nodeType, taxonomy)
			if err != nil {
				return wrapError(err), nil
			}
			results := make([]tengo.Object, len(list))
			for i, t := range list {
				results[i] = termToTengoObj(t)
			}
			return &tengo.ImmutableArray{Value: results}, nil
		}},
		"get": &tengo.UserFunction{Name: "get", Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) < 1 {
				return wrapError(fmt.Errorf("terms.get: requires id argument")), nil
			}
			id := uint(tengoToInt(args[0]))
			res, err := api.GetTerm(ctx, id)
			if err != nil {
				return wrapError(err), nil
			}
			return termToTengoObj(res), nil
		}},
		"create": &tengo.UserFunction{Name: "create", Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) < 1 {
				return wrapError(fmt.Errorf("terms.create: requires input argument")), nil
			}
			m := getTengoMap(args[0])
			if m == nil {
				return wrapError(fmt.Errorf("terms.create: input must be a map")), nil
			}
			term := termFromMap(m)
			res, err := api.CreateTerm(ctx, &term)
			if err != nil {
				return wrapError(err), nil
			}
			return termToTengoObj(res), nil
		}},
		"update": &tengo.UserFunction{Name: "update", Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) < 2 {
				return wrapError(fmt.Errorf("terms.update: requires id and updates arguments")), nil
			}
			id := uint(tengoToInt(args[0]))
			m := getTengoMap(args[1])
			if m == nil {
				return wrapError(fmt.Errorf("terms.update: updates must be a map")), nil
			}
			updates := make(map[string]interface{}, len(m))
			for k, v := range m {
				updates[k] = tengoObjToGo(v)
			}
			res, err := api.UpdateTerm(ctx, id, updates)
			if err != nil {
				return wrapError(err), nil
			}
			return termToTengoObj(res), nil
		}},
		"delete": &tengo.UserFunction{Name: "delete", Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) < 1 {
				return wrapError(fmt.Errorf("terms.delete: requires id argument")), nil
			}
			id := uint(tengoToInt(args[0]))
			if err := api.DeleteTerm(ctx, id); err != nil {
				return wrapError(err), nil
			}
			return tengo.UndefinedValue, nil
		}},
	}
}

func termToTengoObj(t *TaxonomyTerm) tengo.Object {
	if t == nil {
		return tengo.UndefinedValue
	}
	m := map[string]tengo.Object{
		"id":          &tengo.Int{Value: int64(t.ID)},
		"node_type":   &tengo.String{Value: t.NodeType},
		"taxonomy":    &tengo.String{Value: t.Taxonomy},
		"slug":        &tengo.String{Value: t.Slug},
		"name":        &tengo.String{Value: t.Name},
		"description": &tengo.String{Value: t.Description},
		"count":       &tengo.Int{Value: int64(t.Count)},
		"created_at":  &tengo.String{Value: t.CreatedAt.Format(time.RFC3339)},
		"updated_at":  &tengo.String{Value: t.UpdatedAt.Format(time.RFC3339)},
	}
	if t.ParentID != nil {
		m["parent_id"] = &tengo.Int{Value: int64(*t.ParentID)}
	}
	if t.FieldsData != nil {
		m["fields_data"] = goToTengoObj(t.FieldsData)
	}
	return &tengo.ImmutableMap{Value: m}
}

func termFromMap(m map[string]tengo.Object) TaxonomyTerm {
	t := TaxonomyTerm{}
	if v, ok := m["node_type"]; ok {
		t.NodeType = tengoToString(v)
	}
	if v, ok := m["taxonomy"]; ok {
		t.Taxonomy = tengoToString(v)
	}
	if v, ok := m["slug"]; ok {
		t.Slug = tengoToString(v)
	}
	if v, ok := m["name"]; ok {
		t.Name = tengoToString(v)
	}
	if v, ok := m["description"]; ok {
		t.Description = tengoToString(v)
	}
	if v, ok := m["parent_id"]; ok {
		pid := uint(tengoToInt(v))
		if pid > 0 {
			t.ParentID = &pid
		}
	}
	if v, ok := m["fields_data"]; ok {
		if fm := getTengoMap(v); fm != nil {
			t.FieldsData = make(map[string]interface{}, len(fm))
			for k, val := range fm {
				t.FieldsData[k] = tengoObjToGo(val)
			}
		}
	}
	return t
}
