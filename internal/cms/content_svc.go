package cms

import (
	"encoding/json"
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
		Select("id, uuid, parent_id, node_type, status, language_code, slug, full_url, title, seo_settings, fields_data, translation_group_id, version, published_at, created_at, updated_at, deleted_at").
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
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "23505") {
			return fmt.Errorf("slug conflict: full_url %q already exists", node.FullURL)
		}
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

	// Convert JSONB fields from parsed JSON (map/slice) to models.JSONB
	for _, key := range []string{"blocks_data", "seo_settings", "fields_data"} {
		if val, ok := updates[key]; ok && val != nil {
			b, err := json.Marshal(val)
			if err == nil {
				updates[key] = models.JSONB(b)
			}
		}
	}

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
// language code, node type, parent hierarchy, and slug.
// Format for page/post: /{language_code}/{parent_slugs...}/{slug}
// Format for custom types: /{language_code}/{node_type_slug}/{parent_slugs...}/{slug}
// Special case: if slug is "index", full_url is /{language_code} (or / for default).
func buildFullURL(node *models.ContentNode, db *gorm.DB) string {
	langCode := node.LanguageCode
	if langCode == "" {
		langCode = "en"
	}

	// Resolve language slug (URL segment) from language code
	langSlug := resolveLanguageSlug(langCode, db)

	// Special case for index pages
	if node.Slug == "index" {
		if langSlug == "" {
			return "/"
		}
		return "/" + langSlug
	}

	// Build segment chain
	var segments []string

	// Add language slug prefix (empty if hide_prefix is on)
	if langSlug != "" {
		segments = append(segments, langSlug)
	}

	// Custom node types get a URL prefix (translated if available)
	if node.NodeType != "page" && node.NodeType != "" {
		prefix := resolveURLPrefix(node.NodeType, langCode, db)
		if prefix != "" {
			segments = append(segments, prefix)
		}
	}

	if node.ParentID != nil {
		segments = append(segments, collectParentSlugs(*node.ParentID, db)...)
	}
	segments = append(segments, node.Slug)

	return "/" + strings.Join(segments, "/")
}

// resolveLanguageSlug returns the URL slug for a language code.
// Returns empty string if language has hide_prefix enabled.
// Falls back to the code itself if no language record is found.
func resolveLanguageSlug(langCode string, db *gorm.DB) string {
	var lang models.Language
	if err := db.Select("slug, hide_prefix").Where("code = ?", langCode).First(&lang).Error; err != nil {
		return langCode // fallback to code
	}
	if lang.HidePrefix {
		return ""
	}
	if lang.Slug == "" {
		return langCode
	}
	return lang.Slug
}

// resolveURLPrefix returns the URL prefix for a node type in the given language.
// It checks the node_types table for a translated prefix. Falls back to the type slug.
func resolveURLPrefix(nodeType, lang string, db *gorm.DB) string {
	var nt models.NodeType
	if err := db.Select("slug, url_prefixes").Where("slug = ?", nodeType).First(&nt).Error; err != nil {
		return nodeType // fallback to type slug
	}

	// Parse url_prefixes JSONB
	var prefixes map[string]string
	if len(nt.URLPrefixes) > 0 {
		if err := json.Unmarshal([]byte(nt.URLPrefixes), &prefixes); err == nil {
			if p, ok := prefixes[lang]; ok && p != "" {
				return p
			}
		}
	}

	return nodeType // fallback to type slug
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
