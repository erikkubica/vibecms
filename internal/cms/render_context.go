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
	ID           int
	Status       string
	Title        string
	Slug         string
	FullURL      string
	BlocksHTML   template.HTML
	Fields       map[string]interface{}
	SEO          map[string]interface{}
	NodeType     string
	LanguageCode string
	Translations []map[string]interface{}
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
			"id":            td.Node.ID,
			"status":        td.Node.Status,
			"title":         td.Node.Title,
			"slug":          td.Node.Slug,
			"full_url":      td.Node.FullURL,
			"blocks_html":   td.Node.BlocksHTML,
			"fields":        td.Node.Fields,
			"seo":           td.Node.SEO,
			"node_type":     td.Node.NodeType,
			"language_code": td.Node.LanguageCode,
			"translations":  td.Node.Translations,
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
// If usedBlockSlugs is provided, only CSS/JS for those blocks are included.
func (rc *RenderContext) BuildAppData(settings map[string]string, languages []models.Language, currentLang *models.Language, usedBlockSlugs ...[]string) AppData {
	app := AppData{
		Settings:    settings,
		Languages:   languages,
		CurrentLang: currentLang,
	}

	if rc.themeAssets != nil {
		app.HeadStyles = rc.themeAssets.GetHeadStyles()
		app.HeadScripts = rc.themeAssets.GetHeadScripts()
		app.FootScripts = rc.themeAssets.GetFootScripts()
		app.BlockStyles = rc.themeAssets.BuildBlockStyleTags(usedBlockSlugs...)
		app.BlockScripts = rc.themeAssets.BuildBlockScriptTags(usedBlockSlugs...)
	}

	app.ThemeURL = "/theme/assets"

	return app
}

// BuildNodeData creates the .Node namespace from a content node and its rendered blocks HTML.
func (rc *RenderContext) BuildNodeData(node *models.ContentNode, blocksHTML string, languages []models.Language) NodeData {
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

	// Load translations for language switcher
	translations := rc.loadTranslations(node, languages)

	return NodeData{
		ID:           node.ID,
		Status:       node.Status,
		Title:        node.Title,
		Slug:         node.Slug,
		FullURL:      node.FullURL,
		BlocksHTML:   template.HTML(blocksHTML),
		Fields:       fields,
		SEO:          seo,
		NodeType:     node.NodeType,
		LanguageCode: node.LanguageCode,
		Translations: translations,
	}
}

// loadTranslations returns translation siblings for a node, including the current node.
// Each entry has: language_code, full_url, title, flag, language_name, is_current.
func (rc *RenderContext) loadTranslations(node *models.ContentNode, langs []models.Language) []map[string]interface{} {
	if node.TranslationGroupID == nil || *node.TranslationGroupID == "" {
		return nil
	}

	var siblings []models.ContentNode
	rc.db.Where("translation_group_id = ? AND status = 'published' AND deleted_at IS NULL", *node.TranslationGroupID).
		Select("id, title, slug, full_url, language_code").
		Find(&siblings)

	if len(siblings) <= 1 {
		return nil
	}

	langMap := make(map[string]models.Language)
	for _, l := range langs {
		langMap[l.Code] = l
	}

	result := make([]map[string]interface{}, 0, len(siblings))
	for _, s := range siblings {
		lang := langMap[s.LanguageCode]
		result = append(result, map[string]interface{}{
			"language_code": s.LanguageCode,
			"language_name": lang.Name,
			"flag":          lang.Flag,
			"title":         s.Title,
			"full_url":      s.FullURL,
			"is_current":    s.LanguageCode == node.LanguageCode,
		})
	}
	return result
}

// LoadMenus resolves all menus for the current language into a map keyed by slug.
// Performance: Uses ListWithItems for batch loading and single query node URL resolution.
func (rc *RenderContext) LoadMenus(languageID *int) map[string]interface{} {
	menus := make(map[string]interface{})
	
	// Step 1: Batch fetch all menus and their items
	allMenus, err := rc.menuSvc.ListWithItems(languageID)
	if err != nil {
		log.Printf("WARN: failed to load menus: %v", err)
		return menus
	}

	// Step 2: Collect all node IDs across all menus for single batch lookup
	allNodeIDs := make(map[int]bool)
	for _, m := range allMenus {
		rc.collectMenuItemNodeIDs(m.Items, allNodeIDs)
	}

	// Step 3: Batch fetch all node URLs
	nodeMap := make(map[int]string)
	if len(allNodeIDs) > 0 {
		var ids []int
		for id := range allNodeIDs {
			ids = append(ids, id)
		}
		var nodes []models.ContentNode
		if err := rc.db.Select("id, full_url").Where("id IN ?", ids).Find(&nodes).Error; err == nil {
			for _, n := range nodes {
				nodeMap[n.ID] = n.FullURL
			}
		}
	}

	// Step 4: Convert to maps using the pre-fetched nodeMap
	for _, m := range allMenus {
		menus[m.Slug] = map[string]interface{}{
			"id":          m.ID,
			"slug":        m.Slug,
			"name":        m.Name,
			"language_id": m.LanguageID,
			"items":       rc.menuItemsToMaps(m.Items, nodeMap),
		}
	}

	return menus
}



func (rc *RenderContext) collectMenuItemNodeIDs(items []models.MenuItem, nodeIDs map[int]bool) {
	for _, item := range items {
		if item.ItemType == "node" && item.NodeID != nil {
			nodeIDs[*item.NodeID] = true
		}
		if len(item.Children) > 0 {
			rc.collectMenuItemNodeIDs(item.Children, nodeIDs)
		}
	}
}

// menuItemsToMaps converts MenuItem structs to snake_case maps recursively.
// For "node" type items, resolves the URL from the pre-fetched node map.
func (rc *RenderContext) menuItemsToMaps(items []models.MenuItem, nodeMap map[int]string) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(items))
	for _, item := range items {
		url := item.URL
		// Resolve URL for node-type items from the map
		if item.ItemType == "node" && item.NodeID != nil {
			if fullURL, ok := nodeMap[*item.NodeID]; ok {
				url = fullURL
			}
		}
		m := map[string]interface{}{
			"id":        item.ID,
			"title":     item.Title,
			"item_type": item.ItemType,
			"url":       url,
			"target":    item.Target,
			"css_class": item.CSSClass,
			"children":  rc.menuItemsToMaps(item.Children, nodeMap),
		}
		if item.NodeID != nil {
			m["node_id"] = *item.NodeID
		}
		result = append(result, m)
	}
	return result
}
