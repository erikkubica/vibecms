package sdui

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"vibecms/internal/events"
	"vibecms/internal/models"

	"gorm.io/gorm"
)

// Engine generates and caches SDUI layout trees for admin pages.
// It subscribes to state-change events and invalidates its cache
// when extensions or node types are modified.
type Engine struct {
	db       *gorm.DB
	eventBus *events.EventBus
	mu       sync.RWMutex
	cache    map[string]cachedLayout
}

type cachedLayout struct {
	Layout  LayoutNode
	Version string
}

// NewEngine creates a layout engine that invalidates its cache
// on extension and node-type lifecycle events.
func NewEngine(db *gorm.DB, eventBus *events.EventBus) *Engine {
	e := &Engine{
		db:       db,
		eventBus: eventBus,
		cache:    make(map[string]cachedLayout),
	}

	// Invalidate cache on relevant events
	eventBus.Subscribe("extension.activated", e.onStateChange)
	eventBus.Subscribe("extension.deactivated", e.onStateChange)
	eventBus.Subscribe("node_type.created", e.onStateChange)
	eventBus.Subscribe("node_type.updated", e.onStateChange)
	eventBus.Subscribe("node_type.deleted", e.onStateChange)
	eventBus.Subscribe("taxonomy.created", e.onStateChange)
	eventBus.Subscribe("taxonomy.updated", e.onStateChange)
	eventBus.Subscribe("taxonomy.deleted", e.onStateChange)

	return e
}

func (e *Engine) onStateChange(action string, payload events.Payload) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.cache = make(map[string]cachedLayout) // clear all cache
}

// GenerateBootManifest creates the boot manifest for the current state.
func (e *Engine) GenerateBootManifest(user *models.User) (*BootManifest, error) {
	// Get active extensions
	var exts []models.Extension
	if err := e.db.Where("is_active = ?", true).Order("priority ASC").Find(&exts).Error; err != nil {
		return nil, err
	}

	// Get node types
	var nodeTypes []models.NodeType
	if err := e.db.Order("label ASC").Find(&nodeTypes).Error; err != nil {
		return nil, err
	}

	// Build boot extensions
	bootExts := make([]BootExt, 0, len(exts))
	for _, ext := range exts {
		var manifest struct {
			AdminUI *struct {
				Entry      string   `json:"entry"`
				Components []string `json:"components"`
			} `json:"admin_ui"`
		}
		_ = json.Unmarshal(ext.Manifest, &manifest)

		be := BootExt{
			Slug: ext.Slug,
			Name: ext.Name,
		}
		if manifest.AdminUI != nil {
			be.Entry = manifest.AdminUI.Entry
			be.Components = manifest.AdminUI.Components
		}
		bootExts = append(bootExts, be)
	}

	// Get taxonomies (for navigation sub-items under node types)
	var taxonomies []models.Taxonomy
	if err := e.db.Order("label ASC").Find(&taxonomies).Error; err != nil {
		return nil, err
	}

	// Build navigation
	nav := e.buildNavigation(nodeTypes, taxonomies, exts)

	// Build boot node types
	bootNodeTypes := make([]BootNodeType, 0, len(nodeTypes))
	for _, nt := range nodeTypes {
		bootNodeTypes = append(bootNodeTypes, BootNodeType{
			Slug:           nt.Slug,
			Label:          nt.Label,
			LabelPlural:    nt.LabelPlural,
			Icon:           nt.Icon,
			SupportsBlocks: nt.SupportsBlocks,
		})
	}

	// Parse user capabilities
	caps := make(map[string]interface{})
	_ = json.Unmarshal(user.Role.Capabilities, &caps)

	return &BootManifest{
		Version: "1.0.0",
		User: BootUser{
			ID:           user.ID,
			Email:        user.Email,
			FullName:     derefString(user.FullName),
			Role:         user.Role.Slug,
			Capabilities: caps,
		},
		Extensions: bootExts,
		Navigation: nav,
		NodeTypes:  bootNodeTypes,
	}, nil
}

// GenerateLayout creates a layout tree for a given page slug.
// Results are cached until a state-change event invalidates them.
// Dashboard layouts are never cached because they contain live data.
func (e *Engine) GenerateLayout(pageSlug string, params map[string]string, userName string) (*LayoutNode, error) {
	skipCache := pageSlug == "dashboard" || pageSlug == "node-list" || pageSlug == "taxonomy-terms" ||
		pageSlug == "templates" || pageSlug == "layouts" || pageSlug == "block-types" || pageSlug == "layout-blocks" || pageSlug == "menus" ||
		pageSlug == "themes" || pageSlug == "extensions" ||
		pageSlug == "content-types" || pageSlug == "taxonomies"

	if !skipCache {
		e.mu.RLock()
		if cached, ok := e.cache[pageSlug]; ok {
			e.mu.RUnlock()
			return &cached.Layout, nil
		}
		e.mu.RUnlock()
	}

	var layout *LayoutNode

	switch pageSlug {
	case "dashboard":
		layout = e.dashboardLayout(userName)
	case "list":
		layout = e.listLayout(params["nodeType"])
	case "content-types":
		layout = e.contentTypesLayout(params)
	case "taxonomies":
		layout = e.taxonomiesLayout(params)
	case "node-list":
		layout = e.nodeListLayout(params)
	case "taxonomy-terms":
		layout = e.taxonomyTermsLayout(params)
	case "templates":
		layout = e.templatesLayout(params)
	case "layouts":
		layout = e.layoutsLayout(params)
	case "block-types":
		layout = e.blockTypesLayout(params)
	case "layout-blocks":
		layout = e.layoutBlocksLayout(params)
	case "menus":
		layout = e.menusLayout(params)
	case "themes":
		layout = e.themesLayout()
	case "extensions":
		layout = e.extensionsLayout()
	default:
		layout = e.defaultLayout(pageSlug)
	}

	// TODO: Apply filter chain (admin:layout:render) so extensions can modify
	// layout = e.applyFilters(layout, pageSlug)

	if !skipCache {
		e.mu.Lock()
		e.cache[pageSlug] = cachedLayout{Layout: *layout}
		e.mu.Unlock()
	}

	return layout, nil
}

func (e *Engine) buildNavigation(nodeTypes []models.NodeType, taxonomies []models.Taxonomy, exts []models.Extension) []NavItem {
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
					})}
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
	nav = append(nav, NavItem{ID: "section-settings", Label: "Settings", IsSection: true})
	nav = append(nav, []NavItem{
		{ID: "nav-site-settings", Label: "Site", Icon: "Globe", Path: "/admin/settings/site"},
		{ID: "nav-languages", Label: "Languages", Icon: "Languages", Path: "/admin/languages"},
		{ID: "nav-users", Label: "Users", Icon: "Users", Path: "/admin/users"},
		{ID: "nav-roles", Label: "Roles", Icon: "Shield", Path: "/admin/roles"},
		{ID: "nav-mcp-tokens", Label: "MCP Tokens", Icon: "Key", Path: "/admin/mcp-tokens"},
	}...)
	nav = append(nav, extSettings...)

	return nav
}

type recentNode struct {
	ID        int    `json:"id"`
	Title     string `json:"title"`
	NodeType  string `json:"node_type"`
	Status    string `json:"status"`
	UpdatedAt string `json:"updated_at"`
}

// greetingFor returns a time-of-day greeting based on the local hour.
func greetingFor(now time.Time) string {
	h := now.Hour()
	switch {
	case h < 5:
		return "Working late"
	case h < 12:
		return "Good morning"
	case h < 18:
		return "Good afternoon"
	default:
		return "Good evening"
	}
}

// firstName returns the first whitespace-separated token of a full name,
// or "Admin" when empty.
func firstName(full string) string {
	full = strings.TrimSpace(full)
	if full == "" {
		return "Admin"
	}
	if i := strings.IndexAny(full, " \t"); i > 0 {
		return full[:i]
	}
	return full
}

func (e *Engine) dashboardLayout(userName string) *LayoutNode {
	var totalContent, published, drafts, totalUsers int64
	e.db.Model(&models.ContentNode{}).Where("deleted_at IS NULL").Count(&totalContent)
	e.db.Model(&models.ContentNode{}).Where("deleted_at IS NULL AND status = ?", "published").Count(&published)
	e.db.Model(&models.ContentNode{}).Where("deleted_at IS NULL AND status = ?", "draft").Count(&drafts)
	e.db.Model(&models.User{}).Count(&totalUsers)

	// 5 most recent content nodes (any status)
	var nodes []models.ContentNode
	e.db.Where("deleted_at IS NULL").Order("updated_at DESC").Limit(5).Find(&nodes)
	recentNodes := make([]recentNode, 0, len(nodes))
	for _, n := range nodes {
		recentNodes = append(recentNodes, recentNode{
			ID: n.ID, Title: n.Title, NodeType: n.NodeType,
			Status: n.Status, UpdatedAt: n.UpdatedAt.Format("2006-01-02"),
		})
	}

	// 5 most-recently-updated drafts → "Needs attention"
	var draftNodes []models.ContentNode
	e.db.Where("deleted_at IS NULL AND status = ?", "draft").
		Order("updated_at DESC").Limit(5).Find(&draftNodes)
	draftItems := make([]map[string]interface{}, 0, len(draftNodes))
	for _, n := range draftNodes {
		title := n.Title
		if title == "" {
			title = "Untitled"
		}
		draftItems = append(draftItems, map[string]interface{}{
			"id":      n.ID,
			"message": title + " · " + n.NodeType,
			"time":    n.UpdatedAt.Format("Jan 2"),
			"type":    "update",
		})
	}

	now := time.Now()
	greeting := greetingFor(now) + ", " + firstName(userName)

	return &LayoutNode{
		Type:  "VerticalStack",
		Props: map[string]interface{}{"gap": 6},
		Children: []LayoutNode{
			{
				Type: "WelcomeBanner",
				Props: map[string]interface{}{
					"title":       greeting,
					"subtitle":    now.Format("Monday, January 2"),
					"actionLabel": "Create New Page",
					"actionPath":  "/admin/pages/new",
				},
			},
			// Stats
			{
				Type:  "Grid",
				Props: map[string]interface{}{"cols": 4, "gap": 4},
				Children: []LayoutNode{
					{Type: "StatCard", Props: map[string]interface{}{"label": "Total Content", "value": fmt.Sprintf("%d", totalContent), "icon": "FileText", "color": "indigo"}},
					{Type: "StatCard", Props: map[string]interface{}{"label": "Published", "value": fmt.Sprintf("%d", published), "icon": "Eye", "color": "emerald"}},
					{Type: "StatCard", Props: map[string]interface{}{"label": "Drafts", "value": fmt.Sprintf("%d", drafts), "icon": "PenLine", "color": "amber"}},
					{Type: "StatCard", Props: map[string]interface{}{"label": "Users", "value": fmt.Sprintf("%d", totalUsers), "icon": "Users", "color": "violet"}},
				},
			},
			// Working area: drafts + quick actions
			{
				Type:  "Grid",
				Props: map[string]interface{}{"cols": 2, "gap": 6},
				Children: []LayoutNode{
					{
						Type: "ActivityFeed",
						Props: map[string]interface{}{
							"items":        draftItems,
							"title":        "Needs attention",
							"emptyMessage": "No drafts waiting — nice.",
						},
					},
					{
						Type: "QuickActions",
						Props: map[string]interface{}{
							"actions": []map[string]interface{}{
								{"label": "Pages", "path": "/admin/content/page", "icon": "FileText"},
								{"label": "Forms", "path": "/admin/ext/forms", "icon": "FormInput"},
								{"label": "Media", "path": "/admin/ext/media-manager", "icon": "Image"},
								{"label": "Users", "path": "/admin/users", "icon": "Users"},
								{"label": "Themes", "path": "/admin/themes", "icon": "Palette"},
								{"label": "Extensions", "path": "/admin/extensions", "icon": "Puzzle"},
							},
						},
					},
				},
			},
			// Recent content
			{
				Type:  "RecentContentTable",
				Props: map[string]interface{}{"items": recentNodes},
			},
		},
	}
}

