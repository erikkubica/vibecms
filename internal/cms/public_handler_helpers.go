package cms

import (
	"encoding/json"
	"log"

	"squilla/internal/models"
)

// This file collects the small helpers that don't belong to the
// page-render flow but are needed by it: site-setting and language
// caches, blocks_data parsing, theme-asset reference resolution, and
// block-slug extraction.

// loadSiteSettings returns site settings for the site's default language.
// Equivalent to loadSiteSettingsForLocale("") — kept for callers that don't
// have request-language context (admin tools, retention loops, etc.).
func (h *PublicHandler) loadSiteSettings() map[string]string {
	return h.loadSiteSettingsForLocale("")
}

// loadSiteSettingsForLocale loads site settings scoped to a request locale,
// falling back to the default-language row when the locale doesn't override
// a key. Cached per-locale; invalidated by setting.updated and the
// .ClearCache() entry point.
//
// settings rows carry a language_code (migration 0038); without locale
// scoping, GORM's Find returns every row and last-write-wins on duplicate
// keys non-deterministically. That broke seo_robots_index — the public
// renderer would sometimes see "true" even after the operator saved
// "false" on the default-language admin.
func (h *PublicHandler) loadSiteSettingsForLocale(locale string) map[string]string {
	defLang := h.defaultLanguageCode()
	if locale == "" {
		locale = defLang
	}

	h.cacheMu.RLock()
	if h.siteSettingsByLocale != nil {
		if cached, ok := h.siteSettingsByLocale[locale]; ok {
			h.cacheMu.RUnlock()
			return cached
		}
	}
	h.cacheMu.RUnlock()

	h.cacheMu.Lock()
	defer h.cacheMu.Unlock()
	if h.siteSettingsByLocale == nil {
		h.siteSettingsByLocale = map[string]map[string]string{}
	}
	if cached, ok := h.siteSettingsByLocale[locale]; ok {
		return cached
	}

	settings := make(map[string]string)
	var rows []models.SiteSetting
	q := h.db
	switch {
	case defLang == "":
		// No default language seeded yet — fall back to a single
		// last-write-wins read so a fresh install can still surface
		// boot-time defaults.
		_ = q.Find(&rows).Error
	case locale == defLang:
		_ = q.Where("language_code = ?", defLang).Find(&rows).Error
	default:
		// Pull both the requested locale AND the default-language row,
		// then prefer the requested locale per-key with default-language
		// fallback. One query, two values per key max.
		_ = q.Where("language_code IN ?", []string{locale, defLang}).Find(&rows).Error
	}

	type pick struct {
		val    string
		locale string
	}
	chosen := map[string]pick{}
	for _, s := range rows {
		if s.Value == nil {
			continue
		}
		existing, ok := chosen[s.Key]
		if !ok {
			chosen[s.Key] = pick{val: *s.Value, locale: s.LanguageCode}
			continue
		}
		// Prefer the requested locale's value over the default's.
		if existing.locale == defLang && s.LanguageCode == locale {
			chosen[s.Key] = pick{val: *s.Value, locale: s.LanguageCode}
		}
	}
	for k, p := range chosen {
		settings[k] = p.val
	}

	h.siteSettingsByLocale[locale] = settings
	return settings
}

// defaultLanguageCode returns the code of the language flagged is_default
// on the languages table, or "" before any language has been seeded.
func (h *PublicHandler) defaultLanguageCode() string {
	var code string
	_ = h.db.Table("languages").Select("code").Where("is_default = ?", true).Limit(1).Scan(&code).Error
	return code
}

// loadActiveLanguages loads all active languages as a slice.
func (h *PublicHandler) loadActiveLanguages() []models.Language {
	h.cacheMu.RLock()
	languages := h.activeLanguages
	h.cacheMu.RUnlock()

	if languages != nil {
		return languages
	}

	h.cacheMu.Lock()
	defer h.cacheMu.Unlock()
	if h.activeLanguages != nil {
		return h.activeLanguages
	}

	var langs []models.Language
	h.db.Where("is_active = ?", true).Order("sort_order ASC").Find(&langs)

	h.activeLanguages = langs
	return langs
}

// parseBlocks unmarshals JSONB blocks_data into a slice of maps.
func parseBlocks(data models.JSONB) []map[string]interface{} {
	if len(data) == 0 {
		return nil
	}

	var blocks []map[string]interface{}
	if err := json.Unmarshal([]byte(data), &blocks); err != nil {
		log.Printf("warning: failed to parse blocks_data: %v", err)
		return nil
	}
	return blocks
}

// resolveAssetRefsInBlocks walks each block's "fields" map and substitutes
// any `theme-asset:<key>` / `extension-asset:<slug>:<key>` string references
// with the matching media object, using the active theme's asset map.
// Without this, live renders leak "theme-asset:..." strings into templates
// and Go's safeURL sanitiser replaces them with "#ZgotmplZ".
func (h *PublicHandler) resolveAssetRefsInBlocks(blocks []map[string]interface{}) {
	if len(blocks) == 0 {
		return
	}
	var active models.Theme
	if err := h.db.Where("is_active = ?", true).First(&active).Error; err != nil {
		return
	}
	lookup := LoadAssetLookup(h.db, active.Name)
	for i := range blocks {
		fields, ok := blocks[i]["fields"].(map[string]interface{})
		if !ok {
			continue
		}
		blocks[i]["fields"] = ResolveThemeAssetRefs(fields, lookup)
	}
}

// extractBlockSlugs returns the unique block type slugs used in a parsed blocks list.
func extractBlockSlugs(blocks []map[string]interface{}) []string {
	seen := make(map[string]bool, len(blocks))
	slugs := make([]string, 0, len(blocks))
	for _, b := range blocks {
		if t, ok := b["type"].(string); ok && t != "" && !seen[t] {
			seen[t] = true
			slugs = append(slugs, t)
		}
	}
	return slugs
}
