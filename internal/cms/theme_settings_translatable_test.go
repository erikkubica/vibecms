package cms

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"squilla/internal/models"
	"squilla/internal/testutil"
)

// dbSettingsAPI is a settingsAPI implementation backed by a real GORM
// connection. The pre-existing fakeCoreAPI in the older test file uses
// an in-memory map and ignores language_code entirely, which is why
// the recent translatable regressions slipped through unit tests. This
// variant exercises the same per-locale storage the production handler
// hits in production.
type dbSettingsAPI struct{ db *gorm.DB }

func newDBSettingsAPI(db *gorm.DB) *dbSettingsAPI { return &dbSettingsAPI{db: db} }

func (a *dbSettingsAPI) GetSetting(ctx context.Context, key string) (string, error) {
	var s models.SiteSetting
	if err := a.db.WithContext(ctx).Where("\"key\" = ?", key).First(&s).Error; err != nil {
		return "", nil
	}
	if s.Value == nil {
		return "", nil
	}
	return *s.Value, nil
}

func (a *dbSettingsAPI) SetSetting(ctx context.Context, key, value string) error {
	return a.SetSettingLoc(ctx, key, "", value)
}

func (a *dbSettingsAPI) GetSettings(ctx context.Context, prefix string) (map[string]string, error) {
	return a.GetSettingsLoc(ctx, prefix, "")
}

// GetSettingsLoc mirrors coreImpl.GetSettingsLoc — returns the locale's
// row when present, falling back to the default-language row. Empty
// locale resolves through the same default-language fallback so reads
// don't see global rows; this is the production behaviour the theme
// handler now compensates for by querying the empty-locale row
// directly.
func (a *dbSettingsAPI) GetSettingsLoc(ctx context.Context, prefix, locale string) (map[string]string, error) {
	def := a.defaultLocale(ctx)
	if locale == "" {
		locale = def
	}
	q := a.db.WithContext(ctx).Model(&models.SiteSetting{})
	if prefix != "" {
		q = q.Where("\"key\" LIKE ?", prefix+"%")
	}
	switch {
	case locale == "" && def == "":
		return map[string]string{}, nil
	case def == "" || locale == def:
		q = q.Where("language_code = ?", locale)
	default:
		q = q.Where("language_code IN ?", []string{locale, def})
	}
	var rows []models.SiteSetting
	if err := q.Find(&rows).Error; err != nil {
		return nil, err
	}
	type pick struct{ val, loc string }
	chosen := map[string]pick{}
	for _, r := range rows {
		v := ""
		if r.Value != nil {
			v = *r.Value
		}
		exist, ok := chosen[r.Key]
		if !ok || (exist.loc == def && r.LanguageCode == locale) {
			chosen[r.Key] = pick{val: v, loc: r.LanguageCode}
		}
	}
	out := make(map[string]string, len(chosen))
	for k, p := range chosen {
		out[strings.TrimPrefix(k, prefix)] = p.val
	}
	return out, nil
}

func (a *dbSettingsAPI) SetSettingLoc(ctx context.Context, key, locale, value string) error {
	v := value
	row := models.SiteSetting{Key: key, LanguageCode: locale, Value: &v}
	return a.db.WithContext(ctx).Save(&row).Error
}

// GetSettingsGlobal mirrors coreImpl.GetSettingsGlobal so the
// globalSettingsReader type assertion in BuildThemeSettingsContextForLocale
// succeeds in tests that exercise the public render path.
func (a *dbSettingsAPI) GetSettingsGlobal(ctx context.Context, prefix string) (map[string]string, error) {
	q := a.db.WithContext(ctx).Model(&models.SiteSetting{}).Where("language_code = ?", "")
	if prefix != "" {
		q = q.Where("\"key\" LIKE ?", prefix+"%")
	}
	var rows []models.SiteSetting
	if err := q.Find(&rows).Error; err != nil {
		return nil, err
	}
	out := map[string]string{}
	for _, r := range rows {
		v := ""
		if r.Value != nil {
			v = *r.Value
		}
		out[strings.TrimPrefix(r.Key, prefix)] = v
	}
	return out, nil
}

func (a *dbSettingsAPI) defaultLocale(ctx context.Context) string {
	var code string
	_ = a.db.WithContext(ctx).Table("languages").Select("code").Where("is_default = ?", true).Limit(1).Scan(&code).Error
	return code
}