func (e *Engine) listLayout(nodeType string) *LayoutNode {
	label := nodeType
	if nodeType == "" {
		label = "items"
	}

	return &LayoutNode{
		Type:  "VerticalStack",
		Props: map[string]interface{}{"gap": 0},
		Children: []LayoutNode{
			{
				Type: "ListHeader",
				Props: map[string]interface{}{
					"title":   label,
					"newPath": "/admin/content/" + nodeType + "/new",
				},
			},
			{
				Type: "ListToolbar",
				Props: map[string]interface{}{
					"searchPlaceholder": "Search " + label + "...",
				},
			},
			{
				Type: "DataTable",
				Props: map[string]interface{}{
					"endpoint": "nodes",
					"nodeType": nodeType,
					"columns":  []interface{}{},
				},
			},
		},
	}
}

// isBuiltinNodeType returns true for the two types the kernel reserves —
// "page" and "post" cannot be deleted (node_type_svc enforces this).
func isBuiltinNodeType(slug string) bool {
	return slug == "page" || slug == "post"
}

func (e *Engine) contentTypesLayout(params map[string]string) *LayoutNode {
	page, _ := strconv.Atoi(params["page"])
	if page < 1 {
		page = 1
	}
	perPage := getPerPage(params)
	tab := params["tab"]
	search := params["search"]

	sortBy := params["sort"]
	sortOrder := params["order"]
	switch sortBy {
	case "label", "slug", "updated_at":
	default:
		sortBy = "label"
	}
	if sortOrder != "asc" && sortOrder != "desc" {
		sortOrder = "asc"
	}

	q := e.db.Model(&models.NodeType{})
	if search != "" {
		q = q.Where("label ILIKE ? OR slug ILIKE ?", "%"+search+"%", "%"+search+"%")
	}
	var allTypes []models.NodeType
	q.Order(sortBy + " " + sortOrder).Find(&allTypes)

	builtinCount := 0
	customCount := 0
	for _, nt := range allTypes {
		if isBuiltinNodeType(nt.Slug) {
			builtinCount++
		} else {
			customCount++
		}
	}

	filtered := allTypes
	switch tab {
	case "builtin":
		filtered = filtered[:0]
		for _, nt := range allTypes {
			if isBuiltinNodeType(nt.Slug) {
				filtered = append(filtered, nt)
			}
		}
	case "custom":
		filtered = filtered[:0]
		for _, nt := range allTypes {
			if !isBuiltinNodeType(nt.Slug) {
				filtered = append(filtered, nt)
			}
		}
	}

	total := len(filtered)
	offset := (page - 1) * perPage
	end := offset + perPage
	if end > total {
		end = total
	}
	pageData := []models.NodeType{}
	if offset < total {
		pageData = filtered[offset:end]
	}

	rows := make([]map[string]interface{}, 0, len(pageData))
	for _, nt := range pageData {
		var taxSlugs []string
		if err := json.Unmarshal(nt.Taxonomies, &taxSlugs); err != nil {
			taxSlugs = []string{}
		}
		builtin := isBuiltinNodeType(nt.Slug)
		rows = append(rows, map[string]interface{}{
			"id":             nt.ID,
			"slug":           nt.Slug,
			"label":          nt.Label,
			"labelPlural":    nt.LabelPlural,
			"taxonomyCount":  len(taxSlugs),
			"supportsBlocks": nt.SupportsBlocks,
			"sourceLabel":    ternary(builtin, "Built-in", "Custom"),
			"isCustom":       !builtin,
			"updated_at":     nt.UpdatedAt.Format("2006-01-02"),
			"editPath":       fmt.Sprintf("/admin/content-types/%d/edit", nt.ID),
		})
	}

	totalPages := total / perPage
	if total%perPage > 0 {
		totalPages++
	}

	tabs := []map[string]interface{}{
		{"value": "all", "label": "All", "count": len(allTypes)},
	}
	if builtinCount > 0 {
		tabs = append(tabs, map[string]interface{}{"value": "builtin", "label": "Built-in", "count": builtinCount})
	}
	if customCount > 0 {
		tabs = append(tabs, map[string]interface{}{"value": "custom", "label": "Custom", "count": customCount})
	}

	activeTab := tab
	if activeTab == "" {
		activeTab = "all"
	}
	hasFilters := search != "" || (tab != "" && tab != "all")

	return &LayoutNode{
		Type:  "VerticalStack",
		Props: map[string]interface{}{"gap": 0},
		Children: []LayoutNode{
			{Type: "PageHeader", Props: map[string]interface{}{
				"newLabel":  "New Content Type",
				"newPath":   "/admin/content-types/new",
				"tabs":      tabs,
				"activeTab": activeTab,
				"tabParam":  "tab",
			}},
			{Type: "SearchToolbar", Props: map[string]interface{}{
				"searchPlaceholder": "Search content types…",
			}},
			{Type: "GenericListTable", Props: map[string]interface{}{
				"columns": []map[string]interface{}{
					{"key": "label", "label": "Label", "sortable": true},
					{"key": "slug", "label": "Slug", "width": 160, "sortable": true},
					{"key": "taxonomyCount", "label": "Taxonomies", "width": 110, "align": "center"},
					{"key": "supportsBlocks", "label": "Blocks", "width": 80, "align": "center"},
					{"key": "sourceLabel", "label": "Source", "width": 110},
					{"key": "updated_at", "label": "Updated", "width": 110, "sortable": true},
					{"key": "actions", "label": "Actions", "width": 120, "align": "right"},
				},
				"rows":       rows,
				"emptyIcon":  "Shapes",
				"emptyTitle": "No content types found",
				"emptyDesc":  "Create your first content type to model custom data",
				"newPath":    "/admin/content-types/new",
				"newLabel":   "New Content Type",
				"pagination": map[string]interface{}{
					"page": page, "perPage": perPage,
					"total": total, "totalPages": totalPages,
				},
				"label":      "content types",
				"hasFilters": hasFilters,
				"sortBy":     sortBy,
				"sortOrder":  sortOrder,
			}, Actions: map[string]ActionDef{
				"onRowDelete": {
					Type: "SEQUENCE",
					Steps: []ActionDef{
						{Type: "CONFIRM", Message: "Delete this content type? This cannot be undone."},
						{Type: "CORE_API", Method: "node-types:delete", Params: map[string]interface{}{"id": "$event.id"}},
						{Type: "TOAST", Message: "Content type deleted", Variant: "success"},
						{Type: "INVALIDATE", Keys: []string{"layout", "boot"}},
					},
				},
			}},
		},
	}
}

// ternary returns a when cond is true, b otherwise. Small helper to keep
// the row-mapping code readable.
func ternary[T any](cond bool, a, b T) T {
	if cond {
		return a
	}
	return b
}

