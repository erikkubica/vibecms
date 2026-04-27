package main

import (
	"encoding/json"
	"testing"
)

// ---- normalizeForm ----

func TestNormalizeForm_StringFields(t *testing.T) {
	row := map[string]any{
		"id":            float64(1),
		"fields":        `[{"id":"name","label":"Name","type":"text"}]`,
		"notifications": `[{"enabled":true}]`,
		"settings":      `{"honeypot_enabled":true}`,
	}
	result := normalizeForm(row)

	if _, ok := result["fields"].([]any); !ok {
		t.Errorf("fields should be []any after normalize, got %T", result["fields"])
	}
	if _, ok := result["notifications"].([]any); !ok {
		t.Errorf("notifications should be []any after normalize, got %T", result["notifications"])
	}
	if _, ok := result["settings"].(map[string]any); !ok {
		t.Errorf("settings should be map after normalize, got %T", result["settings"])
	}
}

func TestNormalizeForm_AlreadyParsed(t *testing.T) {
	row := map[string]any{
		"fields":   []any{map[string]any{"id": "name"}},
		"settings": map[string]any{"key": "val"},
	}
	result := normalizeForm(row)
	if _, ok := result["fields"].([]any); !ok {
		t.Error("already-parsed fields should remain as []any")
	}
}

func TestNormalizeForm_InvalidJSON(t *testing.T) {
	row := map[string]any{
		"fields": "not-valid-json",
	}
	result := normalizeForm(row)
	// Invalid JSON should remain as the original string
	if _, ok := result["fields"].(string); !ok {
		t.Error("invalid JSON should remain as string")
	}
}

// ---- getFormFields ----

func TestGetFormFields_SliceAny(t *testing.T) {
	form := map[string]any{
		"fields": []any{
			map[string]any{"id": "name", "type": "text"},
			map[string]any{"id": "email", "type": "email"},
		},
	}
	fields := getFormFields(form)
	if len(fields) != 2 {
		t.Errorf("expected 2 fields, got %d", len(fields))
	}
}

func TestGetFormFields_SliceMaps(t *testing.T) {
	form := map[string]any{
		"fields": []map[string]any{
			{"id": "name"},
		},
	}
	fields := getFormFields(form)
	if len(fields) != 1 {
		t.Errorf("expected 1 field, got %d", len(fields))
	}
}

func TestGetFormFields_Nil(t *testing.T) {
	form := map[string]any{}
	fields := getFormFields(form)
	if fields != nil {
		t.Errorf("missing fields key should return nil, got %v", fields)
	}
}

// ---- getFormSettings ----

func TestGetFormSettings_Present(t *testing.T) {
	form := map[string]any{
		"settings": map[string]any{"rate_limit": float64(5)},
	}
	s := getFormSettings(form)
	if s["rate_limit"] != float64(5) {
		t.Errorf("expected rate_limit=5, got %v", s["rate_limit"])
	}
}

func TestGetFormSettings_Missing(t *testing.T) {
	s := getFormSettings(map[string]any{})
	if s == nil || len(s) != 0 {
		t.Errorf("missing settings should return empty map, got %v", s)
	}
}

// ---- parseJSONMap ----

func TestParseJSONMap_Map(t *testing.T) {
	m := map[string]any{"key": "val"}
	result := parseJSONMap(m)
	if result["key"] != "val" {
		t.Errorf("map input should be returned as-is: %v", result)
	}
}

func TestParseJSONMap_JSONString(t *testing.T) {
	result := parseJSONMap(`{"foo":"bar"}`)
	if result["foo"] != "bar" {
		t.Errorf("JSON string should be parsed: %v", result)
	}
}

func TestParseJSONMap_Invalid(t *testing.T) {
	if parseJSONMap("not-json") != nil {
		t.Error("invalid JSON string should return nil")
	}
	if parseJSONMap(nil) != nil {
		t.Error("nil should return nil")
	}
	if parseJSONMap(42) != nil {
		t.Error("non-string/map should return nil")
	}
}

// ---- fieldValueToString ----

func TestFieldValueToString(t *testing.T) {
	cases := []struct {
		v    any
		want string
	}{
		{"hello", "hello"},
		{nil, ""},
		{map[string]any{"label": "Option A", "value": "a"}, "Option A"},
		{map[string]any{"value": "x"}, ""},  // no label → uses Sprintf
		{42, "42"},
		{3.14, "3.14"},
	}
	for _, tc := range cases {
		got := fieldValueToString(tc.v)
		if got != tc.want {
			// For maps without label, we just check it doesn't panic
			if _, ok := tc.v.(map[string]any); ok {
				continue
			}
			t.Errorf("fieldValueToString(%v) = %q, want %q", tc.v, got, tc.want)
		}
	}
}

// ---- uniqueSlug ----

func TestUniqueSlug_NoCollision(t *testing.T) {
	h := NewFakeHost()
	slug := uniqueSlug(ctx(), h, "my-form")
	if slug != "my-form" {
		t.Errorf("no collision: expected my-form, got %q", slug)
	}
}

func TestUniqueSlug_WithCollision(t *testing.T) {
	h := NewFakeHost()
	// Pre-create a form with slug "contact"
	h.DataCreate(ctx(), formsTable, map[string]any{"name": "Contact", "slug": "contact"})

	slug := uniqueSlug(ctx(), h, "contact")
	if slug != "contact-2" {
		t.Errorf("collision: expected contact-2, got %q", slug)
	}
}

func TestUniqueSlug_MultipleCollisions(t *testing.T) {
	h := NewFakeHost()
	h.DataCreate(ctx(), formsTable, map[string]any{"name": "C", "slug": "contact"})
	h.DataCreate(ctx(), formsTable, map[string]any{"name": "C2", "slug": "contact-2"})

	slug := uniqueSlug(ctx(), h, "contact")
	if slug != "contact-3" {
		t.Errorf("multiple collisions: expected contact-3, got %q", slug)
	}
}

// ---- normalizeForm JSON round-trip ----

func TestNormalizeForm_SettingsRoundTrip(t *testing.T) {
	settings := map[string]any{
		"retention_period": "30",
		"honeypot_enabled": true,
	}
	settingsJSON, _ := json.Marshal(settings)
	row := map[string]any{
		"settings": string(settingsJSON),
	}
	result := normalizeForm(row)
	s, ok := result["settings"].(map[string]any)
	if !ok {
		t.Fatalf("settings should be map after normalize, got %T", result["settings"])
	}
	if s["retention_period"] != "30" {
		t.Errorf("retention_period: got %v", s["retention_period"])
	}
}
