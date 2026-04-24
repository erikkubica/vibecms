package coreapi

import (
	"context"
	"fmt"
	"strings"

	"vibecms/internal/models"
)

// GetSetting returns the value for a site setting key.
// Returns an empty string (not an error) if the key is missing.
// Uses Limit(1).Find instead of First so absent keys don't log
// ErrRecordNotFound — "setting not configured" is an expected path,
// not an error condition.
func (c *coreImpl) GetSetting(_ context.Context, key string) (string, error) {
	var rows []models.SiteSetting
	if err := c.db.Where("\"key\" = ?", key).Limit(1).Find(&rows).Error; err != nil {
		return "", fmt.Errorf("coreapi GetSetting: %w", err)
	}
	if len(rows) == 0 || rows[0].Value == nil {
		return "", nil
	}
	return *rows[0].Value, nil
}

// SetSetting upserts a site setting (insert or update).
func (c *coreImpl) SetSetting(_ context.Context, key, value string) error {
	var rows []models.SiteSetting
	if err := c.db.Where("\"key\" = ?", key).Limit(1).Find(&rows).Error; err != nil {
		return fmt.Errorf("coreapi SetSetting lookup: %w", err)
	}

	if len(rows) == 0 {
		s := models.SiteSetting{Key: key, Value: &value}
		if err := c.db.Create(&s).Error; err != nil {
			return fmt.Errorf("coreapi SetSetting create: %w", err)
		}
		return nil
	}

	s := rows[0]
	s.Value = &value
	if err := c.db.Save(&s).Error; err != nil {
		return fmt.Errorf("coreapi SetSetting update: %w", err)
	}
	return nil
}

// GetSettings returns settings matching an optional prefix.
// If prefix is non-empty, only keys starting with that prefix are returned,
// and the prefix is trimmed from the keys in the result map.
func (c *coreImpl) GetSettings(_ context.Context, prefix string) (map[string]string, error) {
	var settings []models.SiteSetting

	query := c.db.Model(&models.SiteSetting{})
	if prefix != "" {
		query = query.Where("\"key\" LIKE ?", prefix+"%")
	}

	if err := query.Find(&settings).Error; err != nil {
		return nil, fmt.Errorf("coreapi GetSettings: %w", err)
	}

	result := make(map[string]string, len(settings))
	for _, s := range settings {
		k := s.Key
		if prefix != "" {
			k = strings.TrimPrefix(k, prefix)
		}
		v := ""
		if s.Value != nil {
			v = *s.Value
		}
		result[k] = v
	}
	return result, nil
}
