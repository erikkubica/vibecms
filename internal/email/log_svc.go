package email

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	"gorm.io/gorm"

	"vibecms/internal/models"
)

// defaultLogRetentionDays is the fallback retention if the setting is unset
// or unparseable. 30 days balances debugging needs against unbounded growth.
const defaultLogRetentionDays = 30

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

// loadSiteSettings reads all site_settings into a map. Used by the
// dispatcher to assemble per-event provider settings; resend/retry
// logic lives in the email-manager extension and goes through
// coreapi.SendEmail rather than constructing providers here — see
// CLAUDE.md ("feature code in core" rule).
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

// CleanOldLogs deletes email logs older than the configured retention.
// Reads `email_log_retention_days` from site_settings; falls back to
// defaultLogRetentionDays when unset/invalid.
func (s *LogService) CleanOldLogs() error {
	days := defaultLogRetentionDays
	var setting models.SiteSetting
	if err := s.db.Where("key = ?", "email_log_retention_days").First(&setting).Error; err == nil && setting.Value != nil {
		if n, err := strconv.Atoi(*setting.Value); err == nil && n > 0 {
			days = n
		}
	}
	cutoff := time.Now().AddDate(0, 0, -days)
	return s.db.Where("created_at < ?", cutoff).Delete(&models.EmailLog{}).Error
}

// StartCleanupLoop runs CleanOldLogs daily until ctx is cancelled.
// Mirrors SessionService.StartCleanupLoop and PasswordResetService.StartCleanupLoop.
func (s *LogService) StartCleanupLoop(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = 24 * time.Hour
	}
	go func() {
		if err := s.CleanOldLogs(); err != nil {
			log.Printf("email log cleanup: initial sweep failed: %v", err)
		}
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := s.CleanOldLogs(); err != nil {
					log.Printf("email log cleanup: %v", err)
				}
			}
		}
	}()
}