func (e *Engine) taxonomiesLayout(params map[string]string) *LayoutNode {
	page, _ := strconv.Atoi(params["page"])
	if page < 1 {
		page = 1
	}
	perPage := getPerPage(params)
	tab := params["tab"]
	search := params["search"]

	sortBy := params["sort"]
	sortOrder := params["order"]
	switch sortBy {
	case "label", "slug", "updated_at":
	default:
		sortBy = "label"
	}
	if sortOrder != "asc" && sortOrder != "desc" {
		sortOrder = "asc"
	}

	// Resolve node-type display labels so pills and tabs read nicely.
	var nodeTypes []models.NodeType
	e.db.Select("slug, label, label_plural").Find(&nodeTypes)
	nodeTypeLabels := make(map[string]string, len(nodeTypes))
	for _, nt := range nodeTypes {
		label := nt.LabelPlural
		if label == "" {
			label = nt.Label
		}
		if label == "" {
			label = nt.Slug
		}
		nodeTypeLabels[nt.Slug] = label
	}

	q := e.db.Model(&models.Taxonomy{})
	if search != "" {
		q = q.Where("label ILIKE ? OR slug ILIKE ?", "%"+search+"%", "%"+search+"%")
	}
	var allTaxonomies []models.Taxonomy
	q.Order(sortBy + " " + sortOrder).Find(&allTaxonomies)

	// Tab counts per node-type slug. A taxonomy attached to multiple node
	// types shows up in each tab so the counts add up to the usage count,
	// not the distinct taxonomy count.
	nodeTypeTabCounts := make(map[string]int)
	for _, t := range allTaxonomies {
		for _, nts := range t.NodeTypes {
			nodeTypeTabCounts[nts]++
		}
	}

	filtered := allTaxonomies
	if tab != "" && tab != "all" {
		filtered = filtered[:0]
		for _, t := range allTaxonomies {
			for _, nts := range t.NodeTypes {
				if nts == tab {
					filtered = append(filtered, t)
					break
				}
			}
		}
	}

	total := len(filtered)
	offset := (page - 1) * perPage
	end := offset + perPage
	if end > total {
		end = total
	}
	pageData := []models.Taxonomy{}
	if offset < total {
		pageData = filtered[offset:end]
	}

	rows := make([]map[string]interface{}, 0, len(pageData))
	for _, t := range pageData {
		nodeTypesArr := make([]string, len(t.NodeTypes))
		labels := make([]string, 0, len(t.NodeTypes))
		for i, s := range t.NodeTypes {
			nodeTypesArr[i] = s
			if l, ok := nodeTypeLabels[s]; ok {
				labels = append(labels, l)
			} else {
				labels = append(labels, s)
			}
		}
		nodeTypesDisplay := "—"
		if len(labels) > 0 {
			nodeTypesDisplay = strings.Join(labels, ", ")
		}

		rows = append(rows, map[string]interface{}{
			"id":               t.ID,
			"slug":             t.Slug,
			"label":            t.Label,
			"labelPlural":      t.LabelPlural,
			"description":      t.Description,
			"hierarchical":     t.Hierarchical,
			"nodeTypes":        nodeTypesArr,
			"nodeTypesDisplay": nodeTypesDisplay,
			"updated_at":       t.UpdatedAt.Format("2006-01-02"),
			"editPath":         fmt.Sprintf("/admin/taxonomies/%s/edit", t.Slug),
		})
	}

	totalPages := total / perPage
	if total%perPage > 0 {
		totalPages++
	}

	// Build tabs: All + one per node-type slug with at least one attached taxonomy.
	tabs := []map[string]interface{}{
		{"value": "all", "label": "All", "count": len(allTaxonomies)},
	}
	slugs := make([]string, 0, len(nodeTypeTabCounts))
	for s := range nodeTypeTabCounts {
		slugs = append(slugs, s)
	}
	sort.Strings(slugs)
	for _, s := range slugs {
		label := s
		if l, ok := nodeTypeLabels[s]; ok {
			label = l
		}
		tabs = append(tabs, map[string]interface{}{
			"value": s,
			"label": label,
			"count": nodeTypeTabCounts[s],
		})
	}

	activeTab := tab
	if activeTab == "" {
		activeTab = "all"
	}
	hasFilters := search != "" || (tab != "" && tab != "all")

	return &LayoutNode{
		Type:  "VerticalStack",
		Props: map[string]interface{}{"gap": 0},
		Children: []LayoutNode{
			{Type: "PageHeader", Props: map[string]interface{}{
				"newLabel":  "New Taxonomy",
				"newPath":   "/admin/taxonomies/new",
				"tabs":      tabs,
				"activeTab": activeTab,
				"tabParam":  "tab",
			}},
			{Type: "SearchToolbar", Props: map[string]interface{}{
				"searchPlaceholder": "Search taxonomies…",
			}},
			{Type: "GenericListTable", Props: map[string]interface{}{
				"columns": []map[string]interface{}{
					{"key": "label", "label": "Label", "sortable": true},
					{"key": "slug", "label": "Slug", "width": 160, "sortable": true},
					{"key": "nodeTypesDisplay", "label": "Content Types"},
					{"key": "hierarchical", "label": "Hierarchical", "width": 110, "align": "center"},
					{"key": "updated_at", "label": "Updated", "width": 110, "sortable": true},
					{"key": "actions", "label": "Actions", "width": 120, "align": "right"},
				},
				"rows":       rows,
				"emptyIcon":  "Tags",
				"emptyTitle": "No taxonomies found",
				"emptyDesc":  "Create your first taxonomy to group content with terms",
				"newPath":    "/admin/taxonomies/new",
				"newLabel":   "New Taxonomy",
				"pagination": map[string]interface{}{
					"page": page, "perPage": perPage,
					"total": total, "totalPages": totalPages,
				},
				"label":      "taxonomies",
				"hasFilters": hasFilters,
				"sortBy":     sortBy,
				"sortOrder":  sortOrder,
			}, Actions: map[string]ActionDef{
				"onRowDelete": {
					Type: "SEQUENCE",
					Steps: []ActionDef{
						{Type: "CONFIRM", Message: "Delete this taxonomy? This cannot be undone."},
						{Type: "CORE_API", Method: "taxonomies:delete", Params: map[string]interface{}{"slug": "$event.slug"}},
						{Type: "TOAST", Message: "Taxonomy deleted", Variant: "success"},
						{Type: "INVALIDATE", Keys: []string{"layout", "boot"}},
					},
				},
			}},
		},
	}
}

func (e *Engine) nodeListLayout(params map[string]string) *LayoutNode {
	nodeTypeSlug := params["nodeType"]
	if nodeTypeSlug == "" {
		return e.defaultLayout("node-list")
	}

	// 1. Get NodeType by slug
	var nt models.NodeType
	if err := e.db.Where("slug = ?", nodeTypeSlug).First(&nt).Error; err != nil {
		return e.defaultLayout("node-list")
	}

	// Resolve display labels
	labelPlural := nt.LabelPlural
	if labelPlural == "" {
		labelPlural = nt.Label
	}

	// 2. Get taxonomy definitions for this node type
	var taxSlugs []string
	if err := json.Unmarshal(nt.Taxonomies, &taxSlugs); err != nil {
		taxSlugs = []string{}
	}

	var taxonomyDefs []models.Taxonomy
	if len(taxSlugs) > 0 {
		e.db.Where("slug IN ?", taxSlugs).Order("label ASC").Find(&taxonomyDefs)
	}

	taxonomyDefsList := make([]map[string]interface{}, 0, len(taxonomyDefs))
	for _, t := range taxonomyDefs {
		taxonomyDefsList = append(taxonomyDefsList, map[string]interface{}{
			"slug":  t.Slug,
			"label": t.Label,
		})
	}

	// 3. Build base query for ContentNode
	page, _ := strconv.Atoi(params["page"])
	if page < 1 {
		page = 1
	}
	perPage := getPerPage(params)
	offset := (page - 1) * perPage

	sortBy := params["sort"]
	sortOrder := params["order"]
	switch sortBy {
	case "title", "updated_at", "created_at":
	default:
		sortBy = "updated_at"
	}
	if sortOrder != "asc" && sortOrder != "desc" {
		sortOrder = "desc"
	}

	query := e.db.Model(&models.ContentNode{}).Where("node_type = ? AND deleted_at IS NULL", nodeTypeSlug)

	// Optional search filter (applied before status counting)
	if search := params["search"]; search != "" {
		query = query.Where("title ILIKE ?", "%"+search+"%")
	}

	// Optional language filter (applied before status counting)
	if lang := params["language"]; lang != "" && lang != "all" {
		query = query.Where("language_code = ?", lang)
	}

	// Apply taxonomy term filters from URL params
	activeTaxFilters := make([]map[string]interface{}, 0)
	for _, taxDef := range taxonomyDefs {
		if termName := params[taxDef.Slug]; termName != "" {
			query = query.Where("taxonomies->? @> ?::jsonb", taxDef.Slug, fmt.Sprintf(`["%s"]`, termName))
			activeTaxFilters = append(activeTaxFilters, map[string]interface{}{
				"taxonomy": taxDef.Slug,
				"term":     termName,
				"label":    taxDef.Label,
			})
		}
	}

	// 4. Query status counts (unfiltered by status, but filtered by node_type, search, language, taxonomy)
	baseCountQuery := "SELECT status, COUNT(*) as count FROM content_nodes WHERE node_type = ? AND deleted_at IS NULL"
	countArgs := []interface{}{nodeTypeSlug}
	if search := params["search"]; search != "" {
		baseCountQuery += " AND title ILIKE ?"
		countArgs = append(countArgs, "%"+search+"%")
	}
	if lang := params["language"]; lang != "" && lang != "all" {
		baseCountQuery += " AND language_code = ?"
		countArgs = append(countArgs, lang)
	}
	for _, taxDef := range taxonomyDefs {
		if termName := params[taxDef.Slug]; termName != "" {
			baseCountQuery += " AND taxonomies->? @> ?::jsonb"
			countArgs = append(countArgs, taxDef.Slug, fmt.Sprintf(`["%s"]`, termName))
		}
	}
	baseCountQuery += " GROUP BY status"

	type statusCount struct {
		Status string
		Count  int64
	}
	var statusCounts []statusCount
	e.db.Raw(baseCountQuery, countArgs...).Scan(&statusCounts)

	statusMap := make(map[string]int64)
	var totalAll int64
	for _, sc := range statusCounts {
		statusMap[sc.Status] = sc.Count
		totalAll += sc.Count
	}

	tabs := []map[string]interface{}{
		{"value": "all", "label": "All", "count": totalAll},
		{"value": "published", "label": "Published", "count": statusMap["published"]},
		{"value": "draft", "label": "Drafts", "count": statusMap["draft"]},
		{"value": "archived", "label": "Archived", "count": statusMap["archived"]},
	}

	// 5. Query languages
	var languages []models.Language
	e.db.Where("is_active = ?", true).Order("sort_order ASC, name ASC").Find(&languages)

	langList := make([]map[string]interface{}, 0, len(languages))
	for _, lang := range languages {
		langList = append(langList, map[string]interface{}{
			"code": lang.Code,
			"name": lang.Name,
			"flag": lang.Flag,
		})
	}

	// 6. Apply status filter for pagination
	if status := params["status"]; status != "" && status != "all" {
		query = query.Where("status = ?", status)
	}

	// 7. Count total matching nodes (with status filter applied)
	var totalCount int64
	query.Count(&totalCount)

	// 8. Fetch paginated nodes
	var nodes []models.ContentNode
	query.Order(sortBy + " " + sortOrder).Offset(offset).Limit(perPage).Find(&nodes)

	// Calculate base path
	basePath := basePathForNodeType(nodeTypeSlug)

	// 9. Build rows
	rows := make([]map[string]interface{}, 0, len(nodes))
	for _, n := range nodes {
		// Parse taxonomies JSONB — map of taxonomy_slug → []term_names
		var nodeTax map[string][]string
		if err := json.Unmarshal(n.Taxonomies, &nodeTax); err != nil {
			nodeTax = map[string][]string{}
		}

		rows = append(rows, map[string]interface{}{
			"id":            n.ID,
			"title":         n.Title,
			"slug":          n.Slug,
			"status":        n.Status,
			"language_code": n.LanguageCode,
			"taxonomies":    nodeTax,
			"updated_at":    n.UpdatedAt.Format("2006-01-02"),
			"editPath":      fmt.Sprintf("%s/%d/edit", basePath, n.ID),
		})
	}

	totalPages := int(totalCount) / perPage
	if int(totalCount)%perPage > 0 {
		totalPages++
	}

	// Build children dynamically to support conditional TaxonomyFilterChips
	children := []LayoutNode{
		{
			Type: "PageHeader",
			Props: map[string]interface{}{
				"title": labelPlural,
				"tabs":  tabs,
				"activeTab": func() string {
					if s := params["status"]; s != "" && s != "all" {
						return s
					}
					return "all"
				}(),
				"newLabel":         "New " + nt.Label,
				"newPath":          basePath + "/new",
				"taxonomyDefs":     taxonomyDefsList,
				"activeTaxFilters": activeTaxFilters,
			},
			Actions: map[string]ActionDef{
				"onNew": {Type: "NAVIGATE", To: basePath + "/new"},
			},
		},
	}

	// Add taxonomy filter chips if any active
	if len(activeTaxFilters) > 0 {
		children = append(children, LayoutNode{
			Type: "TaxonomyFilterChips",
			Props: map[string]interface{}{
				"filters": activeTaxFilters,
			},
		})
	}

	children = append(children, []LayoutNode{
		{
			Type: "SearchToolbar",
			Props: map[string]interface{}{
				"searchPlaceholder": "Search by title or slug…",
				"languages":         langList,
				"activeLanguage":    params["language"],
			},
		},
		{
			Type: "ContentNodeTable",
			Props: map[string]interface{}{
				"nodeType":            nodeTypeSlug,
				"columns":             []string{"title", "status", "taxonomies", "language", "updated_at", "actions"},
				"rows":                rows,
				"pagination":          map[string]interface{}{"page": page, "perPage": perPage, "total": int(totalCount), "totalPages": totalPages},
				"taxonomyDefs":        taxonomyDefsList,
				"basePath":            basePath,
				"nodeTypeLabel":       nt.Label,
				"nodeTypeLabelPlural": labelPlural,
				"hasActiveFilters":    len(activeTaxFilters) > 0 || params["search"] != "" || (params["status"] != "" && params["status"] != "all"),
				"sortBy":              sortBy,
				"sortOrder":           sortOrder,
			},
			Actions: map[string]ActionDef{
				"onRowDelete": {
					Type: "SEQUENCE",
					Steps: []ActionDef{
						{Type: "CONFIRM", Message: "Delete this item? This cannot be undone."},
						{Type: "CORE_API", Method: "nodes:delete", Params: map[string]interface{}{"id": "$event.id"}},
						{Type: "TOAST", Message: "Deleted", Variant: "success"},
						{Type: "INVALIDATE", Keys: []string{"layout"}},
					},
				},
			},
		},
	}...)

	return &LayoutNode{
		Type:     "VerticalStack",
		Props:    map[string]interface{}{"gap": 0},
		Children: children,
	}
}

