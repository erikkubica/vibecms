package coreapi

import (
	"context"
	"fmt"
	"strings"

	"squilla/internal/events"
	"squilla/internal/models"
	"squilla/internal/secrets"
)

// fallbackLocale is the sentinel stored in site_settings.language_code when a
// value applies across every language. Empty string is preferred over NULL so
// it can sit cleanly in a composite primary key.
const fallbackLocale = ""

// readSettingForLocale fetches the raw stored value for (key, locale) with
// fallback to (key, ""). Returns ("", nil) for absent rows so "setting not
// configured" stays an expected path instead of an error.
func (c *coreImpl) readSettingForLocale(ctx context.Context, key, locale string) (string, error) {
	if locale != fallbackLocale {
		v, ok, err := c.readSettingExact(ctx, key, locale)
		if err != nil {
			return "", err
		}
		if ok {
			return v, nil
		}
	}
	v, _, err := c.readSettingExact(ctx, key, fallbackLocale)
	return v, err
}

// readSettingExact looks up the row at exactly (key, locale) without falling
// back. Returns ("", false, nil) when missing.
func (c *coreImpl) readSettingExact(ctx context.Context, key, locale string) (string, bool, error) {
	var rows []models.SiteSetting
	err := c.db.WithContext(ctx).
		Where("\"key\" = ? AND language_code = ?", key, locale).
		Limit(1).Find(&rows).Error
	if err != nil {
		return "", false, fmt.Errorf("coreapi readSetting: %w", err)
	}
	if len(rows) == 0 {
		return "", false, nil
	}
	if rows[0].Value == nil {
		return "", true, nil
	}
	return *rows[0].Value, true, nil
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

// GetSetting returns the value for a site setting key (fallback locale only).
// Locale-aware callers should use GetSettingLoc.
func (c *coreImpl) GetSetting(ctx context.Context, key string) (string, error) {
	return c.GetSettingLoc(ctx, key, fallbackLocale)
}

// GetSettingLoc returns the value for (key, locale), falling back to
// (key, "") when no per-locale row exists. Pass "" to read the fallback row
// directly without a per-locale lookup.
func (c *coreImpl) GetSettingLoc(ctx context.Context, key, locale string) (string, error) {
	raw, err := c.readSettingForLocale(ctx, key, locale)
	if err != nil {
		return "", err
	}
	return c.decryptIfSecret(key, raw)
}

// SetSetting upserts a site setting at the fallback locale (applies to all
// languages). Locale-aware callers should use SetSettingLoc.
func (c *coreImpl) SetSetting(ctx context.Context, key, value string) error {
	return c.SetSettingLoc(ctx, key, fallbackLocale, value)
}

// SetSettingLoc upserts a site setting for the given (key, locale). Pass
// "" for locale to write the fallback row.
func (c *coreImpl) SetSettingLoc(ctx context.Context, key, locale, value string) error {
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
	if err := db.Where("\"key\" = ? AND language_code = ?", key, locale).
		Limit(1).Find(&rows).Error; err != nil {
		return fmt.Errorf("coreapi SetSetting lookup: %w", err)
	}

	if len(rows) == 0 {
		s := models.SiteSetting{Key: key, LanguageCode: locale, Value: &stored}
		if err := db.Create(&s).Error; err != nil {
			return fmt.Errorf("coreapi SetSetting create: %w", err)
		}
		c.publishSettingUpdated(key, locale)
		return nil
	}

	s := rows[0]
	s.Value = &stored
	if err := db.Save(&s).Error; err != nil {
		return fmt.Errorf("coreapi SetSetting update: %w", err)
	}
	c.publishSettingUpdated(key, locale)
	return nil
}

// publishSettingUpdated fires `setting.updated` so subscribers (notably the
// public handler's site-settings cache) can invalidate. Without this,
// scripts that mutate settings — including kernel pointers like
// `homepage_node_id` — leave the in-process cache stale until the next
// theme.activate or process restart.
func (c *coreImpl) publishSettingUpdated(key, locale string) {
	if c.eventBus == nil {
		return
	}
	c.eventBus.Publish("setting.updated", events.Payload{
		"key":           key,
		"language_code": locale,
	})
}

// GetSettings returns settings matching an optional prefix at the fallback
// locale only. Locale-aware callers should use GetSettingsLoc.
func (c *coreImpl) GetSettings(ctx context.Context, prefix string) (map[string]string, error) {
	return c.GetSettingsLoc(ctx, prefix, fallbackLocale)
}

// GetSettingsLoc returns settings matching an optional prefix, with per-key
// fallback semantics: each key returns its (key, locale) value if present,
// otherwise (key, "") if present, otherwise nothing. The result map is keyed
// by trimmed key, same shape as GetSettings.
func (c *coreImpl) GetSettingsLoc(ctx context.Context, prefix, locale string) (map[string]string, error) {
	var settings []models.SiteSetting

	query := c.db.WithContext(ctx).Model(&models.SiteSetting{})
	if prefix != "" {
		query = query.Where("\"key\" LIKE ?", prefix+"%")
	}
	// Fetch both the requested locale and the fallback so we can resolve
	// per-key in one pass without N round-trips.
	if locale != fallbackLocale {
		query = query.Where("language_code IN ?", []string{locale, fallbackLocale})
	} else {
		query = query.Where("language_code = ?", fallbackLocale)
	}

	if err := query.Find(&settings).Error; err != nil {
		return nil, fmt.Errorf("coreapi GetSettings: %w", err)
	}

	// Group by key: prefer the per-locale value when both rows exist.
	type row struct {
		val    string
		locale string
	}
	chosen := make(map[string]row, len(settings))
	for _, s := range settings {
		v := ""
		if s.Value != nil {
			v = *s.Value
		}
		// Always prefer the requested locale; only the fallback gets
		// recorded when nothing for the locale has been seen.
		existing, ok := chosen[s.Key]
		if !ok || (existing.locale == fallbackLocale && s.LanguageCode != fallbackLocale) {
			chosen[s.Key] = row{val: v, locale: s.LanguageCode}
		}
	}

	result := make(map[string]string, len(chosen))
	for fullKey, r := range chosen {
		k := fullKey
		if prefix != "" {
			k = strings.TrimPrefix(k, prefix)
		}
		decrypted, err := c.decryptIfSecret(fullKey, r.val)
		if err != nil {
			return nil, fmt.Errorf("coreapi GetSettings decrypt %q: %w", fullKey, err)
		}
		result[k] = decrypted
	}
	return result, nil
}
