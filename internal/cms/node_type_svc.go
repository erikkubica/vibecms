package cms

import (
	"encoding/json"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"vibecms/internal/models"
)

// NodeTypeService provides business logic for managing custom node types.
type NodeTypeService struct {
	db *gorm.DB
}

// NewNodeTypeService creates a new NodeTypeService with the given database connection.
func NewNodeTypeService(db *gorm.DB) *NodeTypeService {
	return &NodeTypeService{db: db}
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

	// Convert JSONB fields from parsed JSON (map/slice) to models.JSONB
	for _, key := range []string{"field_schema", "url_prefixes"} {
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
	return updated, nil
}

// Delete removes a node type by ID. Built-in types ("page" and "post") cannot be deleted.
func (s *NodeTypeService) Delete(id int) error {
	existing, err := s.GetByID(id)
	if err != nil {
		return err
	}

	// Prevent deleting built-in types
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
	return nil
}
