package cms

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"squilla/internal/events"
	"squilla/internal/models"
)

// defaultRevisionsPerNode caps revisions per content node. 50 is enough to
// recover from accidents while preventing unbounded growth on chatty
// editors who save every few seconds.
const defaultRevisionsPerNode = 50

// defaultJSONB returns src when it has bytes; otherwise returns the given
// fallback as JSONB. Used to keep snapshot rows from violating the
// jsonb NOT NULL constraint when the source column is empty.
func defaultJSONB(src models.JSONB, fallback string) models.JSONB {
	if len(src) > 0 {
		return src
	}
	return models.JSONB(fallback)
}

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
func (s *ContentService) List(page, perPage int, status, nodeType, langCode, search string, taxQuery map[string][]string) ([]models.ContentNode, int64, error) {
	var nodes []models.ContentNode
	var total int64

	query := s.db.Model(&models.ContentNode{}).
		Where("node_type IN (?)", s.db.Model(&models.NodeType{}).Select("slug"))

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

	if len(taxQuery) > 0 {
		for tax, terms := range taxQuery {
			if len(terms) > 0 {
				if len(terms) == 1 {
					query = query.Where("taxonomies->? ? ?", tax, terms[0])
				} else {
					b, _ := json.Marshal(terms)
					query = query.Where("taxonomies->? @> ?", tax, b)
				}
			}
		}
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

	// Stamp the creating user as the author when one is available. Without
	// this, scope='own' RBAC mode (per dev guide §3.4) is unusable — every
	// node would be unowned. userID 0 = anonymous/system caller (MCP, seed,
	// extension); leave AuthorID nil for those.
	if node.AuthorID == nil && userID > 0 {
		uid := userID
		node.AuthorID = &uid
	}

	node.FullURL = buildFullURL(node, s.db)

	// Set published_at if creating as published.
	if node.Status == "published" && node.PublishedAt == nil {
		now := time.Now()
		node.PublishedAt = &now
	}

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

	// Create a revision snapshot before updating. userID 0 means the update
	// was issued by kernel infrastructure (MCP, extensions) rather than an
	// authenticated admin; store NULL instead of violating the FK.
	revision := models.ContentNodeRevision{
		NodeID:             existing.ID,
		Title:              existing.Title,
		Slug:               existing.Slug,
		Status:             existing.Status,
		LanguageCode:       existing.LanguageCode,
		LayoutSlug:         existing.LayoutSlug,
		Excerpt:            existing.Excerpt,
		FeaturedImage:      defaultJSONB(existing.FeaturedImage, "{}"),
		BlocksSnapshot:     defaultJSONB(existing.BlocksData, "[]"),
		FieldsSnapshot:     defaultJSONB(existing.FieldsData, "{}"),
		SeoSnapshot:        defaultJSONB(existing.SeoSettings, "{}"),
		TaxonomiesSnapshot: defaultJSONB(existing.Taxonomies, "{}"),
		VersionNumber:      existing.Version,
	}
	if userID > 0 {
		uid := userID
		revision.CreatedBy = &uid
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
	for _, key := range []string{"blocks_data", "seo_settings", "fields_data", "layout_data", "featured_image", "taxonomies"} {
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
		SeoSettings:        models.JSONB("{}"),
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
