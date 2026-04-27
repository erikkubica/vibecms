package main

import (
	"encoding/json"
	"fmt"
)

// normalizeForm parses JSONB string fields back into proper objects.
// PostgreSQL JSONB columns come back as strings through the DataStore.
func normalizeForm(row map[string]any) map[string]any {
	return normalizeJSONBFields(row, "fields", "notifications", "settings")
}

// normalizeSubmission parses JSONB columns on a form_submissions row so the
// admin UI receives `data` and `metadata` as objects rather than escaped JSON
// strings (which would otherwise be iterated character-by-character).
func normalizeSubmission(row map[string]any) map[string]any {
	return normalizeJSONBFields(row, "data", "metadata")
}

// normalizeJSONBFields decodes any of the listed keys whose value is a JSON
// string into its parsed shape. No-ops on non-string values.
func normalizeJSONBFields(row map[string]any, keys ...string) map[string]any {
	for _, key := range keys {
		s, ok := row[key].(string)
		if !ok || s == "" {
			continue
		}
		var parsed any
		if err := json.Unmarshal([]byte(s), &parsed); err == nil {
			row[key] = parsed
		}
	}
	return row
}

// getFormFields extracts the fields array from a normalized form map.
func getFormFields(form map[string]any) []map[string]any {
	raw := form["fields"]
	switch v := raw.(type) {
	case []any:
		fields := make([]map[string]any, 0, len(v))
		for _, item := range v {
			if m, ok := item.(map[string]any); ok {
				fields = append(fields, m)
			}
		}
		return fields
	case []map[string]any:
		return v
	default:
		return nil
	}
}

// getFormSettings extracts the settings map from a normalized form.
func getFormSettings(form map[string]any) map[string]any {
	if s, ok := form["settings"].(map[string]any); ok {
		return s
	}
	return map[string]any{}
}

// parseJSONMap parses a value that may be a map or a JSON-encoded string into map[string]any.
func parseJSONMap(v any) map[string]any {
	if v == nil {
		return nil
	}
	if m, ok := v.(map[string]any); ok {
		return m
	}
	if s, ok := v.(string); ok {
		var m map[string]any
		if err := json.Unmarshal([]byte(s), &m); err == nil {
			return m
		}
	}
	return nil
}

// fieldValueToString converts a submission field value to a string.
// Handles option objects ({label, value}) by returning the label.
func fieldValueToString(v any) string {
	switch val := v.(type) {
	case string:
		return val
	case map[string]any:
		if label, ok := val["label"].(string); ok {
			return label
		}
		return fmt.Sprintf("%v", val)
	case nil:
		return ""
	default:
		return fmt.Sprintf("%v", val)
	}
}
