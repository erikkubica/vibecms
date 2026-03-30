package cms

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"gorm.io/gorm"

	"vibecms/internal/events"
	"vibecms/internal/models"
)

// LayoutService provides business logic for managing layouts.
type LayoutService struct {
	db          *gorm.DB
	cache       sync.Map
	eventBus    *events.EventBus
	themeAssets *ThemeAssetRegistry
}

// NewLayoutService creates a new LayoutService with the given database connection.
func NewLayoutService(db *gorm.DB, eventBus *events.EventBus, themeAssets *ThemeAssetRegistry) *LayoutService {
	return &LayoutService{db: db, eventBus: eventBus, themeAssets: themeAssets}
}

// List retrieves layouts with optional filters for language_id and source.
func (s *LayoutService) List(languageID *int, source string) ([]models.Layout, error) {
	var layouts []models.Layout
	q := s.db.Order("name ASC")
	if languageID != nil {
		q = q.Where("language_id = ?", *languageID)
	}
	if source != "" {
		q = q.Where("source = ?", source)
	}
	if err := q.Find(&layouts).Error; err != nil {
		return nil, fmt.Errorf("failed to list layouts: %w", err)
	}
	return layouts, nil
}

// GetByID retrieves a single layout by its ID.
func (s *LayoutService) GetByID(id int) (*models.Layout, error) {
	var layout models.Layout
	if err := s.db.First(&layout, id).Error; err != nil {
		return nil, err
	}
	return &layout, nil
}

// Create inserts a new layout after validating slug+language uniqueness.
func (s *LayoutService) Create(layout *models.Layout) error {
	// Check slug+language_id uniqueness
	var count int64
	if layout.LanguageID != nil {
		s.db.Model(&models.Layout{}).Where("slug = ? AND language_id = ?", layout.Slug, *layout.LanguageID).Count(&count)
	} else {
		s.db.Model(&models.Layout{}).Where("slug = ? AND language_id IS NULL", layout.Slug).Count(&count)
	}
	if count > 0 {
		return fmt.Errorf("SLUG_CONFLICT")
	}

	if err := s.db.Create(layout).Error; err != nil {
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "23505") {
			return fmt.Errorf("SLUG_CONFLICT")
		}
		return fmt.Errorf("failed to create layout: %w", err)
	}

	s.InvalidateCache()

	if s.eventBus != nil {
		go s.eventBus.Publish("layout.created", events.Payload{
			"layout_id":   layout.ID,
			"layout_slug": layout.Slug,
			"layout_name": layout.Name,
		})
	}

	return nil
}

// Update performs a partial update on a layout by ID.
func (s *LayoutService) Update(id int, updates map[string]interface{}) (*models.Layout, error) {
	existing, err := s.GetByID(id)
	if err != nil {
		return nil, err
	}

	// Block edits to theme-sourced layouts
	if existing.Source == "theme" {
		return nil, fmt.Errorf("THEME_READONLY")
	}

	// Validate slug+language uniqueness if slug is being changed
	if newSlug, ok := updates["slug"].(string); ok && newSlug != "" && newSlug != existing.Slug {
		langID := existing.LanguageID
		if lid, ok := updates["language_id"]; ok {
			if lid == nil {
				langID = nil
			} else if lidFloat, ok := lid.(float64); ok {
				lidInt := int(lidFloat)
				langID = &lidInt
			}
		}
		var count int64
		if langID != nil {
			s.db.Model(&models.Layout{}).Where("slug = ? AND language_id = ? AND id != ?", newSlug, *langID, id).Count(&count)
		} else {
			s.db.Model(&models.Layout{}).Where("slug = ? AND language_id IS NULL AND id != ?", newSlug, id).Count(&count)
		}
		if count > 0 {
			return nil, fmt.Errorf("SLUG_CONFLICT")
		}
	}

	if err := s.db.Model(existing).Updates(updates).Error; err != nil {
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "23505") {
			return nil, fmt.Errorf("SLUG_CONFLICT")
		}
		return nil, fmt.Errorf("failed to update layout: %w", err)
	}

	s.InvalidateCache()

	// Re-fetch updated layout
	updated, err := s.GetByID(id)
	if err != nil {
		return nil, err
	}

	if s.eventBus != nil {
		go s.eventBus.Publish("layout.updated", events.Payload{
			"layout_id":   updated.ID,
			"layout_slug": updated.Slug,
			"layout_name": updated.Name,
		})
	}

	return updated, nil
}

