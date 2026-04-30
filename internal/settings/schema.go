// Package settings provides a schema-driven settings framework. Core,
// extensions, and (eventually) themes register Schemas describing one
// settings surface (security policy, site identity, an extension's config,
// etc.). The framework handles per-locale vs language-agnostic storage,
// validation against the registered schema, and surfacing the schema
// to the admin UI for generic form rendering.
//
// Schemas live in an in-process Registry. Each Field declares whether
// its value is translatable (per-language) or global (language_code='').
// All values still live in the existing site_settings table — no new
// schema migration is required.
package settings

// Schema describes one settings surface (one admin page or one extension
// config block). Identified by a stable string ID such as "security",
// "site.general", or "ext.<slug>".
type Schema struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description,omitempty"`
	// Capability required to read or write values via the HTTP handler.
	// Empty string means "any authenticated admin".
	Capability string    `json:"capability,omitempty"`
	Sections   []Section `json:"sections"`
}

// Section groups related fields under a card heading.
type Section struct {
	Title       string  `json:"title"`
	Icon        string  `json:"icon,omitempty"`
	Description string  `json:"description,omitempty"`
	FullWidth   bool    `json:"full_width,omitempty"`
	Fields      []Field `json:"fields"`
}

// Field is one input on a settings page. Storage row in site_settings
// is keyed by (Key, language_code); language_code is the admin's locale
// when Translatable=true, or the empty string when Translatable=false.
type Field struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	Type        string `json:"type"`
	// Translatable controls storage routing. Default false at the type
	// level — registrations that want per-locale behaviour set true
	// explicitly. The framework never infers translatability from the
	// field type because the same type can sensibly be either.
	Translatable bool   `json:"translatable"`
	Default      string `json:"default,omitempty"`
	Placeholder  string `json:"placeholder,omitempty"`
	Help         string `json:"help,omitempty"`
	Warning      string `json:"warning,omitempty"`
	// Type-specific (textarea/text)
	Rows     int  `json:"rows,omitempty"`
	FontMono bool `json:"font_mono,omitempty"`
	// Type-specific (toggle)
	TrueValue  string `json:"true_value,omitempty"`
	FalseValue string `json:"false_value,omitempty"`
	// Type-specific (node_select)
	NodeType   string `json:"node_type,omitempty"`
	EmptyLabel string `json:"empty_label,omitempty"`
	// Type-specific (select with static options)
	Options []Option `json:"options,omitempty"`
}

// Option is one entry in a select field's static option list.
type Option struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

// FieldByKey looks up a field across every section. Returns nil when the
// key is unknown — callers use this for write-side validation (reject
// keys not declared in the schema) and locale routing (read Translatable).
func (s Schema) FieldByKey(key string) *Field {
	for i := range s.Sections {
		for j := range s.Sections[i].Fields {
			if s.Sections[i].Fields[j].Key == key {
				return &s.Sections[i].Fields[j]
			}
		}
	}
	return nil
}

// Keys returns every field key declared by the schema. Order follows
// section/field declaration order.
func (s Schema) Keys() []string {
	var keys []string
	for _, sec := range s.Sections {
		for _, f := range sec.Fields {
			keys = append(keys, f.Key)
		}
	}
	return keys
}

// HasTranslatable returns true when at least one field is per-locale.
// Used by the admin UI to decide whether to surface the language picker.
func (s Schema) HasTranslatable() bool {
	for _, sec := range s.Sections {
		for _, f := range sec.Fields {
			if f.Translatable {
				return true
			}
		}
	}
	return false
}
