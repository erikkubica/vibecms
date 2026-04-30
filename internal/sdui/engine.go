package sdui

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"squilla/internal/events"
	"squilla/internal/models"

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

	// Build navigation. The user is passed so node-type entries can be
	// filtered out for roles whose effective access for that type is "none"
	// — that's how a capability change makes sidebar items disappear.
	nav := e.buildNavigation(user, nodeTypes, taxonomies, exts)

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
	case "site-settings":
		layout = e.siteSettingsLayout()
	case "site-settings-general":
		layout = e.siteSettingsGeneralLayout()
	case "site-settings-seo":
		layout = e.siteSettingsSEOLayout()
	case "site-settings-advanced":
		layout = e.siteSettingsAdvancedLayout()
	case "security-settings":
		layout = e.securitySettingsLayout()
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

// buildNavigation moved to engine_navigation.go.

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
