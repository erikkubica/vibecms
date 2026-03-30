package coreapi

import (
	"context"
	"encoding/json"
	"strings"

	"vibecms/internal/models"
)

func (c *coreImpl) RegisterNodeType(_ context.Context, input NodeTypeInput) (*NodeType, error) {
	if input.Slug == "" {
		return nil, NewValidation("slug is required")
	}
	if input.Label == "" {
		return nil, NewValidation("label is required")
	}

	fieldSchemaJSON, err := json.Marshal(input.FieldSchema)
	if err != nil {
		return nil, NewInternal("failed to marshal field_schema: " + err.Error())
	}
	urlPrefixesJSON, err := json.Marshal(input.URLPrefixes)
	if err != nil {
		return nil, NewInternal("failed to marshal url_prefixes: " + err.Error())
	}

	nt := &models.NodeType{
		Slug:        input.Slug,
		Label:       input.Label,
		Icon:        input.Icon,
		Description: input.Description,
		FieldSchema: models.JSONB(fieldSchemaJSON),
		URLPrefixes: models.JSONB(urlPrefixesJSON),
	}
	if nt.Icon == "" {
		nt.Icon = "file-text"
	}

	if err := c.nodeTypeSvc.Create(nt); err != nil {
		if strings.Contains(err.Error(), "slug conflict") {
			return nil, NewValidation(err.Error())
		}
		return nil, NewInternal(err.Error())
	}

	return nodeTypeFromModel(nt), nil
}

func (c *coreImpl) GetNodeType(_ context.Context, slug string) (*NodeType, error) {
	nt, err := c.nodeTypeSvc.GetBySlug(slug)
	if err != nil {
		return nil, NewNotFound("node_type", slug)
	}
	return nodeTypeFromModel(nt), nil
}

func (c *coreImpl) ListNodeTypes(_ context.Context) ([]*NodeType, error) {
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

func (c *coreImpl) UpdateNodeType(_ context.Context, slug string, input NodeTypeInput) (*NodeType, error) {
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
	if input.Icon != "" {
		updates["icon"] = input.Icon
	}
	if input.Description != "" {
		updates["description"] = input.Description
	}
	if input.FieldSchema != nil {
		updates["field_schema"] = input.FieldSchema
	}
	if input.URLPrefixes != nil {
		updates["url_prefixes"] = input.URLPrefixes
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

func (c *coreImpl) DeleteNodeType(_ context.Context, slug string) error {
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
		ID:          nt.ID,
		Slug:        nt.Slug,
		Label:       nt.Label,
		Icon:        nt.Icon,
		Description: nt.Description,
		CreatedAt:   nt.CreatedAt,
		UpdatedAt:   nt.UpdatedAt,
	}

	// Parse FieldSchema from JSONB
	if len(nt.FieldSchema) > 0 {
		var fields []NodeTypeField
		if err := json.Unmarshal([]byte(nt.FieldSchema), &fields); err == nil {
			result.FieldSchema = fields
		}
	}
	if result.FieldSchema == nil {
		result.FieldSchema = []NodeTypeField{}
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
