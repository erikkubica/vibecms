package main

import (
	"testing"
)

func TestValidateSubmission_Required(t *testing.T) {
	fields := []map[string]any{
		{"id": "name", "label": "Name", "type": "text", "required": true},
	}

	t.Run("missing required", func(t *testing.T) {
		errs := validateSubmission(map[string]any{}, fields, nil)
		if errs["name"] != "Name is required" {
			t.Errorf("got %q, want 'Name is required'", errs["name"])
		}
	})

	t.Run("empty string required", func(t *testing.T) {
		errs := validateSubmission(map[string]any{"name": ""}, fields, nil)
		if errs["name"] != "Name is required" {
			t.Errorf("expected required error, got %q", errs["name"])
		}
	})

	t.Run("whitespace only required", func(t *testing.T) {
		errs := validateSubmission(map[string]any{"name": "   "}, fields, nil)
		if errs["name"] != "Name is required" {
			t.Errorf("expected required error, got %q", errs["name"])
		}
	})

	t.Run("present required", func(t *testing.T) {
		errs := validateSubmission(map[string]any{"name": "Erik"}, fields, nil)
		if errs["name"] != "" {
			t.Errorf("expected no error, got %q", errs["name"])
		}
	})
}

func TestValidateSubmission_EmailFormat(t *testing.T) {
	fields := []map[string]any{
		{"id": "email", "label": "Email", "type": "email"},
	}

	cases := []struct {
		input   string
		wantErr bool
	}{
		{"user@example.com", false},
		{"not-an-email", true},
		{"@missing.local", true},
		{"", false}, // empty non-required email is fine
	}
	for _, tc := range cases {
		errs := validateSubmission(map[string]any{"email": tc.input}, fields, nil)
		if tc.wantErr && errs["email"] == "" {
			t.Errorf("email=%q: expected error, got none", tc.input)
		}
		if !tc.wantErr && errs["email"] != "" {
			t.Errorf("email=%q: expected no error, got %q", tc.input, errs["email"])
		}
	}
}

func TestValidateSubmission_URLFormat(t *testing.T) {
	fields := []map[string]any{
		{"id": "website", "label": "Website", "type": "url"},
	}

	cases := []struct {
		input   string
		wantErr bool
	}{
		{"https://example.com", false},
		{"http://foo.bar/path?q=1", false},
		{"not a url", true},
		{"", false}, // empty non-required
	}
	for _, tc := range cases {
		errs := validateSubmission(map[string]any{"website": tc.input}, fields, nil)
		if tc.wantErr && errs["website"] == "" {
			t.Errorf("url=%q: expected error, got none", tc.input)
		}
		if !tc.wantErr && errs["website"] != "" {
			t.Errorf("url=%q: expected no error, got %q", tc.input, errs["website"])
		}
	}
}

func TestValidateSubmission_MinMaxLength(t *testing.T) {
	fields := []map[string]any{
		{"id": "bio", "label": "Bio", "type": "text", "min_length": 5, "max_length": 10},
	}

	cases := []struct {
		input   string
		wantErr bool
	}{
		{"hi", true},           // too short
		{"hello", false},       // exactly 5
		{"0123456789", false},  // exactly 10
		{"01234567890", true},  // 11 chars, too long
		// Note: empty non-required field still triggers min_length if value is present (even empty string)
		// because checkLength runs on all present string values regardless of required
	}
	for _, tc := range cases {
		errs := validateSubmission(map[string]any{"bio": tc.input}, fields, nil)
		if tc.wantErr && errs["bio"] == "" {
			t.Errorf("bio=%q: expected error, got none", tc.input)
		}
		if !tc.wantErr && errs["bio"] != "" {
			t.Errorf("bio=%q: expected no error, got %q", tc.input, errs["bio"])
		}
	}

	// Empty string not submitted (field absent) → no error
	errs := validateSubmission(map[string]any{}, fields, nil)
	if errs["bio"] != "" {
		t.Errorf("absent field should not trigger min_length, got %q", errs["bio"])
	}
}

func TestValidateSubmission_Pattern(t *testing.T) {
	fields := []map[string]any{
		{"id": "zip", "label": "Zip", "type": "text", "pattern": `^\d{5}$`},
	}

	cases := []struct {
		input   string
		wantErr bool
	}{
		{"12345", false},
		{"1234", true},
		{"abcde", true},
		{"", false}, // empty skips pattern
	}
	for _, tc := range cases {
		errs := validateSubmission(map[string]any{"zip": tc.input}, fields, nil)
		if tc.wantErr && errs["zip"] == "" {
			t.Errorf("zip=%q: expected pattern error, got none", tc.input)
		}
		if !tc.wantErr && errs["zip"] != "" {
			t.Errorf("zip=%q: expected no error, got %q", tc.input, errs["zip"])
		}
	}
}

