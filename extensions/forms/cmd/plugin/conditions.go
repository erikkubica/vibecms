package main

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// EvaluateGroup returns true if the condition group matches the given submission data.
// An empty or nil group always returns true (no restriction).
// Supports recursive nesting: items may be conditions or sub-groups.
func EvaluateGroup(group map[string]any, data map[string]any) bool {
	if group == nil || len(group) == 0 {
		return true
	}

	all, hasAll := group["all"].([]any)
	any_, hasAny := group["any"].([]any)

	if !hasAll && !hasAny {
		return true
	}

	if hasAll {
		for _, item := range all {
			m, ok := item.(map[string]any)
			if !ok {
				return false
			}
			if !evaluateItem(m, data) {
				return false
			}
		}
	}

	if hasAny {
		matched := false
		for _, item := range any_ {
			m, ok := item.(map[string]any)
			if !ok {
				continue
			}
			if evaluateItem(m, data) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	return true
}

// EvaluateField returns true if the field's display_when condition evaluates to true
// for the given submission data. If the field has no display_when, returns true.
func EvaluateField(field map[string]any, data map[string]any) bool {
	dw := parseJSONMap(field["display_when"])
	return EvaluateGroup(dw, data)
}

// evaluateItem dispatches to EvaluateGroup (nested group) or evaluateCondition (leaf condition).
func evaluateItem(m map[string]any, data map[string]any) bool {
	if _, hasAll := m["all"]; hasAll {
		return EvaluateGroup(m, data)
	}
	if _, hasAny := m["any"]; hasAny {
		return EvaluateGroup(m, data)
	}
	return evaluateCondition(m, data)
}

// evaluateCondition evaluates a single condition against submission data.
// Supports operators: equals, not_equals, contains, not_contains,
// gt, gte, lt, lte, in, not_in, matches, is_empty, is_not_empty.
func evaluateCondition(c map[string]any, data map[string]any) bool {
	field, _ := c["field"].(string)
	op, _ := c["operator"].(string)
	expected := c["value"]
	actual, exists := data[field]

	switch op {
	case "is_empty":
		return !exists || isCondEmpty(actual)
	case "is_not_empty":
		return exists && !isCondEmpty(actual)
	case "equals":
		return fmt.Sprintf("%v", actual) == fmt.Sprintf("%v", expected)
	case "not_equals":
		return fmt.Sprintf("%v", actual) != fmt.Sprintf("%v", expected)
	case "contains":
		return strings.Contains(
			strings.ToLower(fmt.Sprintf("%v", actual)),
			strings.ToLower(fmt.Sprintf("%v", expected)),
		)
	case "not_contains":
		return !strings.Contains(
			strings.ToLower(fmt.Sprintf("%v", actual)),
			strings.ToLower(fmt.Sprintf("%v", expected)),
		)
	case "gt":
		return condToFloat(actual) > condToFloat(expected)
	case "gte":
		return condToFloat(actual) >= condToFloat(expected)
	case "lt":
		return condToFloat(actual) < condToFloat(expected)
	case "lte":
		return condToFloat(actual) <= condToFloat(expected)
	case "in":
		list, ok := expected.([]any)
		if !ok {
			return false
		}
		for _, v := range list {
			if fmt.Sprintf("%v", v) == fmt.Sprintf("%v", actual) {
				return true
			}
		}
		return false
	case "not_in":
		list, ok := expected.([]any)
		if !ok {
			return true
		}
		for _, v := range list {
			if fmt.Sprintf("%v", v) == fmt.Sprintf("%v", actual) {
				return false
			}
		}
		return true
	case "matches":
		pat, _ := expected.(string)
		re, err := regexp.Compile(pat)
		if err != nil {
			return false
		}
		return re.MatchString(fmt.Sprintf("%v", actual))
	}

	return false
}

// isCondEmpty reports whether a value is absent, nil, empty string, false, or an empty collection.
func isCondEmpty(v any) bool {
	if v == nil {
		return true
	}
	switch x := v.(type) {
	case string:
		return strings.TrimSpace(x) == ""
	case bool:
		return !x
	case []any:
		return len(x) == 0
	case map[string]any:
		return len(x) == 0
	}
	return false
}

// condToFloat converts a value to float64 for numeric comparisons.
// Returns 0 for unrecognised types.
func condToFloat(v any) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case int:
		return float64(n)
	case string:
		f, _ := strconv.ParseFloat(n, 64)
		return f
	}
	return 0
}
