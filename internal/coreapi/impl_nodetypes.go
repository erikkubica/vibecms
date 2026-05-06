package coreapi

import (
	"context"
	"encoding/json"
	"strings"

	"squilla/internal/models"
)

func (c *coreImpl) RegisterNodeType(ctx context.Context, input NodeTypeInput) (*NodeType, error) {
	if input.Slug == "" {
		return nil, NewValidation("slug is required")
	}
	if input.Label == "" {
		return nil, NewValidation("label is required")
	}

	fieldSchemaJSON, err := json.Marshal(input.Fields)
	if err != nil {
		return nil, NewInternal("failed to marshal field_schema: " + err.Error())
	}
	urlPrefixesJSON, err := json.Marshal(input.URLPrefixes)
	if err != nil {
		return nil, NewInternal("failed to marshal url_prefixes: " + err.Error())
	}

	nt := &models.NodeType{
		Slug:           input.Slug,
		Label:          input.Label,
		LabelPlural:    input.LabelPlural,
		Icon:           input.Icon,
		Description:    input.Description,
		Fields:         models.JSONB(fieldSchemaJSON),
		URLPrefixes:    models.JSONB(urlPrefixesJSON),
		SupportsBlocks: true,
	}
	if input.SupportsBlocks != nil {
		nt.SupportsBlocks = *input.SupportsBlocks
	}
	if nt.Icon == "" {
		nt.Icon = "file-text"
	}

	if input.Taxonomies != nil {
		b, err := json.Marshal(input.Taxonomies)
		if err != nil {
			return nil, NewInternal("failed to marshal taxonomies: " + err.Error())
		}
		nt.Taxonomies = models.JSONB(b)
	}

	// Check if exists first to support UPSERT from theme scripts
	existing, err := c.nodeTypeSvc.GetBySlug(input.Slug)
	if err == nil && existing != nil {
		// Update existing
		updates := make(map[string]interface{})
		updates["label"] = input.Label
		updates["label_plural"] = input.LabelPlural
		if input.Icon != "" {
			updates["icon"] = input.Icon
		}
		if input.Description != "" {
			updates["description"] = input.Description
		}
		if input.Taxonomies != nil {
			updates["taxonomies"] = nt.Taxonomies
		}
		// Theme seeds re-run on every activation; only overwrite the
		// JSONB schema/prefix columns when the seed actually declared a
		// value. Without this guard, a register() call that omits
		// url_prefixes wipes operator-set prefixes back to null on every
		// reactivation — symptom: the "blog vs post" prefix bug.
		if input.Fields != nil {
			updates["field_schema"] = nt.Fields
		}
		if input.URLPrefixes != nil {
			updates["url_prefixes"] = nt.URLPrefixes
		}
		if input.SupportsBlocks != nil {
			updates["supports_blocks"] = *input.SupportsBlocks
		}

		updated, err := c.nodeTypeSvc.Update(existing.ID, updates)
		if err != nil {
			return nil, NewInternal("failed to update node type on register: " + err.Error())
		}
		return nodeTypeFromModel(updated), nil
	}

	if err := c.nodeTypeSvc.Create(nt); err != nil {
		if strings.Contains(err.Error(), "slug conflict") {
			return nil, NewValidation(err.Error())
		}
		return nil, NewInternal(err.Error())
	}

	return nodeTypeFromModel(nt), nil
}

func (c *coreImpl) GetNodeType(ctx context.Context, slug string) (*NodeType, error) {
	nt, err := c.nodeTypeSvc.GetBySlug(slug)
	if err != nil {
		return nil, NewNotFound("node_type", slug)
	}
	return nodeTypeFromModel(nt), nil
}

func (c *coreImpl) ListNodeTypes(ctx context.Context) ([]*NodeType, error) {
	list, err := c.nodeTypeSvc.ListAll()
	if err != nil {
		return nil, NewInternal(err.Error())
	}
	out := make([]*NodeType, len(list))
	for i := range list {
		out[i] = nodeTypeFromModel(&list[i])
	}
	return out, nil
}

func (c *coreImpl) UpdateNodeType(ctx context.Context, slug string, input NodeTypeInput) (*NodeType, error) {
	existing, err := c.nodeTypeSvc.GetBySlug(slug)
	if err != nil {
		return nil, NewNotFound("node_type", slug)
	}

	updates := make(map[string]interface{})
	if input.Slug != "" {
		updates["slug"] = input.Slug
	}
	if input.Label != "" {
		updates["label"] = input.Label
	}
	if input.LabelPlural != "" {
		updates["label_plural"] = input.LabelPlural
	}
	if input.Icon != "" {
		updates["icon"] = input.Icon
	}
	if input.Description != "" {
		updates["description"] = input.Description
	}
	if input.Taxonomies != nil {
		updates["taxonomies"] = input.Taxonomies
	}
	if input.Fields != nil {
		updates["field_schema"] = input.Fields
	}
	if input.URLPrefixes != nil {
		updates["url_prefixes"] = input.URLPrefixes
	}
	if input.SupportsBlocks != nil {
		updates["supports_blocks"] = *input.SupportsBlocks
	}

	updated, err := c.nodeTypeSvc.Update(existing.ID, updates)
	if err != nil {
		if strings.Contains(err.Error(), "slug conflict") {
			return nil, NewValidation(err.Error())
		}
		return nil, NewInternal(err.Error())
	}

	return nodeTypeFromModel(updated), nil
}

func (c *coreImpl) DeleteNodeType(ctx context.Context, slug string) error {
	existing, err := c.nodeTypeSvc.GetBySlug(slug)
	if err != nil {
		return NewNotFound("node_type", slug)
	}

	if err := c.nodeTypeSvc.Delete(existing.ID); err != nil {
		if strings.Contains(err.Error(), "cannot delete built-in") {
			return NewValidation(err.Error())
		}
		return NewInternal(err.Error())
	}
	return nil
}

func nodeTypeFromModel(nt *models.NodeType) *NodeType {
	result := &NodeType{
		ID:             nt.ID,
		Slug:           nt.Slug,
		Label:          nt.Label,
		LabelPlural:    nt.LabelPlural,
		Icon:           nt.Icon,
		Description:    nt.Description,
		SupportsBlocks: nt.SupportsBlocks,
		CreatedAt:      nt.CreatedAt,
		UpdatedAt:      nt.UpdatedAt,
	}

	// Parse Taxonomies from JSONB
	if len(nt.Taxonomies) > 0 {
		var taxes []TaxonomyDefinition
		if err := json.Unmarshal([]byte(nt.Taxonomies), &taxes); err == nil {
			result.Taxonomies = taxes
		}
	}
	if result.Taxonomies == nil {
		result.Taxonomies = []TaxonomyDefinition{}
	}

	// Parse field schema from JSONB.
	if len(nt.Fields) > 0 {
		var fields []NodeTypeField
		if err := json.Unmarshal([]byte(nt.Fields), &fields); err == nil {
			result.Fields = fields
		}
	}
	if result.Fields == nil {
		result.Fields = []NodeTypeField{}
	}

	// Parse URLPrefixes from JSONB
	if len(nt.URLPrefixes) > 0 {
		var prefixes map[string]string
		if err := json.Unmarshal([]byte(nt.URLPrefixes), &prefixes); err == nil {
			result.URLPrefixes = prefixes
		}
	}
	if result.URLPrefixes == nil {
		result.URLPrefixes = map[string]string{}
	}

	return result
}
