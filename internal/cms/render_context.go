package cms

import (
	"encoding/json"
	"html/template"
	"log"

	"gorm.io/gorm"

	"vibecms/internal/models"
)

// AppData holds the .App namespace for layout templates.
type AppData struct {
	Menus        map[string]interface{}
	Settings     map[string]string
	Languages    []models.Language
	CurrentLang  *models.Language
	HeadStyles   []string
	HeadScripts  []string
	FootScripts  []string
	BlockStyles  template.HTML
	BlockScripts template.HTML
	ThemeURL     string
}

// NodeData holds the .Node namespace for layout templates.
type NodeData struct {
	Title        string
	Slug         string
	FullURL      string
	BlocksHTML   template.HTML
	Fields       map[string]interface{}
	SEO          map[string]interface{}
	NodeType     string
	LanguageCode string
}

// UserData holds the .user namespace for layout templates.
type UserData struct {
	LoggedIn bool
	ID       int
	Email    string
	Role     string
	FullName string
}

// TemplateData is the top-level context passed to layout templates.
type TemplateData struct {
	App  AppData
	Node NodeData
	User UserData
}

// ToMap converts TemplateData to a map with snake_case keys for template access.
// All keys use snake_case for consistency: {{.app.head_styles}}, {{.node.blocks_html}}.
// All nested structs (languages, current_lang) are also converted to maps.
func (td TemplateData) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"app": map[string]interface{}{
			"menus":         td.App.Menus,
			"settings":      td.App.Settings,
			"languages":     languagesToMaps(td.App.Languages),
			"current_lang":  languageToMap(td.App.CurrentLang),
			"head_styles":   td.App.HeadStyles,
			"head_scripts":  td.App.HeadScripts,
			"foot_scripts":  td.App.FootScripts,
			"block_styles":  td.App.BlockStyles,
			"block_scripts": td.App.BlockScripts,
			"theme_url":     td.App.ThemeURL,
		},
		"node": map[string]interface{}{
			"title":         td.Node.Title,
			"slug":          td.Node.Slug,
			"full_url":      td.Node.FullURL,
			"blocks_html":   td.Node.BlocksHTML,
			"fields":        td.Node.Fields,
			"seo":           td.Node.SEO,
			"node_type":     td.Node.NodeType,
			"language_code": td.Node.LanguageCode,
		},
		"user": map[string]interface{}{
			"logged_in": td.User.LoggedIn,
			"id":        td.User.ID,
			"email":     td.User.Email,
			"role": td.User.Role,
			"full_name": td.User.FullName,
		},
	}
}

// languageToMap converts a Language struct to a snake_case map.
func languageToMap(lang *models.Language) map[string]interface{} {
	if lang == nil {
		return nil
	}
	return map[string]interface{}{
		"code":        lang.Code,
		"slug":        lang.Slug,
		"name":        lang.Name,
		"native_name": lang.NativeName,
		"flag":        lang.Flag,
		"is_default":  lang.IsDefault,
		"is_active":   lang.IsActive,
		"hide_prefix": lang.HidePrefix,
	}
}

// languagesToMaps converts a slice of Language structs to snake_case maps.
func languagesToMaps(langs []models.Language) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(langs))
	for i := range langs {
		result = append(result, languageToMap(&langs[i]))
	}
	return result
}

// RenderContext builds template data for layout rendering.
type RenderContext struct {
	db             *gorm.DB
	layoutSvc      *LayoutService
	layoutBlockSvc *LayoutBlockService
	menuSvc        *MenuService
	themeAssets    *ThemeAssetRegistry
}

// NewRenderContext creates a new RenderContext.
func NewRenderContext(db *gorm.DB, layoutSvc *LayoutService, lbSvc *LayoutBlockService, menuSvc *MenuService, themeAssets *ThemeAssetRegistry) *RenderContext {
	return &RenderContext{
		db:             db,
		layoutSvc:      layoutSvc,
		layoutBlockSvc: lbSvc,
		menuSvc:        menuSvc,
		themeAssets:    themeAssets,
	}
}

