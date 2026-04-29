package cms

import (
	"bytes"
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func captureLog(t *testing.T) (*bytes.Buffer, func()) {
	t.Helper()
	var buf bytes.Buffer
	prev := log.Writer()
	log.SetOutput(&buf)
	return &buf, func() { log.SetOutput(prev) }
}

func TestLoadSettingsPages_ValidPagesParsed(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "settings", "header.json"), `{
		"name": "Header Settings",
		"description": "Logo, tagline, etc.",
		"fields": [
			{"key":"logo","label":"Logo","type":"media"},
			{"key":"tagline","label":"Tagline","type":"text","default":"Welcome"}
		]
	}`)
	writeFile(t, filepath.Join(dir, "settings", "api.json"), `{
		"name": "API Keys",
		"fields": [
			{"key":"stripe_key","label":"Stripe","type":"text"}
		]
	}`)

	manifest := ThemeManifest{
		SettingsPages: []ThemeSettingsPageDef{
			{Slug: "header", Name: "Header Settings", File: "settings/header.json", Icon: "panel-top"},
			{Slug: "api-keys", Name: "API Keys", File: "settings/api.json"},
		},
	}

	pages := LoadSettingsPages(dir, manifest)
	if len(pages) != 2 {
		t.Fatalf("want 2 pages, got %d", len(pages))
	}
	if pages[0].Slug != "header" || pages[1].Slug != "api-keys" {
		t.Fatalf("order not preserved: %+v", pages)
	}
	if pages[0].Icon != "panel-top" {
		t.Errorf("want icon panel-top, got %q", pages[0].Icon)
	}
	if pages[0].Description != "Logo, tagline, etc." {
		t.Errorf("description missing: %q", pages[0].Description)
	}
	if len(pages[0].Fields) != 2 {
		t.Fatalf("want 2 fields, got %d", len(pages[0].Fields))
	}
	if pages[0].Fields[1].Key != "tagline" || string(pages[0].Fields[1].Default) != `"Welcome"` {
		t.Errorf("default not preserved: %+v", pages[0].Fields[1])
	}
}

func TestLoadSettingsPages_InvalidPageSkippedWithLog(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "settings", "ok.json"), `{"name":"OK","fields":[{"key":"a","type":"text"}]}`)
	writeFile(t, filepath.Join(dir, "settings", "bad.json"), `{not valid json`)

	manifest := ThemeManifest{
		SettingsPages: []ThemeSettingsPageDef{
			{Slug: "ok", Name: "OK", File: "settings/ok.json"},
			{Slug: "bad", Name: "Bad", File: "settings/bad.json"},
		},
	}

	buf, restore := captureLog(t)
	defer restore()

	pages := LoadSettingsPages(dir, manifest)
	if len(pages) != 1 {
		t.Fatalf("want 1 page, got %d", len(pages))
	}
	if pages[0].Slug != "ok" {
		t.Errorf("wrong page kept: %+v", pages[0])
	}
	if !strings.Contains(buf.String(), `"bad"`) {
		t.Errorf("expected log mention of bad slug, got: %q", buf.String())
	}
}

func TestLoadSettingsPages_MissingFileSkipped(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "settings", "ok.json"), `{"name":"OK","fields":[{"key":"a","type":"text"}]}`)

	manifest := ThemeManifest{
		SettingsPages: []ThemeSettingsPageDef{
			{Slug: "missing", Name: "Missing", File: "settings/nope.json"},
			{Slug: "ok", Name: "OK", File: "settings/ok.json"},
		},
	}

	_, restore := captureLog(t)
	defer restore()

	pages := LoadSettingsPages(dir, manifest)
	if len(pages) != 1 {
		t.Fatalf("want 1 page, got %d", len(pages))
	}
	if pages[0].Slug != "ok" {
		t.Errorf("wrong page kept: %+v", pages[0])
	}
}

func TestLoadSettingsPages_NoneDeclared(t *testing.T) {
	dir := t.TempDir()
	pages := LoadSettingsPages(dir, ThemeManifest{})
	if len(pages) != 0 {
		t.Fatalf("want empty, got %d", len(pages))
	}
}

func TestLoadSettingsPages_FieldsWithMissingKeySkipped(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "settings", "p.json"), `{
		"name":"P",
		"fields":[
			{"key":"good","label":"Good","type":"text"},
			{"key":"","label":"Missing key","type":"text"},
			{"key":"no_type","label":"Missing type","type":""}
		]
	}`)

	manifest := ThemeManifest{
		SettingsPages: []ThemeSettingsPageDef{
			{Slug: "p", Name: "P", File: "settings/p.json"},
		},
	}

	_, restore := captureLog(t)
	defer restore()

	pages := LoadSettingsPages(dir, manifest)
	if len(pages) != 1 {
		t.Fatalf("want 1 page, got %d", len(pages))
	}
	if len(pages[0].Fields) != 1 || pages[0].Fields[0].Key != "good" {
		t.Errorf("expected only 'good' field, got %+v", pages[0].Fields)
	}
}

func TestLoadSettingsPages_RawAndConfigPreserved(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "settings", "p.json"), `{
		"name":"P",
		"fields":[
			{"key":"k","label":"K","type":"select","options":["a","b","c"],"min":1}
		]
	}`)

	manifest := ThemeManifest{
		SettingsPages: []ThemeSettingsPageDef{
			{Slug: "p", Name: "P", File: "settings/p.json"},
		},
	}

	pages := LoadSettingsPages(dir, manifest)
	if len(pages) != 1 || len(pages[0].Fields) != 1 {
		t.Fatalf("setup failed: %+v", pages)
	}
	f := pages[0].Fields[0]

	if len(f.Raw) == 0 {
		t.Fatal("Raw not preserved")
	}
	var raw map[string]any
	if err := json.Unmarshal(f.Raw, &raw); err != nil {
		t.Fatalf("Raw not valid JSON: %v", err)
	}
	if _, ok := raw["options"]; !ok {
		t.Errorf("Raw missing options: %s", string(f.Raw))
	}

	if f.Config == nil {
		t.Fatal("Config not populated")
	}
	if _, ok := f.Config["options"]; !ok {
		t.Errorf("Config missing options: %+v", f.Config)
	}
	if _, ok := f.Config["min"]; !ok {
		t.Errorf("Config missing min: %+v", f.Config)
	}
	// Standard keys should not be in Config (only renderer-specific extras).
	if _, ok := f.Config["key"]; ok {
		t.Errorf("Config should not include 'key': %+v", f.Config)
	}
	if _, ok := f.Config["type"]; ok {
		t.Errorf("Config should not include 'type': %+v", f.Config)
	}
}

func TestLoadSettingsPages_NameFallback(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "settings", "p.json"), `{
		"fields":[{"key":"a","label":"A","type":"text"}]
	}`)

	manifest := ThemeManifest{
		SettingsPages: []ThemeSettingsPageDef{
			{Slug: "p", Name: "Manifest Name", File: "settings/p.json"},
		},
	}

	pages := LoadSettingsPages(dir, manifest)
	if len(pages) != 1 {
		t.Fatalf("want 1 page, got %d", len(pages))
	}
	if pages[0].Name != "Manifest Name" {
		t.Errorf("expected fallback name, got %q", pages[0].Name)
	}
}
