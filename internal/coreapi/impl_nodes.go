package coreapi

import (
	"context"
	"encoding/json"
	"fmt"

	"vibecms/internal/models"
)

// nodeFromModel converts a GORM ContentNode model to a CoreAPI Node.
func nodeFromModel(m *models.ContentNode) *Node {
	n := &Node{
		ID:           uint(m.ID),
		UUID:         m.UUID,
		NodeType:     m.NodeType,
		Status:       m.Status,
		LanguageCode: m.LanguageCode,
		Slug:         m.Slug,
		FullURL:      m.FullURL,
		Title:        m.Title,
		Excerpt:      m.Excerpt,
		PublishedAt:  m.PublishedAt,
		CreatedAt:    m.CreatedAt,
		UpdatedAt:    m.UpdatedAt,
	}

	if m.ParentID != nil {
		pid := uint(*m.ParentID)
		n.ParentID = &pid
	}

	// Unmarshal JSONB fields into their CoreAPI representations.
	if len(m.Taxonomies) > 0 {
		var tx map[string][]string
		if err := json.Unmarshal([]byte(m.Taxonomies), &tx); err == nil {
			n.Taxonomies = tx
		}
	}
	if len(m.FeaturedImage) > 0 {
		var fi any
		if err := json.Unmarshal([]byte(m.FeaturedImage), &fi); err == nil {
			n.FeaturedImage = fi
		}
	}
	if len(m.BlocksData) > 0 {
		var bd any
		if err := json.Unmarshal([]byte(m.BlocksData), &bd); err == nil {
			n.BlocksData = bd
		}
	}
	if len(m.FieldsData) > 0 {
		var fd map[string]any
		if err := json.Unmarshal([]byte(m.FieldsData), &fd); err == nil {
			n.FieldsData = fd
		}
	}
	if len(m.SeoSettings) > 0 {
		var ss map[string]string
		if err := json.Unmarshal([]byte(m.SeoSettings), &ss); err == nil {
			n.SeoSettings = ss
		}
	}

	return n
}

// GetNode retrieves a single content node by ID.
func (c *coreImpl) GetNode(_ context.Context, id uint) (*Node, error) {
	m, err := c.contentSvc.GetByID(int(id))
	if err != nil {
		return nil, fmt.Errorf("coreapi GetNode: %w", err)
	}
	n := nodeFromModel(m)
	if n.FieldsData != nil {
		if lookup := c.loadThemeAssetLookup(); len(lookup) > 0 {
			n.FieldsData = resolveAssetRefsIn(n.FieldsData, lookup)
		}
	}
	return n, nil
}

// QueryNodes searches content nodes with optional filters and pagination.
// Nodes whose node_type is no longer registered are treated as dormant and
// excluded (they return on their own once the type is re-registered).
func (c *coreImpl) QueryNodes(_ context.Context, q NodeQuery) (*NodeList, error) {
	query := c.db.Model(&models.ContentNode{}).
		Where("node_type IN (?)", c.db.Model(&models.NodeType{}).Select("slug"))

	if q.NodeType != "" {
		query = query.Where("node_type = ?", q.NodeType)
	}
	if q.Status != "" {
		query = query.Where("status = ?", q.Status)
	}
	if q.ParentID != nil {
		query = query.Where("parent_id = ?", *q.ParentID)
	}
	if q.LanguageCode != "" {
		query = query.Where("language_code = ?", q.LanguageCode)
	}
	if q.Slug != "" {
		query = query.Where("slug = ?", q.Slug)
	}
	if q.Search != "" {
		query = query.Where("title ILIKE ?", "%"+q.Search+"%")
	}
	if q.Category != "" {
		// Compatibility: search in "category" taxonomy
		query = query.Where("taxonomies->'category' ? ?", q.Category)
	}
	if len(q.TaxQuery) > 0 {
		for tax, terms := range q.TaxQuery {
			if len(terms) > 0 {
				b, _ := json.Marshal(terms)
				// Check if taxonomies[tax] contains ANY of the terms
				// Using JSONB @> for exact match or ? for single term existence
				if len(terms) == 1 {
					query = query.Where("taxonomies->? ? ?", tax, terms[0])
				} else {
					// For multiple terms, we'd need a more complex OR query or use JSONB operators
					// Simple implementation: must contain ALL provided terms
					query = query.Where("taxonomies->? @> ?", tax, b)
				}
			}
		}
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("coreapi QueryNodes count: %w", err)
	}

	orderBy := "created_at DESC"
	if q.OrderBy != "" {
		orderBy = q.OrderBy
	}
	query = query.Order(orderBy)

	if q.Limit > 0 {
		query = query.Limit(q.Limit)
	}
	if q.Offset > 0 {
		query = query.Offset(q.Offset)
	}

	var rows []models.ContentNode
	if err := query.Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("coreapi QueryNodes: %w", err)
	}

	// Pre-compute theme-asset lookup once so every node in this batch
	// gets its fields_data references resolved without N extra queries.
	lookup := c.loadThemeAssetLookup()

	nodes := make([]*Node, len(rows))
	for i := range rows {
		nodes[i] = nodeFromModel(&rows[i])
		if len(lookup) > 0 && nodes[i].FieldsData != nil {
			nodes[i].FieldsData = resolveAssetRefsIn(nodes[i].FieldsData, lookup)
		}
	}

	return &NodeList{Nodes: nodes, Total: total}, nil
}

