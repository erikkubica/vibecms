package main

import (
	"testing"
)

// ---------- evaluateCondition ----------

func TestEvaluateCondition_Equals(t *testing.T) {
	data := map[string]any{"color": "blue"}

	cases := []struct {
		name     string
		expected any
		want     bool
	}{
		{"match", "blue", true},
		{"no match", "red", false},
		{"empty expected", "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := map[string]any{"field": "color", "operator": "equals", "value": tc.expected}
			if got := evaluateCondition(c, data); got != tc.want {
				t.Errorf("equals(%q) = %v, want %v", tc.expected, got, tc.want)
			}
		})
	}
}

func TestEvaluateCondition_NotEquals(t *testing.T) {
	data := map[string]any{"x": "foo"}
	c := map[string]any{"field": "x", "operator": "not_equals", "value": "bar"}
	if !evaluateCondition(c, data) {
		t.Error("not_equals: foo != bar should be true")
	}
	c2 := map[string]any{"field": "x", "operator": "not_equals", "value": "foo"}
	if evaluateCondition(c2, data) {
		t.Error("not_equals: foo != foo should be false")
	}
}

func TestEvaluateCondition_Contains(t *testing.T) {
	data := map[string]any{"msg": "Hello World"}
	cases := []struct {
		val  string
		want bool
	}{
		{"hello", true},
		{"WORLD", true},
		{"xyz", false},
	}
	for _, tc := range cases {
		c := map[string]any{"field": "msg", "operator": "contains", "value": tc.val}
		if got := evaluateCondition(c, data); got != tc.want {
			t.Errorf("contains(%q) = %v, want %v", tc.val, got, tc.want)
		}
	}
}

func TestEvaluateCondition_NotContains(t *testing.T) {
	data := map[string]any{"msg": "Hello World"}
	c := map[string]any{"field": "msg", "operator": "not_contains", "value": "xyz"}
	if !evaluateCondition(c, data) {
		t.Error("not_contains xyz in Hello World should be true")
	}
	c2 := map[string]any{"field": "msg", "operator": "not_contains", "value": "hello"}
	if evaluateCondition(c2, data) {
		t.Error("not_contains hello in Hello World should be false (case-insensitive)")
	}
}

func TestEvaluateCondition_NumericComparisons(t *testing.T) {
	data := map[string]any{"age": float64(25)}

	cases := []struct {
		op   string
		val  float64
		want bool
	}{
		{"gt", 20, true},
		{"gt", 25, false},
		{"gte", 25, true},
		{"gte", 26, false},
		{"lt", 30, true},
		{"lt", 25, false},
		{"lte", 25, true},
		{"lte", 24, false},
	}
	for _, tc := range cases {
		c := map[string]any{"field": "age", "operator": tc.op, "value": tc.val}
		if got := evaluateCondition(c, data); got != tc.want {
			t.Errorf("op=%s val=%v: got %v, want %v", tc.op, tc.val, got, tc.want)
		}
	}
}

func TestEvaluateCondition_In(t *testing.T) {
	data := map[string]any{"color": "blue"}
	c := map[string]any{"field": "color", "operator": "in", "value": []any{"red", "blue", "green"}}
	if !evaluateCondition(c, data) {
		t.Error("in: blue should be in list")
	}
	c2 := map[string]any{"field": "color", "operator": "in", "value": []any{"red", "green"}}
	if evaluateCondition(c2, data) {
		t.Error("in: blue should NOT be in [red, green]")
	}
	// non-slice value → false
	c3 := map[string]any{"field": "color", "operator": "in", "value": "blue"}
	if evaluateCondition(c3, data) {
		t.Error("in: non-slice value should return false")
	}
}

func TestEvaluateCondition_NotIn(t *testing.T) {
	data := map[string]any{"color": "blue"}
	c := map[string]any{"field": "color", "operator": "not_in", "value": []any{"red", "green"}}
	if !evaluateCondition(c, data) {
		t.Error("not_in: blue not in [red,green] should be true")
	}
	c2 := map[string]any{"field": "color", "operator": "not_in", "value": []any{"blue", "green"}}
	if evaluateCondition(c2, data) {
		t.Error("not_in: blue in [blue,green] should be false")
	}
}

