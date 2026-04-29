package cms

import (
	"encoding/json"
	"strconv"
)

// CoerceResult describes the outcome of converting a stored string value
// into the runtime type expected by a settings field. Compatible=false
// means the stored raw value didn't match the current field type, so
// Value carries the field's declared default (or zero value) instead.
// Raw always holds the original stored string so admin UI can surface a
// "previous value: <raw>" hint without losing data.
type CoerceResult struct {
	Value      any
	Compatible bool
	Raw        string
}

// CoerceValue converts a stored string into the runtime type for the given
// field type. ok=false signals the raw value couldn't be coerced (caller
// substitutes a default). An empty raw string is always ok=true with a
// nil Value — "no input" is not a type error.
func CoerceValue(fieldType, raw string) (value any, ok bool) {
	if raw == "" {
		return nil, true
	}
	switch fieldType {
	case "text", "textarea", "richtext", "email", "url", "color", "date",
		"select", "radio":
		return raw, true
	case "number", "range":
		if f, err := strconv.ParseFloat(raw, 64); err == nil {
			return f, true
		}
		return nil, false
	case "toggle":
		switch raw {
		case "true", "1":
			return true, true
		case "false", "0":
			return false, true
		}
		return nil, false
	default:
		// JSON-shaped types — checkbox/image/file/gallery/link/group/repeater/
		// node/term, plus any extension-contributed type. Stored as JSON.
		var v any
		if err := json.Unmarshal([]byte(raw), &v); err == nil {
			return v, true
		}
		return nil, false
	}
}

// CoerceWithDefault wraps CoerceValue and substitutes the field's declared
// default (also raw JSON) when the stored value is incompatible. The DB is
// never mutated here — only the rendered/handed-back value changes.
func CoerceWithDefault(field ThemeSettingsField, raw string) CoerceResult {
	v, ok := CoerceValue(field.Type, raw)
	if ok {
		return CoerceResult{Value: v, Compatible: true, Raw: raw}
	}
	var dflt any
	if len(field.Default) > 0 {
		_ = json.Unmarshal(field.Default, &dflt)
	}
	return CoerceResult{Value: dflt, Compatible: false, Raw: raw}
}
