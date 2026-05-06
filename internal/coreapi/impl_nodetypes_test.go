package coreapi

import (
	"context"
	"testing"

	"squilla/internal/cms"
	"squilla/internal/models"
	"squilla/internal/testutil"
)

// TestRegisterNodeType_PreservesUserURLPrefixes pins the rule that a theme
// seed calling nodetypes.register({slug:"post", ...}) WITHOUT supplying
// url_prefixes must not clobber whatever the operator previously set in
// admin. The squilla theme's setup/nodetypes.tengo registers "post" with
// fields but no url_prefixes; without this guard, every theme reactivation
// resets url_prefixes to "null" — the "blog vs post" bug.
func TestRegisterNodeType_PreservesUserURLPrefixes(t *testing.T) {
	db := testutil.NewSQLiteDB(t)
	if err := db.AutoMigrate(&models.NodeType{}); err != nil {
		t.Fatalf("automigrate: %v", err)
	}
	// Seed an existing nodetype with operator-set url_prefixes.
	if err := db.Create(&models.NodeType{
		Slug:        "post",
		Label:       "Post",
		LabelPlural: "Posts",
		URLPrefixes: models.JSONB(`{"en":"blog"}`),
		Fields:      models.JSONB(`[]`),
		Taxonomies:  models.JSONB(`[]`),
	}).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}

	c := &coreImpl{
		db:          db,
		nodeTypeSvc: cms.NewNodeTypeService(db, nil),
	}

	// Mirrors the squilla theme seed: slug + label + fields, NO url_prefixes.
	_, err := c.RegisterNodeType(context.Background(), NodeTypeInput{
		Slug:   "post",
		Label:  "Post",
		Fields: []NodeTypeField{{Name: "excerpt", Type: "string"}},
	})
	if err != nil {
		t.Fatalf("RegisterNodeType: %v", err)
	}

	var refreshed models.NodeType
	if err := db.First(&refreshed, "slug = ?", "post").Error; err != nil {
		t.Fatalf("reload: %v", err)
	}
	got := string(refreshed.URLPrefixes)
	if got != `{"en":"blog"}` {
		t.Fatalf("url_prefixes = %s, want preserved {\"en\":\"blog\"} (theme seed clobbered admin setting)", got)
	}
}