// BuildAppData populates the AppData struct with theme assets and other app-level data.
func (rc *RenderContext) BuildAppData(settings map[string]string, languages []models.Language, currentLang *models.Language) AppData {
	app := AppData{
		Settings:    settings,
		Languages:   languages,
		CurrentLang: currentLang,
	}

	if rc.themeAssets != nil {
		app.HeadStyles = rc.themeAssets.GetHeadStyles()
		app.HeadScripts = rc.themeAssets.GetHeadScripts()
		app.FootScripts = rc.themeAssets.GetFootScripts()
		app.BlockStyles = rc.themeAssets.BuildBlockStyleTags()
		app.BlockScripts = rc.themeAssets.BuildBlockScriptTags()
	}

	app.ThemeURL = "/theme/assets"

	return app
}

// BuildNodeData creates the .Node namespace from a content node and its rendered blocks HTML.
func (rc *RenderContext) BuildNodeData(node *models.ContentNode, blocksHTML string) NodeData {
	fields := make(map[string]interface{})
	if len(node.FieldsData) > 0 {
		if err := json.Unmarshal([]byte(node.FieldsData), &fields); err != nil {
			log.Printf("WARN: failed to parse node fields_data: %v", err)
		}
	}

	seo := make(map[string]interface{})
	if len(node.SeoSettings) > 0 {
		if err := json.Unmarshal([]byte(node.SeoSettings), &seo); err != nil {
			log.Printf("WARN: failed to parse node seo_settings: %v", err)
		}
	}

	return NodeData{
		Title:        node.Title,
		Slug:         node.Slug,
		FullURL:      node.FullURL,
		BlocksHTML:   template.HTML(blocksHTML),
		Fields:       fields,
		SEO:          seo,
		NodeType:     node.NodeType,
		LanguageCode: node.LanguageCode,
	}
}

// LoadMenus resolves all menus for the current language into a map keyed by slug.
// Menus are converted to snake_case maps for consistent template access.
func (rc *RenderContext) LoadMenus(languageID *int) map[string]interface{} {
	menus := make(map[string]interface{})
	allMenus, err := rc.menuSvc.List(nil)
	if err != nil {
		log.Printf("WARN: failed to load menus: %v", err)
		return menus
	}
	slugs := make(map[string]bool)
	for _, m := range allMenus {
		slugs[m.Slug] = true
	}
	for slug := range slugs {
		menu, err := rc.menuSvc.Resolve(slug, languageID)
		if err != nil {
			continue
		}
		menus[slug] = rc.menuToMap(menu)
	}
	return menus
}

// menuToMap converts a Menu struct to a snake_case map for template access.
func (rc *RenderContext) menuToMap(menu *models.Menu) map[string]interface{} {
	return map[string]interface{}{
		"id":          menu.ID,
		"slug":        menu.Slug,
		"name":        menu.Name,
		"language_id": menu.LanguageID,
		"items":       rc.menuItemsToMaps(menu.Items),
	}
}

// menuItemsToMaps converts MenuItem structs to snake_case maps recursively.
// For "node" type items, resolves the URL from the content node's full_url.
func (rc *RenderContext) menuItemsToMaps(items []models.MenuItem) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(items))
	for _, item := range items {
		url := item.URL
		// Resolve URL for node-type items from the content node
		if item.ItemType == "node" && item.NodeID != nil {
			var node models.ContentNode
			if err := rc.db.Select("full_url").First(&node, *item.NodeID).Error; err == nil {
				url = node.FullURL
			}
		}
		m := map[string]interface{}{
			"id":        item.ID,
			"title":     item.Title,
			"item_type": item.ItemType,
			"url":       url,
			"target":    item.Target,
			"css_class": item.CSSClass,
			"children":  rc.menuItemsToMaps(item.Children),
		}
		if item.NodeID != nil {
			m["node_id"] = *item.NodeID
		}
		result = append(result, m)
	}
	return result
}
