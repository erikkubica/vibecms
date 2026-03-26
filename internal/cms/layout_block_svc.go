package cms

import (
	"fmt"
	"strings"
	"sync"

	"gorm.io/gorm"

	"vibecms/internal/models"
)

// LayoutBlockService provides business logic for managing layout blocks (partials).
type LayoutBlockService struct {
	db    *gorm.DB
	cache sync.Map
}

// NewLayoutBlockService creates a new LayoutBlockService with the given database connection.
func NewLayoutBlockService(db *gorm.DB) *LayoutBlockService {
	return &LayoutBlockService{db: db}
}

// List retrieves layout blocks with optional filters for language_code and source.
func (s *LayoutBlockService) List(languageCode, source string) ([]models.LayoutBlock, error) {
	var blocks []models.LayoutBlock
	q := s.db.Order("name ASC")
	if languageCode != "" {
		q = q.Where("language_code = ?", languageCode)
	}
	if source != "" {
		q = q.Where("source = ?", source)
	}
	if err := q.Find(&blocks).Error; err != nil {
		return nil, fmt.Errorf("failed to list layout blocks: %w", err)
	}
	return blocks, nil
}

// GetByID retrieves a single layout block by its ID.
func (s *LayoutBlockService) GetByID(id int) (*models.LayoutBlock, error) {
	var block models.LayoutBlock
	if err := s.db.First(&block, id).Error; err != nil {
		return nil, err
	}
	return &block, nil
}

// Resolve finds a layout block by slug, trying the requested language first then falling back to the default language.
// Results are cached for performance.
func (s *LayoutBlockService) Resolve(slug, lang, defaultLang string) (*models.LayoutBlock, error) {
	langs := []string{lang}
	if defaultLang != lang {
		langs = append(langs, defaultLang)
	}

	for _, l := range langs {
		cacheKey := fmt.Sprintf("slug:%s:lang:%s", slug, l)
		if cached, ok := s.cache.Load(cacheKey); ok {
			if cached == nil {
				continue
			}
			return cached.(*models.LayoutBlock), nil
		}

		var block models.LayoutBlock
		err := s.db.Where("slug = ? AND language_code = ?", slug, l).First(&block).Error
		if err != nil {
			s.cache.Store(cacheKey, nil)
			continue
		}
		s.cache.Store(cacheKey, &block)
		return &block, nil
	}

	return nil, fmt.Errorf("no layout block found for slug %q", slug)
}

// Create inserts a new layout block after validating slug+language uniqueness.
func (s *LayoutBlockService) Create(block *models.LayoutBlock) error {
	// Check slug+language_code uniqueness
	var count int64
	s.db.Model(&models.LayoutBlock{}).Where("slug = ? AND language_code = ?", block.Slug, block.LanguageCode).Count(&count)
	if count > 0 {
		return fmt.Errorf("SLUG_CONFLICT")
	}

	if err := s.db.Create(block).Error; err != nil {
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "23505") {
			return fmt.Errorf("SLUG_CONFLICT")
		}
		return fmt.Errorf("failed to create layout block: %w", err)
	}

	s.InvalidateCache()
	return nil
}

// Update performs a partial update on a layout block by ID.
func (s *LayoutBlockService) Update(id int, updates map[string]interface{}) (*models.LayoutBlock, error) {
	existing, err := s.GetByID(id)
	if err != nil {
		return nil, err
	}

	// Block edits to theme-sourced layout blocks
	if existing.Source == "theme" {
		return nil, fmt.Errorf("THEME_READONLY")
	}

	// Validate slug+language uniqueness if slug is being changed
	if newSlug, ok := updates["slug"].(string); ok && newSlug != "" && newSlug != existing.Slug {
		langCode := existing.LanguageCode
		if lc, ok := updates["language_code"].(string); ok && lc != "" {
			langCode = lc
		}
		var count int64
		s.db.Model(&models.LayoutBlock{}).Where("slug = ? AND language_code = ? AND id != ?", newSlug, langCode, id).Count(&count)
		if count > 0 {
			return nil, fmt.Errorf("SLUG_CONFLICT")
		}
	}

	if err := s.db.Model(existing).Updates(updates).Error; err != nil {
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "23505") {
			return nil, fmt.Errorf("SLUG_CONFLICT")
		}
		return nil, fmt.Errorf("failed to update layout block: %w", err)
	}

	s.InvalidateCache()

	// Re-fetch updated layout block
	updated, err := s.GetByID(id)
	if err != nil {
		return nil, err
	}
	return updated, nil
}

// Delete removes a layout block by ID. Theme-sourced layout blocks cannot be deleted.
func (s *LayoutBlockService) Delete(id int) error {
	existing, err := s.GetByID(id)
	if err != nil {
		return err
	}

	if existing.Source == "theme" {
		return fmt.Errorf("THEME_READONLY")
	}

	result := s.db.Delete(&models.LayoutBlock{}, id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete layout block: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	s.InvalidateCache()
	return nil
}

// Detach converts a theme-sourced layout block to a custom layout block.
func (s *LayoutBlockService) Detach(id int) (*models.LayoutBlock, error) {
	existing, err := s.GetByID(id)
	if err != nil {
		return nil, err
	}

	if err := s.db.Model(existing).Updates(map[string]interface{}{
		"source":     "custom",
		"theme_name": nil,
	}).Error; err != nil {
		return nil, fmt.Errorf("failed to detach layout block: %w", err)
	}

	s.InvalidateCache()

	updated, err := s.GetByID(id)
	if err != nil {
		return nil, err
	}
	return updated, nil
}

// InvalidateCache resets the entire layout block cache.
func (s *LayoutBlockService) InvalidateCache() {
	s.cache.Range(func(key, value interface{}) bool {
		s.cache.Delete(key)
		return true
	})
}
