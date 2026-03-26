package email

import (
	"fmt"

	"gorm.io/gorm"

	"vibecms/internal/models"
)

// TemplateService provides CRUD operations for email templates.
type TemplateService struct {
	db *gorm.DB
}

// NewTemplateService creates a new TemplateService with the given database connection.
func NewTemplateService(db *gorm.DB) *TemplateService {
	return &TemplateService{db: db}
}

// List retrieves all email templates ordered by name.
func (s *TemplateService) List() ([]models.EmailTemplate, error) {
	var templates []models.EmailTemplate
	if err := s.db.Order("name ASC").Find(&templates).Error; err != nil {
		return nil, fmt.Errorf("failed to list email templates: %w", err)
	}
	return templates, nil
}

// GetByID retrieves a single email template by its ID.
func (s *TemplateService) GetByID(id int) (*models.EmailTemplate, error) {
	var tmpl models.EmailTemplate
	if err := s.db.First(&tmpl, id).Error; err != nil {
		return nil, err
	}
	return &tmpl, nil
}

// GetBySlug retrieves a single email template by its slug.
func (s *TemplateService) GetBySlug(slug string) (*models.EmailTemplate, error) {
	var tmpl models.EmailTemplate
	if err := s.db.Where("slug = ?", slug).First(&tmpl).Error; err != nil {
		return nil, err
	}
	return &tmpl, nil
}

// Create inserts a new email template.
func (s *TemplateService) Create(t *models.EmailTemplate) error {
	if err := s.db.Create(t).Error; err != nil {
		return fmt.Errorf("failed to create email template: %w", err)
	}
	return nil
}

// Update performs a partial update on an email template by ID.
func (s *TemplateService) Update(id int, updates map[string]interface{}) (*models.EmailTemplate, error) {
	existing, err := s.GetByID(id)
	if err != nil {
		return nil, err
	}

	if err := s.db.Model(existing).Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("failed to update email template: %w", err)
	}

	updated, err := s.GetByID(id)
	if err != nil {
		return nil, err
	}
	return updated, nil
}

// Delete removes an email template by ID.
func (s *TemplateService) Delete(id int) error {
	result := s.db.Delete(&models.EmailTemplate{}, id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete email template: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
