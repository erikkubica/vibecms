package scripting

import (
	"encoding/json"
	"fmt"

	"vibecms/internal/models"

	"github.com/d5/tengo/v2"
	"gorm.io/gorm"
)

// nodesModule returns the cms/nodes built-in module.
func (e *ScriptEngine) nodesModule() map[string]tengo.Object {
	return map[string]tengo.Object{
		"list":        &tengo.UserFunction{Name: "list", Value: e.nodesList},
		"get":         &tengo.UserFunction{Name: "get", Value: e.nodesGet},
		"get_by_slug": &tengo.UserFunction{Name: "get_by_slug", Value: e.nodesGetBySlug},
		"create":      &tengo.UserFunction{Name: "create", Value: e.nodesCreate},
		"update":      &tengo.UserFunction{Name: "update", Value: e.nodesUpdate},
		"delete":      &tengo.UserFunction{Name: "delete", Value: e.nodesDelete},
		"query":       &tengo.UserFunction{Name: "query", Value: e.nodesQuery},
	}
}

// nodesList handles nodes.list(options) -> [{id, title, slug, ...}]
// Options: {status, node_type, language_code, search, page, per_page}
func (e *ScriptEngine) nodesList(args ...tengo.Object) (tengo.Object, error) {
	page, perPage := 1, 50
	var status, nodeType, langCode, search string

	if len(args) > 0 {
		if m := getMap(args[0]); m != nil {
			if v, ok := m["page"]; ok {
				page = getInt(v)
			}
			if v, ok := m["per_page"]; ok {
				perPage = getInt(v)
			}
			if v, ok := m["status"]; ok {
				status = getString(v)
			}
			if v, ok := m["node_type"]; ok {
				nodeType = getString(v)
			}
			if v, ok := m["language_code"]; ok {
				langCode = getString(v)
			}
			if v, ok := m["search"]; ok {
				search = getString(v)
			}
		}
	}

	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 200 {
		perPage = 50
	}

	nodes, total, err := e.contentSvc.List(page, perPage, status, nodeType, langCode, search)
	if err != nil {
		return tengo.UndefinedValue, fmt.Errorf("nodes.list: %w", err)
	}

	items := make([]tengo.Object, len(nodes))
	for i, n := range nodes {
		items[i] = nodeToTengo(&n)
	}

	return &tengo.ImmutableMap{Value: map[string]tengo.Object{
		"items": &tengo.ImmutableArray{Value: items},
		"total": &tengo.Int{Value: total},
		"page":  &tengo.Int{Value: int64(page)},
	}}, nil
}

// nodesGet handles nodes.get(id) -> node map
func (e *ScriptEngine) nodesGet(args ...tengo.Object) (tengo.Object, error) {
	if len(args) < 1 {
		return tengo.UndefinedValue, fmt.Errorf("nodes.get: requires id argument")
	}
	id := getInt(args[0])
	if id <= 0 {
		return tengo.UndefinedValue, fmt.Errorf("nodes.get: invalid id")
	}

	node, err := e.contentSvc.GetByID(id)
	if err != nil {
		return tengo.UndefinedValue, nil // return undefined for not found
	}
	return nodeToTengo(node), nil
}

// nodesGetBySlug handles nodes.get_by_slug(full_url) -> node map
func (e *ScriptEngine) nodesGetBySlug(args ...tengo.Object) (tengo.Object, error) {
	if len(args) < 1 {
		return tengo.UndefinedValue, fmt.Errorf("nodes.get_by_slug: requires slug argument")
	}
	slug := getString(args[0])
	if slug == "" {
		return tengo.UndefinedValue, fmt.Errorf("nodes.get_by_slug: empty slug")
	}

	var node models.ContentNode
	if err := e.db.Where("full_url = ?", slug).First(&node).Error; err != nil {
		return tengo.UndefinedValue, nil
	}
	return nodeToTengo(&node), nil
}

// nodesCreate handles nodes.create(data) -> node map
// Data: {title, slug, node_type, status, language_code, parent_id, blocks_data, fields_data, seo_settings}
func (e *ScriptEngine) nodesCreate(args ...tengo.Object) (tengo.Object, error) {
	if len(args) < 1 {
		return tengo.UndefinedValue, fmt.Errorf("nodes.create: requires data argument")
	}
	m := getMap(args[0])
	if m == nil {
		return tengo.UndefinedValue, fmt.Errorf("nodes.create: argument must be a map")
	}

	node := &models.ContentNode{
		NodeType:     "page",
		Status:       "draft",
		LanguageCode: "en",
	}

	if v, ok := m["title"]; ok {
		node.Title = getString(v)
	}
	if v, ok := m["slug"]; ok {
		node.Slug = getString(v)
	}
	if v, ok := m["node_type"]; ok {
		node.NodeType = getString(v)
	}
	if v, ok := m["status"]; ok {
		node.Status = getString(v)
	}
	if v, ok := m["language_code"]; ok {
		node.LanguageCode = getString(v)
	}
	if v, ok := m["parent_id"]; ok {
		pid := getInt(v)
		if pid > 0 {
			node.ParentID = &pid
		}
	}
	if v, ok := m["blocks_data"]; ok {
		b, _ := json.Marshal(tengoToGo(v))
		node.BlocksData = models.JSONB(b)
	}
	if v, ok := m["fields_data"]; ok {
		b, _ := json.Marshal(tengoToGo(v))
		node.FieldsData = models.JSONB(b)
	}
	if v, ok := m["seo_settings"]; ok {
		b, _ := json.Marshal(tengoToGo(v))
		node.SeoSettings = models.JSONB(b)
	}

	if err := e.contentSvc.Create(node, 0); err != nil {
		return tengo.UndefinedValue, fmt.Errorf("nodes.create: %w", err)
	}

	return nodeToTengo(node), nil
}

