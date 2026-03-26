package cms

import (
	"encoding/json"
	"html/template"
	"log"

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

// TemplateData is the top-level context passed to layout templates.
type TemplateData struct {
	App  AppData
	Node NodeData
}

// RenderContext builds template data for layout rendering.
type RenderContext struct {
	layoutSvc      *LayoutService
	layoutBlockSvc *LayoutBlockService
	menuSvc        *MenuService
	themeAssets    *ThemeAssetRegistry
}

// NewRenderContext creates a new RenderContext.
func NewRenderContext(layoutSvc *LayoutService, lbSvc *LayoutBlockService, menuSvc *MenuService, themeAssets *ThemeAssetRegistry) *RenderContext {
	return &RenderContext{
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
func (rc *RenderContext) LoadMenus(lang, defaultLang string) map[string]interface{} {
	menus := make(map[string]interface{})
	allMenus, err := rc.menuSvc.List("")
	if err != nil {
		log.Printf("WARN: failed to load menus: %v", err)
		return menus
	}
	slugs := make(map[string]bool)
	for _, m := range allMenus {
		slugs[m.Slug] = true
	}
	for slug := range slugs {
		menu, err := rc.menuSvc.Resolve(slug, lang, defaultLang)
		if err != nil {
			continue
		}
		menus[slug] = menu
	}
	return menus
}
