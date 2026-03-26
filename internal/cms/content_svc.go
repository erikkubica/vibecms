package cms

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"vibecms/internal/events"
	"vibecms/internal/models"
)

// ContentService provides business logic for managing content nodes.
type ContentService struct {
	db       *gorm.DB
	eventBus *events.EventBus
}

// NewContentService creates a new ContentService with the given database connection.
func NewContentService(db *gorm.DB, eventBus *events.EventBus) *ContentService {
	return &ContentService{db: db, eventBus: eventBus}
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

	if s.eventBus != nil {
		go s.eventBus.Publish("node.created", events.Payload{
			"node_id":   node.ID,
			"node_type": node.NodeType,
			"node_title": node.Title,
			"node_slug": node.Slug,
			"full_url":  node.FullURL,
			"author_id": node.AuthorID,
		})
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

		if s.eventBus != nil {
			go s.eventBus.Publish("node.updated", events.Payload{
				"node_id":    updated.ID,
				"node_type":  updated.NodeType,
				"node_title": updated.Title,
				"node_slug":  updated.Slug,
				"full_url":   updated.FullURL,
				"author_id":  updated.AuthorID,
			})

			// Check for publish/unpublish status transitions
			if newStatus, ok := updates["status"].(string); ok {
				if newStatus == "published" && existing.Status != "published" {
					go s.eventBus.Publish("node.published", events.Payload{
						"node_id":    updated.ID,
						"node_type":  updated.NodeType,
						"node_title": updated.Title,
						"node_slug":  updated.Slug,
						"full_url":   updated.FullURL,
						"author_id":  updated.AuthorID,
					})
				} else if newStatus != "published" && existing.Status == "published" {
					go s.eventBus.Publish("node.unpublished", events.Payload{
						"node_id":    updated.ID,
						"node_type":  updated.NodeType,
						"node_title": updated.Title,
						"node_slug":  updated.Slug,
						"full_url":   updated.FullURL,
						"author_id":  updated.AuthorID,
					})
				}
			}
		}

		return updated, nil
	}

	// Re-fetch updated node
	updated, err := s.GetByID(id)
	if err != nil {
		return nil, err
	}

	if s.eventBus != nil {
		go s.eventBus.Publish("node.updated", events.Payload{
			"node_id":   updated.ID,
			"node_type": updated.NodeType,
			"node_title": updated.Title,
			"node_slug": updated.Slug,
			"full_url":  updated.FullURL,
			"author_id": updated.AuthorID,
		})

		// Check for publish/unpublish status transitions
		if newStatus, ok := updates["status"].(string); ok {
			if newStatus == "published" && existing.Status != "published" {
				go s.eventBus.Publish("node.published", events.Payload{
					"node_id":   updated.ID,
					"node_type": updated.NodeType,
					"node_title": updated.Title,
					"node_slug": updated.Slug,
					"full_url":  updated.FullURL,
					"author_id": updated.AuthorID,
				})
			} else if newStatus != "published" && existing.Status == "published" {
				go s.eventBus.Publish("node.unpublished", events.Payload{
					"node_id":   updated.ID,
					"node_type": updated.NodeType,
					"node_title": updated.Title,
					"node_slug": updated.Slug,
					"full_url":  updated.FullURL,
					"author_id": updated.AuthorID,
				})
			}
		}
	}

	return updated, nil
}

// Delete performs a soft delete on a content node.
func (s *ContentService) Delete(id int) error {
	// Fetch node before deletion for event payload
	var node models.ContentNode
	if s.eventBus != nil {
		if n, err := s.GetByID(id); err == nil {
			node = *n
		}
	}

	result := s.db.Delete(&models.ContentNode{}, id)
	if result.Error != nil {
		return fmt.Errorf("deleting content node: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	if s.eventBus != nil {
		go s.eventBus.Publish("node.deleted", events.Payload{
			"node_id":   node.ID,
			"node_type": node.NodeType,
			"node_title": node.Title,
			"node_slug": node.Slug,
			"full_url":  node.FullURL,
			"author_id": node.AuthorID,
		})
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

// GetTranslations returns all translation siblings of the given node.
// If the node has no translation_group_id, an empty slice is returned.
func (s *ContentService) GetTranslations(nodeID int) ([]models.ContentNode, error) {
	node, err := s.GetByID(nodeID)
	if err != nil {
		return nil, err
	}

	if node.TranslationGroupID == nil || *node.TranslationGroupID == "" {
		return []models.ContentNode{}, nil
	}

	var translations []models.ContentNode
	err = s.db.
		Where("translation_group_id = ? AND id != ?", *node.TranslationGroupID, nodeID).
		Find(&translations).Error
	if err != nil {
		return nil, fmt.Errorf("fetching translations: %w", err)
	}

	return translations, nil
}

// CreateTranslation clones a content node as a translation in a new language.
func (s *ContentService) CreateTranslation(sourceID int, targetLangCode string) (*models.ContentNode, error) {
	source, err := s.GetByID(sourceID)
	if err != nil {
		return nil, err
	}

	// Ensure the source node has a translation_group_id
	if source.TranslationGroupID == nil || *source.TranslationGroupID == "" {
		groupID := uuid.New().String()
		source.TranslationGroupID = &groupID
		if err := s.db.Model(source).Update("translation_group_id", groupID).Error; err != nil {
			return nil, fmt.Errorf("setting translation group on source: %w", err)
		}
	}

	newNode := models.ContentNode{
		Title:              fmt.Sprintf("[%s] %s", targetLangCode, source.Title),
		Slug:               source.Slug,
		NodeType:           source.NodeType,
		BlocksData:         source.BlocksData,
		FieldsData:         source.FieldsData,
		SeoSettings:        source.SeoSettings,
		Status:             "draft",
		LanguageCode:       targetLangCode,
		ParentID:           source.ParentID,
		TranslationGroupID: source.TranslationGroupID,
	}

	newNode.FullURL = buildFullURL(&newNode, s.db)

	// Handle slug conflicts by appending -2, -3, etc.
	baseFullURL := newNode.FullURL
	baseSlug := newNode.Slug
	for suffix := 2; ; suffix++ {
		var count int64
		s.db.Model(&models.ContentNode{}).Where("full_url = ?", newNode.FullURL).Count(&count)
		if count == 0 {
			break
		}
		newNode.Slug = fmt.Sprintf("%s-%d", baseSlug, suffix)
		newNode.FullURL = fmt.Sprintf("%s-%d", baseFullURL, suffix)
	}

	if err := s.db.Create(&newNode).Error; err != nil {
		return nil, fmt.Errorf("creating translation node: %w", err)
	}

	return &newNode, nil
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
