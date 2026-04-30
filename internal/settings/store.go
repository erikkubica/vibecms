package settings

import (
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"squilla/internal/models"
	"squilla/internal/secrets"
)

// Store loads and saves schema-driven values against the existing
// site_settings table. Per-field locale routing is the whole point: a
// translatable field reads/writes the row at the admin's current locale,
// a non-translatable field reads/writes language_code=''. One mixed
// settings page can have both.
type Store struct {
	db      *gorm.DB
	secrets *secrets.Service
}

// NewStore constructs a Store. secretsSvc may be nil — secret-looking
// keys (token suffixes, etc.) are then stored plaintext, matching the
// legacy SettingsHandler behaviour. Pass non-nil to opt into AES-GCM
// at-rest encryption.
func NewStore(db *gorm.DB, secretsSvc *secrets.Service) *Store {
	return &Store{db: db, secrets: secretsSvc}
}

// Load returns every value declared by the schema, merged across the
// translatable and non-translatable rows. locale is the admin's current
// language; pass "" only on first-install paths where no language has
// been seeded. Missing rows resolve to the schema's Default.
//
// Secret-looking keys (token suffixes, etc.) are masked to "***" — the
// admin UI displays a "set" / "not set" indicator without ever rendering
// the real value. Writes that round-trip "***" are skipped at the Save
// layer so the masking doesn't destroy the stored secret.
func (s *Store) Load(schema Schema, locale string) (map[string]string, error) {
	keys := schema.Keys()
	if len(keys) == 0 {
		return map[string]string{}, nil
	}

	// Pull every relevant row in one query: the per-locale row for
	// translatable fields and the empty-locale row for the rest. We
	// post-filter in Go because the locale-vs-key relationship is
	// schema-driven, not encodable as a single SQL WHERE.
	locales := []string{""}
	if locale != "" {
		locales = append(locales, locale)
	}

	var rows []models.SiteSetting
	if err := s.db.
		Where("key IN ? AND language_code IN ?", keys, locales).
		Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("settings: load %q: %w", schema.ID, err)
	}

	byKey := make(map[string]map[string]string, len(rows))
	for _, r := range rows {
		v := ""
		if r.Value != nil {
			v = *r.Value
		}
		if byKey[r.Key] == nil {
			byKey[r.Key] = map[string]string{}
		}
		byKey[r.Key][r.LanguageCode] = v
	}

	out := make(map[string]string, len(keys))
	for _, sec := range schema.Sections {
		for _, f := range sec.Fields {
			row := byKey[f.Key]
			var v string
			var found bool
			if f.Translatable {
				if locale != "" {
					v, found = row[locale]
				}
			} else {
				v, found = row[""]
			}
			if !found {
				v = f.Default
			}
			if secrets.IsSecretKey(f.Key) {
				if v != "" {
					out[f.Key] = "***"
				} else {
					out[f.Key] = ""
				}
				continue
			}
			out[f.Key] = v
		}
	}
	return out, nil
}

// Save writes the supplied values, validating each key against the
// schema and routing to the correct row by Translatable. Unknown keys
// are rejected — the schema is the single source of truth for what
// can be persisted.
//
// Translatable writes require a non-empty locale. The first-install
// edge case (no language seeded) is the caller's problem: the HTTP
// handler returns NO_LANGUAGE before reaching Save.
func (s *Store) Save(schema Schema, locale string, values map[string]string) error {
	if len(values) == 0 {
		return nil
	}
	for key := range values {
		f := schema.FieldByKey(key)
		if f == nil {
			return fmt.Errorf("settings: %q: unknown field %q", schema.ID, key)
		}
		if f.Translatable && locale == "" {
			return fmt.Errorf("settings: %q: field %q is translatable but no locale was supplied", schema.ID, key)
		}
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		for key, raw := range values {
			f := schema.FieldByKey(key) // safe — validated above
			stored := raw
			if secrets.IsSecretKey(key) {
				// Round-trip masking guard: the admin UI always echoes
				// the masked "***" placeholder back on save. Skip it so
				// we don't overwrite the real secret.
				if raw == "***" {
					continue
				}
				if s.secrets != nil {
					enc, err := s.secrets.MaybeEncrypt(raw)
					if err != nil {
						return fmt.Errorf("encrypt %q: %w", key, err)
					}
					stored = enc
				}
			}
			rowLocale := ""
			if f.Translatable {
				rowLocale = locale
			}
			v := stored
			row := models.SiteSetting{Key: key, LanguageCode: rowLocale, Value: &v}
			if err := tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "key"}, {Name: "language_code"}},
				DoUpdates: clause.AssignmentColumns([]string{"value", "updated_at"}),
			}).Create(&row).Error; err != nil {
				return fmt.Errorf("persist %q: %w", key, err)
			}
		}
		return nil
	})
}