func (e *Engine) taxonomyTermsLayout(params map[string]string) *LayoutNode {
	taxonomySlug := params["taxonomy"]
	nodeTypeSlug := params["nodeType"]
	if taxonomySlug == "" || nodeTypeSlug == "" {
		return e.defaultLayout("taxonomy-terms")
	}

	// 1. Get Taxonomy by slug
	var tax models.Taxonomy
	if err := e.db.Where("slug = ?", taxonomySlug).First(&tax).Error; err != nil {
		return e.defaultLayout("taxonomy-terms")
	}

	// Resolve display labels
	labelPlural := tax.LabelPlural
	if labelPlural == "" {
		labelPlural = tax.Label
	}

	// 2. Get NodeType for context (best-effort, doesn't fail layout)
	var nt models.NodeType
	e.db.Where("slug = ?", nodeTypeSlug).First(&nt)

	basePath := basePathForNodeType(nodeTypeSlug)

	// 3. Sort + search params
	termPage, _ := strconv.Atoi(params["page"])
	if termPage < 1 {
		termPage = 1
	}
	termPerPage := getPerPage(params)
	termSearch := params["search"]

	termSortBy := params["sort"]
	termSortOrder := params["order"]
	switch termSortBy {
	case "name", "count":
	default:
		termSortBy = "name"
	}
	if termSortOrder != "asc" && termSortOrder != "desc" {
		if termSortBy == "count" {
			termSortOrder = "desc"
		} else {
			termSortOrder = "asc"
		}
	}

	// 4. Query taxonomy terms with search + sort + pagination
	termQuery := e.db.Model(&models.TaxonomyTerm{}).
		Where("node_type = ? AND taxonomy = ?", nodeTypeSlug, taxonomySlug)
	if termSearch != "" {
		termQuery = termQuery.Where("name ILIKE ? OR slug ILIKE ?", "%"+termSearch+"%", "%"+termSearch+"%")
	}

	var termTotal int64
	termQuery.Count(&termTotal)

	termOffset := (termPage - 1) * termPerPage
	var terms []models.TaxonomyTerm
	termQuery.Order(termSortBy + " " + termSortOrder).Offset(termOffset).Limit(termPerPage).Find(&terms)

	// 5. Build rows
	rows := make([]map[string]interface{}, 0, len(terms))
	for _, t := range terms {
		editPath := fmt.Sprintf("/admin/content/%s/taxonomies/%s/%d/edit", nodeTypeSlug, taxonomySlug, t.ID)
		rows = append(rows, map[string]interface{}{
			"id":          t.ID,
			"name":        t.Name,
			"slug":        t.Slug,
			"description": t.Description,
			"count":       t.Count,
			"editPath":    editPath,
		})
	}

	termTotalPages := int(termTotal) / termPerPage
	if int(termTotal)%termPerPage > 0 {
		termTotalPages++
	}

	hasFilters := termSearch != ""

	return &LayoutNode{
		Type:  "VerticalStack",
		Props: map[string]interface{}{"gap": 0},
		Children: []LayoutNode{
			{
				Type: "PageHeader",
				Props: map[string]interface{}{
					"tabs":      []map[string]interface{}{{"value": "all", "label": "All", "count": int(termTotal)}},
					"activeTab": "all",
					"newLabel":  "New " + tax.Label,
				},
				Actions: map[string]ActionDef{
					"onBack": {Type: "NAVIGATE", To: basePath},
					"onNew":  {Type: "NAVIGATE", To: fmt.Sprintf("/admin/content/%s/taxonomies/%s/new", nodeTypeSlug, taxonomySlug)},
				},
			},
			{
				Type: "SearchToolbar",
				Props: map[string]interface{}{
					"searchPlaceholder": "Search terms…",
				},
			},
			{
				Type: "TaxonomyTermsTable",
				Props: map[string]interface{}{
					"taxonomy":         taxonomySlug,
					"nodeType":         nodeTypeSlug,
					"rows":             rows,
					"sortBy":           termSortBy,
					"sortOrder":        termSortOrder,
					"hasActiveFilters": hasFilters,
					"pagination": map[string]interface{}{
						"page": termPage, "perPage": termPerPage,
						"total": int(termTotal), "totalPages": termTotalPages,
					},
				},
				Actions: map[string]ActionDef{
					"onRowDelete": {
						Type: "SEQUENCE",
						Steps: []ActionDef{
							{Type: "CONFIRM", Message: "Delete this term? This cannot be undone."},
							{Type: "CORE_API", Method: "terms:delete", Params: map[string]interface{}{"id": "$event.id"}},
							{Type: "TOAST", Message: "Term deleted", Variant: "success"},
							{Type: "INVALIDATE", Keys: []string{"layout"}},
						},
					},
				},
			},
		},
	}
}

func sourceTabs(counts map[string]int, total int) []map[string]interface{} {
	tabs := []map[string]interface{}{
		{"value": "all", "label": "All", "count": total},
	}
	for _, s := range []string{"custom", "theme", "extension"} {
		if n := counts[s]; n > 0 {
			label := s
			label = string([]rune(label)[:1]) + label[1:]
			tabs = append(tabs, map[string]interface{}{"value": s, "label": label[:1] + label[1:], "count": n})
		}
	}
	return tabs
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return string([]rune(s[:1])) + s[1:]
}

// prettySlug converts a kebab-case slug to Title Case ("my-theme" → "My Theme").
func prettySlug(s string) string {
	words := strings.Split(s, "-")
	for i, w := range words {
		words[i] = capitalize(w)
	}
	return strings.Join(words, " ")
}

// themeTabInfo holds per-theme tab display data.
type themeTabInfo struct {
	name  string
	count int
}

// sourceDisplayLabel returns a human-readable label for a record's source field.
// themeNames maps theme/extension slug → display name.
func sourceDisplayLabel(source string, themeName *string, themeNames map[string]string, extNames map[string]string) string {
	switch source {
	case "custom":
		return "Custom"
	case "extension":
		if themeName != nil {
			if name, ok := extNames[*themeName]; ok && name != "" {
				return name
			}
			return prettySlug(*themeName)
		}
		return "Extension"
	case "theme":
		if themeName != nil {
			if name, ok := themeNames[*themeName]; ok && name != "" {
				return name
			}
			return prettySlug(*themeName)
		}
		return "Theme"
	}
	return capitalize(source)
}

