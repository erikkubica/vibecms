package cms

import (
	"encoding/json"
	"regexp"

	"gorm.io/gorm"

	"squilla/internal/models"
)

// themeAssetRefRegexp matches "theme-asset:<key>" — references the active
// theme's asset manifest.
var themeAssetRefRegexp = regexp.MustCompile(`^theme-asset:([a-z0-9_-]+)$`)

// extensionAssetRefRegexp matches "extension-asset:<slug>:<key>" — references
// an extension's asset manifest. The slug is part of the key because multiple
// extensions may be active simultaneously (unlike themes, which are a
// singleton).
var extensionAssetRefRegexp = regexp.MustCompile(`^extension-asset:([a-z0-9_-]+):([a-z0-9_-]+)$`)

// mediaAssetRow is a lightweight view into media_files used by the resolver.
// Shared between theme-owned and extension-owned rows — the source/owner
// distinction lives in the lookup map keys, not the row struct.
type mediaAssetRow struct {
	ID        uint   `gorm:"column:id"`
	URL       string `gorm:"column:url"`
	AssetKey  string `gorm:"column:asset_key"`
	Alt       string `gorm:"column:alt"`
	MimeType  string `gorm:"column:mime_type"`
	Width     int    `gorm:"column:width"`
	Height    int    `gorm:"column:height"`
	// Owner fields — only one is populated per row, based on source.
	ThemeName     string `gorm:"column:theme_name"`
	ExtensionSlug string `gorm:"column:extension_slug"`
}

// asMediaValue renders the row as the JSON shape expected by the frontend's
// normalizeToMediaValue (matches what MediaPickerModal returns).
func (r mediaAssetRow) asMediaValue() map[string]interface{} {
	return map[string]interface{}{
		"id":        r.ID,
		"url":       r.URL,
		"alt":       r.Alt,
		"mime_type": r.MimeType,
		"width":     r.Width,
		"height":    r.Height,
	}
}

// AssetLookup is the resolver's index: theme assets keyed by asset_key,
// extension assets keyed by "<slug>:<asset_key>".
type AssetLookup struct {
	Theme     map[string]mediaAssetRow
	Extension map[string]mediaAssetRow
}

// Empty reports whether the lookup has no usable entries.
func (l AssetLookup) Empty() bool {
	return len(l.Theme) == 0 && len(l.Extension) == 0
}

// LoadAssetLookup fetches every theme- and extension-owned media row and
// returns them indexed for the resolver. Degrades gracefully when the
// media-manager extension hasn't installed its ownership migrations —
// missing columns return an empty map, not an error.
func LoadAssetLookup(db *gorm.DB, themeName string) AssetLookup {
	out := AssetLookup{
		Theme:     map[string]mediaAssetRow{},
		Extension: map[string]mediaAssetRow{},
	}
	if db == nil {
		return out
	}

	// Theme-owned rows for the currently-active theme.
	if themeName != "" {
		var rows []mediaAssetRow
		err := db.Table("media_files").
			Select("id, url, asset_key, alt, mime_type, width, height, theme_name, extension_slug").
			Where("source = ? AND theme_name = ? AND asset_key IS NOT NULL AND asset_key <> ''", "theme", themeName).
			Scan(&rows).Error
		if err == nil {
			for _, r := range rows {
				out.Theme[r.AssetKey] = r
			}
		}
	}

	// Extension-owned rows for ALL active extensions (multiple can be
	// active, so we don't filter by a single slug). We just load them all
	// and let the resolver pick by the "<slug>:<key>" combination.
	var extRows []mediaAssetRow
	err := db.Table("media_files").
		Select("id, url, asset_key, alt, mime_type, width, height, theme_name, extension_slug").
		Where("source = ? AND extension_slug IS NOT NULL AND extension_slug <> '' AND asset_key IS NOT NULL AND asset_key <> ''", "extension").
		Scan(&extRows).Error
	if err == nil {
		for _, r := range extRows {
			out.Extension[r.ExtensionSlug+":"+r.AssetKey] = r
		}
	}

	return out
}

// LoadThemeAssetMap is a back-compat shim returning just the theme map.
// Prefer LoadAssetLookup.
func LoadThemeAssetMap(db *gorm.DB, themeName string) map[string]mediaAssetRow {
	return LoadAssetLookup(db, themeName).Theme
}

