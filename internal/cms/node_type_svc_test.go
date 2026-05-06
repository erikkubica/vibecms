package cms

import (
	"encoding/json"
	"testing"

	"squilla/internal/models"
	"squilla/internal/testutil"
)

// TestNodeTypeServiceUpdate_RebuildsNodeURLsOnPrefixChange pins the
// invariant that changing a node-type's url_prefixes triggers a backfill
// of full_url on every existing content node of that type. Without it,
// the stored URL stays stale until each node is individually re-saved —
// which is the "blog vs post" symptom users hit when they change the
// English prefix from /blog to /post in admin and the public site keeps
// rendering the old prefix.
func TestNodeTypeServiceUpdate_RebuildsNodeURLsOnPrefixChange(t *testing.T) {
	db := testutil.NewSQLiteDB(t)
	// AutoMigrate(&models.ContentNode{}) trips on the gen_random_uuid()
	// default (SQLite has no such function), so create a minimal schema
	// with just the columns buildFullURL and the rebuild loop touch.
	if err := db.AutoMigrate(&models.NodeType{}, &models.Language{}, &models.SiteSetting{}); err != nil {
		t.Fatalf("automigrate: %v", err)
	}
	if err := db.Exec(`
		CREATE TABLE content_nodes (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			parent_id INTEGER,
			node_type TEXT NOT NULL DEFAULT 'page',
			status TEXT NOT NULL DEFAULT 'draft',
			language_code TEXT NOT NULL DEFAULT 'en',
			language_id INTEGER,
			slug TEXT NOT NULL,
			full_url TEXT NOT NULL,
			title TEXT NOT NULL,
			translation_group_id TEXT,
			created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
		)
	`).Error; err != nil {
		t.Fatalf("create content_nodes: %v", err)
	}
	if err := db.Create(&models.Language{Code: "en", Slug: "en", Name: "English"}).Error; err != nil {
		t.Fatalf("seed language: %v", err)
	}
	nt := models.NodeType{
		Slug:        "post",
		Label:       "Post",
		LabelPlural: "Posts",
		URLPrefixes: models.JSONB(`{"en":"blog"}`),
		Fields:      models.JSONB(`[]`),
		Taxonomies:  models.JSONB(`[]`),
	}
	if err := db.Create(&nt).Error; err != nil {
		t.Fatalf("seed node type: %v", err)
	}
	if err := db.Exec(
		`INSERT INTO content_nodes (id, slug, node_type, language_code, status, title, full_url) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		1, "hello", "post", "en", "published", "Hello", "/en/blog/hello",
	).Error; err != nil {
		t.Fatalf("seed node: %v", err)
	}
	nodeID := 1

	svc := NewNodeTypeService(db, nil)
	if _, err := svc.Update(nt.ID, map[string]interface{}{
		"url_prefixes": map[string]interface{}{"en": "post"},
	}); err != nil {
		t.Fatalf("Update: %v", err)
	}

	var fullURL string
	if err := db.Raw(`SELECT full_url FROM content_nodes WHERE id = ?`, nodeID).Scan(&fullURL).Error; err != nil {
		t.Fatalf("reload node: %v", err)
	}
	want := "/en/post/hello"
	if fullURL != want {
		t.Fatalf("full_url = %q, want %q (backfill missing — change to url_prefixes did not propagate)", fullURL, want)
	}
}

// TestNodeTypeServiceUpdate_FieldsKeyMapsToColumn pins the wire-vs-column
// translation: the admin UI sends "fields" (canonical vocabulary), but the
// stored column is named "field_schema". Without the rename the update
// would fail with "no such column: fields" / "column \"fields\" does not
// exist", roll back the whole row, and silently drop unrelated changes
// (url_prefixes, label, etc.) submitted in the same PATCH.
func TestNodeTypeServiceUpdate_FieldsKeyMapsToColumn(t *testing.T) {
	db := testutil.NewSQLiteDB(t)
	if err := db.AutoMigrate(&models.NodeType{}); err != nil {
		t.Fatalf("automigrate: %v", err)
	}
	nt := models.NodeType{
		Slug:        "post",
		Label:       "Post",
		LabelPlural: "Posts",
		URLPrefixes: models.JSONB(`{"en":"/post"}`),
		Fields:      models.JSONB(`[]`),
		Taxonomies:  models.JSONB(`[]`),
	}
	if err := db.Create(&nt).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}

	svc := NewNodeTypeService(db, nil)

	// Mirrors what the admin UI sends after the canonical-vocabulary
	// refactor: "fields" (not "field_schema") plus a sibling change.
	updates := map[string]interface{}{
		"fields": []map[string]interface{}{
			{"name": "subtitle", "title": "Subtitle", "type": "string"},
		},
		"url_prefixes": map[string]interface{}{"en": "/blog"},
	}
	updated, err := svc.Update(nt.ID, updates)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}

	// Sibling field must persist — would silently fail if the bad
	// "fields" key tripped the row rollback.
	var prefixes map[string]string
	if err := json.Unmarshal([]byte(updated.URLPrefixes), &prefixes); err != nil {
		t.Fatalf("unmarshal url_prefixes: %v", err)
	}
	if prefixes["en"] != "/blog" {
		t.Fatalf("url_prefixes en = %q, want /blog", prefixes["en"])
	}

	// And the canonical "fields" payload must land in field_schema.
	var fields []map[string]interface{}
	if err := json.Unmarshal([]byte(updated.Fields), &fields); err != nil {
		t.Fatalf("unmarshal fields: %v", err)
	}
	if len(fields) != 1 || fields[0]["name"] != "subtitle" {
		t.Fatalf("fields not persisted: %v", fields)
	}
}