// buildSourceTabs creates tab entries where theme/extension items are each broken out by
// their individual display name. Tab value = slug, so ?source=<slug> filters precisely.
func buildSourceTabs(total, customCount int, themeTabMap, extTabMap map[string]*themeTabInfo) []map[string]interface{} {
	tabs := []map[string]interface{}{{"value": "all", "label": "All", "count": total}}
	if customCount > 0 {
		tabs = append(tabs, map[string]interface{}{"value": "custom", "label": "Custom", "count": customCount})
	}
	// Theme tabs sorted alphabetically by slug
	themeSlugs := make([]string, 0, len(themeTabMap))
	for slug := range themeTabMap {
		themeSlugs = append(themeSlugs, slug)
	}
	sort.Strings(themeSlugs)
	for _, slug := range themeSlugs {
		info := themeTabMap[slug]
		tabs = append(tabs, map[string]interface{}{"value": slug, "label": info.name, "count": info.count})
	}
	// Extension tabs sorted alphabetically by slug
	extSlugs := make([]string, 0, len(extTabMap))
	for slug := range extTabMap {
		extSlugs = append(extSlugs, slug)
	}
	sort.Strings(extSlugs)
	for _, slug := range extSlugs {
		info := extTabMap[slug]
		tabs = append(tabs, map[string]interface{}{"value": "ext:" + slug, "label": info.name, "count": info.count})
	}
	return tabs
}

// isThemeSlugFilter returns true when sourceFilter holds a theme slug.
func isThemeSlugFilter(s string) bool {
	switch s {
	case "", "all", "custom":
		return false
	}
	return !strings.HasPrefix(s, "ext:")
}

// isExtSlugFilter returns true when sourceFilter holds an extension slug (prefixed with "ext:").
func isExtSlugFilter(s string) bool {
	return strings.HasPrefix(s, "ext:")
}

// getPerPage returns the per-page size from params, clamped to [5, 100], defaulting to 10.
func getPerPage(params map[string]string) int {
	if v, err := strconv.Atoi(params["per_page"]); err == nil && v >= 5 && v <= 100 {
		return v
	}
	return 10
}

// themeNameMap fetches a lookup table for resolving a stored ThemeName value
// to its proper display name. Indexed by both slug and display name so that
// it works regardless of which value the model field stores.
func (e *Engine) themeNameMap() map[string]string {
	var themes []models.Theme
	e.db.Select("slug, name").Find(&themes)
	m := make(map[string]string, len(themes)*2)
	for _, t := range themes {
		if t.Slug != "" {
			m[t.Slug] = t.Name
		}
		if t.Name != "" {
			m[t.Name] = t.Name
		}
	}
	return m
}

// extensionNameMap fetches a slug → display name map for active extensions.
func (e *Engine) extensionNameMap() map[string]string {
	var exts []models.Extension
	e.db.Select("slug, name").Find(&exts)
	m := make(map[string]string, len(exts))
	for _, ex := range exts {
		if ex.Slug != "" {
			m[ex.Slug] = ex.Name
		}
	}
	return m
}

func (e *Engine) templatesLayout(params map[string]string) *LayoutNode {
	page, _ := strconv.Atoi(params["page"])
	if page < 1 {
		page = 1
	}
	perPage := getPerPage(params)
	sourceFilter := params["source"]
	search := params["search"]

	sortBy := params["sort"]
	sortOrder := params["order"]
	switch sortBy {
	case "label", "slug", "updated_at":
	default:
		sortBy = "updated_at"
	}
	if sortOrder != "asc" && sortOrder != "desc" {
		sortOrder = "desc"
	}

	themeNames := e.themeNameMap()
	extNames := e.extensionNameMap()

	// Load all (with search filter) for tab counts
	baseQuery := e.db.Model(&models.Template{})
	if search != "" {
		baseQuery = baseQuery.Where("label ILIKE ? OR slug ILIKE ?", "%"+search+"%", "%"+search+"%")
	}
	var allTemplates []models.Template
	baseQuery.Order(sortBy + " " + sortOrder).Find(&allTemplates)

	// Build per-source counts (theme/extension items grouped by their slug)
	customCount := 0
	themeTabMap := map[string]*themeTabInfo{}
	extTabMap := map[string]*themeTabInfo{}
	for _, t := range allTemplates {
		switch t.Source {
		case "custom":
			customCount++
		case "extension":
			slug := ""
			if t.ThemeName != nil {
				slug = *t.ThemeName
			}
			if _, ok := extTabMap[slug]; !ok {
				extTabMap[slug] = &themeTabInfo{name: sourceDisplayLabel("extension", t.ThemeName, themeNames, extNames)}
			}
			extTabMap[slug].count++
		case "theme":
			slug := ""
			if t.ThemeName != nil {
				slug = *t.ThemeName
			}
			if _, ok := themeTabMap[slug]; !ok {
				themeTabMap[slug] = &themeTabInfo{name: sourceDisplayLabel("theme", t.ThemeName, themeNames, extNames)}
			}
			themeTabMap[slug].count++
		}
	}

	// Filter in-memory: theme slug filters by source=theme + theme_name=slug,
	// ext:<slug> filters by source=extension + theme_name=slug.
	filtered := allTemplates
	if sourceFilter != "" && sourceFilter != "all" {
		filtered = filtered[:0]
		for _, t := range allTemplates {
			if isExtSlugFilter(sourceFilter) {
				extSlug := strings.TrimPrefix(sourceFilter, "ext:")
				if t.Source == "extension" && t.ThemeName != nil && *t.ThemeName == extSlug {
					filtered = append(filtered, t)
				}
			} else if isThemeSlugFilter(sourceFilter) {
				if t.Source == "theme" && t.ThemeName != nil && *t.ThemeName == sourceFilter {
					filtered = append(filtered, t)
				}
			} else if t.Source == sourceFilter {
				filtered = append(filtered, t)
			}
		}
	}

	total := len(filtered)
	offset := (page - 1) * perPage
	end := offset + perPage
	if end > total {
		end = total
	}
	pageData := filtered
	if offset < total {
		pageData = filtered[offset:end]
	} else {
		pageData = []models.Template{}
	}

	rows := make([]map[string]interface{}, 0, len(pageData))
	for _, t := range pageData {
		var blockConfigs []interface{}
		if err := json.Unmarshal(t.BlockConfig, &blockConfigs); err != nil {
			blockConfigs = []interface{}{}
		}
		description := t.Description
		if description == "" {
			description = "—"
		}
		rows = append(rows, map[string]interface{}{
			"id":          t.ID,
			"label":       t.Label,
			"slug":        t.Slug,
			"description": description,
			"blockCount":  len(blockConfigs),
			"source":      t.Source,
			"sourceLabel": sourceDisplayLabel(t.Source, t.ThemeName, themeNames, extNames),
			"isCustom":    t.Source == "custom",
			"updated_at":  t.UpdatedAt.Format("2006-01-02"),
			"editPath":    fmt.Sprintf("/admin/templates/%d/edit", t.ID),
		})
	}

	totalPages := total / perPage
	if total%perPage > 0 {
		totalPages++
	}

	tabs := buildSourceTabs(len(allTemplates), customCount, themeTabMap, extTabMap)

	activeTab := sourceFilter
	if activeTab == "" {
		activeTab = "all"
	}

	hasFilters := search != "" || (sourceFilter != "" && sourceFilter != "all")

	return &LayoutNode{
		Type:  "VerticalStack",
		Props: map[string]interface{}{"gap": 0},
		Children: []LayoutNode{
			{Type: "PageHeader", Props: map[string]interface{}{
				"newLabel":  "New Template",
				"newPath":   "/admin/templates/new",
				"tabs":      tabs,
				"activeTab": activeTab,
				"tabParam":  "source",
			}},
			{Type: "SearchToolbar", Props: map[string]interface{}{
				"searchPlaceholder": "Search templates…",
			}},
			{Type: "GenericListTable", Props: map[string]interface{}{
				"columns": []map[string]interface{}{
					{"key": "label", "label": "Label", "sortable": true},
					{"key": "blockCount", "label": "Blocks", "width": 80, "align": "center"},
					{"key": "sourceLabel", "label": "Source", "width": 130},
					{"key": "description", "label": "Description"},
					{"key": "updated_at", "label": "Updated", "width": 110, "sortable": true},
					{"key": "actions", "label": "Actions", "width": 120, "align": "right"},
				},
				"rows":       rows,
				"emptyIcon":  "LayoutTemplate",
				"emptyTitle": "No templates found",
				"emptyDesc":  "Create your first template to get started",
				"newPath":    "/admin/templates/new",
				"newLabel":   "New Template",
				"pagination": map[string]interface{}{
					"page": page, "perPage": perPage,
					"total": total, "totalPages": totalPages,
				},
				"label":      "templates",
				"hasFilters": hasFilters,
				"sortBy":     sortBy,
				"sortOrder":  sortOrder,
			}, Actions: map[string]ActionDef{
				"onRowDelete": {
					Type: "SEQUENCE",
					Steps: []ActionDef{
						{Type: "CONFIRM", Message: "Delete this template? This cannot be undone."},
						{Type: "CORE_API", Method: "templates:delete", Params: map[string]interface{}{"id": "$event.id"}},
						{Type: "TOAST", Message: "Template deleted", Variant: "success"},
						{Type: "INVALIDATE", Keys: []string{"layout"}},
					},
				},
				"onRowDetach": {
					Type: "SEQUENCE",
					Steps: []ActionDef{
						{Type: "CORE_API", Method: "templates:detach", Params: map[string]interface{}{"id": "$event.id"}},
						{Type: "TOAST", Message: "Detached from source", Variant: "success"},
						{Type: "INVALIDATE", Keys: []string{"layout"}},
					},
				},
			}},
		},
	}
}