// hasAny reports whether the lookup has any resolvable entries. Used to
// short-circuit work when no theme/extension is active.
func (l AssetLookup) hasAny() bool {
	return len(l.Theme) > 0 || len(l.Extension) > 0
}

// loadActiveAssetLookup returns the AssetLookup for the currently active
// theme — convenience wrapper used by live render paths that don't already
// carry a theme name.
func loadActiveAssetLookup(db *gorm.DB) AssetLookup {
	var t models.Theme
	if err := db.Where("is_active = ?", true).First(&t).Error; err != nil {
		return AssetLookup{}
	}
	return LoadAssetLookup(db, t.Name)
}

// ResolveThemeAssetRefs walks a JSON-decoded value and replaces any
// "theme-asset:<key>" or "extension-asset:<slug>:<key>" string with the
// matching media object. Unresolved refs (and strings that don't match
// either pattern) pass through unchanged.
//
// When the match is at an object's "url" key (e.g. {"url": "theme-asset:foo"}),
// the enclosing object is replaced by the resolved media value, with any
// other sibling keys (like alt) merged on top.
func ResolveThemeAssetRefs(value any, lookup AssetLookup) any {
	switch v := value.(type) {
	case string:
		if row, ok := resolveRefString(v, lookup); ok {
			return row.asMediaValue()
		}
		return v
	case map[string]interface{}:
		if rawURL, ok := v["url"].(string); ok {
			if row, hit := resolveRefString(rawURL, lookup); hit {
				resolved := row.asMediaValue()
				for k, val := range v {
					if k == "url" {
						continue
					}
					resolved[k] = val
				}
				return resolved
			}
		}
		for k, val := range v {
			v[k] = ResolveThemeAssetRefs(val, lookup)
		}
		return v
	case []interface{}:
		for i, item := range v {
			v[i] = ResolveThemeAssetRefs(item, lookup)
		}
		return v
	case json.RawMessage:
		var decoded any
		if err := json.Unmarshal(v, &decoded); err != nil {
			return v
		}
		resolved := ResolveThemeAssetRefs(decoded, lookup)
		if out, err := json.Marshal(resolved); err == nil {
			return json.RawMessage(out)
		}
		return v
	}
	return value
}

// LoadActiveAssetLookupExported is the exported form of loadActiveAssetLookup
// for callers outside the cms package (the rendering layer needs to resolve
// theme-asset: URIs at template-render time).
func LoadActiveAssetLookupExported(db *gorm.DB) AssetLookup {
	return loadActiveAssetLookup(db)
}

// ResolveAssetURI maps a "theme-asset:<key>" / "extension-asset:<slug>:<key>"
// URI to its real public URL using the given lookup. Returns "", false when
// the URI is malformed or the asset isn't registered.
func ResolveAssetURI(uri string, lookup AssetLookup) (string, bool) {
	row, ok := resolveRefString(uri, lookup)
	if !ok {
		return "", false
	}
	return row.URL, true
}

// resolveRefString tries both the theme and extension patterns against a
// string value and returns the matched row if found.
func resolveRefString(s string, lookup AssetLookup) (mediaAssetRow, bool) {
	if m := themeAssetRefRegexp.FindStringSubmatch(s); len(m) == 2 {
		if row, ok := lookup.Theme[m[1]]; ok {
			return row, true
		}
	}
	if m := extensionAssetRefRegexp.FindStringSubmatch(s); len(m) == 3 {
		if row, ok := lookup.Extension[m[1]+":"+m[2]]; ok {
			return row, true
		}
	}
	return mediaAssetRow{}, false
}

// ResolveThemeAssetRefsInJSON decodes raw JSON bytes, resolves any
// theme-asset: / extension-asset: references, and re-encodes.
func ResolveThemeAssetRefsInJSON(raw []byte, lookup AssetLookup) []byte {
	if len(raw) == 0 {
		return raw
	}
	var decoded any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return raw
	}
	resolved := ResolveThemeAssetRefs(decoded, lookup)
	out, err := json.Marshal(resolved)
	if err != nil {
		return raw
	}
	return out
}

// ActiveThemeName reads the name of the currently-active theme, or "" if
// none is marked active. Cheap single-row query; callers cache per-request.
func ActiveThemeName(db *gorm.DB) string {
	if db == nil {
		return ""
	}
	var name string
	_ = db.Table("themes").Select("name").Where("is_active = ?", true).Limit(1).Scan(&name).Error
	return name
}