func TestEvaluateCondition_IsEmpty(t *testing.T) {
	cases := []struct {
		name  string
		data  map[string]any
		field string
		want  bool
	}{
		{"missing field", map[string]any{}, "x", true},
		{"empty string", map[string]any{"x": ""}, "x", true},
		{"whitespace", map[string]any{"x": "  "}, "x", true},
		{"non-empty string", map[string]any{"x": "hello"}, "x", false},
		{"false bool", map[string]any{"x": false}, "x", true},
		{"true bool", map[string]any{"x": true}, "x", false},
		{"empty slice", map[string]any{"x": []any{}}, "x", true},
		{"non-empty slice", map[string]any{"x": []any{1}}, "x", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := map[string]any{"field": tc.field, "operator": "is_empty"}
			if got := evaluateCondition(c, tc.data); got != tc.want {
				t.Errorf("is_empty(%q) = %v, want %v", tc.name, got, tc.want)
			}
		})
	}
}

func TestEvaluateCondition_IsNotEmpty(t *testing.T) {
	c := map[string]any{"field": "x", "operator": "is_not_empty"}
	if evaluateCondition(c, map[string]any{}) {
		t.Error("is_not_empty: missing field should be false")
	}
	if evaluateCondition(c, map[string]any{"x": ""}) {
		t.Error("is_not_empty: empty string should be false")
	}
	if !evaluateCondition(c, map[string]any{"x": "hello"}) {
		t.Error("is_not_empty: non-empty string should be true")
	}
}

func TestEvaluateCondition_Matches(t *testing.T) {
	data := map[string]any{"phone": "+1-555-000-1234"}
	c := map[string]any{"field": "phone", "operator": "matches", "value": `^\+[0-9\-]+$`}
	if !evaluateCondition(c, data) {
		t.Error("matches: should match phone pattern")
	}
	c2 := map[string]any{"field": "phone", "operator": "matches", "value": `^[a-z]+$`}
	if evaluateCondition(c2, data) {
		t.Error("matches: should not match alpha-only pattern")
	}
	// Invalid regex → false
	c3 := map[string]any{"field": "phone", "operator": "matches", "value": `[invalid(`}
	if evaluateCondition(c3, data) {
		t.Error("matches: invalid regex should return false")
	}
}

func TestEvaluateCondition_UnknownOperator(t *testing.T) {
	data := map[string]any{"x": "val"}
	c := map[string]any{"field": "x", "operator": "nonexistent", "value": "val"}
	if evaluateCondition(c, data) {
		t.Error("unknown operator should return false")
	}
}

// ---------- EvaluateGroup ----------

func TestEvaluateGroup_Nil(t *testing.T) {
	if !EvaluateGroup(nil, map[string]any{}) {
		t.Error("nil group should return true")
	}
	if !EvaluateGroup(map[string]any{}, map[string]any{}) {
		t.Error("empty group should return true")
	}
}

func TestEvaluateGroup_All(t *testing.T) {
	data := map[string]any{"a": "1", "b": "2"}
	group := map[string]any{
		"all": []any{
			map[string]any{"field": "a", "operator": "equals", "value": "1"},
			map[string]any{"field": "b", "operator": "equals", "value": "2"},
		},
	}
	if !EvaluateGroup(group, data) {
		t.Error("all conditions true → group should be true")
	}
	data2 := map[string]any{"a": "1", "b": "X"}
	if EvaluateGroup(group, data2) {
		t.Error("one condition false → group should be false")
	}
}

func TestEvaluateGroup_Any(t *testing.T) {
	data := map[string]any{"a": "1"}
	group := map[string]any{
		"any": []any{
			map[string]any{"field": "a", "operator": "equals", "value": "1"},
			map[string]any{"field": "a", "operator": "equals", "value": "2"},
		},
	}
	if !EvaluateGroup(group, data) {
		t.Error("at least one true → group should be true")
	}
	data2 := map[string]any{"a": "3"}
	if EvaluateGroup(group, data2) {
		t.Error("none matching → group should be false")
	}
}

