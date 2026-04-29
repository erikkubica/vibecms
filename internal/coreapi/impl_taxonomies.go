package coreapi

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"squilla/internal/models"

	"github.com/lib/pq"
	"gorm.io/gorm"
)

// --- Taxonomy Definitions ---

func (c *coreImpl) RegisterTaxonomy(ctx context.Context, input TaxonomyInput) (*Taxonomy, error) {
	// UPSERT behavior
	var existing models.Taxonomy
	err := c.db.WithContext(ctx).Where("slug = ?", input.Slug).First(&existing).Error
	
	m := &models.Taxonomy{
		Slug:         input.Slug,
		Label:        input.Label,
		LabelPlural:  input.LabelPlural,
		Description:  input.Description,
		NodeTypes:    pq.StringArray(input.NodeTypes),
	}
	if input.Hierarchical != nil {
		m.Hierarchical = *input.Hierarchical
	}
	if input.ShowUI != nil {
		m.ShowUI = *input.ShowUI
	} else {
		m.ShowUI = true // Default
	}

	if input.FieldSchema != nil {
		b, _ := json.Marshal(input.FieldSchema)
		m.FieldSchema = models.JSONB(b)
	}

	if err == nil {
		// Update — only set label/description if provided, never overwrite node_types
		// (node_types are managed by the admin UI, not by theme scripts on boot)
		updates := map[string]interface{}{}
		if input.Label != "" {
			updates["label"] = input.Label
		}
		if input.LabelPlural != "" {
			updates["label_plural"] = input.LabelPlural
		}
		if input.Description != "" {
			updates["description"] = input.Description
		}
		if input.Hierarchical != nil {
			updates["hierarchical"] = *input.Hierarchical
		}
		if input.ShowUI != nil {
			updates["show_ui"] = *input.ShowUI
		}
		if input.FieldSchema != nil {
			b, _ := json.Marshal(input.FieldSchema)
			updates["field_schema"] = models.JSONB(b)
		}
		// Merge node_types: add any from input that aren't already present
		if len(input.NodeTypes) > 0 {
			existingSet := map[string]bool{}
			for _, nt := range existing.NodeTypes {
				existingSet[nt] = true
			}
			merged := make([]string, len(existing.NodeTypes))
			copy(merged, existing.NodeTypes)
			for _, nt := range input.NodeTypes {
				if !existingSet[nt] {
					merged = append(merged, nt)
				}
			}
			// Only update if there are new additions
			if len(merged) > len(existing.NodeTypes) {
				literal := "{" + strings.Join(merged, ",") + "}"
				updates["node_types"] = gorm.Expr("?::text[]", literal)
			}
		}
		if len(updates) > 0 {
			if err := c.db.WithContext(ctx).Model(&models.Taxonomy{}).Where("slug = ?", input.Slug).Updates(updates).Error; err != nil {
				return nil, fmt.Errorf("coreapi RegisterTaxonomy update: %w", err)
			}
		}
		c.db.WithContext(ctx).Where("slug = ?", input.Slug).First(&existing)
		return taxonomyFromModel(&existing), nil
	}

	// Create
	if err := c.db.WithContext(ctx).Create(m).Error; err != nil {
		return nil, fmt.Errorf("coreapi RegisterTaxonomy create: %w", err)
	}
	return taxonomyFromModel(m), nil
}

func (c *coreImpl) GetTaxonomy(ctx context.Context, slug string) (*Taxonomy, error) {
	var m models.Taxonomy
	if err := c.db.WithContext(ctx).Where("slug = ?", slug).First(&m).Error; err != nil {
		return nil, fmt.Errorf("coreapi GetTaxonomy: %w", err)
	}
	return taxonomyFromModel(&m), nil
}

func (c *coreImpl) ListTaxonomies(ctx context.Context) ([]*Taxonomy, error) {
	var mList []models.Taxonomy
	if err := c.db.WithContext(ctx).Order("label ASC").Find(&mList).Error; err != nil {
		return nil, fmt.Errorf("coreapi ListTaxonomies: %w", err)
	}
	results := make([]*Taxonomy, len(mList))
	for i, m := range mList {
		results[i] = taxonomyFromModel(&m)
	}
	return results, nil
}

func (c *coreImpl) UpdateTaxonomy(ctx context.Context, slug string, input TaxonomyInput) (*Taxonomy, error) {
	var m models.Taxonomy
	if err := c.db.WithContext(ctx).Where("slug = ?", slug).First(&m).Error; err != nil {
		return nil, fmt.Errorf("coreapi UpdateTaxonomy: %w", err)
	}

	updates := map[string]interface{}{}
	if input.Label != "" {
		updates["label"] = input.Label
	}
	if input.LabelPlural != "" {
		updates["label_plural"] = input.LabelPlural
	}
	if input.Description != "" {
		updates["description"] = input.Description
	}
	if input.NodeTypes != nil {
		updates["node_types"] = pq.StringArray(input.NodeTypes)
	}
	if input.Hierarchical != nil {
		updates["hierarchical"] = *input.Hierarchical
	}
	if input.ShowUI != nil {
		updates["show_ui"] = *input.ShowUI
	}
	if input.FieldSchema != nil {
		b, _ := json.Marshal(input.FieldSchema)
		updates["field_schema"] = models.JSONB(b)
	}
	if len(updates) == 0 {
		return taxonomyFromModel(&m), nil
	}

	if err := c.db.WithContext(ctx).Model(&m).Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("coreapi UpdateTaxonomy: %w", err)
	}
	return taxonomyFromModel(&m), nil
}

