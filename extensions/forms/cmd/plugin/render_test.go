package main

import (
	"strings"
	"testing"
)

// simpleForm creates a minimal form with the given fields and layout.
func simpleForm(layout string, fields []map[string]any) map[string]any {
	return map[string]any{
		"id":     float64(1),
		"name":   "Test Form",
		"fields": fields,
		"layout": layout,
	}
}

// ---- renderFormHTML ----

func TestRenderFormHTML_EmptyLayout(t *testing.T) {
	p := newPlugin(NewFakeHost())
	form := simpleForm("", nil)
	_, err := p.renderFormHTML(form)
	if err == nil || !strings.Contains(err.Error(), "layout is empty") {
		t.Errorf("expected layout-empty error, got %v", err)
	}
}

func TestRenderFormHTML_InvalidLayout(t *testing.T) {
	p := newPlugin(NewFakeHost())
	form := simpleForm("{{.Unclosed", nil)
	_, err := p.renderFormHTML(form)
	if err == nil {
		t.Error("invalid template should error")
	}
}

func TestRenderFormHTML_SimpleLayout(t *testing.T) {
	p := newPlugin(NewFakeHost())
	fields := []map[string]any{
		{"id": "email", "label": "Email", "type": "email"},
	}
	form := simpleForm(`<form>{{range .fields_list}}<input name="{{.id}}">{{end}}</form>`, fields)
	html, err := p.renderFormHTML(form)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(html, `name="email"`) {
		t.Errorf("rendered HTML should contain email input, got: %s", html)
	}
}

func TestRenderFormHTML_FieldByIDAccess(t *testing.T) {
	p := newPlugin(NewFakeHost())
	fields := []map[string]any{
		{"id": "name", "label": "Full Name", "type": "text"},
	}
	form := simpleForm(`<form>{{.name.label}}</form>`, fields)
	html, err := p.renderFormHTML(form)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(html, "Full Name") {
		t.Errorf("template shorthand .name.label should render label, got: %s", html)
	}
}

func TestRenderFormHTML_MetaScript(t *testing.T) {
	p := newPlugin(NewFakeHost())
	form := simpleForm(`<form></form>`, nil)
	html, err := p.renderFormHTML(form)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(html, `data-vibe-form-meta`) {
		t.Error("rendered HTML should contain meta script tag")
	}
}

func TestRenderFormHTML_HoneypotInjected(t *testing.T) {
	p := newPlugin(NewFakeHost())
	form := simpleForm(`<form></form>`, nil)
	html, err := p.renderFormHTML(form)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(html, `name="website_url"`) {
		t.Error("honeypot field should be injected by default")
	}
}

func TestRenderFormHTML_HoneypotDisabled(t *testing.T) {
	p := newPlugin(NewFakeHost())
	form := map[string]any{
		"id":     float64(1),
		"name":   "Test",
		"fields": []any{},
		"layout": `<form></form>`,
		"settings": map[string]any{
			"honeypot_enabled": false,
		},
	}
	html, err := p.renderFormHTML(form)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(html, `name="website_url"`) {
		t.Error("honeypot should not be injected when disabled")
	}
}

func TestRenderFormHTML_CaptchaInjected(t *testing.T) {
	p := newPlugin(NewFakeHost())
	form := map[string]any{
		"id":     float64(1),
		"name":   "Test",
		"fields": []any{},
		"layout": `<form></form>`,
		"settings": map[string]any{
			"captcha_provider": "recaptcha",
			"captcha_site_key": "key123",
		},
	}
	html, err := p.renderFormHTML(form)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(html, "recaptcha") {
		t.Error("captcha script should be injected")
	}
}

func TestRenderFormHTML_PrivacyPolicyURL(t *testing.T) {
	p := newPlugin(NewFakeHost())
	form := map[string]any{
		"id":     float64(1),
		"name":   "Test",
		"fields": []any{},
		"layout": `<form><a href="{privacy_policy_url}">Privacy</a></form>`,
		"settings": map[string]any{
			"privacy_policy_url": "https://example.com/privacy",
		},
	}
	html, err := p.renderFormHTML(form)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(html, "https://example.com/privacy") {
		t.Errorf("privacy policy URL should be substituted, got: %s", html)
	}
}

func TestRenderFormHTML_FormCSSClass(t *testing.T) {
	p := newPlugin(NewFakeHost())
	form := map[string]any{
		"id":     float64(1),
		"name":   "Test",
		"fields": []any{},
		"layout": `<form></form>`,
		"settings": map[string]any{
			"form_css_class": "my-custom-class",
		},
	}
	html, err := p.renderFormHTML(form)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(html, "my-custom-class") {
		t.Errorf("form CSS class should be injected, got: %s", html)
	}
}

// ---- defaultLayoutForStyle ----

func TestDefaultLayoutForStyle(t *testing.T) {
	cases := []struct {
		style    string
		expected string
	}{
		{"grid", defaultLayoutGrid},
		{"card", defaultLayoutCard},
		{"inline", defaultLayoutInline},
		{"simple", defaultLayoutSimple},
		{"", defaultLayoutSimple},
		{"unknown", defaultLayoutSimple},
	}
	for _, tc := range cases {
		got := defaultLayoutForStyle(tc.style)
		if got != tc.expected {
			t.Errorf("style=%q: returned wrong layout", tc.style)
		}
	}
}

// ---- normalizeFieldOptions ----

func TestNormalizeFieldOptions_StringOptions(t *testing.T) {
	field := map[string]any{
		"id":      "color",
		"options": []any{"red", "blue"},
	}
	normalizeFieldOptions(field)
	opts, ok := field["options"].([]any)
	if !ok {
		t.Fatal("options should be []any")
	}
	if len(opts) != 2 {
		t.Fatalf("expected 2 options, got %d", len(opts))
	}
	first := opts[0].(map[string]any)
	if first["label"] != "red" || first["value"] != "red" {
		t.Errorf("string option not normalized: %v", first)
	}
}

func TestNormalizeFieldOptions_MapOptions(t *testing.T) {
	field := map[string]any{
		"id": "color",
		"options": []any{
			map[string]any{"label": "Red", "value": "red"},
		},
	}
	normalizeFieldOptions(field)
	opts := field["options"].([]any)
	first := opts[0].(map[string]any)
	if first["label"] != "Red" || first["value"] != "red" {
		t.Errorf("map option should be preserved: %v", first)
	}
}

func TestNormalizeFieldOptions_MissingLabelValue(t *testing.T) {
	field := map[string]any{
		"id": "x",
		"options": []any{
			map[string]any{"label": "Only Label"},
		},
	}
	normalizeFieldOptions(field)
	opts := field["options"].([]any)
	first := opts[0].(map[string]any)
	if first["value"] != "" {
		t.Errorf("missing value should default to empty string, got %v", first["value"])
	}
}

// ---- applyFormPostProcessing ----

func TestApplyFormPostProcessing_EmptySettings(t *testing.T) {
	result := applyFormPostProcessing(`<form></form>`, map[string]any{})
	// Honeypot should be injected by default
	if !strings.Contains(result, "website_url") {
		t.Error("honeypot should be present with empty settings")
	}
}

func TestApplyFormPostProcessing_PrivacyURLEmpty(t *testing.T) {
	html := `<form><a href="{privacy_policy_url}">Privacy</a></form>`
	result := applyFormPostProcessing(html, map[string]any{"privacy_policy_url": ""})
	if strings.Contains(result, `href=""`) {
		t.Error("empty privacy_policy_url should remove the href=")
	}
}
