package cms

import (
	"encoding/json"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"squilla/internal/events"
	"squilla/internal/models"
)

// NodeTypeService provides business logic for managing custom node types.
type NodeTypeService struct {
	db       *gorm.DB
	eventBus *events.EventBus
}

// NewNodeTypeService creates a new NodeTypeService with the given database connection.
func NewNodeTypeService(db *gorm.DB, eventBus *events.EventBus) *NodeTypeService {
	return &NodeTypeService{db: db, eventBus: eventBus}
}

// DB exposes the underlying GORM connection for handlers that need to run
// auxiliary queries (e.g. merging the standalone taxonomies table into a
// node type response).
func (s *NodeTypeService) DB() *gorm.DB { return s.db }

func (s *NodeTypeService) emit(action string, id int, slug string) {
	if s.eventBus == nil {
		return
	}
	s.eventBus.Publish(action, events.Payload{"id": id, "slug": slug})
}

// List retrieves a paginated list of node types ordered by label.
func (s *NodeTypeService) List(page, perPage int) ([]models.NodeType, int64, error) {
	var total int64
	if err := s.db.Model(&models.NodeType{}).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("counting node types: %w", err)
	}

	var nodeTypes []models.NodeType
	err := s.db.
		Order("label ASC").
		Offset((page - 1) * perPage).
		Limit(perPage).
		Find(&nodeTypes).Error
	if err != nil {
		return nil, 0, fmt.Errorf("listing node types: %w", err)
	}
	return nodeTypes, total, nil
}

// ListAll retrieves all node types ordered by label (for internal use).
func (s *NodeTypeService) ListAll() ([]models.NodeType, error) {
	var nodeTypes []models.NodeType
	if err := s.db.Order("label ASC").Find(&nodeTypes).Error; err != nil {
		return nil, fmt.Errorf("listing node types: %w", err)
	}
	return nodeTypes, nil
}

// GetByID retrieves a single node type by its ID.
func (s *NodeTypeService) GetByID(id int) (*models.NodeType, error) {
	var nt models.NodeType
	if err := s.db.First(&nt, id).Error; err != nil {
		return nil, err
	}
	return &nt, nil
}

// GetBySlug retrieves a single node type by its slug.
func (s *NodeTypeService) GetBySlug(slug string) (*models.NodeType, error) {
	var nt models.NodeType
	if err := s.db.Where("slug = ?", slug).First(&nt).Error; err != nil {
		return nil, err
	}
	return &nt, nil
}

// Create inserts a new node type after validating slug uniqueness.
func (s *NodeTypeService) Create(nt *models.NodeType) error {
	if nt.Slug == "" {
		return fmt.Errorf("validation error: slug is required")
	}
	if nt.Label == "" {
		return fmt.Errorf("validation error: label is required")
	}

	// Check slug uniqueness
	var count int64
	s.db.Model(&models.NodeType{}).Where("slug = ?", nt.Slug).Count(&count)
	if count > 0 {
		return fmt.Errorf("slug conflict: node type with slug %q already exists", nt.Slug)
	}

	if err := s.db.Create(nt).Error; err != nil {
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "23505") {
			return fmt.Errorf("slug conflict: node type with slug %q already exists", nt.Slug)
		}
		return fmt.Errorf("creating node type: %w", err)
	}

	s.emit("node_type.created", nt.ID, nt.Slug)
	return nil
}

// Update performs a partial update on a node type by ID.
func (s *NodeTypeService) Update(id int, updates map[string]interface{}) (*models.NodeType, error) {
	existing, err := s.GetByID(id)
	if err != nil {
		return nil, err
	}

	// Validate slug uniqueness if slug is being changed
	if newSlug, ok := updates["slug"].(string); ok && newSlug != "" && newSlug != existing.Slug {
		var count int64
		s.db.Model(&models.NodeType{}).Where("slug = ? AND id != ?", newSlug, id).Count(&count)
		if count > 0 {
			return nil, fmt.Errorf("slug conflict: node type with slug %q already exists", newSlug)
		}
	}

	// The wire vocabulary uses "fields"; the DB column is "field_schema".
	// GORM's Updates(map) treats keys as column names verbatim, so rename
	// before handing it off — otherwise we'd issue UPDATE ... SET fields=...
	// against a column that doesn't exist and roll back the whole row.
	if val, ok := updates["fields"]; ok {
		delete(updates, "fields")
		updates["field_schema"] = val
	}

	// Convert JSONB fields from parsed JSON (map/slice) to models.JSONB
	for _, key := range []string{"field_schema", "url_prefixes", "taxonomies"} {
		if val, ok := updates[key]; ok && val != nil {
			b, err := json.Marshal(val)
			if err == nil {
				updates[key] = models.JSONB(b)
			}
		}
	}

	if err := s.db.Model(existing).Updates(updates).Error; err != nil {
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "23505") {
			slug := updates["slug"]
			return nil, fmt.Errorf("slug conflict: node type with slug %q already exists", slug)
		}
		return nil, fmt.Errorf("updating node type: %w", err)
	}

	// Re-fetch updated node type
	updated, err := s.GetByID(id)
	if err != nil {
		return nil, err
	}

	// full_url is materialized on each content_node at save time, so a
	// url_prefixes (or slug) change here would otherwise leave existing
	// nodes pointing at the old prefix until each was individually
	// re-saved. Mirrors LanguageService.rebuildAllURLsForLanguage.
	_, prefixesChanged := updates["url_prefixes"]
	_, slugChanged := updates["slug"]
	if prefixesChanged || slugChanged {
		s.rebuildAllURLsForNodeType(updated.Slug)
	}

	s.emit("node_type.updated", updated.ID, updated.Slug)
	return updated, nil
}

// rebuildAllURLsForNodeType recomputes full_url for every content node of
// the given type using the canonical buildFullURL logic. Called whenever
// a url_prefixes or slug change would invalidate the materialized URLs.
func (s *NodeTypeService) rebuildAllURLsForNodeType(nodeTypeSlug string) {
	var nodes []models.ContentNode
	s.db.Where("node_type = ? AND deleted_at IS NULL", nodeTypeSlug).Find(&nodes)
	for _, node := range nodes {
		newURL := buildFullURL(&node, s.db)
		if newURL != node.FullURL {
			s.db.Model(&node).Update("full_url", newURL)
		}
	}
}

// Delete removes a node type by ID. Built-in types ("page" and "post") cannot be deleted.
// Content nodes of this type are preserved as "dormant" rows — invisible to
// queries (which filter by active node_types) but resurrectable if the type
// is re-registered later.
func (s *NodeTypeService) Delete(id int) error {
	existing, err := s.GetByID(id)
	if err != nil {
		return err
	}

	if existing.Slug == "page" || existing.Slug == "post" {
		return fmt.Errorf("cannot delete built-in node type %q", existing.Slug)
	}

	result := s.db.Delete(&models.NodeType{}, id)
	if result.Error != nil {
		return fmt.Errorf("deleting node type: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	s.emit("node_type.deleted", existing.ID, existing.Slug)
	return nil
}
