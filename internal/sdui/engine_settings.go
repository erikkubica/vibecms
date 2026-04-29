package sdui

// engine_settings.go — SDUI layout generators for the various settings pages.
// The schema is described as plain Go data so the layout cache stays cheap and
// the React Shell can render any settings page through the same SettingsForm
// component.

func (e *Engine) siteSettingsLayout() *LayoutNode {
	schema := []map[string]any{
		{
			"title":       "General",
			"icon":        "Globe",
			"description": "Basic site identity",
			"fields": []map[string]any{
				{"key": "site_name", "label": "Site Name", "type": "text", "placeholder": "My Website"},
				{
					"key":         "site_url",
					"label":       "Site URL",
					"type":        "text",
					"placeholder": "https://example.com",
					"help":        "Used for sitemaps, canonical URLs, and absolute links",
				},
				{
					"key":         "site_description",
					"label":       "Site Description",
					"type":        "textarea",
					"rows":        2,
					"placeholder": "A short description of your website...",
				},
			},
		},
		{
			"title":       "Homepage",
			"icon":        "Home",
			"description": "Choose which page visitors see first",
			"fields": []map[string]any{
				{
					"key":         "homepage_node_id",
					"label":       "Homepage",
					"type":        "node_select",
					"node_type":   "page",
					"empty_label": "No homepage set",
					"placeholder": "Select a page...",
					"help":        "This page will be displayed when visitors access your site root",
				},
			},
		},
		{
			"title":       "Code Injection",
			"icon":        "FileText",
			"description": "Add custom code to your site's <head> section",
			"full_width":  true,
			"fields": []map[string]any{
				{
					"key":         "analytics_code",
					"label":       "Analytics Code",
					"type":        "textarea",
					"rows":        5,
					"font_mono":   true,
					"placeholder": "<!-- Google Analytics, Plausible, etc. -->",
					"help":        "Injected into <head> on every public page",
				},
				{
					"key":         "custom_head_code",
					"label":       "Custom Head Code",
					"type":        "textarea",
					"rows":        5,
					"font_mono":   true,
					"placeholder": "<!-- Custom meta tags, fonts, etc. -->",
					"help":        "Injected into <head> on every public page",
				},
				{
					"key":         "custom_footer_code",
					"label":       "Footer Code",
					"type":        "textarea",
					"rows":        5,
					"font_mono":   true,
					"placeholder": "<!-- Chat widgets, tracking pixels, etc. -->",
					"help":        "Injected before </body> on every public page",
				},
			},
		},
	}

	// SettingsForm owns its own page-level spacing (title row + 2-col grid).
	// The admin shell's <main> already provides outer padding, so we don't
	// wrap with another padded VerticalStack here.
	return &LayoutNode{
		Type: "SettingsForm",
		Props: map[string]any{
			"title":            "Site Settings",
			"description":      "Configure your site's core settings",
			"schema":           schema,
			"show_clear_cache": true,
		},
	}
}
