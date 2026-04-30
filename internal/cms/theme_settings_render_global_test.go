package cms

import (
	"context"
	"testing"

	"squilla/internal/models"
	"squilla/internal/testutil"
)

// TestBuildThemeSettingsContext_NonTranslatableVisible is the regression
// test for the public-render-side leak of the same bug we already fixed
// in the admin handler. Symptom: header pill, CTA, footer copy all
// rendered empty / default on the live site even after the operator
// saved values, because BuildThemeSettingsContextForLocale called
// GetSettingsLoc which doesn't surface language_code='' rows.
//
// Steps: seed a translatable field at locale 'en' and a global field at
// language_code='', then build the render context for 'en'. Both must
// resolve to their stored values.
func TestBuildThemeSettingsContext_NonTranslatableVisible(t *testing.T) {
	db := testutil.NewSQLiteDB(t)
	if err := db.AutoMigrate(&models.SiteSetting{}, &models.Language{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := db.Create(&models.Language{Code: "en", Slug: "en", Name: "English", IsDefault: true, IsActive: true}).Error; err != nil {
		t.Fatalf("seed lang: %v", err)
	}

	prefix := ThemePrefix("squilla")
	en := "Hello"
	tagline := models.SiteSetting{Key: prefix + "branding:tagline", LanguageCode: "en", Value: &en}
	if err := db.Create(&tagline).Error; err != nil {
		t.Fatalf("seed tagline: %v", err)
	}
	violet := "violet"
	accent := models.SiteSetting{Key: prefix + "branding:accent", LanguageCode: "", Value: &violet}
	if err := db.Create(&accent).Error; err != nil {
		t.Fatalf("seed accent: %v", err)
	}
	yes := "true"
	pill := models.SiteSetting{Key: prefix + "header:show_meta_pill", LanguageCode: "", Value: &yes}
	if err := db.Create(&pill).Error; err != nil {
		t.Fatalf("seed pill: %v", err)
	}

	tt := true
	ff := false
	branding := ThemeSettingsPage{
		Slug: "branding",
		Fields: []ThemeSettingsField{
			{Key: "tagline", Type: "text", Translatable: &tt},
			{Key: "accent", Type: "select", Translatable: &ff},
		},
	}
	header := ThemeSettingsPage{
		Slug: "header",
		Fields: []ThemeSettingsField{
			{Key: "show_meta_pill", Type: "toggle", Translatable: &ff},
		},
	}
	reg := NewThemeSettingsRegistry()
	reg.SetActive("squilla", []ThemeSettingsPage{branding, header})

	api := newDBSettingsAPI(db)
	ctx, err := BuildThemeSettingsContextForLocale(context.Background(), reg, api, "en")
	if err != nil {
		t.Fatalf("build context: %v", err)
	}

	if v, _ := ctx["branding"]["tagline"].(string); v != "Hello" {
		t.Errorf("translatable tagline en: got %v, want %q", ctx["branding"]["tagline"], "Hello")
	}
	if v, _ := ctx["branding"]["accent"].(string); v != "violet" {
		t.Errorf("global accent (regression): got %v, want %q", ctx["branding"]["accent"], "violet")
	}
	if v, _ := ctx["header"]["show_meta_pill"].(bool); !v {
		t.Errorf("global toggle (regression): got %v, want true", ctx["header"]["show_meta_pill"])
	}
}
