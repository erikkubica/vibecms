package cms

import (
	"fmt"
	"strings"

	"gorm.io/gorm"

	"vibecms/internal/models"
)

// ContentService provides business logic for managing content nodes.
type ContentService struct {
	db *gorm.DB
}

// NewContentService creates a new ContentService with the given database connection.
func NewContentService(db *gorm.DB) *ContentService {
	return &ContentService{db: db}
}

// List retrieves a paginated list of content nodes with optional filters.
// For performance, blocks_data is excluded from the result set.
func (s *ContentService) List(page, perPage int, status, nodeType, langCode, search string) ([]models.ContentNode, int64, error) {
	var nodes []models.ContentNode
	var total int64

	query := s.db.Model(&models.ContentNode{})

	if status != "" {
		query = query.Where("status = ?", status)
	}
	if nodeType != "" {
		query = query.Where("node_type = ?", nodeType)
	}
	if langCode != "" {
		query = query.Where("language_code = ?", langCode)
	}
	if search != "" {
		searchTerm := "%" + search + "%"
		query = query.Where("title ILIKE ? OR slug ILIKE ?", searchTerm, searchTerm)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("counting content nodes: %w", err)
	}

	offset := (page - 1) * perPage
	err := query.
		Select("id, uuid, parent_id, node_type, status, language_code, slug, full_url, title, seo_settings, translation_group_id, version, published_at, created_at, updated_at, deleted_at").
		Order("created_at DESC").
		Offset(offset).
		Limit(perPage).
		Find(&nodes).Error
	if err != nil {
		return nil, 0, fmt.Errorf("listing content nodes: %w", err)
	}

	return nodes, total, nil
}

// GetByID retrieves a single content node by its ID, including blocks_data.
func (s *ContentService) GetByID(id int) (*models.ContentNode, error) {
	var node models.ContentNode
	if err := s.db.First(&node, id).Error; err != nil {
		return nil, err
	}
	return &node, nil
}

// Create inserts a new content node. If the slug is empty, it is auto-generated
// from the title. The full_url is built from the parent chain, language code, and slug.
func (s *ContentService) Create(node *models.ContentNode, userID int) error {
	if node.Slug == "" {
		node.Slug = Slugify(node.Title)
	}

	if err := ValidateSlug(node.Slug); err != nil {
		return fmt.Errorf("invalid slug: %w", err)
	}

	node.FullURL = buildFullURL(node, s.db)

	// Check full_url uniqueness
	var count int64
	s.db.Model(&models.ContentNode{}).Where("full_url = ?", node.FullURL).Count(&count)
	if count > 0 {
		return fmt.Errorf("slug conflict: full_url %q already exists", node.FullURL)
	}

	if err := s.db.Create(node).Error; err != nil {
		return fmt.Errorf("creating content node: %w", err)
	}

	return nil
}

// Update performs a partial update on a content node. A revision is created before
// applying changes. The version is incremented, and full_url is rebuilt if the slug changes.
func (s *ContentService) Update(id int, updates map[string]interface{}, userID int) (*models.ContentNode, error) {
	existing, err := s.GetByID(id)
	if err != nil {
		return nil, err
	}

	// Create a revision snapshot before updating
	revision := models.ContentNodeRevision{
		NodeID:         existing.ID,
		BlocksSnapshot: existing.BlocksData,
		SeoSnapshot:    existing.SeoSettings,
		CreatedBy:      &userID,
	}
	if err := s.db.Create(&revision).Error; err != nil {
		return nil, fmt.Errorf("creating revision: %w", err)
	}

	// Validate slug if it is being changed
	if newSlug, ok := updates["slug"].(string); ok && newSlug != "" {
		if err := ValidateSlug(newSlug); err != nil {
			return nil, fmt.Errorf("invalid slug: %w", err)
		}
	}

	// Increment version
	updates["version"] = existing.Version + 1

	if err := s.db.Model(existing).Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("updating content node: %w", err)
	}

	// Rebuild full_url if slug, parent_id, or language_code changed
	rebuildNeeded := false
	for key := range updates {
		if key == "slug" || key == "parent_id" || key == "language_code" {
			rebuildNeeded = true
			break
		}
	}

	if rebuildNeeded {
		// Re-fetch to get the updated fields
		updated, err := s.GetByID(id)
		if err != nil {
			return nil, err
		}

		newFullURL := buildFullURL(updated, s.db)

		// Check uniqueness excluding current node
		var count int64
		s.db.Model(&models.ContentNode{}).
			Where("full_url = ? AND id != ?", newFullURL, id).
			Count(&count)
		if count > 0 {
			return nil, fmt.Errorf("slug conflict: full_url %q already exists", newFullURL)
		}

		s.db.Model(updated).Update("full_url", newFullURL)
		updated.FullURL = newFullURL
		return updated, nil
	}

	// Re-fetch updated node
	updated, err := s.GetByID(id)
	if err != nil {
		return nil, err
	}
	return updated, nil
}

// Delete performs a soft delete on a content node.
func (s *ContentService) Delete(id int) error {
	result := s.db.Delete(&models.ContentNode{}, id)
	if result.Error != nil {
		return fmt.Errorf("deleting content node: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// ValidateSlugUnique checks whether a slug/full_url is unique, optionally
// excluding a specific node ID (for update scenarios).
func (s *ContentService) ValidateSlugUnique(slug string, excludeID int) (bool, error) {
	var count int64
	query := s.db.Model(&models.ContentNode{}).Where("slug = ?", slug)
	if excludeID > 0 {
		query = query.Where("id != ?", excludeID)
	}
	if err := query.Count(&count).Error; err != nil {
		return false, fmt.Errorf("checking slug uniqueness: %w", err)
	}
	return count == 0, nil
}

// buildFullURL constructs the full URL path for a content node based on its
// language code, parent hierarchy, and slug.
// Format: /{language_code}/{parent_slugs...}/{slug}
// Special case: if slug is "index", full_url is /{language_code} (or / for default).
func buildFullURL(node *models.ContentNode, db *gorm.DB) string {
	lang := node.LanguageCode
	if lang == "" {
		lang = "en"
	}

	// Special case for index pages
	if node.Slug == "index" {
		return "/" + lang
	}

	// Build parent slug chain
	var segments []string
	if node.ParentID != nil {
		segments = collectParentSlugs(*node.ParentID, db)
	}
	segments = append(segments, node.Slug)

	return "/" + lang + "/" + strings.Join(segments, "/")
}

// collectParentSlugs recursively collects slugs from parent nodes.
func collectParentSlugs(parentID int, db *gorm.DB) []string {
	var parent models.ContentNode
	if err := db.Select("id, parent_id, slug").First(&parent, parentID).Error; err != nil {
		return nil
	}

	var segments []string
	if parent.ParentID != nil {
		segments = collectParentSlugs(*parent.ParentID, db)
	}
	segments = append(segments, parent.Slug)
	return segments
}