func (e *Engine) layoutsLayout(params map[string]string) *LayoutNode {
	page, _ := strconv.Atoi(params["page"])
	if page < 1 {
		page = 1
	}
	perPage := getPerPage(params)
	sourceFilter := params["source"]
	search := params["search"]

	sortBy := params["sort"]
	sortOrder := params["order"]
	switch sortBy {
	case "name", "slug", "updated_at":
	default:
		sortBy = "updated_at"
	}
	if sortOrder != "asc" && sortOrder != "desc" {
		sortOrder = "desc"
	}

	// Fetch languages for display and filter
	var languages []models.Language
	e.db.Where("is_active = ?", true).Order("sort_order ASC, name ASC").Find(&languages)
	langMap := make(map[int]models.Language)
	langList := make([]map[string]interface{}, 0, len(languages))
	for _, l := range languages {
		langMap[l.ID] = l
		langList = append(langList, map[string]interface{}{
			"id":   l.ID,
			"code": l.Code,
			"name": l.Name,
			"flag": l.Flag,
		})
	}

	themeNames := e.themeNameMap()
	extNames := e.extensionNameMap()

	// Base query with language + search filters
	baseQuery := e.db.Model(&models.Layout{})
	if lang := params["language"]; lang != "" && lang != "all" {
		if langID, err := strconv.Atoi(lang); err == nil {
			baseQuery = baseQuery.Where("language_id = ?", langID)
		}
	}
	if search != "" {
		baseQuery = baseQuery.Where("name ILIKE ? OR slug ILIKE ?", "%"+search+"%", "%"+search+"%")
	}

	// Load all (for source counts, preserving sort order)
	var allLayouts []models.Layout
	baseQuery.Order(sortBy + " " + sortOrder).Find(&allLayouts)

	customCount := 0
	themeTabMap := map[string]*themeTabInfo{}
	extTabMap := map[string]*themeTabInfo{}
	for _, l := range allLayouts {
		switch l.Source {
		case "custom":
			customCount++
		case "extension":
			slug := ""
			if l.ThemeName != nil {
				slug = *l.ThemeName
			}
			if _, ok := extTabMap[slug]; !ok {
				extTabMap[slug] = &themeTabInfo{name: sourceDisplayLabel("extension", l.ThemeName, themeNames, extNames)}
			}
			extTabMap[slug].count++
		case "theme":
			slug := ""
			if l.ThemeName != nil {
				slug = *l.ThemeName
			}
			if _, ok := themeTabMap[slug]; !ok {
				themeTabMap[slug] = &themeTabInfo{name: sourceDisplayLabel("theme", l.ThemeName, themeNames, extNames)}
			}
			themeTabMap[slug].count++
		}
	}

	filtered := allLayouts
	if sourceFilter != "" && sourceFilter != "all" {
		filtered = filtered[:0]
		for _, l := range allLayouts {
			if isExtSlugFilter(sourceFilter) {
				extSlug := strings.TrimPrefix(sourceFilter, "ext:")
				if l.Source == "extension" && l.ThemeName != nil && *l.ThemeName == extSlug {
					filtered = append(filtered, l)
				}
			} else if isThemeSlugFilter(sourceFilter) {
				if l.Source == "theme" && l.ThemeName != nil && *l.ThemeName == sourceFilter {
					filtered = append(filtered, l)
				}
			} else if l.Source == sourceFilter {
				filtered = append(filtered, l)
			}
		}
	}

	total := len(filtered)
	offset := (page - 1) * perPage
	end := offset + perPage
	if end > total {
		end = total
	}
	pageData := filtered
	if offset < total {
		pageData = filtered[offset:end]
	} else {
		pageData = []models.Layout{}
	}

	rows := make([]map[string]interface{}, 0, len(pageData))
	for _, l := range pageData {
		langDisplay := "All"
		var langFlag, langCode string
		if l.LanguageID != nil {
			if lang, ok := langMap[*l.LanguageID]; ok {
				langDisplay = lang.Name
				langFlag = lang.Flag
				langCode = lang.Code
			} else {
				langDisplay = fmt.Sprintf("ID %d", *l.LanguageID)
			}
		}
		rows = append(rows, map[string]interface{}{
			"id":          l.ID,
			"name":        l.Name,
			"slug":        l.Slug,
			"source":      l.Source,
			"sourceLabel": sourceDisplayLabel(l.Source, l.ThemeName, themeNames, extNames),
			"isCustom":    l.Source == "custom",
			"isDefault":   l.IsDefault,
			"languageID":  l.LanguageID,
			"langDisplay": langDisplay,
			"langFlag":    langFlag,
			"langCode":    langCode,
			"updated_at":  l.UpdatedAt.Format("2006-01-02"),
			"editPath":    fmt.Sprintf("/admin/layouts/%d", l.ID),
		})
	}

	totalPages := total / perPage
	if total%perPage > 0 {
		totalPages++
	}

	tabs := buildSourceTabs(len(allLayouts), customCount, themeTabMap, extTabMap)

	activeTab := sourceFilter
	if activeTab == "" {
		activeTab = "all"
	}

	hasFilters := search != "" || (params["language"] != "" && params["language"] != "all") || (sourceFilter != "" && sourceFilter != "all")

	return &LayoutNode{
		Type:  "VerticalStack",
		Props: map[string]interface{}{"gap": 0},
		Children: []LayoutNode{
			{Type: "PageHeader", Props: map[string]interface{}{
				"newLabel":  "New Layout",
				"newPath":   "/admin/layouts/new",
				"tabs":      tabs,
				"activeTab": activeTab,
				"tabParam":  "source",
			}},
			{Type: "SearchToolbar", Props: map[string]interface{}{
				"searchPlaceholder": "Search layouts…",
				"languages":         langList,
				"activeLanguage":    params["language"],
			}},
			{Type: "GenericListTable", Props: map[string]interface{}{
				"columns": []map[string]interface{}{
					{"key": "name", "label": "Name", "sortable": true},
					{"key": "slug", "label": "Slug", "width": 180},
					{"key": "langDisplay", "label": "Language", "width": 130},
					{"key": "sourceLabel", "label": "Source", "width": 130},
					{"key": "isDefault", "label": "Default", "width": 90},
					{"key": "updated_at", "label": "Updated", "width": 110, "sortable": true},
					{"key": "actions", "label": "Actions", "width": 140, "align": "right"},
				},
				"rows":       rows,
				"emptyIcon":  "LayoutTemplate",
				"emptyTitle": "No layouts found",
				"emptyDesc":  "Create your first layout to get started",
				"newPath":    "/admin/layouts/new",
				"newLabel":   "New Layout",
				"pagination": map[string]interface{}{
					"page": page, "perPage": perPage,
					"total": total, "totalPages": totalPages,
				},
				"label":      "layouts",
				"hasFilters": hasFilters,
				"sortBy":     sortBy,
				"sortOrder":  sortOrder,
			}, Actions: map[string]ActionDef{
				"onRowDelete": {
					Type: "SEQUENCE",
					Steps: []ActionDef{
						{Type: "CONFIRM", Message: "Delete this layout? This cannot be undone."},
						{Type: "CORE_API", Method: "layouts:delete", Params: map[string]interface{}{"id": "$event.id"}},
						{Type: "TOAST", Message: "Layout deleted", Variant: "success"},
						{Type: "INVALIDATE", Keys: []string{"layout"}},
					},
				},
				"onRowDetach": {
					Type: "SEQUENCE",
					Steps: []ActionDef{
						{Type: "CORE_API", Method: "layouts:detach", Params: map[string]interface{}{"id": "$event.id"}},
						{Type: "TOAST", Message: "Detached from source", Variant: "success"},
						{Type: "INVALIDATE", Keys: []string{"layout"}},
					},
				},
			}},
		},
	}
}

