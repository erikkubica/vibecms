package sdui

import (
	"encoding/json"
	"fmt"
	"strconv"
	"sync"

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
		pageSlug == "themes" || pageSlug == "extensions"

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
		layout = e.contentTypesLayout()
	case "taxonomies":
		layout = e.taxonomiesLayout()
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

	// ── Top level ──
	nav = append(nav, NavItem{
		ID: "nav-dashboard", Label: "Dashboard", Icon: "LayoutDashboard",
		Path: "/admin/dashboard",
	})

	// ── Content section ──
	nav = append(nav, NavItem{ID: "section-content", Label: "Content", IsSection: true})

	for _, nt := range nodeTypes {
		// Build sub-items for this node type: main listing + any taxonomies.
		basePath := pathForType(nt.Slug)
		displayLabel := labelFor(nt)

		taxChildren := []NavItem{}
		// First child: link to the main listing (so clicking "All X" still
		// navigates even though the parent item toggles the dropdown).
		taxChildren = append(taxChildren, NavItem{
			ID:    "nav-content-" + nt.Slug + "-all",
			Label: "All " + displayLabel,
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

	// ── Design section ──
	nav = append(nav, NavItem{ID: "section-design", Label: "Design", IsSection: true})
	nav = append(nav, []NavItem{
		{ID: "nav-templates", Label: "Templates", Icon: "FileCode", Path: "/admin/templates"},
		{ID: "nav-layouts", Label: "Layouts", Icon: "LayoutPanelTop", Path: "/admin/layouts"},
		{ID: "nav-block-types", Label: "Block Types", Icon: "Blocks", Path: "/admin/block-types"},
		{ID: "nav-layout-blocks", Label: "Layout Blocks", Icon: "Component", Path: "/admin/layout-blocks"},
		{ID: "nav-menus", Label: "Menus", Icon: "ListTree", Path: "/admin/menus"},
	}...)

	// ── Development section ──
	nav = append(nav, NavItem{ID: "section-dev", Label: "Development", IsSection: true})
	nav = append(nav, []NavItem{
		{ID: "nav-content-types", Label: "Content Types", Icon: "Shapes", Path: "/admin/content-types"},
		{ID: "nav-taxonomies", Label: "Taxonomies", Icon: "Tags", Path: "/admin/taxonomies"},
		{ID: "nav-themes", Label: "Themes", Icon: "Brush", Path: "/admin/themes"},
		{ID: "nav-extensions", Label: "Extensions", Icon: "Puzzle", Path: "/admin/extensions"},
	}...)

	// ── Settings section ──
	nav = append(nav, NavItem{ID: "section-settings", Label: "Settings", IsSection: true})
	nav = append(nav, []NavItem{
		{ID: "nav-site-settings", Label: "Site", Icon: "Globe", Path: "/admin/settings/site"},
		{ID: "nav-languages", Label: "Languages", Icon: "Languages", Path: "/admin/languages"},
		{ID: "nav-users", Label: "Users", Icon: "Users", Path: "/admin/users"},
		{ID: "nav-roles", Label: "Roles", Icon: "Shield", Path: "/admin/roles"},
		{ID: "nav-mcp-tokens", Label: "MCP Tokens", Icon: "Key", Path: "/admin/mcp-tokens"},
	}...)

	// ── Extension navigation items ──
	// Extensions declare which section their items belong to via the
	// manifest's menu.section field. If no section or no children, the
	// item is placed at the top level.
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
			} `json:"admin_ui"`
		}
		_ = json.Unmarshal(ext.Manifest, &manifest)

		if manifest.AdminUI == nil || manifest.AdminUI.Menu == nil {
			continue
		}
		m := manifest.AdminUI.Menu

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
			nav = append(nav, NavItem{
				ID:       "nav-ext-" + ext.Slug,
				Label:    m.Label,
				Icon:     m.Icon,
				Children: children,
			})
		} else {
			// Extension with no menu children — link directly to its route.
			nav = append(nav, NavItem{
				ID:    "nav-ext-" + ext.Slug,
				Label: m.Label,
				Icon:  m.Icon,
				Path:  "/admin/ext/" + ext.Slug + "/",
			})
		}
	}

	return nav
}