// nodesUpdate handles nodes.update(id, data) -> node map
func (e *ScriptEngine) nodesUpdate(args ...tengo.Object) (tengo.Object, error) {
	if len(args) < 2 {
		return tengo.UndefinedValue, fmt.Errorf("nodes.update: requires id and data arguments")
	}
	id := getInt(args[0])
	if id <= 0 {
		return tengo.UndefinedValue, fmt.Errorf("nodes.update: invalid id")
	}

	m := getMap(args[1])
	if m == nil {
		return tengo.UndefinedValue, fmt.Errorf("nodes.update: data must be a map")
	}

	updates := make(map[string]interface{})
	for k, v := range m {
		updates[k] = tengoToGo(v)
	}

	node, err := e.contentSvc.Update(id, updates, 0)
	if err != nil {
		return tengo.UndefinedValue, fmt.Errorf("nodes.update: %w", err)
	}

	return nodeToTengo(node), nil
}

// nodesDelete handles nodes.delete(id) -> bool
func (e *ScriptEngine) nodesDelete(args ...tengo.Object) (tengo.Object, error) {
	if len(args) < 1 {
		return tengo.UndefinedValue, fmt.Errorf("nodes.delete: requires id argument")
	}
	id := getInt(args[0])
	if id <= 0 {
		return tengo.UndefinedValue, fmt.Errorf("nodes.delete: invalid id")
	}

	if err := e.contentSvc.Delete(id); err != nil {
		return tengo.FalseValue, nil
	}
	return tengo.TrueValue, nil
}

// nodesQuery handles nodes.query(options) -> [nodes]
// A flexible query builder: {where, order, limit, offset, select}
func (e *ScriptEngine) nodesQuery(args ...tengo.Object) (tengo.Object, error) {
	if len(args) < 1 {
		return tengo.UndefinedValue, fmt.Errorf("nodes.query: requires options argument")
	}
	m := getMap(args[0])
	if m == nil {
		return tengo.UndefinedValue, fmt.Errorf("nodes.query: argument must be a map")
	}

	query := e.db.Model(&models.ContentNode{})

	if v, ok := m["where"]; ok {
		if wm := getMap(v); wm != nil {
			for field, val := range wm {
				query = query.Where(field+" = ?", tengoToGo(val))
			}
		}
	}
	if v, ok := m["order"]; ok {
		query = query.Order(getString(v))
	}
	if v, ok := m["limit"]; ok {
		limit := getInt(v)
		if limit > 0 && limit <= 500 {
			query = query.Limit(limit)
		}
	} else {
		query = query.Limit(50)
	}
	if v, ok := m["offset"]; ok {
		query = query.Offset(getInt(v))
	}

	var nodes []models.ContentNode
	if err := query.Find(&nodes).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return &tengo.ImmutableArray{Value: []tengo.Object{}}, nil
		}
		return tengo.UndefinedValue, fmt.Errorf("nodes.query: %w", err)
	}

	items := make([]tengo.Object, len(nodes))
	for i, n := range nodes {
		items[i] = nodeToTengo(&n)
	}
	return &tengo.ImmutableArray{Value: items}, nil
}

// nodeToTengo converts a ContentNode to a Tengo ImmutableMap.
func nodeToTengo(n *models.ContentNode) tengo.Object {
	m := map[string]tengo.Object{
		"id":            &tengo.Int{Value: int64(n.ID)},
		"uuid":          &tengo.String{Value: n.UUID},
		"node_type":     &tengo.String{Value: n.NodeType},
		"status":        &tengo.String{Value: n.Status},
		"language_code": &tengo.String{Value: n.LanguageCode},
		"slug":          &tengo.String{Value: n.Slug},
		"full_url":      &tengo.String{Value: n.FullURL},
		"title":         &tengo.String{Value: n.Title},
		"version":       &tengo.Int{Value: int64(n.Version)},
		"created_at":    &tengo.String{Value: n.CreatedAt.Format("2006-01-02T15:04:05Z")},
		"updated_at":    &tengo.String{Value: n.UpdatedAt.Format("2006-01-02T15:04:05Z")},
	}

	if n.ParentID != nil {
		m["parent_id"] = &tengo.Int{Value: int64(*n.ParentID)}
	} else {
		m["parent_id"] = tengo.UndefinedValue
	}

	if n.AuthorID != nil {
		m["author_id"] = &tengo.Int{Value: int64(*n.AuthorID)}
	} else {
		m["author_id"] = tengo.UndefinedValue
	}

	if n.PublishedAt != nil {
		m["published_at"] = &tengo.String{Value: n.PublishedAt.Format("2006-01-02T15:04:05Z")}
	} else {
		m["published_at"] = tengo.UndefinedValue
	}

	// Parse JSONB fields into Tengo objects
	if len(n.BlocksData) > 0 {
		var blocks interface{}
		if err := json.Unmarshal([]byte(n.BlocksData), &blocks); err == nil {
			m["blocks_data"] = goToTengo(blocks)
		}
	}
	if len(n.FieldsData) > 0 {
		var fields interface{}
		if err := json.Unmarshal([]byte(n.FieldsData), &fields); err == nil {
			m["fields_data"] = goToTengo(fields)
		}
	}
	if len(n.SeoSettings) > 0 {
		var seo interface{}
		if err := json.Unmarshal([]byte(n.SeoSettings), &seo); err == nil {
			m["seo_settings"] = goToTengo(seo)
		}
	}

	return &tengo.ImmutableMap{Value: m}
}
