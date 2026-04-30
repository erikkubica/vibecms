package sdui

import (
	"encoding/json"
	"strings"

	"squilla/internal/models"
)

// canReadNodeType mirrors auth.GetNodeAccess(...).CanRead() but is duplicated
// here to avoid an import cycle (sdui ← api ← auth → api). Resolution order:
// per-type override in `nodes.<slug>` → `default_node_access` → "none".
//
// Keep in sync with auth/rbac_middleware.go's GetNodeAccess.
func canReadNodeType(user *models.User, slug string) bool {
	if user == nil {
		return false
	}
	var caps map[string]interface{}
	if err := json.Unmarshal(user.Role.Capabilities, &caps); err != nil {
		return false
	}
	access := "none"
	if nodes, ok := caps["nodes"].(map[string]interface{}); ok {
		if override, ok := nodes[slug].(map[string]interface{}); ok {
			if a, ok := override["access"].(string); ok {
				access = a
			}
			return access == "read" || access == "write"
		}
	}
	if def, ok := caps["default_node_access"].(map[string]interface{}); ok {
		if a, ok := def["access"].(string); ok {
			access = a
		}
	}
	return access == "read" || access == "write"
}

// buildNavigation assembles the admin sidebar tree returned by
// GenerateBootManifest. Lives in its own file because it's the
// largest non-page-layout method on Engine and conceptually
// independent from the per-page layout dispatch in engine.go.
//
// Sidebar shape: top-level dashboard + extension-supplied top-level
// items, then four kernel-fixed sections (Content / Design /
// Development / Settings) each with kernel items followed by any
// extension items routed to that section via
// admin_ui.menu.section in the extension manifest.
//
// Per-user filtering: node-type entries are dropped when the user's
// effective access for that type resolves to "none". Defense in depth —
// API capability guards still reject direct calls — but hiding the link
// is what the user sees, so this is where capability edits visibly land.
func (e *Engine) buildNavigation(user *models.User, nodeTypes []models.NodeType, taxonomies []models.Taxonomy, exts []models.Extension) []NavItem {
	var nav []NavItem

	// Helper: resolve display label with fallback.
	labelFor := func(nt models.NodeType) string {
		if nt.LabelPlural != "" {
			return nt.LabelPlural
		}
		return nt.Label
	}

	// Build a lookup: node_type_slug → []Taxonomy
	taxesByNodeType := make(map[string][]models.Taxonomy)
	for _, tax := range taxonomies {
		for _, ntSlug := range tax.NodeTypes {
			taxesByNodeType[ntSlug] = append(taxesByNodeType[ntSlug], tax)
		}
	}

	// Helper: resolve path for a node type slug.
	pathForType := func(slug string) string {
		switch slug {
		case "page":
			return "/admin/pages"
		case "post":
			return "/admin/posts"
		default:
			return "/admin/content/" + slug
		}
	}

	// Helper: resolve icon for a node type slug.
	iconForType := func(slug, fallback string) string {
		if fallback != "" {
			return fallback
		}
		switch slug {
		case "page":
			return "FileText"
		case "post":
			return "Newspaper"
		default:
			return "Boxes"
		}
	}

	// ── Parse extension manifests once and bucket items by section ──
	// Extensions declare placement via:
	//   admin_ui.menu.section      → "content" | "design" | "development" | "settings"
	//   admin_ui.settings_menu[]   → shortcut; always lands in the Settings section
	// Anything without a section (or an unknown value) is treated as top-level.
	extContent := []NavItem{}
	extDesign := []NavItem{}
	extDev := []NavItem{}
	extSettings := []NavItem{}
	extTopLevel := []NavItem{}

	for _, ext := range exts {
		var manifest struct {
			AdminUI *struct {
				Menu *struct {
					Label    string `json:"label"`
					Icon     string `json:"icon"`
					Section  string `json:"section"`
					Children []struct {
						Label string `json:"label"`
						Route string `json:"route"`
						Icon  string `json:"icon"`
					} `json:"children"`
				} `json:"menu"`
				SettingsMenu []struct {
					Label string `json:"label"`
					Route string `json:"route"`
					Icon  string `json:"icon"`
				} `json:"settings_menu"`
			} `json:"admin_ui"`
		}
		_ = json.Unmarshal(ext.Manifest, &manifest)
		if manifest.AdminUI == nil {
			continue
		}

		if m := manifest.AdminUI.Menu; m != nil {
			var navItem NavItem
			if len(m.Children) > 0 {
				children := make([]NavItem, 0, len(m.Children))
				for _, c := range m.Children {
					children = append(children, NavItem{
						ID:    "nav-ext-" + ext.Slug + "-" + c.Route,
						Label: c.Label,
						Icon:  c.Icon,
						Path:  c.Route,
					})
				}
				navItem = NavItem{
					ID:       "nav-ext-" + ext.Slug,
					Label:    m.Label,
					Icon:     m.Icon,
					Children: children,
				}
			} else {
				navItem = NavItem{
					ID:    "nav-ext-" + ext.Slug,
					Label: m.Label,
					Icon:  m.Icon,
					Path:  "/admin/ext/" + ext.Slug + "/",
				}
			}
			switch strings.ToLower(m.Section) {
			case "content":
				extContent = append(extContent, navItem)
			case "design":
				extDesign = append(extDesign, navItem)
			case "development", "dev":
				extDev = append(extDev, navItem)
			case "settings":
				extSettings = append(extSettings, navItem)
			default:
				extTopLevel = append(extTopLevel, navItem)
			}
		}

		for _, item := range manifest.AdminUI.SettingsMenu {
			extSettings = append(extSettings, NavItem{
				ID:    "nav-ext-" + ext.Slug + "-settings-" + item.Route,
				Label: item.Label,
				Icon:  item.Icon,
				Path:  item.Route,
			})
		}
	}

	// ── Top level ──
	nav = append(nav, NavItem{
		ID: "nav-dashboard", Label: "Dashboard", Icon: "LayoutDashboard",
		Path: "/admin/dashboard",
	})
	nav = append(nav, extTopLevel...)

	// ── Content section ──
	nav = append(nav, NavItem{ID: "section-content", Label: "Content", IsSection: true})

	for _, nt := range nodeTypes {
		// Capability gate: skip types the current user has no read access to.
		// Resolves per-type override first then default_node_access — same
		// rules as the API guards, so the sidebar is exactly the set of types
		// the user could actually open.
		if !canReadNodeType(user, nt.Slug) {
			continue
		}

		// Build sub-items for this node type: main listing + any taxonomies.
		basePath := pathForType(nt.Slug)
		displayLabel := labelFor(nt)

		taxChildren := []NavItem{}
		// First child: link to the main listing.
		taxChildren = append(taxChildren, NavItem{
			ID:    "nav-content-" + nt.Slug + "-all",
			Label: displayLabel,
			Icon:  iconForType(nt.Slug, nt.Icon),
			Path:  basePath,
		})
		for _, tax := range taxesByNodeType[nt.Slug] {
			taxLabel := tax.LabelPlural
			if taxLabel == "" {
				taxLabel = tax.Label
			}

			taxChildren = append(taxChildren, NavItem{
				ID:    "nav-content-" + nt.Slug + "-tax-" + tax.Slug,
				Label: taxLabel,
				Icon:  "Tags",
				Path:  "/admin/content/" + nt.Slug + "/taxonomies/" + tax.Slug,
			})
		}

		item := NavItem{
			ID:    "nav-content-" + nt.Slug,
			Label: displayLabel,
			Icon:  iconForType(nt.Slug, nt.Icon),
			Path:  basePath,
		}
		if len(taxChildren) > 1 {
			// Only add children when there are taxonomies beyond the
			// "All X" entry — otherwise keep it as a flat link.
			item.Children = taxChildren
		}
		nav = append(nav, item)
	}
	nav = append(nav, extContent...)

	// ── Design section ──
	nav = append(nav, NavItem{ID: "section-design", Label: "Design", IsSection: true})
	nav = append(nav, []NavItem{
		{ID: "nav-templates", Label: "Templates", Icon: "FileCode", Path: "/admin/templates"},
		{ID: "nav-layouts", Label: "Layouts", Icon: "LayoutPanelTop", Path: "/admin/layouts"},
		{ID: "nav-block-types", Label: "Block Types", Icon: "Blocks", Path: "/admin/block-types"},
		{ID: "nav-layout-blocks", Label: "Layout Blocks", Icon: "Component", Path: "/admin/layout-blocks"},
		{ID: "nav-menus", Label: "Menus", Icon: "ListTree", Path: "/admin/menus"},
	}...)
	nav = append(nav, extDesign...)

	// ── Development section ──
	nav = append(nav, NavItem{ID: "section-dev", Label: "Development", IsSection: true})
	nav = append(nav, []NavItem{
		{ID: "nav-content-types", Label: "Content Types", Icon: "Shapes", Path: "/admin/content-types"},
		{ID: "nav-taxonomies", Label: "Taxonomies", Icon: "Tags", Path: "/admin/taxonomies"},
		{ID: "nav-themes", Label: "Themes", Icon: "Brush", Path: "/admin/themes"},
		{ID: "nav-extensions", Label: "Extensions", Icon: "Puzzle", Path: "/admin/extensions"},
	}...)
	nav = append(nav, extDev...)

	// ── Settings section ──
	// Top-level groups: Site Settings (with sub-pages), Security
	// (with sub-pages). Extension-contributed settings still slot in
	// after the built-in groups via extSettings.
	nav = append(nav, NavItem{ID: "section-settings", Label: "Settings", IsSection: true})
	nav = append(nav, NavItem{
		ID:    "nav-site-settings",
		Label: "Site Settings",
		Icon:  "Globe",
		Path:  "/admin/settings/site/general",
		Children: []NavItem{
			{ID: "nav-site-settings-general", Label: "General", Icon: "Globe", Path: "/admin/settings/site/general"},
			{ID: "nav-site-settings-seo", Label: "SEO", Icon: "Globe", Path: "/admin/settings/site/seo"},
			{ID: "nav-site-settings-advanced", Label: "Advanced", Icon: "FileCode", Path: "/admin/settings/site/advanced"},
			{ID: "nav-site-settings-languages", Label: "Languages", Icon: "Languages", Path: "/admin/settings/site/languages"},
		},
	})
	nav = append(nav, NavItem{
		ID:    "nav-security",
		Label: "Security",
		Icon:  "Shield",
		Path:  "/admin/security/users",
		Children: []NavItem{
			{ID: "nav-security-users", Label: "Users", Icon: "Users", Path: "/admin/security/users"},
			{ID: "nav-security-roles", Label: "Roles", Icon: "Shield", Path: "/admin/security/roles"},
			{ID: "nav-security-mcp-tokens", Label: "MCP Tokens", Icon: "Key", Path: "/admin/security/mcp-tokens"},
			{ID: "nav-security-settings", Label: "Settings", Icon: "Settings", Path: "/admin/security/settings"},
		},
	})
	nav = append(nav, extSettings...)

	return nav
}
