package settings

import (
	"testing"

	"squilla/internal/models"
	"squilla/internal/testutil"
)

// TestStore_TranslatableRoutesPerLocale is the regression test for the
// bug that surfaced after schema-driven settings shipped: a translatable
// field saved in 'en' must not appear under 'sk', and a global field
// must round-trip the same value regardless of which locale the admin
// is editing in.
func TestStore_TranslatableRoutesPerLocale(t *testing.T) {
	db := testutil.NewSQLiteDB(t)
	if err := db.AutoMigrate(&models.SiteSetting{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	store := NewStore(db, nil)

	schema := Schema{
		ID: "test", Title: "T",
		Sections: []Section{{Title: "A", Fields: []Field{
			{Key: "tagline", Type: "text", Translatable: true, Default: "hello"},
			{Key: "accent", Type: "select", Translatable: false, Default: "teal"},
		}}},
	}

	if err := store.Save(schema, "en", map[string]string{
		"tagline": "Hello world",
		"accent":  "violet",
	}); err != nil {
		t.Fatalf("save en: %v", err)
	}

	gotEn, err := store.Load(schema, "en")
	if err != nil {
		t.Fatalf("load en: %v", err)
	}
	if gotEn["tagline"] != "Hello world" {
		t.Errorf("translatable en read: got %q, want %q", gotEn["tagline"], "Hello world")
	}
	if gotEn["accent"] != "violet" {
		t.Errorf("global en read: got %q, want %q", gotEn["accent"], "violet")
	}

	gotSk, err := store.Load(schema, "sk")
	if err != nil {
		t.Fatalf("load sk: %v", err)
	}
	if gotSk["tagline"] != "hello" {
		t.Errorf("untranslated sk should fall back to default: got %q, want %q", gotSk["tagline"], "hello")
	}
	if gotSk["accent"] != "violet" {
		t.Errorf("global field should be visible from any locale: got %q, want %q", gotSk["accent"], "violet")
	}
}

// TestStore_NonTranslatableWritesGlobalRow confirms that flipping a
// non-translatable field while editing in 'en' does NOT create a row at
// language_code='en' — only at language_code=''. This was the reload
// regression: the value persisted under the wrong locale and the read
// path missed it.
func TestStore_NonTranslatableWritesGlobalRow(t *testing.T) {
	db := testutil.NewSQLiteDB(t)
	if err := db.AutoMigrate(&models.SiteSetting{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	store := NewStore(db, nil)

	schema := Schema{
		ID: "security", Title: "S",
		Sections: []Section{{Title: "S", Fields: []Field{
			{Key: "allow_registration", Type: "toggle", Translatable: false, TrueValue: "true", FalseValue: "false", Default: "false"},
		}}},
	}

	if err := store.Save(schema, "en", map[string]string{"allow_registration": "true"}); err != nil {
		t.Fatalf("save: %v", err)
	}

	var rows []models.SiteSetting
	if err := db.Where("key = ?", "allow_registration").Find(&rows).Error; err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d: %+v", len(rows), rows)
	}
	if rows[0].LanguageCode != "" {
		t.Errorf("non-translatable should land at language_code='', got %q", rows[0].LanguageCode)
	}
}

// TestStore_RejectsUnknownKey ensures the schema is the single source of
// truth on the write path. A value for a key not declared in the schema
// is rejected — otherwise an attacker / typo could pollute the table
// with arbitrary settings via the schema endpoint.
func TestStore_RejectsUnknownKey(t *testing.T) {
	db := testutil.NewSQLiteDB(t)
	if err := db.AutoMigrate(&models.SiteSetting{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	store := NewStore(db, nil)
	schema := Schema{
		ID: "test", Title: "T",
		Sections: []Section{{Title: "A", Fields: []Field{
			{Key: "ok", Type: "text", Translatable: true},
		}}},
	}
	err := store.Save(schema, "en", map[string]string{"unknown": "x"})
	if err == nil {
		t.Fatal("expected error for unknown key")
	}
}

// TestStore_TranslatableRequiresLocale guards against a regression where
// an empty locale would silently route translatable writes to the global
// row, conflating them across languages.
func TestStore_TranslatableRequiresLocale(t *testing.T) {
	db := testutil.NewSQLiteDB(t)
	if err := db.AutoMigrate(&models.SiteSetting{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	store := NewStore(db, nil)
	schema := Schema{
		ID: "test", Title: "T",
		Sections: []Section{{Title: "A", Fields: []Field{
			{Key: "tagline", Type: "text", Translatable: true},
		}}},
	}
	err := store.Save(schema, "", map[string]string{"tagline": "hi"})
	if err == nil {
		t.Fatal("expected error when saving translatable field with empty locale")
	}
}

// TestStore_LoadFallsBackToDefault is the "fresh database" path — no
// rows yet, the schema declares defaults, the load path should surface
// those defaults so the admin UI doesn't render empty inputs.
func TestStore_LoadFallsBackToDefault(t *testing.T) {
	db := testutil.NewSQLiteDB(t)
	if err := db.AutoMigrate(&models.SiteSetting{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	store := NewStore(db, nil)
	schema := Schema{
		ID: "test", Title: "T",
		Sections: []Section{{Title: "A", Fields: []Field{
			{Key: "size", Type: "text", Translatable: false, Default: "medium"},
		}}},
	}
	got, err := store.Load(schema, "en")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if got["size"] != "medium" {
		t.Errorf("default fallback: got %q, want %q", got["size"], "medium")
	}
}

// TestStore_SaveIsIdempotent verifies the upsert path — saving the same
// value twice produces one row, not two, and the second save updates
// rather than failing on the (key, language_code) primary key.
func TestStore_SaveIsIdempotent(t *testing.T) {
	db := testutil.NewSQLiteDB(t)
	if err := db.AutoMigrate(&models.SiteSetting{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	store := NewStore(db, nil)
	schema := Schema{
		ID: "test", Title: "T",
		Sections: []Section{{Title: "A", Fields: []Field{
			{Key: "tagline", Type: "text", Translatable: true},
		}}},
	}
	for i, val := range []string{"a", "b", "c"} {
		if err := store.Save(schema, "en", map[string]string{"tagline": val}); err != nil {
			t.Fatalf("save %d: %v", i, err)
		}
	}
	var rows []models.SiteSetting
	if err := db.Where("key = ?", "tagline").Find(&rows).Error; err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row after idempotent saves, got %d", len(rows))
	}
	if rows[0].Value == nil || *rows[0].Value != "c" {
		got := "<nil>"
		if rows[0].Value != nil {
			got = *rows[0].Value
		}
		t.Errorf("last value should win: got %q, want %q", got, "c")
	}
}

// TestStore_SecretMasking confirms credential-like keys round-trip as
// "***" on the load path even when a real value is stored, so the admin
// UI can never accidentally render the secret.
func TestStore_SecretMasking(t *testing.T) {
	db := testutil.NewSQLiteDB(t)
	if err := db.AutoMigrate(&models.SiteSetting{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	store := NewStore(db, nil)
	schema := Schema{
		ID: "smtp", Title: "S",
		Sections: []Section{{Title: "S", Fields: []Field{
			{Key: "smtp_password", Type: "text", Translatable: false},
		}}},
	}
	if err := store.Save(schema, "en", map[string]string{"smtp_password": "shh"}); err != nil {
		t.Fatalf("save: %v", err)
	}
	got, err := store.Load(schema, "en")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if got["smtp_password"] != "***" {
		t.Errorf("secret should be masked on read: got %q, want %q", got["smtp_password"], "***")
	}
	// Round-trip "***" must NOT overwrite the stored secret.
	if err := store.Save(schema, "en", map[string]string{"smtp_password": "***"}); err != nil {
		t.Fatalf("re-save mask: %v", err)
	}
	var row models.SiteSetting
	if err := db.Where("key = ?", "smtp_password").First(&row).Error; err != nil {
		t.Fatalf("query: %v", err)
	}
	if row.Value == nil || *row.Value != "shh" {
		got := "<nil>"
		if row.Value != nil {
			got = *row.Value
		}
		t.Errorf("masked round-trip overwrote secret: got %q, want %q", got, "shh")
	}
}
