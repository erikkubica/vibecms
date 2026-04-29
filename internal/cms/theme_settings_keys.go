package cms

import (
	"context"
	"strings"

	"squilla/internal/models"

	"gorm.io/gorm"
)

// themeKeyPrefix is the namespace under which all theme settings live in
// the site_settings table. Keep in sync with ParseSettingKey.
const themeKeyPrefix = "theme:"

// SettingKey returns the canonical site_settings key for a given theme
// slug, settings page slug, and field key — e.g. "theme:hello-vietnam:header:logo".
func SettingKey(themeSlug, pageSlug, fieldKey string) string {
	return themeKeyPrefix + themeSlug + ":" + pageSlug + ":" + fieldKey
}

// ThemePrefix returns the site_settings prefix that scopes all settings
// rows for the given theme — used as a LIKE prefix or a GetSettings(prefix)
// argument.
func ThemePrefix(themeSlug string) string {
	return themeKeyPrefix + themeSlug + ":"
}

// ParseSettingKey reverses SettingKey. ok=false for keys that are not
// theme-scoped or are malformed (fewer than 3 colon-separated segments
// after the "theme:" prefix).
func ParseSettingKey(k string) (themeSlug, pageSlug, fieldKey string, ok bool) {
	if !strings.HasPrefix(k, themeKeyPrefix) {
		return "", "", "", false
	}
	rest := strings.TrimPrefix(k, themeKeyPrefix)
	parts := strings.SplitN(rest, ":", 3)
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		return "", "", "", false
	}
	return parts[0], parts[1], parts[2], true
}

// DeleteThemeSettings removes every site_settings row whose key is scoped
// to the given theme slug. Called on full theme deletion (NOT deactivation,
// where values are preserved for later reactivation). Idempotent — deleting
// a theme that never had settings simply removes zero rows.
func DeleteThemeSettings(ctx context.Context, db *gorm.DB, themeSlug string) error {
	if themeSlug == "" {
		return nil
	}
	return db.WithContext(ctx).
		Where("\"key\" LIKE ?", ThemePrefix(themeSlug)+"%").
		Delete(&models.SiteSetting{}).Error
}