func (c *coreImpl) DeleteTaxonomy(ctx context.Context, slug string) error {
	if err := c.db.WithContext(ctx).Where("slug = ?", slug).Delete(&models.Taxonomy{}).Error; err != nil {
		return fmt.Errorf("coreapi DeleteTaxonomy: %w", err)
	}
	return nil
}

// --- Taxonomy Terms ---

func (c *coreImpl) ListTerms(ctx context.Context, nodeType string, taxonomy string) ([]*TaxonomyTerm, error) {
	var mTerms []models.TaxonomyTerm
	query := c.db.WithContext(ctx).Where("taxonomy = ?", taxonomy)
	if nodeType != "" {
		query = query.Where("node_type = ?", nodeType)
	}
	
	err := query.Order("name ASC").Find(&mTerms).Error
	if err != nil {
		return nil, fmt.Errorf("coreapi ListTerms: %w", err)
	}

	results := make([]*TaxonomyTerm, len(mTerms))
	for i, t := range mTerms {
		results[i] = termFromModel(&t)
	}
	return results, nil
}

func (c *coreImpl) GetTerm(ctx context.Context, id uint) (*TaxonomyTerm, error) {
	var t models.TaxonomyTerm
	if err := c.db.WithContext(ctx).First(&t, id).Error; err != nil {
		return nil, fmt.Errorf("coreapi GetTerm: %w", err)
	}
	return termFromModel(&t), nil
}

func (c *coreImpl) CreateTerm(ctx context.Context, term *TaxonomyTerm) (*TaxonomyTerm, error) {
	m := &models.TaxonomyTerm{
		NodeType:           term.NodeType,
		Taxonomy:           term.Taxonomy,
		LanguageCode:       term.LanguageCode,
		TranslationGroupID: term.TranslationGroupID,
		Slug:               term.Slug,
		Name:               term.Name,
		Description:        term.Description,
	}
	if m.LanguageCode == "" {
		m.LanguageCode = c.defaultLocale(ctx)
	}
	if term.ParentID != nil {
		pid := int(*term.ParentID)
		m.ParentID = &pid
	}
	if term.FieldsData != nil {
		b, _ := json.Marshal(term.FieldsData)
		m.FieldsData = models.JSONB(b)
	}

	if err := c.db.WithContext(ctx).Create(m).Error; err != nil {
		return nil, fmt.Errorf("coreapi CreateTerm: %w", err)
	}
	return termFromModel(m), nil
}

func (c *coreImpl) UpdateTerm(ctx context.Context, id uint, updates map[string]interface{}) (*TaxonomyTerm, error) {
	var m models.TaxonomyTerm
	if err := c.db.WithContext(ctx).First(&m, id).Error; err != nil {
		return nil, fmt.Errorf("coreapi UpdateTerm: %w", err)
	}

	// Handle FieldsData marshaling if present
	if fd, ok := updates["fields_data"]; ok && fd != nil {
		b, _ := json.Marshal(fd)
		updates["fields_data"] = models.JSONB(b)
	}

	if err := c.db.WithContext(ctx).Model(&m).Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("coreapi UpdateTerm: %w", err)
	}

	return termFromModel(&m), nil
}

func (c *coreImpl) DeleteTerm(ctx context.Context, id uint) error {
	if err := c.db.WithContext(ctx).Delete(&models.TaxonomyTerm{}, id).Error; err != nil {
		return fmt.Errorf("coreapi DeleteTerm: %w", err)
	}
	return nil
}

// --- Helpers ---

func taxonomyFromModel(m *models.Taxonomy) *Taxonomy {
	t := &Taxonomy{
		ID:           uint(m.ID),
		Slug:         m.Slug,
		Label:        m.Label,
		LabelPlural:  m.LabelPlural,
		Description:  m.Description,
		Hierarchical: m.Hierarchical,
		ShowUI:       m.ShowUI,
		NodeTypes:    []string(m.NodeTypes),
		CreatedAt:    m.CreatedAt,
		UpdatedAt:    m.UpdatedAt,
	}
	if len(m.FieldSchema) > 0 {
		var schema []NodeTypeField
		json.Unmarshal([]byte(m.FieldSchema), &schema)
		t.FieldSchema = schema
	}
	return t
}

func termFromModel(m *models.TaxonomyTerm) *TaxonomyTerm {
	t := &TaxonomyTerm{
		ID:                 uint(m.ID),
		NodeType:           m.NodeType,
		Taxonomy:           m.Taxonomy,
		LanguageCode:       m.LanguageCode,
		TranslationGroupID: m.TranslationGroupID,
		Slug:               m.Slug,
		Name:               m.Name,
		Description:        m.Description,
		Count:              m.Count,
		CreatedAt:          m.CreatedAt,
		UpdatedAt:          m.UpdatedAt,
	}
	if m.ParentID != nil {
		pid := uint(*m.ParentID)
		t.ParentID = &pid
	}
	if len(m.FieldsData) > 0 {
		var fields map[string]interface{}
		json.Unmarshal([]byte(m.FieldsData), &fields)
		t.FieldsData = fields
	}
	return t
}
