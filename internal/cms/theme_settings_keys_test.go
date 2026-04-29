// Tests for the pure-string helpers in theme_settings_keys.go.
//
// The DB-cleanup function DeleteThemeSettings is intentionally NOT covered
// by a unit test here: the project has no GORM test-DB harness, and the only
// in-process option (gorm.io/driver/sqlite) drags in mattn/go-sqlite3 which
// requires CGO and breaks the Alpine Docker build. Since DeleteThemeSettings
// is a one-line LIKE query against PostgreSQL, it is exercised end-to-end by
// the theme deactivation E2E test in Task 11 instead.
package cms

import (
	"testing"
)

func TestSettingKey_Format(t *testing.T) {
	got := SettingKey("hello-vietnam", "header", "logo")
	want := "theme:hello-vietnam:header:logo"
	if got != want {
		t.Fatalf("SettingKey: got %q, want %q", got, want)
	}
}

func TestThemePrefix(t *testing.T) {
	got := ThemePrefix("hello-vietnam")
	want := "theme:hello-vietnam:"
	if got != want {
		t.Fatalf("ThemePrefix: got %q, want %q", got, want)
	}
}

func TestParseSettingKey_Roundtrip(t *testing.T) {
	a, b, c, ok := ParseSettingKey("theme:a:b:c")
	if !ok {
		t.Fatalf("expected ok=true")
	}
	if a != "a" || b != "b" || c != "c" {
		t.Fatalf("got (%q,%q,%q), want (a,b,c)", a, b, c)
	}
}

func TestParseSettingKey_RejectsNonTheme(t *testing.T) {
	if _, _, _, ok := ParseSettingKey("homepage_node_id"); ok {
		t.Fatalf("expected ok=false for non-theme key")
	}
}

func TestParseSettingKey_RejectsMalformed(t *testing.T) {
	cases := []string{
		"theme:",
		"theme:a",
		"theme:a:",
		"theme:a:b",
		"theme::field",
		"theme:slug::field",
	}
	for _, k := range cases {
		t.Run(k, func(t *testing.T) {
			if _, _, _, ok := ParseSettingKey(k); ok {
				t.Fatalf("expected ok=false for %q", k)
			}
		})
	}
}

func TestParseSettingKey_FieldKeyContainingColon(t *testing.T) {
	a, b, c, ok := ParseSettingKey("theme:a:b:foo:bar")
	if !ok {
		t.Fatalf("expected ok=true")
	}
	if a != "a" || b != "b" || c != "foo:bar" {
		t.Fatalf("got (%q,%q,%q), want (a,b,foo:bar)", a, b, c)
	}
}