// Delete removes a layout by ID. Theme-sourced layouts cannot be deleted.
func (s *LayoutService) Delete(id int) error {
	existing, err := s.GetByID(id)
	if err != nil {
		return err
	}

	if existing.Source == "theme" {
		return fmt.Errorf("THEME_READONLY")
	}

	result := s.db.Delete(&models.Layout{}, id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete layout: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	s.InvalidateCache()

	if s.eventBus != nil {
		go s.eventBus.Publish("layout.deleted", events.Payload{
			"layout_id":   existing.ID,
			"layout_slug": existing.Slug,
			"layout_name": existing.Name,
		})
	}

	return nil
}

// Detach converts a theme-sourced layout to a custom layout.
func (s *LayoutService) Detach(id int) (*models.Layout, error) {
	existing, err := s.GetByID(id)
	if err != nil {
		return nil, err
	}

	if err := s.db.Model(existing).Updates(map[string]interface{}{
		"source":     "custom",
		"theme_name": nil,
	}).Error; err != nil {
		return nil, fmt.Errorf("failed to detach layout: %w", err)
	}

	s.InvalidateCache()

	updated, err := s.GetByID(id)
	if err != nil {
		return nil, err
	}
	return updated, nil
}

// Reattach restores a detached layout to its theme version by re-reading the theme file.
func (s *LayoutService) Reattach(id int) (*models.Layout, error) {
	existing, err := s.GetByID(id)
	if err != nil {
		return nil, err
	}
	if existing.Source == "theme" {
		return existing, nil
	}

	if s.themeAssets == nil {
		return nil, fmt.Errorf("no theme loaded")
	}

	s.themeAssets.mu.RLock()
	themeDir := s.themeAssets.themeDir
	s.themeAssets.mu.RUnlock()

	if themeDir == "" {
		return nil, fmt.Errorf("no theme directory configured")
	}

	manifestData, err := os.ReadFile(filepath.Join(themeDir, "theme.json"))
	if err != nil {
		return nil, fmt.Errorf("failed to read theme.json: %w", err)
	}
	var manifest ThemeManifest
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse theme.json: %w", err)
	}

	for _, def := range manifest.Layouts {
		if def.Slug == existing.Slug {
			code, err := os.ReadFile(filepath.Join(themeDir, "layouts", def.File))
			if err != nil {
				return nil, fmt.Errorf("failed to read theme layout file: %w", err)
			}
			themeName := manifest.Name
			if err := s.db.Model(existing).Updates(map[string]interface{}{
				"template_code": string(code),
				"source":        "theme",
				"theme_name":    &themeName,
				"is_default":    def.IsDefault,
			}).Error; err != nil {
				return nil, fmt.Errorf("failed to reattach layout: %w", err)
			}
			s.InvalidateCache()
			return s.GetByID(id)
		}
	}

	return nil, fmt.Errorf("layout %q not found in theme", existing.Slug)
}