// makeBrandingPage returns a settings page with one translatable text
// field plus one global select field — mirrors the squilla theme's
// branding + appearance pages closely enough to catch routing
// regressions.
func makeBrandingPage(t *testing.T) ThemeSettingsPage {
	t.Helper()
	tagline := makeField(t, "tagline", "Tagline", "text", "")
	tt := true
	tagline.Translatable = &tt
	accent := makeField(t, "accent", "Accent", "select", "teal")
	ff := false
	accent.Translatable = &ff
	return ThemeSettingsPage{
		Slug:   "branding",
		Name:   "Branding",
		Fields: []ThemeSettingsField{tagline, accent},
	}
}

// TestThemeSettings_TranslatableRoundTrip is the regression test for the
// "saving stopped working" symptom. The pre-fix bug: non-translatable
// fields were saved at language_code='' (correct) but read back via
// GetSettingsLoc with empty locale, which the helper resolves to the
// default-language row. The admin UI then showed the default value
// instead of the saved one.
//
// Steps: seed languages, save in 'en' across both fields, GET back in
// 'en' and 'sk'. The translatable tagline must only appear in the
// locale where it was saved; the global accent must appear in both.
func TestThemeSettings_TranslatableRoundTrip(t *testing.T) {
	db := testutil.NewSQLiteDB(t)
	if err := db.AutoMigrate(&models.SiteSetting{}, &models.Language{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := db.Create(&models.Language{Code: "en", Slug: "en", Name: "English", IsDefault: true, IsActive: true}).Error; err != nil {
		t.Fatalf("seed lang en: %v", err)
	}
	if err := db.Create(&models.Language{Code: "sk", Slug: "sk", Name: "Slovak", IsActive: true}).Error; err != nil {
		t.Fatalf("seed lang sk: %v", err)
	}

	reg := NewThemeSettingsRegistry()
	reg.SetActive("squilla", []ThemeSettingsPage{makeBrandingPage(t)})

	api := newDBSettingsAPI(db)
	h := NewThemeSettingsHandler(reg, api, db, nil, nil)

	app := fiber.New()
	app.Get("/theme-settings/:page", h.Get)
	app.Put("/theme-settings/:page", h.Save)

	saveBody := `{"values":{"tagline":"Hello, world","accent":"violet"}}`
	saveReq := httptest.NewRequest("PUT", "/theme-settings/branding", strings.NewReader(saveBody))
	saveReq.Header.Set("Content-Type", "application/json")
	saveReq.Header.Set("X-Admin-Language", "en")
	saveResp, err := app.Test(saveReq)
	if err != nil {
		t.Fatalf("save request: %v", err)
	}
	if saveResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(saveResp.Body)
		t.Fatalf("save status %d: %s", saveResp.StatusCode, body)
	}

	gotEn := decodeThemeSettingsGet(t, app, "en")
	if v := strFromValue(gotEn["tagline"]); v != "Hello, world" {
		t.Errorf("translatable en read: got %q, want %q", v, "Hello, world")
	}
	if v := strFromValue(gotEn["accent"]); v != "violet" {
		t.Errorf("global en read (regression): got %q, want %q", v, "violet")
	}

	gotSk := decodeThemeSettingsGet(t, app, "sk")
	if v := strFromValue(gotSk["accent"]); v != "violet" {
		t.Errorf("global field invisible in sk locale (regression): got %q, want %q", v, "violet")
	}

	var rows []models.SiteSetting
	if err := db.Where("\"key\" LIKE ?", ThemePrefix("squilla")+"%").Find(&rows).Error; err != nil {
		t.Fatalf("query: %v", err)
	}
	for _, r := range rows {
		switch r.Key {
		case SettingKey("squilla", "branding", "tagline"):
			if r.LanguageCode != "en" {
				t.Errorf("translatable tagline stored at %q, want %q", r.LanguageCode, "en")
			}
		case SettingKey("squilla", "branding", "accent"):
			if r.LanguageCode != "" {
				t.Errorf("global accent stored at %q, want empty string", r.LanguageCode)
			}
		default:
			t.Errorf("unexpected row key %q", r.Key)
		}
	}
}

func decodeThemeSettingsGet(t *testing.T, app *fiber.App, locale string) map[string]valueDTO {
	t.Helper()
	req := httptest.NewRequest("GET", "/theme-settings/branding", nil)
	req.Header.Set("X-Admin-Language", locale)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("get request: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("get status %d: %s", resp.StatusCode, body)
	}
	body, _ := io.ReadAll(resp.Body)
	var env struct {
		Data getResponse `json:"data"`
	}
	if err := json.NewDecoder(bytes.NewReader(body)).Decode(&env); err != nil {
		t.Fatalf("decode: %v (%s)", err, body)
	}
	return env.Data.Values
}

func strFromValue(v valueDTO) string {
	if s, ok := v.Value.(string); ok {
		return s
	}
	return ""
}
