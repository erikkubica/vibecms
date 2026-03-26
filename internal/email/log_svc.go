package email

import (
	"fmt"

	"gorm.io/gorm"

	"vibecms/internal/models"
)

// LogFilters defines filtering options for listing email logs.
type LogFilters struct {
	Status    string
	Action    string
	Recipient string
	DateFrom  string
	DateTo    string
	Page      int
	PerPage   int
}

// LogService provides operations for email logs.
type LogService struct {
	db *gorm.DB
}

// NewLogService creates a new LogService with the given database connection.
func NewLogService(db *gorm.DB) *LogService {
	return &LogService{db: db}
}

// List retrieves paginated email logs with optional filters.
// Returns matching logs, total count, and any error.
func (s *LogService) List(filters LogFilters) ([]models.EmailLog, int64, error) {
	q := s.db.Model(&models.EmailLog{})

	if filters.Status != "" {
		q = q.Where("status = ?", filters.Status)
	}
	if filters.Action != "" {
		q = q.Where("action = ?", filters.Action)
	}
	if filters.Recipient != "" {
		q = q.Where("recipient_email ILIKE ?", "%"+filters.Recipient+"%")
	}
	if filters.DateFrom != "" {
		q = q.Where("created_at >= ?", filters.DateFrom)
	}
	if filters.DateTo != "" {
		q = q.Where("created_at <= ?", filters.DateTo)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count email logs: %w", err)
	}

	page := filters.Page
	if page < 1 {
		page = 1
	}
	perPage := filters.PerPage
	if perPage < 1 {
		perPage = 25
	}

	var logs []models.EmailLog
	offset := (page - 1) * perPage
	if err := q.Order("created_at DESC").Offset(offset).Limit(perPage).Find(&logs).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list email logs: %w", err)
	}

	return logs, total, nil
}

// GetByID retrieves a single email log by its ID.
func (s *LogService) GetByID(id int) (*models.EmailLog, error) {
	var logEntry models.EmailLog
	if err := s.db.First(&logEntry, id).Error; err != nil {
		return nil, err
	}
	return &logEntry, nil
}

// Create inserts a new email log entry.
func (s *LogService) Create(logEntry *models.EmailLog) error {
	if err := s.db.Create(logEntry).Error; err != nil {
		return fmt.Errorf("failed to create email log: %w", err)
	}
	return nil
}

// Resend loads an existing log entry, re-sends its rendered body via the active provider,
// and creates a new log entry recording the result.
func (s *LogService) Resend(id int) error {
	original, err := s.GetByID(id)
	if err != nil {
		return fmt.Errorf("failed to load original email log: %w", err)
	}

	// Load site settings to determine the active provider.
	settings := loadSiteSettings(s.db)
	providerName := settings["email_provider"]

	newLog := &models.EmailLog{
		RuleID:         original.RuleID,
		TemplateSlug:   original.TemplateSlug,
		Action:         original.Action,
		RecipientEmail: original.RecipientEmail,
		Subject:        original.Subject,
		RenderedBody:   original.RenderedBody,
	}

	provider := NewProvider(providerName, settings)
	if provider == nil {
		errMsg := "no email provider configured"
		newLog.Status = "failed"
		newLog.ErrorMessage = &errMsg
		s.Create(newLog)
		return fmt.Errorf(errMsg)
	}

	pName := provider.Name()
	newLog.Provider = &pName

	if err := provider.Send([]string{original.RecipientEmail}, original.Subject, original.RenderedBody); err != nil {
		errMsg := err.Error()
		newLog.Status = "failed"
		newLog.ErrorMessage = &errMsg
		s.Create(newLog)
		return err
	}

	newLog.Status = "sent"
	return s.Create(newLog)
}

// loadSiteSettings reads all site_settings into a map.
func loadSiteSettings(db *gorm.DB) map[string]string {
	var settings []models.SiteSetting
	db.Find(&settings)
	m := make(map[string]string, len(settings))
	for _, s := range settings {
		if s.Value != nil {
			m[s.Key] = *s.Value
		}
	}
	return m
}