func TestValidateSubmission_InvalidPatternSilentSkip(t *testing.T) {
	fields := []map[string]any{
		{"id": "f", "label": "F", "type": "text", "pattern": `[invalid(`},
	}
	errs := validateSubmission(map[string]any{"f": "anything"}, fields, nil)
	if errs["f"] != "" {
		t.Errorf("invalid pattern should be silently skipped, got %q", errs["f"])
	}
}

func TestValidateSubmission_Numeric(t *testing.T) {
	fields := []map[string]any{
		{"id": "age", "label": "Age", "type": "number", "min": float64(18), "max": float64(99)},
	}

	cases := []struct {
		input   any
		wantErr bool
	}{
		{float64(18), false},
		{float64(99), false},
		{float64(17), true},
		{float64(100), true},
		{"25", false},   // string numeric
		{"abc", false},  // unparseable → no error (skip)
	}
	for _, tc := range cases {
		errs := validateSubmission(map[string]any{"age": tc.input}, fields, nil)
		if tc.wantErr && errs["age"] == "" {
			t.Errorf("age=%v: expected error, got none", tc.input)
		}
		if !tc.wantErr && errs["age"] != "" {
			t.Errorf("age=%v: expected no error, got %q", tc.input, errs["age"])
		}
	}
}

func TestValidateSubmission_NumericStep(t *testing.T) {
	fields := []map[string]any{
		{"id": "qty", "label": "Qty", "type": "number", "step": float64(5)},
	}
	cases := []struct {
		input   any
		wantErr bool
	}{
		{float64(10), false},
		{float64(15), false},
		{float64(7), true},
	}
	for _, tc := range cases {
		errs := validateSubmission(map[string]any{"qty": tc.input}, fields, nil)
		if tc.wantErr && errs["qty"] == "" {
			t.Errorf("qty=%v: expected step error, got none", tc.input)
		}
		if !tc.wantErr && errs["qty"] != "" {
			t.Errorf("qty=%v: expected no error, got %q", tc.input, errs["qty"])
		}
	}
}

func TestValidateSubmission_GDPRConsent(t *testing.T) {
	fields := []map[string]any{
		{"id": "gdpr", "label": "GDPR", "type": "gdpr_consent", "required": true},
	}

	t.Run("not checked required", func(t *testing.T) {
		errs := validateSubmission(map[string]any{"gdpr": false}, fields, nil)
		if errs["gdpr"] == "" {
			t.Error("unchecked required GDPR should produce error")
		}
	})

	t.Run("checked", func(t *testing.T) {
		errs := validateSubmission(map[string]any{"gdpr": true}, fields, nil)
		if errs["gdpr"] != "" {
			t.Errorf("checked GDPR should have no error, got %q", errs["gdpr"])
		}
	})
}

func TestValidateSubmission_FileRequired(t *testing.T) {
	fields := []map[string]any{
		{"id": "doc", "label": "Document", "type": "file", "required": true},
	}

	t.Run("no file uploaded", func(t *testing.T) {
		errs := validateSubmission(map[string]any{}, fields, nil)
		if errs["doc"] == "" {
			t.Error("required file missing should produce error")
		}
	})

	t.Run("file provided", func(t *testing.T) {
		files := map[string][]uploadedFile{
			"doc": {{FieldName: "doc", FileName: "test.pdf", MimeType: "application/pdf", Size: 100, Body: []byte("x")}},
		}
		errs := validateSubmission(map[string]any{}, fields, files)
		if errs["doc"] != "" {
			t.Errorf("file uploaded should have no error, got %q", errs["doc"])
		}
	})
}

func TestValidateSubmission_FileMaxFiles(t *testing.T) {
	fields := []map[string]any{
		{"id": "docs", "label": "Docs", "type": "file", "multiple": true, "max_files": 2},
	}
	files := map[string][]uploadedFile{
		"docs": {
			{FileName: "a.pdf", Size: 100, Body: []byte("a")},
			{FileName: "b.pdf", Size: 100, Body: []byte("b")},
			{FileName: "c.pdf", Size: 100, Body: []byte("c")},
		},
	}
	// Note: the file field max_files validation requires the field value to exist in data
	// (the "if !exists { continue }" guard fires before the file type switch).
	// To trigger the check, the field must also be present in data.
	data := map[string]any{"docs": nil} // mark as present so validation runs
	errs := validateSubmission(data, fields, files)
	if errs["docs"] == "" {
		t.Error("exceeding max_files should produce error")
	}
}