// assetRef maps an asset_key like "cave-obsidian" → media row's URL/alt/dims.
type assetRef struct {
	URL    string
	Alt    string
	Width  int
	Height int
}

// loadThemeAssetLookup reads media_files rows owned by the currently active
// theme and returns a map keyed by asset_key. Returns nil if no active theme
// or no theme-owned assets.
func (c *coreImpl) loadThemeAssetLookup() map[string]assetRef {
	type row struct {
		AssetKey string
		URL      string
		Alt      string
		Width    int
		Height   int
	}
	var rows []row
	err := c.db.Table("media_files").
		Select("asset_key, url, alt, width, height").
		Where("source = ?", "theme").
		Where("theme_name = (SELECT name FROM themes WHERE is_active = true LIMIT 1)").
		Scan(&rows).Error
	if err != nil || len(rows) == 0 {
		return nil
	}
	out := make(map[string]assetRef, len(rows))
	for _, r := range rows {
		if r.AssetKey == "" {
			continue
		}
		out[r.AssetKey] = assetRef{URL: r.URL, Alt: r.Alt, Width: r.Width, Height: r.Height}
	}
	return out
}

// resolveAssetRefsIn walks a map (usually fields_data) and replaces any
// `theme-asset:<key>` string with a resolved `{url, alt, width, height}`
// media object. When the ref sits at an object's "url" key (e.g.
// `{"url": "theme-asset:foo", "alt": "..."}` — the common block-image shape),
// the url is replaced in place so sibling keys (like a custom alt) survive.
func resolveAssetRefsIn(v map[string]interface{}, lookup map[string]assetRef) map[string]interface{} {
	for k, val := range v {
		v[k] = resolveAssetRefAny(val, lookup)
	}
	return v
}

func resolveAssetRefAny(val interface{}, lookup map[string]assetRef) interface{} {
	switch vv := val.(type) {
	case string:
		if ref, ok := matchThemeAssetRef(vv, lookup); ok {
			return map[string]interface{}{"url": ref.URL, "alt": ref.Alt, "width": ref.Width, "height": ref.Height}
		}
		return vv
	case map[string]interface{}:
		if rawURL, ok := vv["url"].(string); ok {
			if ref, hit := matchThemeAssetRef(rawURL, lookup); hit {
				vv["url"] = ref.URL
				if _, has := vv["alt"]; !has {
					vv["alt"] = ref.Alt
				}
				if _, has := vv["width"]; !has && ref.Width > 0 {
					vv["width"] = ref.Width
				}
				if _, has := vv["height"]; !has && ref.Height > 0 {
					vv["height"] = ref.Height
				}
				return vv
			}
		}
		for k, iv := range vv {
			vv[k] = resolveAssetRefAny(iv, lookup)
		}
		return vv
	case []interface{}:
		for i, iv := range vv {
			vv[i] = resolveAssetRefAny(iv, lookup)
		}
		return vv
	}
	return val
}

// matchThemeAssetRef returns the resolved row when the string matches the
// `theme-asset:<key>` pattern AND the key is present in the lookup.
func matchThemeAssetRef(s string, lookup map[string]assetRef) (assetRef, bool) {
	const prefix = "theme-asset:"
	if len(s) <= len(prefix) || s[:len(prefix)] != prefix {
		return assetRef{}, false
	}
	ref, ok := lookup[s[len(prefix):]]
	return ref, ok
}

func (c *coreImpl) ListTaxonomyTerms(_ context.Context, nodeType string, taxonomy string) ([]string, error) {
	var terms []string
	// Subquery to extract array elements as text
	// select distinct term from (select jsonb_array_elements_text(taxonomies->'category') as term from content_nodes where node_type = 'post') as t
	err := c.db.Table("(?) as t",
		c.db.Table("content_nodes").
			Select("jsonb_array_elements_text(taxonomies->?) as term", taxonomy).
			Where("node_type = ? AND deleted_at IS NULL", nodeType)).
		Select("DISTINCT term").
		Order("term ASC").
		Scan(&terms).Error

	if err != nil {
		return nil, fmt.Errorf("coreapi ListTaxonomyTerms: %w", err)
	}
	return terms, nil
}