type recentNode struct {
	ID        int    `json:"id"`
	Title     string `json:"title"`
	NodeType  string `json:"node_type"`
	Status    string `json:"status"`
	UpdatedAt string `json:"updated_at"`
}

func (e *Engine) dashboardLayout(userName string) *LayoutNode {
	// Query total content nodes (non-deleted)
	var totalContent int64
	e.db.Model(&models.ContentNode{}).Where("deleted_at IS NULL").Count(&totalContent)

	// Query published content nodes
	var published int64
	e.db.Model(&models.ContentNode{}).Where("deleted_at IS NULL AND status = ?", "published").Count(&published)

	// Query draft content nodes
	var drafts int64
	e.db.Model(&models.ContentNode{}).Where("deleted_at IS NULL AND status = ?", "draft").Count(&drafts)

	// Query total users
	var totalUsers int64
	e.db.Model(&models.User{}).Count(&totalUsers)

	// Query 5 most recent content nodes
	var nodes []models.ContentNode
	e.db.Where("deleted_at IS NULL").Order("updated_at DESC").Limit(5).Find(&nodes)

	recentNodes := make([]recentNode, 0, len(nodes))
	for _, n := range nodes {
		recentNodes = append(recentNodes, recentNode{
			ID:        n.ID,
			Title:     n.Title,
			NodeType:  n.NodeType,
			Status:    n.Status,
			UpdatedAt: n.UpdatedAt.Format("2006-01-02"),
		})
	}

	totalStr := fmt.Sprintf("%d", totalContent)
	pubStr := fmt.Sprintf("%d", published)
	draftStr := fmt.Sprintf("%d", drafts)
	usersStr := fmt.Sprintf("%d", totalUsers)

	return &LayoutNode{
		Type:  "VerticalStack",
		Props: map[string]interface{}{"gap": 6, "className": "p-6"},
		Children: []LayoutNode{
			// Welcome banner
			{
				Type: "WelcomeBanner",
				Props: map[string]interface{}{
					"title":       "Welcome back, " + userName,
					"subtitle":    "Here's what's happening with your site.",
					"actionLabel": "Create New Page",
					"actionPath":  "/admin/pages/new",
				},
			},
			// Stats grid
			{
				Type:  "Grid",
				Props: map[string]interface{}{"cols": 4, "gap": 4},
				Children: []LayoutNode{
					{Type: "StatCard", Props: map[string]interface{}{"label": "Total Content", "value": totalStr, "icon": "FileText", "color": "indigo"}},
					{Type: "StatCard", Props: map[string]interface{}{"label": "Published", "value": pubStr, "icon": "Eye", "color": "emerald"}},
					{Type: "StatCard", Props: map[string]interface{}{"label": "Drafts", "value": draftStr, "icon": "PenLine", "color": "amber"}},
					{Type: "StatCard", Props: map[string]interface{}{"label": "Users", "value": usersStr, "icon": "Users", "color": "violet"}},
				},
			},
			// Recent content table
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
		Props: map[string]interface{}{"gap": 4},
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

func (e *Engine) contentTypesLayout() *LayoutNode {
	var nodeTypes []models.NodeType
	if err := e.db.Order("label ASC").Find(&nodeTypes).Error; err != nil {
		return e.defaultLayout("content-types")
	}

	cards := make([]LayoutNode, 0, len(nodeTypes))
	for _, nt := range nodeTypes {
		// Count taxonomies from JSONB field
		var taxSlugs []string
		if err := json.Unmarshal(nt.Taxonomies, &taxSlugs); err != nil {
			taxSlugs = []string{}
		}
		taxCount := len(taxSlugs)

		// Resolve display label
		displayLabel := nt.LabelPlural
		if displayLabel == "" {
			displayLabel = nt.Label
		}

		editPath := fmt.Sprintf("/admin/content-types/%d/edit", nt.ID)
		confirmMsg := fmt.Sprintf("Delete content type '%s'? This cannot be undone.", nt.Label)

		cardProps := map[string]interface{}{
			"id":             nt.ID,
			"slug":           nt.Slug,
			"label":          nt.Label,
			"labelPlural":    nt.LabelPlural,
			"icon":           nt.Icon,
			"description":    nt.Description,
			"supportsBlocks": nt.SupportsBlocks,
			"taxonomyCount":  taxCount,
			"editPath":       editPath,
		}

		cardActions := map[string]ActionDef{
			"onEdit": {
				Type: "NAVIGATE",
				To:   editPath,
			},
			"onDelete": {
				Type: "SEQUENCE",
				Steps: []ActionDef{
					{Type: "CONFIRM", Message: confirmMsg},
					{Type: "CORE_API", Method: "node-types:delete", Params: map[string]interface{}{"id": nt.ID}},
					{Type: "TOAST", Message: "Content type deleted", Variant: "success"},
					{Type: "INVALIDATE", Keys: []string{"layout", "boot"}},
				},
			},
		}

		cards = append(cards, LayoutNode{
			Type:    "ContentTypeCard",
			Props:   cardProps,
			Actions: cardActions,
		})
	}

	return &LayoutNode{
		Type:  "VerticalStack",
		Props: map[string]interface{}{"gap": 6, "className": "p-6"},
		Children: []LayoutNode{
			{
				Type:  "AdminHeader",
				Props: map[string]interface{}{"title": "Content Types"},
			},
			{
				Type:  "HorizontalStack",
				Props: map[string]interface{}{"gap": 3},
				Children: []LayoutNode{
					{
						Type:  "TextBlock",
						Props: map[string]interface{}{"text": "Manage your content types..."},
					},
					{
						Type: "VibeButton",
						Props: map[string]interface{}{
							"label":   "New Content Type",
							"variant": "default",
						},
						Actions: map[string]ActionDef{
							"onClick": {Type: "NAVIGATE", To: "/admin/content-types/new"},
						},
					},
				},
			},
			{
				Type:     "Grid",
				Props:    map[string]interface{}{"cols": 3, "gap": 4},
				Children: cards,
			},
		},
	}
}

func (e *Engine) taxonomiesLayout() *LayoutNode {
	var taxonomies []models.Taxonomy
	if err := e.db.Order("label ASC").Find(&taxonomies).Error; err != nil {
		return e.defaultLayout("taxonomies")
	}

	cards := make([]LayoutNode, 0, len(taxonomies))
	for _, tax := range taxonomies {
		// Resolve display label
		displayLabel := tax.LabelPlural
		if displayLabel == "" {
			displayLabel = tax.Label
		}

		// Convert pq.StringArray to []string for JSON serialization
		nodeTypesArr := make([]string, len(tax.NodeTypes))
		for i, s := range tax.NodeTypes {
			nodeTypesArr[i] = s
		}

		editPath := fmt.Sprintf("/admin/taxonomies/%s/edit", tax.Slug)
		confirmMsg := fmt.Sprintf("Delete taxonomy '%s'? This cannot be undone.", tax.Label)

		cardProps := map[string]interface{}{
			"id":           tax.ID,
			"slug":         tax.Slug,
			"label":        tax.Label,
			"labelPlural":  tax.LabelPlural,
			"description":  tax.Description,
			"hierarchical": tax.Hierarchical,
			"nodeTypes":    nodeTypesArr,
			"editPath":     editPath,
		}

		cardActions := map[string]ActionDef{
			"onEdit": {
				Type: "NAVIGATE",
				To:   editPath,
			},
			"onDelete": {
				Type: "SEQUENCE",
				Steps: []ActionDef{
					{Type: "CONFIRM", Message: confirmMsg},
					{Type: "CORE_API", Method: "taxonomies:delete", Params: map[string]interface{}{"slug": tax.Slug}},
					{Type: "TOAST", Message: "Taxonomy deleted", Variant: "success"},
					{Type: "INVALIDATE", Keys: []string{"layout", "boot"}},
				},
			},
		}

		cards = append(cards, LayoutNode{
			Type:    "TaxonomyCard",
			Props:   cardProps,
			Actions: cardActions,
		})
	}

	return &LayoutNode{
		Type:  "VerticalStack",
		Props: map[string]interface{}{"gap": 6, "className": "p-6"},
		Children: []LayoutNode{
			{
				Type:  "AdminHeader",
				Props: map[string]interface{}{"title": "Taxonomies"},
			},
			{
				Type:  "HorizontalStack",
				Props: map[string]interface{}{"gap": 3},
				Children: []LayoutNode{
					{
						Type:  "TextBlock",
						Props: map[string]interface{}{"text": "Manage your taxonomies..."},
					},
					{
						Type: "VibeButton",
						Props: map[string]interface{}{
							"label":   "New Taxonomy",
							"variant": "default",
						},
						Actions: map[string]ActionDef{
							"onClick": {Type: "NAVIGATE", To: "/admin/taxonomies/new"},
						},
					},
				},
			},
			{
				Type:     "Grid",
				Props:    map[string]interface{}{"cols": 3, "gap": 4},
				Children: cards,
			},
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
	perPage := 20
	offset := (page - 1) * perPage

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
	query.Order("updated_at DESC").Offset(offset).Limit(perPage).Find(&nodes)

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
		Props:    map[string]interface{}{"gap": 4, "className": "p-6"},
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

	// 3. Query taxonomy terms
	var terms []models.TaxonomyTerm
	e.db.Where("node_type = ? AND taxonomy = ?", nodeTypeSlug, taxonomySlug).
		Order("name ASC").Find(&terms)

	// 4. Build rows
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

	return &LayoutNode{
		Type:  "VerticalStack",
		Props: map[string]interface{}{"gap": 4, "className": "p-6"},
		Children: []LayoutNode{
			{
				Type: "PageHeader",
				Props: map[string]interface{}{
					"title": labelPlural,
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
					"taxonomy": taxonomySlug,
					"nodeType": nodeTypeSlug,
					"rows":     rows,
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

func (e *Engine) templatesLayout(params map[string]string) *LayoutNode {
	page, _ := strconv.Atoi(params["page"])
	if page < 1 {
		page = 1
	}
	perPage := 25
	offset := (page - 1) * perPage

	var totalCount int64
	e.db.Model(&models.Template{}).Count(&totalCount)

	var templates []models.Template
	e.db.Order("label ASC").Offset(offset).Limit(perPage).Find(&templates)

	rows := make([]map[string]interface{}, 0, len(templates))
	for _, t := range templates {
		// Parse block_config to get count
		var blockConfigs []interface{}
		if err := json.Unmarshal(t.BlockConfig, &blockConfigs); err != nil {
			blockConfigs = []interface{}{}
		}

		isCustom := t.Source == "custom"
		sourceLabel := t.Source
		if t.Source == "theme" && t.ThemeName != nil {
			sourceLabel = *t.ThemeName
		}

		description := t.Description
		if description == "" {
			description = "—"
		}

		row := map[string]interface{}{
			"id":          t.ID,
			"label":       t.Label,
			"slug":        t.Slug,
			"description": description,
			"blockCount":  len(blockConfigs),
			"source":      t.Source,
			"sourceLabel": sourceLabel,
			"isCustom":    isCustom,
			"editPath":    fmt.Sprintf("/admin/templates/%d/edit", t.ID),
		}
		rows = append(rows, row)
	}

	totalPages := int(totalCount) / perPage
	if int(totalCount)%perPage > 0 {
		totalPages++
	}

	return &LayoutNode{
		Type:  "VerticalStack",
		Props: map[string]interface{}{"gap": 4, "className": "p-6"},
		Children: []LayoutNode{
			{Type: "PageHeader", Props: map[string]interface{}{
				"title":    "Templates",
				"newLabel": "New Template",
				"newPath":  "/admin/templates/new",
			}},
			{Type: "GenericListTable", Props: map[string]interface{}{
				"columns": []map[string]interface{}{
					{"key": "label", "label": "Label"},
					{"key": "blockCount", "label": "Blocks", "width": 100, "align": "center"},
					{"key": "sourceLabel", "label": "Source", "width": 140},
					{"key": "description", "label": "Description"},
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
					"total": int(totalCount), "totalPages": totalPages,
				},
				"label": "templates",
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
	perPage := 25
	offset := (page - 1) * perPage

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

	query := e.db.Model(&models.Layout{})
	if lang := params["language"]; lang != "" && lang != "all" {
		if langID, err := strconv.Atoi(lang); err == nil {
			query = query.Where("language_id = ?", langID)
		}
	}

	var totalCount int64
	query.Count(&totalCount)

	var layouts []models.Layout
	query.Order("name ASC").Offset(offset).Limit(perPage).Find(&layouts)

	rows := make([]map[string]interface{}, 0, len(layouts))
	for _, l := range layouts {
		isCustom := l.Source == "custom"
		sourceLabel := l.Source
		if l.Source == "theme" && l.ThemeName != nil {
			sourceLabel = *l.ThemeName
		}
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
			"sourceLabel": sourceLabel,
			"isCustom":    isCustom,
			"isDefault":   l.IsDefault,
			"languageID":  l.LanguageID,
			"langDisplay": langDisplay,
			"langFlag":    langFlag,
			"langCode":    langCode,
			"editPath":    fmt.Sprintf("/admin/layouts/%d", l.ID),
		})
	}

	totalPages := int(totalCount) / perPage
	if int(totalCount)%perPage > 0 {
		totalPages++
	}

	hasFilters := params["language"] != "" && params["language"] != "all"

	return &LayoutNode{
		Type:  "VerticalStack",
		Props: map[string]interface{}{"gap": 4, "className": "p-6"},
		Children: []LayoutNode{
			{Type: "PageHeader", Props: map[string]interface{}{
				"title":          "Layouts",
				"newLabel":       "New Layout",
				"newPath":        "/admin/layouts/new",
				"languages":      langList,
				"activeLanguage": params["language"],
			}},
			{Type: "GenericListTable", Props: map[string]interface{}{
				"columns": []map[string]interface{}{
					{"key": "name", "label": "Name"},
					{"key": "slug", "label": "Slug", "width": 200},
					{"key": "langDisplay", "label": "Language", "width": 140},
					{"key": "sourceLabel", "label": "Source", "width": 140},
					{"key": "isDefault", "label": "Default", "width": 110},
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
					"total": int(totalCount), "totalPages": totalPages,
				},
				"label":      "layouts",
				"hasFilters": hasFilters,
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
	perPage := 25
	offset := (page - 1) * perPage

	var totalCount int64
	e.db.Model(&models.BlockType{}).Count(&totalCount)

	var blockTypes []models.BlockType
	e.db.Order("label ASC").Offset(offset).Limit(perPage).Find(&blockTypes)

	rows := make([]map[string]interface{}, 0, len(blockTypes))
	for _, bt := range blockTypes {
		// Parse field_schema to get field count
		var fields []interface{}
		if err := json.Unmarshal(bt.FieldSchema, &fields); err != nil {
			fields = []interface{}{}
		}

		isCustom := bt.Source == "custom"
		sourceLabel := bt.Source
		if bt.Source == "theme" && bt.ThemeName != nil {
			sourceLabel = *bt.ThemeName
		}

		description := bt.Description
		if description == "" {
			description = "—"
		}

		row := map[string]interface{}{
			"id":          bt.ID,
			"label":       bt.Label,
			"slug":        bt.Slug,
			"icon":        bt.Icon,
			"description": description,
			"fieldCount":  len(fields),
			"source":      bt.Source,
			"sourceLabel": sourceLabel,
			"isCustom":    isCustom,
			"editPath":    fmt.Sprintf("/admin/block-types/%d/edit", bt.ID),
		}
		rows = append(rows, row)
	}

	totalPages := int(totalCount) / perPage
	if int(totalCount)%perPage > 0 {
		totalPages++
	}

	return &LayoutNode{
		Type:  "VerticalStack",
		Props: map[string]interface{}{"gap": 4, "className": "p-6"},
		Children: []LayoutNode{
			{Type: "PageHeader", Props: map[string]interface{}{
				"title":    "Block Types",
				"newLabel": "New Block Type",
				"newPath":  "/admin/block-types/new",
			}},
			{Type: "GenericListTable", Props: map[string]interface{}{
				"columns": []map[string]interface{}{
					{"key": "label", "label": "Label"},
					{"key": "slug", "label": "Slug", "width": 160},
					{"key": "fieldCount", "label": "Fields", "width": 100, "align": "center"},
					{"key": "sourceLabel", "label": "Source", "width": 140},
					{"key": "description", "label": "Description"},
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
					"total": int(totalCount), "totalPages": totalPages,
				},
				"label": "block-types",
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
	perPage := 25
	offset := (page - 1) * perPage

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

	query := e.db.Model(&models.LayoutBlock{})
	if lang := params["language"]; lang != "" && lang != "all" {
		if langID, err := strconv.Atoi(lang); err == nil {
			query = query.Where("language_id = ?", langID)
		}
	}

	var totalCount int64
	query.Count(&totalCount)

	var layoutBlocks []models.LayoutBlock
	query.Order("name ASC").Offset(offset).Limit(perPage).Find(&layoutBlocks)

	rows := make([]map[string]interface{}, 0, len(layoutBlocks))
	for _, lb := range layoutBlocks {
		isCustom := lb.Source == "custom"
		sourceLabel := lb.Source
		if lb.Source == "theme" && lb.ThemeName != nil {
			sourceLabel = *lb.ThemeName
		}

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
			"sourceLabel": sourceLabel,
			"isCustom":    isCustom,
			"languageID":  lb.LanguageID,
			"langDisplay": langDisplay,
			"langFlag":    langFlag,
			"langCode":    langCode,
			"editPath":    fmt.Sprintf("/admin/layout-blocks/%d/edit", lb.ID),
		})
	}

	totalPages := int(totalCount) / perPage
	if int(totalCount)%perPage > 0 {
		totalPages++
	}

	hasFilters := params["language"] != "" && params["language"] != "all"

	return &LayoutNode{
		Type:  "VerticalStack",
		Props: map[string]interface{}{"gap": 4, "className": "p-6"},
		Children: []LayoutNode{
			{Type: "PageHeader", Props: map[string]interface{}{
				"title":          "Layout Blocks",
				"newLabel":       "New Layout Block",
				"newPath":        "/admin/layout-blocks/new",
				"languages":      langList,
				"activeLanguage": params["language"],
			}},
			{Type: "GenericListTable", Props: map[string]interface{}{
				"columns": []map[string]interface{}{
					{"key": "name", "label": "Name"},
					{"key": "slug", "label": "Slug", "width": 200},
					{"key": "langDisplay", "label": "Language", "width": 140},
					{"key": "sourceLabel", "label": "Source", "width": 140},
					{"key": "description", "label": "Description"},
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
					"total": int(totalCount), "totalPages": totalPages,
				},
				"label":      "layout-blocks",
				"hasFilters": hasFilters,
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
	// Menus are usually few, so no pagination needed
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

	var menus []models.Menu
	query.Order("name ASC").Find(&menus)

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
			"editPath":    fmt.Sprintf("/admin/menus/%d/edit", m.ID),
		})
	}

	hasFilters := params["language"] != "" && params["language"] != "all"

	return &LayoutNode{
		Type:  "VerticalStack",
		Props: map[string]interface{}{"gap": 4, "className": "p-6"},
		Children: []LayoutNode{
			{Type: "PageHeader", Props: map[string]interface{}{
				"title":          "Menus",
				"newLabel":       "New Menu",
				"newPath":        "/admin/menus/new",
				"languages":      langList,
				"activeLanguage": params["language"],
			}},
			{Type: "GenericListTable", Props: map[string]interface{}{
				"columns": []map[string]interface{}{
					{"key": "name", "label": "Name"},
					{"key": "slug", "label": "Slug", "width": 200},
					{"key": "langDisplay", "label": "Language", "width": 140},
					{"key": "version", "label": "Version", "width": 100, "align": "center"},
					{"key": "itemCount", "label": "Items", "width": 100, "align": "center"},
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
		Props: map[string]interface{}{"gap": 4, "className": "p-6"},
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
