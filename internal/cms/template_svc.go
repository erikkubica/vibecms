package cms

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gorm.io/gorm"

	"vibecms/internal/models"
)

// TemplateService provides business logic for managing templates.
type TemplateService struct {
	db          *gorm.DB
	themeAssets *ThemeAssetRegistry
}

// NewTemplateService creates a new TemplateService with the given database connection.
func NewTemplateService(db *gorm.DB, themeAssets *ThemeAssetRegistry) *TemplateService {
	return &TemplateService{db: db, themeAssets: themeAssets}
}

// List retrieves templates ordered by label with pagination.
func (s *TemplateService) List(page, perPage int) ([]models.Template, int64, error) {
	var total int64
	if err := s.db.Model(&models.Template{}).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("counting templates: %w", err)
	}
	var templates []models.Template
	offset := (page - 1) * perPage
	if err := s.db.Order("label ASC").Offset(offset).Limit(perPage).Find(&templates).Error; err != nil {
		return nil, 0, fmt.Errorf("listing templates: %w", err)
	}
	return templates, total, nil
}

// GetByID retrieves a single template by its ID.
func (s *TemplateService) GetByID(id int) (*models.Template, error) {
	var t models.Template
	if err := s.db.First(&t, id).Error; err != nil {
		return nil, err
	}
	return &t, nil
}

// GetBySlug retrieves a single template by its slug.
func (s *TemplateService) GetBySlug(slug string) (*models.Template, error) {
	var t models.Template
	if err := s.db.Where("slug = ?", slug).First(&t).Error; err != nil {
		return nil, err
	}
	return &t, nil
}

// Create inserts a new template after validating slug uniqueness.
func (s *TemplateService) Create(t *models.Template) error {
	if t.Slug == "" {
		return fmt.Errorf("validation error: slug is required")
	}
	if t.Label == "" {
		return fmt.Errorf("validation error: label is required")
	}

	// Check slug uniqueness
	var count int64
	s.db.Model(&models.Template{}).Where("slug = ?", t.Slug).Count(&count)
	if count > 0 {
		return fmt.Errorf("slug conflict: template with slug %q already exists", t.Slug)
	}

	if err := s.db.Create(t).Error; err != nil {
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "23505") {
			return fmt.Errorf("slug conflict: template with slug %q already exists", t.Slug)
		}
		return fmt.Errorf("creating template: %w", err)
	}

	return nil
}

// Update performs a partial update on a template by ID.
func (s *TemplateService) Update(id int, updates map[string]interface{}) (*models.Template, error) {
	existing, err := s.GetByID(id)
	if err != nil {
		return nil, err
	}

	// Validate slug uniqueness if slug is being changed
	if newSlug, ok := updates["slug"].(string); ok && newSlug != "" && newSlug != existing.Slug {
		var count int64
		s.db.Model(&models.Template{}).Where("slug = ? AND id != ?", newSlug, id).Count(&count)
		if count > 0 {
			return nil, fmt.Errorf("slug conflict: template with slug %q already exists", newSlug)
		}
	}

	// Convert JSONB fields from parsed JSON (map/slice) to models.JSONB
	for _, key := range []string{"block_config"} {
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
			return nil, fmt.Errorf("slug conflict: template with slug %q already exists", slug)
		}
		return nil, fmt.Errorf("updating template: %w", err)
	}

	// Re-fetch updated template
	updated, err := s.GetByID(id)
	if err != nil {
		return nil, err
	}
	return updated, nil
}

// Delete removes a template by ID.
func (s *TemplateService) Delete(id int) error {
	_, err := s.GetByID(id)
	if err != nil {
		return err
	}

	result := s.db.Delete(&models.Template{}, id)
	if result.Error != nil {
		return fmt.Errorf("deleting template: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// Detach converts a theme/extension-sourced template to custom.
func (s *TemplateService) Detach(id int) (*models.Template, error) {
	var existing models.Template
	if err := s.db.First(&existing, id).Error; err != nil {
		return nil, err
	}
	if err := s.db.Model(&existing).Updates(map[string]interface{}{
		"source":     "custom",
		"theme_name": nil,
	}).Error; err != nil {
		return nil, fmt.Errorf("failed to detach template: %w", err)
	}
	s.db.First(&existing, id)
	return &existing, nil
}

// Reattach restores a detached template to its theme version.
func (s *TemplateService) Reattach(id int) (*models.Template, error) {
	var existing models.Template
	if err := s.db.First(&existing, id).Error; err != nil {
		return nil, err
	}
	if existing.Source == "theme" || existing.Source == "extension" {
		return &existing, nil // already attached
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

	// Read theme.json to find the matching template definition.
	manifestData, err := os.ReadFile(filepath.Join(themeDir, "theme.json"))
	if err != nil {
		return nil, fmt.Errorf("failed to read theme.json: %w", err)
	}
	var manifest ThemeManifest
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse theme.json: %w", err)
	}

	themeName := manifest.Name
	for _, def := range manifest.Templates {
		if def.Slug == existing.Slug {
			filePath := filepath.Join(themeDir, "templates", def.File)
			if err := RegisterTemplateFromFile(s.db, filePath, def.Slug, "theme", themeName); err != nil {
				return nil, fmt.Errorf("failed to reattach template: %w", err)
			}
			s.db.First(&existing, id)
			return &existing, nil
		}
	}

	return nil, fmt.Errorf("template %q not found in current theme", existing.Slug)
}
