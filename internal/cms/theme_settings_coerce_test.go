package cms

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestCoerceValue(t *testing.T) {
	cases := []struct {
		name      string
		fieldType string
		raw       string
		wantValue any
		wantOk    bool
	}{
		{"empty raw / text", "text", "", nil, true},
		{"empty raw / number", "number", "", nil, true},
		{"empty raw / repeater", "repeater", "", nil, true},
		{"text passes through", "text", "hello", "hello", true},
		{"textarea passes through", "textarea", "line1\nline2", "line1\nline2", true},
		{"richtext passes through", "richtext", "<p>hi</p>", "<p>hi</p>", true},
		{"email passes through", "email", "a@b", "a@b", true},
		{"url passes through", "url", "https://x", "https://x", true},
		{"date passes through", "date", "2026-01-01", "2026-01-01", true},
		{"color passes through", "color", "#abcdef", "#abcdef", true},
		{"select passes through", "select", "one", "one", true},
		{"radio passes through", "radio", "two", "two", true},
		{"number ok int", "number", "42", float64(42), true},
		{"number ok decimal", "number", "3.14", float64(3.14), true},
		{"number ok negative", "number", "-5", float64(-5), true},
		{"number garbage", "number", "abc", nil, false},
		{"range ok", "range", "0.5", float64(0.5), true},
		{"range garbage", "range", "huge", nil, false},
		{"toggle true", "toggle", "true", true, true},
		{"toggle false", "toggle", "false", false, true},
		{"toggle 1", "toggle", "1", true, true},
		{"toggle 0", "toggle", "0", false, true},
		{"toggle bad", "toggle", "yesplease", nil, false},
		{"toggle uppercase rejected", "toggle", "TRUE", nil, false},
		{"checkbox ok array", "checkbox", `["a","b"]`, []any{"a", "b"}, true},
		{"checkbox bad", "checkbox", "not-json", nil, false},
		{"image ok object", "image", `{"id":1,"url":"x"}`, map[string]any{"id": float64(1), "url": "x"}, true},
		{"image bad", "image", "broken", nil, false},
		{"gallery ok", "gallery", `[1,2]`, []any{float64(1), float64(2)}, true},
		{"link ok", "link", `{"url":"/x"}`, map[string]any{"url": "/x"}, true},
		{"group ok", "group", `{"a":1}`, map[string]any{"a": float64(1)}, true},
		{"repeater ok", "repeater", `[{"x":1}]`, []any{map[string]any{"x": float64(1)}}, true},
		{"node ok", "node", `123`, float64(123), true},
		{"term ok", "term", `"slug"`, "slug", true},
		{"extension type still tries JSON", "custom-x", `{"k":"v"}`, map[string]any{"k": "v"}, true},
		{"extension type bad JSON", "custom-x", `not json`, nil, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotValue, gotOk := CoerceValue(tc.fieldType, tc.raw)
			if gotOk != tc.wantOk {
				t.Fatalf("ok mismatch: got %v want %v", gotOk, tc.wantOk)
			}
			if !reflect.DeepEqual(gotValue, tc.wantValue) {
				t.Fatalf("value mismatch:\n  got:  %#v\n  want: %#v", gotValue, tc.wantValue)
			}
		})
	}
}

func TestCoerceWithDefault_FallsBackOnMismatch(t *testing.T) {
	field := ThemeSettingsField{Type: "number", Default: json.RawMessage(`7`)}
	got := CoerceWithDefault(field, "abc")
	want := CoerceResult{Value: float64(7), Compatible: false, Raw: "abc"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v want %#v", got, want)
	}
}

func TestCoerceWithDefault_NoDefaultGivesNilValue(t *testing.T) {
	field := ThemeSettingsField{Type: "number", Default: nil}
	got := CoerceWithDefault(field, "abc")
	want := CoerceResult{Value: nil, Compatible: false, Raw: "abc"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v want %#v", got, want)
	}
}

func TestCoerceWithDefault_CompatibleKeepsRaw(t *testing.T) {
	field := ThemeSettingsField{Type: "text", Default: json.RawMessage(`"d"`)}
	got := CoerceWithDefault(field, "hello")
	want := CoerceResult{Value: "hello", Compatible: true, Raw: "hello"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v want %#v", got, want)
	}
}

func TestCoerceWithDefault_EmptyRawCompatibleNoDefault(t *testing.T) {
	field := ThemeSettingsField{Type: "text"}
	got := CoerceWithDefault(field, "")
	want := CoerceResult{Value: nil, Compatible: true, Raw: ""}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v want %#v", got, want)
	}
}

func TestCoerceWithDefault_DefaultJsonObject(t *testing.T) {
	field := ThemeSettingsField{Type: "image", Default: json.RawMessage(`{"id":1}`)}
	got := CoerceWithDefault(field, "broken")
	if got.Compatible {
		t.Fatalf("expected Compatible=false")
	}
	wantValue := map[string]any{"id": float64(1)}
	if !reflect.DeepEqual(got.Value, wantValue) {
		t.Fatalf("value mismatch:\n  got:  %#v\n  want: %#v", got.Value, wantValue)
	}
	if got.Raw != "broken" {
		t.Fatalf("raw mismatch: %q", got.Raw)
	}
}
