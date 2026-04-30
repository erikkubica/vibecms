package sdui

// engine_settings.go — SDUI layout generators for settings pages.
//
// Pre-registry, every settings page declared its schema inline here as
// nested map[string]any. That worked, but it conflated "what does this
// page look like" with "where does the data live", and it gave themes
// and extensions no way to share the mechanism.
//
// Post-registry, settings pages are tiny wrappers that emit a single
// SchemaSettings node referencing a registered schema by ID. The React
// component fetches /admin/api/settings/schemas/<id> on mount and
// renders the schema generically. Schema declarations live in
// internal/settings/builtin.go (core), at extension activation
// (extensions), and eventually in theme settings.json (themes).

func (e *Engine) siteSettingsLayout() *LayoutNode {
	return e.siteSettingsGeneralLayout()
}

func (e *Engine) siteSettingsGeneralLayout() *LayoutNode {
	return schemaSettingsNode("site.general", true)
}

func (e *Engine) siteSettingsSEOLayout() *LayoutNode {
	return schemaSettingsNode("site.seo", true)
}

func (e *Engine) siteSettingsAdvancedLayout() *LayoutNode {
	return schemaSettingsNode("site.advanced", true)
}

func (e *Engine) securitySettingsLayout() *LayoutNode {
	// Security has no translatable fields, so the React component
	// suppresses the language picker — show_clear_cache stays false
	// because flipping registration policy doesn't invalidate render
	// caches.
	return schemaSettingsNode("security", false)
}

// schemaSettingsNode is the canonical SchemaSettings layout. The
// show_clear_cache prop is preserved for the legacy site-settings
// pages where flipping a value (homepage selection, code injection)
// can leave stale rendered HTML in the cache.
func schemaSettingsNode(schemaID string, showClearCache bool) *LayoutNode {
	return &LayoutNode{
		Type: "SchemaSettings",
		Props: map[string]any{
			"schema_id":        schemaID,
			"show_clear_cache": showClearCache,
		},
	}
}