func (e *Engine) blockTypesLayout(params map[string]string) *LayoutNode {
	page, _ := strconv.Atoi(params["page"])
	if page < 1 {
		page = 1
	}
	perPage := getPerPage(params)
	sourceFilter := params["source"]
	search := params["search"]

	sortBy := params["sort"]
	sortOrder := params["order"]
	switch sortBy {
	case "label", "slug", "updated_at":
	default:
		sortBy = "updated_at"
	}
	if sortOrder != "asc" && sortOrder != "desc" {
		sortOrder = "desc"
	}

	themeNames := e.themeNameMap()
	extNames := e.extensionNameMap()

	baseQuery := e.db.Model(&models.BlockType{})
	if search != "" {
		baseQuery = baseQuery.Where("label ILIKE ? OR slug ILIKE ?", "%"+search+"%", "%"+search+"%")
	}
	var allBlockTypes []models.BlockType
	baseQuery.Order(sortBy + " " + sortOrder).Find(&allBlockTypes)

	customCount := 0
	themeTabMap := map[string]*themeTabInfo{}
	extTabMap := map[string]*themeTabInfo{}
	for _, bt := range allBlockTypes {
		switch bt.Source {
		case "custom":
			customCount++
		case "extension":
			slug := ""
			if bt.ThemeName != nil {
				slug = *bt.ThemeName
			}
			if _, ok := extTabMap[slug]; !ok {
				extTabMap[slug] = &themeTabInfo{name: sourceDisplayLabel("extension", bt.ThemeName, themeNames, extNames)}
			}
			extTabMap[slug].count++
		case "theme":
			slug := ""
			if bt.ThemeName != nil {
				slug = *bt.ThemeName
			}
			if _, ok := themeTabMap[slug]; !ok {
				themeTabMap[slug] = &themeTabInfo{name: sourceDisplayLabel("theme", bt.ThemeName, themeNames, extNames)}
			}
			themeTabMap[slug].count++
		}
	}

	filtered := allBlockTypes
	if sourceFilter != "" && sourceFilter != "all" {
		filtered = filtered[:0]
		for _, bt := range allBlockTypes {
			if isExtSlugFilter(sourceFilter) {
				extSlug := strings.TrimPrefix(sourceFilter, "ext:")
				if bt.Source == "extension" && bt.ThemeName != nil && *bt.ThemeName == extSlug {
					filtered = append(filtered, bt)
				}
			} else if isThemeSlugFilter(sourceFilter) {
				if bt.Source == "theme" && bt.ThemeName != nil && *bt.ThemeName == sourceFilter {
					filtered = append(filtered, bt)
				}
			} else if bt.Source == sourceFilter {
				filtered = append(filtered, bt)
			}
		}
	}

	total := len(filtered)
	offset := (page - 1) * perPage
	end := offset + perPage
	if end > total {
		end = total
	}
	pageData := filtered
	if offset < total {
		pageData = filtered[offset:end]
	} else {
		pageData = []models.BlockType{}
	}

	rows := make([]map[string]interface{}, 0, len(pageData))
	for _, bt := range pageData {
		var fields []interface{}
		if err := json.Unmarshal(bt.FieldSchema, &fields); err != nil {
			fields = []interface{}{}
		}
		description := bt.Description
		if description == "" {
			description = "—"
		}
		rows = append(rows, map[string]interface{}{
			"id":          bt.ID,
			"label":       bt.Label,
			"slug":        bt.Slug,
			"icon":        bt.Icon,
			"description": description,
			"fieldCount":  len(fields),
			"source":      bt.Source,
			"sourceLabel": sourceDisplayLabel(bt.Source, bt.ThemeName, themeNames, extNames),
			"isCustom":    bt.Source == "custom",
			"updated_at":  bt.UpdatedAt.Format("2006-01-02"),
			"editPath":    fmt.Sprintf("/admin/block-types/%d/edit", bt.ID),
		})
	}

	totalPages := total / perPage
	if total%perPage > 0 {
		totalPages++
	}

	tabs := buildSourceTabs(len(allBlockTypes), customCount, themeTabMap, extTabMap)

	activeTab := sourceFilter
	if activeTab == "" {
		activeTab = "all"
	}

	hasFilters := search != "" || (sourceFilter != "" && sourceFilter != "all")

	return &LayoutNode{
		Type:  "VerticalStack",
		Props: map[string]interface{}{"gap": 0},
		Children: []LayoutNode{
			{Type: "PageHeader", Props: map[string]interface{}{
				"newLabel":  "New Block Type",
				"newPath":   "/admin/block-types/new",
				"tabs":      tabs,
				"activeTab": activeTab,
				"tabParam":  "source",
			}},
			{Type: "SearchToolbar", Props: map[string]interface{}{
				"searchPlaceholder": "Search block types…",
			}},
			{Type: "GenericListTable", Props: map[string]interface{}{
				"columns": []map[string]interface{}{
					{"key": "label", "label": "Label", "sortable": true},
					{"key": "slug", "label": "Slug", "width": 150},
					{"key": "fieldCount", "label": "Fields", "width": 80, "align": "center"},
					{"key": "sourceLabel", "label": "Source", "width": 130},
					{"key": "description", "label": "Description"},
					{"key": "updated_at", "label": "Updated", "width": 110, "sortable": true},
					{"key": "actions", "label": "Actions", "width": 120, "align": "right"},
				},
				"rows":       rows,
				"emptyIcon":  "Blocks",
				"emptyTitle": "No block types found",
				"emptyDesc":  "Create your first block type to get started",
				"newPath":    "/admin/block-types/new",
				"newLabel":   "New Block Type",
				"pagination": map[string]interface{}{
					"page": page, "perPage": perPage,
					"total": total, "totalPages": totalPages,
				},
				"label":      "block-types",
				"hasFilters": hasFilters,
				"sortBy":     sortBy,
				"sortOrder":  sortOrder,
			}, Actions: map[string]ActionDef{
				"onRowDelete": {
					Type: "SEQUENCE",
					Steps: []ActionDef{
						{Type: "CONFIRM", Message: "Delete this block type? This cannot be undone."},
						{Type: "CORE_API", Method: "block-types:delete", Params: map[string]interface{}{"id": "$event.id"}},
						{Type: "TOAST", Message: "Block type deleted", Variant: "success"},
						{Type: "INVALIDATE", Keys: []string{"layout"}},
					},
				},
				"onRowDetach": {
					Type: "SEQUENCE",
					Steps: []ActionDef{
						{Type: "CORE_API", Method: "block-types:detach", Params: map[string]interface{}{"id": "$event.id"}},
						{Type: "TOAST", Message: "Detached from source", Variant: "success"},
						{Type: "INVALIDATE", Keys: []string{"layout"}},
					},
				},
			}},
		},
	}
}

func (e *Engine) layoutBlocksLayout(params map[string]string) *LayoutNode {
	page, _ := strconv.Atoi(params["page"])
	if page < 1 {
		page = 1
	}
	perPage := getPerPage(params)
	sourceFilter := params["source"]
	search := params["search"]

	sortBy := params["sort"]
	sortOrder := params["order"]
	switch sortBy {
	case "name", "slug", "updated_at":
	default:
		sortBy = "updated_at"
	}
	if sortOrder != "asc" && sortOrder != "desc" {
		sortOrder = "desc"
	}

	// Fetch languages for display and filter
	var languages []models.Language
	e.db.Where("is_active = ?", true).Order("sort_order ASC, name ASC").Find(&languages)
	langMap := make(map[int]models.Language)
	langList := make([]map[string]interface{}, 0, len(languages))
	for _, l := range languages {
		langMap[l.ID] = l
		langList = append(langList, map[string]interface{}{
			"id":   l.ID,
			"code": l.Code,
			"name": l.Name,
			"flag": l.Flag,
		})
	}

	themeNames := e.themeNameMap()
	extNames := e.extensionNameMap()

	baseQuery := e.db.Model(&models.LayoutBlock{})
	if lang := params["language"]; lang != "" && lang != "all" {
		if langID, err := strconv.Atoi(lang); err == nil {
			baseQuery = baseQuery.Where("language_id = ?", langID)
		}
	}
	if search != "" {
		baseQuery = baseQuery.Where("name ILIKE ? OR slug ILIKE ?", "%"+search+"%", "%"+search+"%")
	}

	var allLayoutBlocks []models.LayoutBlock
	baseQuery.Order(sortBy + " " + sortOrder).Find(&allLayoutBlocks)

	customCount := 0
	themeTabMap := map[string]*themeTabInfo{}
	extTabMap := map[string]*themeTabInfo{}
	for _, lb := range allLayoutBlocks {
		switch lb.Source {
		case "custom":
			customCount++
		case "extension":
			slug := ""
			if lb.ThemeName != nil {
				slug = *lb.ThemeName
			}
			if _, ok := extTabMap[slug]; !ok {
				extTabMap[slug] = &themeTabInfo{name: sourceDisplayLabel("extension", lb.ThemeName, themeNames, extNames)}
			}
			extTabMap[slug].count++
		case "theme":
			slug := ""
			if lb.ThemeName != nil {
				slug = *lb.ThemeName
			}
			if _, ok := themeTabMap[slug]; !ok {
				themeTabMap[slug] = &themeTabInfo{name: sourceDisplayLabel("theme", lb.ThemeName, themeNames, extNames)}
			}
			themeTabMap[slug].count++
		}
	}

	filtered := allLayoutBlocks
	if sourceFilter != "" && sourceFilter != "all" {
		filtered = filtered[:0]
		for _, lb := range allLayoutBlocks {
			if isExtSlugFilter(sourceFilter) {
				extSlug := strings.TrimPrefix(sourceFilter, "ext:")
				if lb.Source == "extension" && lb.ThemeName != nil && *lb.ThemeName == extSlug {
					filtered = append(filtered, lb)
				}
			} else if isThemeSlugFilter(sourceFilter) {
				if lb.Source == "theme" && lb.ThemeName != nil && *lb.ThemeName == sourceFilter {
					filtered = append(filtered, lb)
				}
			} else if lb.Source == sourceFilter {
				filtered = append(filtered, lb)
			}
		}
	}

	total := len(filtered)
	offset := (page - 1) * perPage
	end := offset + perPage
	if end > total {
		end = total
	}
	pageData := filtered
	if offset < total {
		pageData = filtered[offset:end]
	} else {
		pageData = []models.LayoutBlock{}
	}

	rows := make([]map[string]interface{}, 0, len(pageData))
	for _, lb := range pageData {
		langDisplay := "All"
		var langFlag, langCode string
		if lb.LanguageID != nil {
			if lang, ok := langMap[*lb.LanguageID]; ok {
				langDisplay = lang.Name
				langFlag = lang.Flag
				langCode = lang.Code
			} else {
				langDisplay = fmt.Sprintf("ID %d", *lb.LanguageID)
			}
		}
		description := lb.Description
		if description == "" {
			description = "—"
		}
		rows = append(rows, map[string]interface{}{
			"id":          lb.ID,
			"name":        lb.Name,
			"slug":        lb.Slug,
			"description": description,
			"source":      lb.Source,
			"sourceLabel": sourceDisplayLabel(lb.Source, lb.ThemeName, themeNames, extNames),
			"isCustom":    lb.Source == "custom",
			"languageID":  lb.LanguageID,
			"langDisplay": langDisplay,
			"langFlag":    langFlag,
			"langCode":    langCode,
			"updated_at":  lb.UpdatedAt.Format("2006-01-02"),
			"editPath":    fmt.Sprintf("/admin/layout-blocks/%d/edit", lb.ID),
		})
	}

	totalPages := total / perPage
	if total%perPage > 0 {
		totalPages++
	}

	tabs := buildSourceTabs(len(allLayoutBlocks), customCount, themeTabMap, extTabMap)

	activeTab := sourceFilter
	if activeTab == "" {
		activeTab = "all"
	}

	hasFilters := search != "" || (params["language"] != "" && params["language"] != "all") || (sourceFilter != "" && sourceFilter != "all")

	return &LayoutNode{
		Type:  "VerticalStack",
		Props: map[string]interface{}{"gap": 0},
		Children: []LayoutNode{
			{Type: "PageHeader", Props: map[string]interface{}{
				"newLabel":  "New Layout Block",
				"newPath":   "/admin/layout-blocks/new",
				"tabs":      tabs,
				"activeTab": activeTab,
				"tabParam":  "source",
			}},
			{Type: "SearchToolbar", Props: map[string]interface{}{
				"searchPlaceholder": "Search layout blocks…",
				"languages":         langList,
				"activeLanguage":    params["language"],
			}},
			{Type: "GenericListTable", Props: map[string]interface{}{
				"columns": []map[string]interface{}{
					{"key": "name", "label": "Name", "sortable": true},
					{"key": "slug", "label": "Slug", "width": 180},
					{"key": "langDisplay", "label": "Language", "width": 130},
					{"key": "sourceLabel", "label": "Source", "width": 130},
					{"key": "description", "label": "Description"},
					{"key": "updated_at", "label": "Updated", "width": 110, "sortable": true},
					{"key": "actions", "label": "Actions", "width": 120, "align": "right"},
				},
				"rows":       rows,
				"emptyIcon":  "Component",
				"emptyTitle": "No layout blocks found",
				"emptyDesc":  "Create your first layout block to get started",
				"newPath":    "/admin/layout-blocks/new",
				"newLabel":   "New Layout Block",
				"pagination": map[string]interface{}{
					"page": page, "perPage": perPage,
					"total": total, "totalPages": totalPages,
				},
				"label":      "layout-blocks",
				"hasFilters": hasFilters,
				"sortBy":     sortBy,
				"sortOrder":  sortOrder,
			}, Actions: map[string]ActionDef{
				"onRowDelete": {
					Type: "SEQUENCE",
					Steps: []ActionDef{
						{Type: "CONFIRM", Message: "Delete this layout block? This cannot be undone."},
						{Type: "CORE_API", Method: "layout-blocks:delete", Params: map[string]interface{}{"id": "$event.id"}},
						{Type: "TOAST", Message: "Layout block deleted", Variant: "success"},
						{Type: "INVALIDATE", Keys: []string{"layout"}},
					},
				},
				"onRowDetach": {
					Type: "SEQUENCE",
					Steps: []ActionDef{
						{Type: "CORE_API", Method: "layout-blocks:detach", Params: map[string]interface{}{"id": "$event.id"}},
						{Type: "TOAST", Message: "Detached from source", Variant: "success"},
						{Type: "INVALIDATE", Keys: []string{"layout"}},
					},
				},
			}},
		},
	}
}