// CreateNode creates a new content node, defaulting type to "page" and status to "draft".
func (c *coreImpl) CreateNode(_ context.Context, input NodeInput) (*Node, error) {
	m := &models.ContentNode{
		Title:        input.Title,
		Slug:         input.Slug,
		NodeType:     input.NodeType,
		Status:       input.Status,
		LanguageCode: input.LanguageCode,
		Excerpt:      input.Excerpt,
	}

	if input.Taxonomies != nil {
		b, err := json.Marshal(input.Taxonomies)
		if err != nil {
			return nil, fmt.Errorf("coreapi CreateNode marshal taxonomies: %w", err)
		}
		m.Taxonomies = models.JSONB(b)
	}

	if m.NodeType == "" {
		m.NodeType = "page"
	}
	if m.Status == "" {
		m.Status = "draft"
	}
	if m.LanguageCode == "" {
		m.LanguageCode = "en"
	}

	if input.ParentID != nil {
		pid := int(*input.ParentID)
		m.ParentID = &pid
	}

	if input.LayoutSlug != "" {
		s := input.LayoutSlug
		m.LayoutSlug = &s
	}

	if input.FeaturedImage != nil {
		b, err := json.Marshal(input.FeaturedImage)
		if err != nil {
			return nil, fmt.Errorf("coreapi CreateNode marshal featured_image: %w", err)
		}
		m.FeaturedImage = models.JSONB(b)
	}
	if input.BlocksData != nil {
		b, err := json.Marshal(input.BlocksData)
		if err != nil {
			return nil, fmt.Errorf("coreapi CreateNode marshal blocks: %w", err)
		}
		m.BlocksData = models.JSONB(b)
	}
	if input.FieldsData != nil {
		b, err := json.Marshal(input.FieldsData)
		if err != nil {
			return nil, fmt.Errorf("coreapi CreateNode marshal fields: %w", err)
		}
		m.FieldsData = models.JSONB(b)
	}
	if input.SeoSettings != nil {
		b, err := json.Marshal(input.SeoSettings)
		if err != nil {
			return nil, fmt.Errorf("coreapi CreateNode marshal seo: %w", err)
		}
		m.SeoSettings = models.JSONB(b)
	}

	// Use ContentService.Create (userID 0 = system/extension).
	if err := c.contentSvc.Create(m, 0); err != nil {
		return nil, fmt.Errorf("coreapi CreateNode: %w", err)
	}

	return nodeFromModel(m), nil
}

// UpdateNode updates an existing node, applying only non-zero fields from input.
func (c *coreImpl) UpdateNode(_ context.Context, id uint, input NodeInput) (*Node, error) {
	updates := make(map[string]interface{})

	if input.Title != "" {
		updates["title"] = input.Title
	}
	if input.Slug != "" {
		updates["slug"] = input.Slug
	}
	if input.NodeType != "" {
		updates["node_type"] = input.NodeType
	}
	if input.Status != "" {
		updates["status"] = input.Status
	}
	if input.LanguageCode != "" {
		updates["language_code"] = input.LanguageCode
	}
	if input.ParentID != nil {
		updates["parent_id"] = int(*input.ParentID)
	}
	if input.LayoutSlug != "" {
		updates["layout_slug"] = input.LayoutSlug
	}
	if input.FeaturedImage != nil {
		if b, err := json.Marshal(input.FeaturedImage); err == nil {
			updates["featured_image"] = models.JSONB(b)
		}
	}
	if input.Excerpt != "" {
		updates["excerpt"] = input.Excerpt
	}
	if input.Taxonomies != nil {
		if b, err := json.Marshal(input.Taxonomies); err == nil {
			updates["taxonomies"] = models.JSONB(b)
		}
	}
	if input.BlocksData != nil {
		if b, err := json.Marshal(input.BlocksData); err == nil {
			updates["blocks_data"] = models.JSONB(b)
		}
	}
	if input.FieldsData != nil {
		if b, err := json.Marshal(input.FieldsData); err == nil {
			updates["fields_data"] = models.JSONB(b)
		}
	}
	if input.SeoSettings != nil {
		if b, err := json.Marshal(input.SeoSettings); err == nil {
			updates["seo_settings"] = models.JSONB(b)
		}
	}

	// Use ContentService.Update (userID 0 = system/extension).
	m, err := c.contentSvc.Update(int(id), updates, 0)
	if err != nil {
		return nil, fmt.Errorf("coreapi UpdateNode: %w", err)
	}

	return nodeFromModel(m), nil
}

// DeleteNode soft-deletes a content node by ID.
func (c *coreImpl) DeleteNode(_ context.Context, id uint) error {
	if err := c.contentSvc.Delete(int(id)); err != nil {
		return fmt.Errorf("coreapi DeleteNode: %w", err)
	}
	return nil
}
