package main

import (
	"fmt"
	"math"
	"net/mail"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

// validateSubmission returns a per-field error map (empty if valid).
// `files` is the parsed file uploads keyed by field id (nil for JSON-only submits).
func validateSubmission(data map[string]any, fields []map[string]any, files map[string][]uploadedFile) map[string]string {
	errs := map[string]string{}

	for _, field := range fields {
		id, _ := field["id"].(string)
		if id == "" {
			continue
		}
		if !shouldValidateField(field, data) {
			continue
		}

		fType, _ := field["type"].(string)
		label, _ := field["label"].(string)
		if label == "" {
			label = id
		}

		val, exists := data[id]

		// 1. Required
		if req, _ := field["required"].(bool); req {
			if fType == "file" {
				if len(files[id]) == 0 {
					errs[id] = label + " is required"
					continue
				}
			} else if fType == "gdpr_consent" {
				if checked, _ := val.(bool); !checked {
					errs[id] = "You must agree to the privacy policy"
					continue
				}
			} else if !exists || isEmpty(val) {
				errs[id] = label + " is required"
				continue
			}
		}

		// File fields may have no data[id] entry but still have uploaded files via multipart.
		// Skip the !exists guard for file fields so the case "file" branch can run.
		if !exists && fType != "file" {
			continue
		}

		// 2. Type-specific format checks
		switch fType {
		case "email":
			if s, ok := val.(string); ok && s != "" {
				if _, err := mail.ParseAddress(s); err != nil {
					errs[id] = "Invalid email format"
					continue
				}
			}
		case "url":
			if s, ok := val.(string); ok && s != "" {
				if _, err := url.ParseRequestURI(s); err != nil {
					errs[id] = "Invalid URL format"
					continue
				}
			}
		case "gdpr_consent":
			if req, _ := field["required"].(bool); req {
				if checked, _ := val.(bool); !checked {
					errs[id] = "You must agree to the privacy policy"
					continue
				}
			}
		case "file":
			if ffiles := files[id]; len(ffiles) > 0 {
				// max_files check
				if multiple, _ := field["multiple"].(bool); multiple {
					if maxF := intField(field, "max_files"); maxF > 0 && len(ffiles) > maxF {
						errs[id] = fmt.Sprintf("At most %d files allowed", maxF)
						continue
					}
				}
				// per-file allowed_types / max_size
				for _, f := range ffiles {
					if msg := validateFile(field, f); msg != "" {
						errs[id] = msg
						break
					}
				}
			}
			continue // file fields skip length/pattern checks
		case "number", "range":
			if msg := checkNumeric(field, val); msg != "" {
				errs[id] = msg
				continue
			}
		}

		// 3. Length / pattern (text-like types)
		switch fType {
		case "text", "textarea", "url", "email", "tel":
			if msg := checkLength(field, val); msg != "" {
				errs[id] = msg
				continue
			}
			if msg := checkPattern(field, val); msg != "" {
				errs[id] = msg
				continue
			}
		}
	}

	return errs
}

// shouldValidateField returns false when the field's display_when condition
// evaluates to false — meaning the field is hidden and must be skipped.
func shouldValidateField(field map[string]any, data map[string]any) bool {
	return EvaluateField(field, data)
}

// isEmpty reports whether a value is nil, empty string, or whitespace-only string.
// Used by validation; isCondEmpty in conditions.go handles the richer condition semantics.
func isEmpty(v any) bool {
	if v == nil {
		return true
	}
	if s, ok := v.(string); ok {
		return strings.TrimSpace(s) == ""
	}
	return false
}

// intField reads an int-valued key from a field definition (float64 or int).
func intField(field map[string]any, key string) int {
	v, ok := field[key]
	if !ok {
		return 0
	}
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	}
	return 0
}

// floatField reads a float64-valued key from a field definition.
func floatField(field map[string]any, key string) (float64, bool) {
	v, ok := field[key]
	if !ok {
		return 0, false
	}
	switch n := v.(type) {
	case float64:
		return n, true
	case int:
		return float64(n), true
	}
	return 0, false
}

// checkLength validates min_length / max_length for string values.
func checkLength(field map[string]any, val any) string {
	s, ok := val.(string)
	if !ok {
		return ""
	}
	if n := intField(field, "min_length"); n > 0 && len(s) < n {
		return fmt.Sprintf("Minimum %d characters", n)
	}
	if n := intField(field, "max_length"); n > 0 && len(s) > n {
		return fmt.Sprintf("Maximum %d characters", n)
	}
	return ""
}

// checkNumeric validates min / max / step for numeric values.
func checkNumeric(field map[string]any, val any) string {
	var num float64
	switch n := val.(type) {
	case float64:
		num = n
	case int:
		num = float64(n)
	case string:
		var err error
		num, err = strconv.ParseFloat(n, 64)
		if err != nil {
			return ""
		}
	default:
		return ""
	}

	if minVal, ok := floatField(field, "min"); ok && num < minVal {
		return fmt.Sprintf("Must be at least %g", minVal)
	}
	if maxVal, ok := floatField(field, "max"); ok && num > maxVal {
		return fmt.Sprintf("Must be at most %g", maxVal)
	}
	if step, ok := floatField(field, "step"); ok && step > 0 {
		// Check divisibility within floating-point tolerance
		if rem := math.Mod(num, step); math.Abs(rem) > 1e-9 && math.Abs(rem-step) > 1e-9 {
			return fmt.Sprintf("Must be a multiple of %g", step)
		}
	}
	return ""
}

// checkPattern validates a regex pattern constraint.
func checkPattern(field map[string]any, val any) string {
	pattern, _ := field["pattern"].(string)
	if pattern == "" {
		return ""
	}
	s, ok := val.(string)
	if !ok || s == "" {
		return ""
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return "" // invalid pattern — skip silently
	}
	if !re.MatchString(s) {
		return "Invalid format"
	}
	return ""
}

