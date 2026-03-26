package email

import (
	"fmt"

	"gorm.io/gorm"

	"vibecms/internal/models"
)

// RuleService provides CRUD operations for email rules.
type RuleService struct {
	db *gorm.DB
}

// NewRuleService creates a new RuleService with the given database connection.
func NewRuleService(db *gorm.DB) *RuleService {
	return &RuleService{db: db}
}

// List retrieves all email rules with their templates preloaded, ordered by action.
func (s *RuleService) List() ([]models.EmailRule, error) {
	var rules []models.EmailRule
	if err := s.db.Preload("Template").Order("action ASC").Find(&rules).Error; err != nil {
		return nil, fmt.Errorf("failed to list email rules: %w", err)
	}
	return rules, nil
}

// GetByID retrieves a single email rule by its ID with template preloaded.
func (s *RuleService) GetByID(id int) (*models.EmailRule, error) {
	var rule models.EmailRule
	if err := s.db.Preload("Template").First(&rule, id).Error; err != nil {
		return nil, err
	}
	return &rule, nil
}

// FindByAction finds enabled rules matching the given action, optionally filtered by node_type.
// Rules with a NULL node_type match all node types.
func (s *RuleService) FindByAction(action string, nodeType string) ([]models.EmailRule, error) {
	var rules []models.EmailRule
	q := s.db.Preload("Template").Where("action = ? AND enabled = ?", action, true)

	if nodeType != "" {
		q = q.Where("node_type IS NULL OR node_type = ?", nodeType)
	}

	if err := q.Find(&rules).Error; err != nil {
		return nil, fmt.Errorf("failed to find email rules for action %q: %w", action, err)
	}
	return rules, nil
}

// Create inserts a new email rule.
func (s *RuleService) Create(r *models.EmailRule) error {
	if err := s.db.Create(r).Error; err != nil {
		return fmt.Errorf("failed to create email rule: %w", err)
	}
	return nil
}

// Update performs a partial update on an email rule by ID.
func (s *RuleService) Update(id int, updates map[string]interface{}) (*models.EmailRule, error) {
	existing, err := s.GetByID(id)
	if err != nil {
		return nil, err
	}

	if err := s.db.Model(existing).Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("failed to update email rule: %w", err)
	}

	updated, err := s.GetByID(id)
	if err != nil {
		return nil, err
	}
	return updated, nil
}

// Delete removes an email rule by ID.
func (s *RuleService) Delete(id int) error {
	result := s.db.Delete(&models.EmailRule{}, id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete email rule: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
