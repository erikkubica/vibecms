package coreapi

import (
	"context"
	"fmt"
	"strings"

	"squilla/internal/events"
	"squilla/internal/models"
	"squilla/internal/secrets"
)

// readSetting fetches the raw stored value for key. Returns ("", nil)
// for an absent or NULL row so "setting not configured" stays an
// expected path instead of an error.
func (c *coreImpl) readSetting(ctx context.Context, key string) (string, error) {
	var rows []models.SiteSetting
	if err := c.db.WithContext(ctx).Where("\"key\" = ?", key).Limit(1).Find(&rows).Error; err != nil {
		return "", fmt.Errorf("coreapi readSetting: %w", err)
	}
	if len(rows) == 0 || rows[0].Value == nil {
		return "", nil
	}
	return *rows[0].Value, nil
}

// decryptIfSecret transparently decrypts when the key is secret-shaped
// and the stored value is wrapped in the encryption envelope. Plaintext
// values pass through, so legacy rows from before encryption was
// configured still work.
func (c *coreImpl) decryptIfSecret(key, raw string) (string, error) {
	if raw == "" || c.secrets == nil || !secrets.IsSecretKey(key) {
		return raw, nil
	}
	return c.secrets.Decrypt(raw)
}

// GetSetting returns the value for a site setting key.
// Returns an empty string (not an error) if the key is missing.
// Secret-shaped keys whose stored value is encrypted are decrypted
// transparently. Internal callers (extensions, scripts via the
// capability guard) get the plaintext; the wire-side admin
// SettingsHandler.List redacts separately.
func (c *coreImpl) GetSetting(ctx context.Context, key string) (string, error) {
	raw, err := c.readSetting(ctx, key)
	if err != nil {
		return "", err
	}
	return c.decryptIfSecret(key, raw)
}

// SetSetting upserts a site setting (insert or update). Secret-shaped
// keys are encrypted at rest; non-secret keys store plaintext as
// before.
func (c *coreImpl) SetSetting(ctx context.Context, key, value string) error {
	stored := value
	if c.secrets != nil && secrets.IsSecretKey(key) {
		enc, err := c.secrets.MaybeEncrypt(value)
		if err != nil {
			return fmt.Errorf("coreapi SetSetting encrypt: %w", err)
		}
		stored = enc
	}

	db := c.db.WithContext(ctx)
	var rows []models.SiteSetting
	if err := db.Where("\"key\" = ?", key).Limit(1).Find(&rows).Error; err != nil {
		return fmt.Errorf("coreapi SetSetting lookup: %w", err)
	}

	if len(rows) == 0 {
		s := models.SiteSetting{Key: key, Value: &stored}
		if err := db.Create(&s).Error; err != nil {
			return fmt.Errorf("coreapi SetSetting create: %w", err)
		}
		c.publishSettingUpdated(key)
		return nil
	}

	s := rows[0]
	s.Value = &stored
	if err := db.Save(&s).Error; err != nil {
		return fmt.Errorf("coreapi SetSetting update: %w", err)
	}
	c.publishSettingUpdated(key)
	return nil
}

// publishSettingUpdated fires `setting.updated` so subscribers (notably the
// public handler's site-settings cache) can invalidate. Without this,
// scripts that mutate settings — including kernel pointers like
// `homepage_node_id` — leave the in-process cache stale until the next
// theme.activate or process restart.
func (c *coreImpl) publishSettingUpdated(key string) {
	if c.eventBus == nil {
		return
	}
	c.eventBus.Publish("setting.updated", events.Payload{"key": key})
}

// GetSettings returns settings matching an optional prefix.
// If prefix is non-empty, only keys starting with that prefix are returned,
// and the prefix is trimmed from the keys in the result map. Secret-shaped
// keys are decrypted transparently — callers see plaintext.
func (c *coreImpl) GetSettings(ctx context.Context, prefix string) (map[string]string, error) {
	var settings []models.SiteSetting

	query := c.db.WithContext(ctx).Model(&models.SiteSetting{})
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
		// Decrypt under the original (full) key — that's what the
		// secret heuristic matches against; the trimmed key is only
		// for the result map shape.
		decrypted, err := c.decryptIfSecret(s.Key, v)
		if err != nil {
			return nil, fmt.Errorf("coreapi GetSettings decrypt %q: %w", s.Key, err)
		}
		result[k] = decrypted
	}
	return result, nil
}