// ResolveForNode resolves the best layout for a content node using cascade resolution.
// Priority: 1) explicit LayoutID, 2) layout-{type}-{slug} by language_id, 3) layout-{type} by language_id,
// 4) is_default=true by language_id, 5) error.
func (s *LayoutService) ResolveForNode(node *models.ContentNode, languageID *int) (*models.Layout, error) {
	// 1. Explicit LayoutID on the node
	if node.LayoutID != nil {
		cacheKey := fmt.Sprintf("id:%d", *node.LayoutID)
		if cached, ok := s.cache.Load(cacheKey); ok {
			if cached == nil {
				return nil, fmt.Errorf("no layout found")
			}
			return cached.(*models.Layout), nil
		}
		layout, err := s.GetByID(*node.LayoutID)
		if err != nil {
			s.cache.Store(cacheKey, nil)
			return nil, fmt.Errorf("no layout found")
		}
		s.cache.Store(cacheKey, layout)
		return layout, nil
	}

	// 2. layout-{nodeType}-{slug}
	specificSlug := fmt.Sprintf("layout-%s-%s", node.NodeType, node.Slug)
	if layout := s.findBySlugAndLang(specificSlug, languageID); layout != nil {
		return layout, nil
	}

	// 3. layout-{nodeType}
	typeSlug := fmt.Sprintf("layout-%s", node.NodeType)
	if layout := s.findBySlugAndLang(typeSlug, languageID); layout != nil {
		return layout, nil
	}

	// 4. is_default=true
	if layout := s.findDefault(languageID); layout != nil {
		return layout, nil
	}

	return nil, fmt.Errorf("no layout found")
}

// findBySlugAndLang looks up a layout by slug, trying specific language_id first, then NULL (all languages).
func (s *LayoutService) findBySlugAndLang(slug string, languageID *int) *models.Layout {
	type langQuery struct {
		id       *int
		cacheKey string
	}

	queries := []langQuery{}
	if languageID != nil {
		queries = append(queries, langQuery{id: languageID, cacheKey: fmt.Sprintf("slug:%s:lang:%d", slug, *languageID)})
	}
	queries = append(queries, langQuery{id: nil, cacheKey: fmt.Sprintf("slug:%s:lang:null", slug)})

	for _, q := range queries {
		if cached, ok := s.cache.Load(q.cacheKey); ok {
			if cached == nil {
				continue
			}
			return cached.(*models.Layout)
		}

		var layout models.Layout
		var err error
		if q.id != nil {
			err = s.db.Where("slug = ? AND language_id = ?", slug, *q.id).First(&layout).Error
		} else {
			err = s.db.Where("slug = ? AND language_id IS NULL", slug).First(&layout).Error
		}
		if err != nil {
			s.cache.Store(q.cacheKey, nil)
			continue
		}
		s.cache.Store(q.cacheKey, &layout)
		return &layout
	}
	return nil
}

// findDefault looks up the default layout, trying specific language_id first, then NULL (all languages).
func (s *LayoutService) findDefault(languageID *int) *models.Layout {
	type langQuery struct {
		id       *int
		cacheKey string
	}

	queries := []langQuery{}
	if languageID != nil {
		queries = append(queries, langQuery{id: languageID, cacheKey: fmt.Sprintf("default:lang:%d", *languageID)})
	}
	queries = append(queries, langQuery{id: nil, cacheKey: "default:lang:null"})

	for _, q := range queries {
		if cached, ok := s.cache.Load(q.cacheKey); ok {
			if cached == nil {
				continue
			}
			return cached.(*models.Layout)
		}

		var layout models.Layout
		var err error
		if q.id != nil {
			err = s.db.Where("is_default = ? AND language_id = ?", true, *q.id).First(&layout).Error
		} else {
			err = s.db.Where("is_default = ? AND language_id IS NULL", true).First(&layout).Error
		}
		if err != nil {
			s.cache.Store(q.cacheKey, nil)
			continue
		}
		s.cache.Store(q.cacheKey, &layout)
		return &layout
	}
	return nil
}

// ResolveDefault resolves the default layout for the given language.
func (s *LayoutService) ResolveDefault(languageID *int) (*models.Layout, error) {
	if layout := s.findDefault(languageID); layout != nil {
		return layout, nil
	}
	return nil, fmt.Errorf("no default layout found")
}

// InvalidateCache resets the entire layout cache.
func (s *LayoutService) InvalidateCache() {
	s.cache.Range(func(key, value interface{}) bool {
		s.cache.Delete(key)
		return true
	})
}