func TestEvaluateGroup_Nested3Levels(t *testing.T) {
	// (subject==Sales OR subject==Other) AND country!=US
	data1 := map[string]any{"subject": "Sales", "country": "CA"}
	data2 := map[string]any{"subject": "Other", "country": "US"} // fails country
	data3 := map[string]any{"subject": "Support", "country": "CA"} // fails subject

	group := map[string]any{
		"all": []any{
			map[string]any{
				"any": []any{
					map[string]any{"field": "subject", "operator": "equals", "value": "Sales"},
					map[string]any{"field": "subject", "operator": "equals", "value": "Other"},
				},
			},
			map[string]any{"field": "country", "operator": "not_equals", "value": "US"},
		},
	}

	if !EvaluateGroup(group, data1) {
		t.Error("Sales+CA should pass")
	}
	if EvaluateGroup(group, data2) {
		t.Error("Other+US should fail (country==US)")
	}
	if EvaluateGroup(group, data3) {
		t.Error("Support+CA should fail (subject not in Sales/Other)")
	}
}

func TestEvaluateGroup_BothAllAndAny(t *testing.T) {
	// group has both "all" and "any" — both must pass
	data := map[string]any{"x": "1", "y": "2"}
	group := map[string]any{
		"all": []any{
			map[string]any{"field": "x", "operator": "equals", "value": "1"},
		},
		"any": []any{
			map[string]any{"field": "y", "operator": "equals", "value": "2"},
			map[string]any{"field": "y", "operator": "equals", "value": "3"},
		},
	}
	if !EvaluateGroup(group, data) {
		t.Error("both all and any pass → true")
	}
	data2 := map[string]any{"x": "1", "y": "9"}
	if EvaluateGroup(group, data2) {
		t.Error("any fails → false")
	}
}

// ---------- EvaluateField ----------

func TestEvaluateField_NoDisplayWhen(t *testing.T) {
	field := map[string]any{"id": "name", "type": "text"}
	if !EvaluateField(field, map[string]any{}) {
		t.Error("field without display_when should always be visible")
	}
}

func TestEvaluateField_WithDisplayWhen(t *testing.T) {
	field := map[string]any{
		"id":   "extra",
		"type": "text",
		"display_when": map[string]any{
			"all": []any{
				map[string]any{"field": "subject", "operator": "equals", "value": "Sales"},
			},
		},
	}
	if EvaluateField(field, map[string]any{"subject": "Support"}) {
		t.Error("display_when false → field should be hidden")
	}
	if !EvaluateField(field, map[string]any{"subject": "Sales"}) {
		t.Error("display_when true → field should be visible")
	}
}

func TestEvaluateField_JSONStringDisplayWhen(t *testing.T) {
	// display_when stored as JSON string (from JSONB)
	field := map[string]any{
		"id":           "extra",
		"display_when": `{"all":[{"field":"show","operator":"equals","value":"yes"}]}`,
	}
	if EvaluateField(field, map[string]any{"show": "no"}) {
		t.Error("display_when JSON string false → hidden")
	}
	if !EvaluateField(field, map[string]any{"show": "yes"}) {
		t.Error("display_when JSON string true → visible")
	}
}

// ---------- isCondEmpty ----------

func TestIsCondEmpty(t *testing.T) {
	cases := []struct {
		v    any
		want bool
	}{
		{nil, true},
		{"", true},
		{"   ", true},
		{"x", false},
		{false, true},
		{true, false},
		{[]any{}, true},
		{[]any{1}, false},
		{map[string]any{}, true},
		{map[string]any{"k": "v"}, false},
	}
	for _, tc := range cases {
		if got := isCondEmpty(tc.v); got != tc.want {
			t.Errorf("isCondEmpty(%T %v) = %v, want %v", tc.v, tc.v, got, tc.want)
		}
	}
}

// ---------- condToFloat ----------

func TestCondToFloat(t *testing.T) {
	cases := []struct {
		v    any
		want float64
	}{
		{float64(3.14), 3.14},
		{int(5), 5.0},
		{"2.5", 2.5},
		{"abc", 0},
		{nil, 0},
	}
	for _, tc := range cases {
		if got := condToFloat(tc.v); got != tc.want {
			t.Errorf("condToFloat(%v) = %v, want %v", tc.v, got, tc.want)
		}
	}
}