func (e *Engine) menusLayout(params map[string]string) *LayoutNode {
	page, _ := strconv.Atoi(params["page"])
	if page < 1 {
		page = 1
	}
	perPage := getPerPage(params)
	search := params["search"]

	sortBy := params["sort"]
	sortOrder := params["order"]
	switch sortBy {
	case "name", "slug", "updated_at":
	default:
		sortBy = "updated_at"
	}
	if sortOrder != "asc" && sortOrder != "desc" {
		sortOrder = "desc"
	}

	// Fetch languages for display and filter
	var languages []models.Language
	e.db.Where("is_active = ?", true).Order("sort_order ASC, name ASC").Find(&languages)
	langMap := make(map[int]models.Language)
	langList := make([]map[string]interface{}, 0, len(languages))
	for _, l := range languages {
		langMap[l.ID] = l
		langList = append(langList, map[string]interface{}{
			"id":   l.ID,
			"code": l.Code,
			"name": l.Name,
			"flag": l.Flag,
		})
	}

	query := e.db.Model(&models.Menu{})
	if lang := params["language"]; lang != "" && lang != "all" {
		if langID, err := strconv.Atoi(lang); err == nil {
			query = query.Where("language_id = ?", langID)
		}
	}
	if search != "" {
		query = query.Where("name ILIKE ? OR slug ILIKE ?", "%"+search+"%", "%"+search+"%")
	}

	var total int64
	query.Count(&total)

	offset := (page - 1) * perPage
	var menus []models.Menu
	query.Order(sortBy + " " + sortOrder).Offset(offset).Limit(perPage).Find(&menus)

	// Count menu items per menu via separate query
	menuIDs := make([]int, 0, len(menus))
	for _, m := range menus {
		menuIDs = append(menuIDs, m.ID)
	}

	itemCountMap := make(map[int]int)
	if len(menuIDs) > 0 {
		type itemCount struct {
			MenuID int
			Count  int
		}
		var counts []itemCount
		e.db.Raw("SELECT menu_id, COUNT(*) as count FROM menu_items WHERE menu_id IN (?) GROUP BY menu_id", menuIDs).Scan(&counts)
		for _, c := range counts {
			itemCountMap[c.MenuID] = c.Count
		}
	}

	rows := make([]map[string]interface{}, 0, len(menus))
	for _, m := range menus {
		langDisplay := "All"
		var langFlag, langCode string
		if m.LanguageID != nil {
			if lang, ok := langMap[*m.LanguageID]; ok {
				langDisplay = lang.Name
				langFlag = lang.Flag
				langCode = lang.Code
			} else {
				langDisplay = fmt.Sprintf("ID %d", *m.LanguageID)
			}
		}

		rows = append(rows, map[string]interface{}{
			"id":          m.ID,
			"name":        m.Name,
			"slug":        m.Slug,
			"version":     m.Version,
			"itemCount":   itemCountMap[m.ID],
			"languageID":  m.LanguageID,
			"langDisplay": langDisplay,
			"langFlag":    langFlag,
			"langCode":    langCode,
			"updated_at":  m.UpdatedAt.Format("2006-01-02"),
			"editPath":    fmt.Sprintf("/admin/menus/%d/edit", m.ID),
		})
	}

	totalPages := int(total) / perPage
	if int(total)%perPage > 0 {
		totalPages++
	}

	hasFilters := search != "" || (params["language"] != "" && params["language"] != "all")

	return &LayoutNode{
		Type:  "VerticalStack",
		Props: map[string]interface{}{"gap": 0},
		Children: []LayoutNode{
			{Type: "PageHeader", Props: map[string]interface{}{
				"newLabel":  "New Menu",
				"newPath":   "/admin/menus/new",
				"tabs":      []map[string]interface{}{{"value": "all", "label": "All", "count": int(total)}},
				"activeTab": "all",
			}},
			{Type: "SearchToolbar", Props: map[string]interface{}{
				"searchPlaceholder": "Search menus…",
				"languages":         langList,
				"activeLanguage":    params["language"],
			}},
			{Type: "GenericListTable", Props: map[string]interface{}{
				"columns": []map[string]interface{}{
					{"key": "name", "label": "Name", "sortable": true},
					{"key": "slug", "label": "Slug", "width": 180},
					{"key": "langDisplay", "label": "Language", "width": 130},
					{"key": "version", "label": "Version", "width": 90, "align": "center"},
					{"key": "itemCount", "label": "Items", "width": 80, "align": "center"},
					{"key": "updated_at", "label": "Updated", "width": 110, "sortable": true},
					{"key": "actions", "label": "Actions", "width": 140, "align": "right"},
				},
				"rows":       rows,
				"emptyIcon":  "ListTree",
				"emptyTitle": "No menus found",
				"emptyDesc":  "Create your first menu to get started",
				"newPath":    "/admin/menus/new",
				"newLabel":   "New Menu",
				"label":      "menus",
				"hasFilters": hasFilters,
				"sortBy":     sortBy,
				"sortOrder":  sortOrder,
				"pagination": map[string]interface{}{
					"page": page, "perPage": perPage,
					"total": int(total), "totalPages": totalPages,
				},
			}, Actions: map[string]ActionDef{
				"onRowDelete": {
					Type: "SEQUENCE",
					Steps: []ActionDef{
						{Type: "CONFIRM", Message: "Delete this menu? This cannot be undone."},
						{Type: "CORE_API", Method: "menus:delete", Params: map[string]interface{}{"id": "$event.id"}},
						{Type: "TOAST", Message: "Menu deleted", Variant: "success"},
						{Type: "INVALIDATE", Keys: []string{"layout"}},
					},
				},
			}},
		},
	}
}

func (e *Engine) themesLayout() *LayoutNode {
	var themes []models.Theme
	e.db.Order("is_active DESC, name ASC").Find(&themes)

	themeList := make([]interface{}, 0, len(themes))
	for _, t := range themes {
		gitURL := ""
		if t.GitURL != nil {
			gitURL = *t.GitURL
		}
		thumbnail := ""
		if t.Thumbnail != nil {
			thumbnail = *t.Thumbnail
		}
		themeList = append(themeList, map[string]interface{}{
			"id":            t.ID,
			"slug":          t.Slug,
			"name":          t.Name,
			"description":   t.Description,
			"version":       t.Version,
			"author":        t.Author,
			"source":        t.Source,
			"git_url":       gitURL,
			"git_branch":    t.GitBranch,
			"has_git_token": t.GitToken != nil && *t.GitToken != "",
			"is_active":     t.IsActive,
			"thumbnail":     thumbnail,
		})
	}

	return &LayoutNode{
		Type: "ThemesGrid",
		Props: map[string]interface{}{
			"themes": themeList,
		},
	}
}

func (e *Engine) extensionsLayout() *LayoutNode {
	var exts []models.Extension
	e.db.Order("is_active DESC, name ASC").Find(&exts)

	extList := make([]interface{}, 0, len(exts))
	for _, ext := range exts {
		extList = append(extList, map[string]interface{}{
			"id":           ext.ID,
			"slug":         ext.Slug,
			"name":         ext.Name,
			"version":      ext.Version,
			"description":  ext.Description,
			"author":       ext.Author,
			"is_active":    ext.IsActive,
			"priority":     ext.Priority,
			"installed_at": ext.InstalledAt.Format("2006-01-02"),
			"updated_at":   ext.UpdatedAt.Format("2006-01-02"),
		})
	}

	return &LayoutNode{
		Type: "ExtensionsGrid",
		Props: map[string]interface{}{
			"extensions": extList,
		},
	}
}

// basePathForNodeType returns the admin base path for a given node type slug.
func basePathForNodeType(slug string) string {
	switch slug {
	case "page":
		return "/admin/pages"
	case "post":
		return "/admin/posts"
	default:
		return "/admin/content/" + slug
	}
}

func (e *Engine) defaultLayout(pageSlug string) *LayoutNode {
	return &LayoutNode{
		Type:  "VerticalStack",
		Props: map[string]interface{}{"gap": 0},
		Children: []LayoutNode{
			{
				Type:  "AdminHeader",
				Props: map[string]interface{}{"title": pageSlug},
			},
		},
	}
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