func TestValidateSubmission_DisplayWhenSkip(t *testing.T) {
	// Field with display_when that evaluates to false should be skipped
	fields := []map[string]any{
		{
			"id": "subject", "label": "Subject", "type": "text",
		},
		{
			"id": "extra", "label": "Extra", "type": "text", "required": true,
			"display_when": map[string]any{
				"all": []any{
					map[string]any{"field": "subject", "operator": "equals", "value": "Sales"},
				},
			},
		},
	}

	t.Run("hidden field not validated", func(t *testing.T) {
		data := map[string]any{"subject": "Support"} // display_when false
		errs := validateSubmission(data, fields, nil)
		if errs["extra"] != "" {
			t.Error("hidden field (display_when=false) should not be validated")
		}
	})

	t.Run("visible field validated", func(t *testing.T) {
		data := map[string]any{"subject": "Sales"} // display_when true
		errs := validateSubmission(data, fields, nil)
		if errs["extra"] == "" {
			t.Error("visible required field should be validated")
		}
	})
}

func TestValidateSubmission_MultipleErrors(t *testing.T) {
	fields := []map[string]any{
		{"id": "name", "label": "Name", "type": "text", "required": true},
		{"id": "email", "label": "Email", "type": "email"},
	}
	data := map[string]any{"email": "not-valid"}
	errs := validateSubmission(data, fields, nil)
	if errs["name"] == "" {
		t.Error("missing name should produce error")
	}
	if errs["email"] == "" {
		t.Error("invalid email should produce error")
	}
}

func TestValidateSubmission_FieldWithNoID(t *testing.T) {
	fields := []map[string]any{
		{"label": "No ID", "type": "text", "required": true}, // no id
		{"id": "name", "label": "Name", "type": "text"},
	}
	// Should not panic, field without id is skipped
	errs := validateSubmission(map[string]any{"name": "Erik"}, fields, nil)
	if len(errs) != 0 {
		t.Errorf("expected no errors, got %v", errs)
	}
}

// ---- checkLength ----

func TestCheckLength(t *testing.T) {
	field := map[string]any{"min_length": 3, "max_length": 6}
	if msg := checkLength(field, "ab"); msg == "" {
		t.Error("too short should produce error")
	}
	if msg := checkLength(field, "1234567"); msg == "" {
		t.Error("too long should produce error")
	}
	if msg := checkLength(field, "abc"); msg != "" {
		t.Errorf("3 chars with min=3 should pass, got %q", msg)
	}
	// Non-string value is skipped
	if msg := checkLength(field, 42); msg != "" {
		t.Errorf("non-string should be skipped, got %q", msg)
	}
}

// ---- intField / floatField ----

func TestIntField(t *testing.T) {
	f := map[string]any{"n": float64(5), "m": 3, "s": "x"}
	if v := intField(f, "n"); v != 5 {
		t.Errorf("float64 key: got %d, want 5", v)
	}
	if v := intField(f, "m"); v != 3 {
		t.Errorf("int key: got %d, want 3", v)
	}
	if v := intField(f, "missing"); v != 0 {
		t.Errorf("missing key: got %d, want 0", v)
	}
	if v := intField(f, "s"); v != 0 {
		t.Errorf("string key: got %d, want 0", v)
	}
}

func TestFloatField(t *testing.T) {
	f := map[string]any{"a": float64(3.14), "b": 2}
	if v, ok := floatField(f, "a"); !ok || v != 3.14 {
		t.Errorf("float64: got %v %v", v, ok)
	}
	if v, ok := floatField(f, "b"); !ok || v != 2.0 {
		t.Errorf("int: got %v %v", v, ok)
	}
	if _, ok := floatField(f, "missing"); ok {
		t.Error("missing key should return false")
	}
}

// ---- isEmpty ----

func TestIsEmpty(t *testing.T) {
	cases := []struct {
		v    any
		want bool
	}{
		{nil, true},
		{"", true},
		{"  ", true},
		{"x", false},
		{0, false},
	}
	for _, tc := range cases {
		if got := isEmpty(tc.v); got != tc.want {
			t.Errorf("isEmpty(%v) = %v, want %v", tc.v, got, tc.want)
		}
	}
}
