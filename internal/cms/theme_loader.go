package cms

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"vibecms/internal/events"
	"vibecms/internal/models"

	"gorm.io/gorm"
)

// ThemeManifest represents the parsed theme.json file.
type ThemeManifest struct {
	Name        string               `json:"name"`
	Version     string               `json:"version"`
	Description string               `json:"description"`
	Author      string               `json:"author"`
	Styles      []ThemeAssetDef      `json:"styles"`
	Scripts     []ThemeAssetDef      `json:"scripts"`
	Layouts     []ThemeLayoutDef     `json:"layouts"`
	Partials    []ThemePartialDef    `json:"partials"`
	Blocks      []ThemeBlockDef      `json:"blocks"`
	Templates   []ThemeTemplateDef   `json:"templates"`
	Assets      []ThemeMediaAssetDef `json:"assets"`
}

// ThemeMediaAssetDef defines an image or media asset shipped with the theme.
// When the theme is activated, the theme loader emits theme.activated with
// these assets in the payload (resolved abs_paths included). Extensions such
// as media-manager subscribe to that event and import the files into their
// own storage (media_files table) tagged source='theme'.
type ThemeMediaAssetDef struct {
	Key    string `json:"key"`
	Src    string `json:"src"`
	Alt    string `json:"alt,omitempty"`
	Width  int    `json:"width,omitempty"`
	Height int    `json:"height,omitempty"`
}

// ThemeAssetDef defines a CSS or JS asset declared in theme.json.
type ThemeAssetDef struct {
	Handle   string   `json:"handle"`
	Src      string   `json:"src"`
	Position string   `json:"position"` // "head" or "footer", default "footer"
	Defer    bool     `json:"defer"`
	Deps     []string `json:"deps"`
}

// ThemeLayoutDef defines a layout declared in theme.json.
type ThemeLayoutDef struct {
	Slug      string `json:"slug"`
	Name      string `json:"name"`
	File      string `json:"file"`
	IsDefault bool   `json:"is_default"`
}

// ThemePartialDef defines a partial declared in theme.json.
type ThemePartialDef struct {
	Slug        string          `json:"slug"`
	Name        string          `json:"name"`
	File        string          `json:"file"`
	FieldSchema json.RawMessage `json:"field_schema"`
}

// ThemeBlockDef defines a block type declared in theme.json.
type ThemeBlockDef struct {
	Slug string `json:"slug"`
	Dir  string `json:"dir"`
}

// ThemeTemplateDef defines a page template declared in theme.json.
type ThemeTemplateDef struct {
	Slug string `json:"slug"`
	File string `json:"file"`
}

// BlockAsset holds scoped CSS and JS for a block type.
type BlockAsset struct {
	CSS string
	JS  string
}

// ThemeAssetRegistry is an in-memory store for resolved theme assets.
type ThemeAssetRegistry struct {
	mu          sync.RWMutex
	headStyles  []string
	headScripts []string
	footScripts []string
	blockAssets map[string]*BlockAsset
	themeDir    string
}

// NewThemeAssetRegistry creates a new ThemeAssetRegistry.
func NewThemeAssetRegistry() *ThemeAssetRegistry {
	return &ThemeAssetRegistry{
		blockAssets: make(map[string]*BlockAsset),
	}
}

// LoadBlockAssetsFromDB seeds the in-memory block asset registry from the
// block_types table. Disk-based sync (ThemeLoader, ExtensionLoader) skips the
// registry write when the content hash is unchanged, which leaves the registry
// empty on every restart after first install. Calling this at boot guarantees
// the registry reflects whatever block_css/block_js is currently persisted.
func (r *ThemeAssetRegistry) LoadBlockAssetsFromDB(db *gorm.DB) error {
	var bts []models.BlockType
	if err := db.Select("slug, block_css, block_js").Find(&bts).Error; err != nil {
		return fmt.Errorf("load block assets from db: %w", err)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, bt := range bts {
		if bt.BlockCSS == "" && bt.BlockJS == "" {
			continue
		}
		r.blockAssets[bt.Slug] = &BlockAsset{
			CSS: bt.BlockCSS,
			JS:  bt.BlockJS,
		}
	}
	return nil
}

// GetHeadStyles returns the resolved head style URLs.
func (r *ThemeAssetRegistry) GetHeadStyles() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]string, len(r.headStyles))
	copy(out, r.headStyles)
	return out
}

// GetHeadScripts returns the resolved head script URLs.
func (r *ThemeAssetRegistry) GetHeadScripts() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]string, len(r.headScripts))
	copy(out, r.headScripts)
	return out
}

// GetFootScripts returns the resolved footer script URLs.
func (r *ThemeAssetRegistry) GetFootScripts() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]string, len(r.footScripts))
	copy(out, r.footScripts)
	return out
}

// GetBlockAssets returns the scoped CSS/JS for a block slug.
func (r *ThemeAssetRegistry) GetBlockAssets(slug string) (*BlockAsset, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	ba, ok := r.blockAssets[slug]
	return ba, ok
}

// BuildBlockStyleTags returns inline <style> tags for block assets.
// If usedSlugs is provided, only those blocks are included; otherwise all blocks.
func (r *ThemeAssetRegistry) BuildBlockStyleTags(usedSlugs ...[]string) template.HTML {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var sb strings.Builder

	if len(usedSlugs) > 0 && len(usedSlugs[0]) > 0 {
		// Only include CSS for blocks used on this page.
		seen := make(map[string]bool, len(usedSlugs[0]))
		for _, slug := range usedSlugs[0] {
			if seen[slug] {
				continue
			}
			seen[slug] = true
			if ba, ok := r.blockAssets[slug]; ok && ba.CSS != "" {
				sb.WriteString(fmt.Sprintf("<style data-block=\"%s\">\n%s\n</style>\n", slug, ba.CSS))
			}
		}
	} else {
		for slug, ba := range r.blockAssets {
			if ba.CSS != "" {
				sb.WriteString(fmt.Sprintf("<style data-block=\"%s\">\n%s\n</style>\n", slug, ba.CSS))
			}
		}
	}
	return template.HTML(sb.String())
}

// BuildBlockScriptTags returns inline <script> tags for block assets.
// If usedSlugs is provided, only those blocks are included; otherwise all blocks.
func (r *ThemeAssetRegistry) BuildBlockScriptTags(usedSlugs ...[]string) template.HTML {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var sb strings.Builder

	if len(usedSlugs) > 0 && len(usedSlugs[0]) > 0 {
		seen := make(map[string]bool, len(usedSlugs[0]))
		for _, slug := range usedSlugs[0] {
			if seen[slug] {
				continue
			}
			seen[slug] = true
			if ba, ok := r.blockAssets[slug]; ok && ba.JS != "" {
				sb.WriteString(fmt.Sprintf("<script data-block=\"%s\">\n%s\n</script>\n", slug, ba.JS))
			}
		}
	} else {
		for slug, ba := range r.blockAssets {
			if ba.JS != "" {
				sb.WriteString(fmt.Sprintf("<script data-block=\"%s\">\n%s\n</script>\n", slug, ba.JS))
			}
		}
	}
	return template.HTML(sb.String())
}

// ThemeLoader reads theme.json and registers layouts, partials, blocks, and assets.
type ThemeLoader struct {
	db       *gorm.DB
	registry *ThemeAssetRegistry
	eventBus *events.EventBus
}

// NewThemeLoader creates a new ThemeLoader.
func NewThemeLoader(db *gorm.DB, registry *ThemeAssetRegistry, eventBus *events.EventBus) *ThemeLoader {
	return &ThemeLoader{
		db:       db,
		registry: registry,
		eventBus: eventBus,
	}
}

// LoadTheme reads theme.json from themeDir and registers everything.
func (tl *ThemeLoader) LoadTheme(themeDir string) error {
	manifestPath := filepath.Join(themeDir, "theme.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("WARN: theme.json not found at %s, skipping theme load", manifestPath)
			return nil
		}
		log.Printf("WARN: failed to read theme.json at %s: %v", manifestPath, err)
		return nil
	}

	var manifest ThemeManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		log.Printf("ERROR: failed to parse theme.json at %s: %v", manifestPath, err)
		return nil
	}

	log.Printf("loading theme: %s v%s", manifest.Name, manifest.Version)

	// Store theme dir in registry.
	tl.registry.mu.Lock()
	tl.registry.themeDir = themeDir
	tl.registry.mu.Unlock()

	// Register assets.
	tl.registerAssets(manifest)

	// Register layouts.
	for _, def := range manifest.Layouts {
		filePath := filepath.Join(themeDir, "layouts", def.File)
		code, err := os.ReadFile(filePath)
		if err != nil {
			log.Printf("WARN: layout file not found %s: %v", filePath, err)
			continue
		}
		tl.upsertLayout(manifest.Name, def, string(code))
	}

	// Register partials.
	for _, def := range manifest.Partials {
		filePath := filepath.Join(themeDir, "partials", def.File)
		code, err := os.ReadFile(filePath)
		if err != nil {
			log.Printf("WARN: partial file not found %s: %v", filePath, err)
			continue
		}
		tl.upsertPartial(manifest.Name, def, string(code))
	}

	// Register blocks.
	for _, def := range manifest.Blocks {
		tl.registerBlock(manifest.Name, def, themeDir)
	}

	// Register page templates.
	for _, def := range manifest.Templates {
		tl.registerTemplate(manifest.Name, def, themeDir)
	}

	// Clean up stale partial records from the layouts table.
	// Partials belong in layout_blocks, not layouts. Earlier versions
	// incorrectly created them in layouts — remove those orphans.
	for _, def := range manifest.Partials {
		tl.db.Where("slug = ? AND source = 'theme'", def.Slug).Delete(&models.Layout{})
	}

	// Upsert theme record in the themes table.
	tl.upsertThemeRecord(manifest, themeDir)

	log.Printf("theme loaded: %s (%d layouts, %d partials, %d blocks, %d styles, %d scripts)",
		manifest.Name, len(manifest.Layouts), len(manifest.Partials), len(manifest.Blocks),
		len(manifest.Styles), len(manifest.Scripts))

	// Trigger cache invalidation across the system
	if tl.eventBus != nil {
		tl.eventBus.Publish("taxonomies:register", events.Payload{
			"theme": manifest.Name,
		})
		// theme.activated is published SYNCHRONOUSLY so that subscribing
		// extensions (e.g. media-manager importing theme assets into
		// media_files) complete before this function returns. First-request
		// renders can then rely on imported assets being in place.
		tl.eventBus.PublishSync("theme.activated", events.Payload{
			"name":    manifest.Name,
			"path":    themeDir,
			"version": manifest.Version,
			"assets":  themeAssetsPayload(themeDir, manifest.Assets),
		})
	}

	return nil
}

// upsertThemeRecord creates or updates the theme record in the themes table.
func (tl *ThemeLoader) upsertThemeRecord(manifest ThemeManifest, themeDir string) {
	slug := strings.ToLower(strings.ReplaceAll(manifest.Name, " ", "-"))

	var existing models.Theme
	result := tl.db.Where("slug = ?", slug).First(&existing)

	if result.Error == nil {
		existing.Name = manifest.Name
		existing.Version = manifest.Version
		existing.Description = manifest.Description
		existing.Author = manifest.Author
		existing.Path = themeDir
		existing.Source = "local"
		existing.IsActive = true
		if err := tl.db.Save(&existing).Error; err != nil {
			log.Printf("WARN: failed to update theme record %s: %v", slug, err)
		}
	} else {
		theme := models.Theme{
			Slug:        slug,
			Name:        manifest.Name,
			Version:     manifest.Version,
			Description: manifest.Description,
			Author:      manifest.Author,
			Path:        themeDir,
			Source:      "local",
			IsActive:    true,
		}
		if err := tl.db.Create(&theme).Error; err != nil {
			log.Printf("WARN: failed to create theme record %s: %v", slug, err)
		}
	}
}

// themeTemplateFile represents the JSON structure of a theme template file.
type themeTemplateFile struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Blocks      []struct {
		Type   string                 `json:"type"`
		Fields map[string]interface{} `json:"fields"`
	} `json:"blocks"`
}

// RegisterTemplateFromFile reads a template JSON file and upserts the template record.
// Shared helper used by both ThemeLoader and ExtensionLoader.
func RegisterTemplateFromFile(db *gorm.DB, filePath string, slug string, source string, sourceName string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("template file not found %s: %w", filePath, err)
	}

	var tmplFile themeTemplateFile
	if err := json.Unmarshal(data, &tmplFile); err != nil {
		return fmt.Errorf("failed to parse template %s: %w", slug, err)
	}

	// Convert {type, fields} to {block_type_slug, default_values}
	blockConfig := make([]map[string]interface{}, 0, len(tmplFile.Blocks))
	for _, b := range tmplFile.Blocks {
		blockConfig = append(blockConfig, map[string]interface{}{
			"block_type_slug": b.Type,
			"default_values":  b.Fields,
		})
	}

	configJSON, err := json.Marshal(blockConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal template block_config for %s: %w", slug, err)
	}

	label := tmplFile.Name
	if label == "" {
		label = slug
	}

	var sourceNamePtr *string
	if sourceName != "" {
		sourceNamePtr = &sourceName
	}

	h := sha256.New()
	h.Write(data)
	contentHash := hex.EncodeToString(h.Sum(nil))

	var existing models.Template
	result := db.Where("slug = ?", slug).First(&existing)

	if result.Error == nil {
		if existing.Source == "custom" {
			return nil
		}
		if existing.ContentHash == contentHash && existing.Source == source {
			return nil
		}
		existing.Label = label
		existing.Description = tmplFile.Description
		existing.BlockConfig = models.JSONB(configJSON)
		existing.Source = source
		existing.ThemeName = sourceNamePtr
		existing.ContentHash = contentHash
		if err := db.Save(&existing).Error; err != nil {
			return fmt.Errorf("failed to update template %s: %w", slug, err)
		}
	} else {
		tmpl := models.Template{
			Slug:        slug,
			Label:       label,
			Description: tmplFile.Description,
			BlockConfig: models.JSONB(configJSON),
			Source:      source,
			ThemeName:   sourceNamePtr,
			ContentHash: contentHash,
		}
		if err := db.Create(&tmpl).Error; err != nil {
			return fmt.Errorf("failed to create template %s: %w", slug, err)
		}
	}
	return nil
}

// registerTemplate reads a theme template file and upserts the template record.
func (tl *ThemeLoader) registerTemplate(themeName string, def ThemeTemplateDef, themeDir string) {
	filePath := filepath.Join(themeDir, "templates", def.File)
	if err := RegisterTemplateFromFile(tl.db, filePath, def.Slug, "theme", themeName); err != nil {
		log.Printf("WARN: %v", err)
	}
}

// registerAssets resolves dependency order and populates the registry.
// Resets any previously-registered theme assets so switching themes doesn't
// leak the old theme's stylesheets/scripts into the new active theme.
func (tl *ThemeLoader) registerAssets(manifest ThemeManifest) {
	tl.registry.mu.Lock()
	defer tl.registry.mu.Unlock()

	tl.registry.headStyles = tl.registry.headStyles[:0]
	tl.registry.headScripts = tl.registry.headScripts[:0]
	tl.registry.footScripts = tl.registry.footScripts[:0]

	// Resolve styles (styles typically don't have deps but support it).
	for _, s := range manifest.Styles {
		url := "/theme/assets/" + s.Src
		tl.registry.headStyles = append(tl.registry.headStyles, url)
	}

	// Resolve scripts with dependency ordering.
	sorted := tl.resolveDeps(manifest.Scripts)
	for _, s := range sorted {
		url := "/theme/assets/" + s.Src
		pos := s.Position
		if pos == "" {
			pos = "footer"
		}
		if pos == "head" {
			tl.registry.headScripts = append(tl.registry.headScripts, url)
		} else {
			tl.registry.footScripts = append(tl.registry.footScripts, url)
		}
	}
}

// upsertLayout creates or updates a layout from a theme definition.
func (tl *ThemeLoader) upsertLayout(themeName string, def ThemeLayoutDef, code string) {
	if err := RegisterLayoutFromFile(tl.db, def, code, "theme", themeName); err != nil {
		log.Printf("WARN: %v", err)
	}
}

// upsertPartial creates or updates a layout block from a theme partial definition.
func (tl *ThemeLoader) upsertPartial(themeName string, def ThemePartialDef, code string) {
	if err := RegisterPartialFromFile(tl.db, def, code, "theme", themeName); err != nil {
		log.Printf("WARN: %v", err)
	}
}

// RegisterLayoutFromFile upserts a layout with hash-based change detection.
func RegisterLayoutFromFile(db *gorm.DB, def ThemeLayoutDef, code string, source string, sourceName string) error {
	h := sha256.New()
	h.Write([]byte(code))
	contentHash := hex.EncodeToString(h.Sum(nil))

	var sourceNamePtr *string
	if sourceName != "" {
		sourceNamePtr = &sourceName
	}

	var existing models.Layout
	result := db.Where("slug = ? AND language_id IS NULL", def.Slug).First(&existing)

	if result.Error == nil {
		if existing.Source == "custom" {
			return nil
		}
		if existing.ContentHash == contentHash && existing.Source == source {
			return nil
		}
		existing.Name = def.Name
		existing.TemplateCode = code
		existing.Source = source
		existing.ThemeName = sourceNamePtr
		existing.IsDefault = def.IsDefault
		existing.ContentHash = contentHash
		if err := db.Save(&existing).Error; err != nil {
			return fmt.Errorf("failed to update layout %s: %w", def.Slug, err)
		}
	} else {
		layout := models.Layout{
			Slug:         def.Slug,
			Name:         def.Name,
			LanguageID:   nil,
			TemplateCode: code,
			Source:       source,
			ThemeName:    sourceNamePtr,
			IsDefault:    def.IsDefault,
			ContentHash:  contentHash,
		}
		if err := db.Create(&layout).Error; err != nil {
			return fmt.Errorf("failed to create layout %s: %w", def.Slug, err)
		}
	}
	return nil
}

// RegisterPartialFromFile upserts a layout block (partial) with hash-based change detection.
func RegisterPartialFromFile(db *gorm.DB, def ThemePartialDef, code string, source string, sourceName string) error {
	h := sha256.New()
	h.Write([]byte(code))
	h.Write(def.FieldSchema) // include field_schema in hash
	contentHash := hex.EncodeToString(h.Sum(nil))

	var sourceNamePtr *string
	if sourceName != "" {
		sourceNamePtr = &sourceName
	}

	// Prepare field_schema JSONB
	fieldSchema := models.JSONB("[]")
	if len(def.FieldSchema) > 0 {
		fieldSchema = models.JSONB(def.FieldSchema)
	}

	var existing models.LayoutBlock
	result := db.Where("slug = ? AND language_id IS NULL", def.Slug).First(&existing)

	if result.Error == nil {
		if existing.Source == "custom" {
			return nil
		}
		if existing.ContentHash == contentHash && existing.Source == source {
			return nil
		}
		existing.Name = def.Name
		existing.TemplateCode = code
		existing.FieldSchema = fieldSchema
		existing.Source = source
		existing.ThemeName = sourceNamePtr
		existing.ContentHash = contentHash
		if err := db.Save(&existing).Error; err != nil {
			return fmt.Errorf("failed to update partial %s: %w", def.Slug, err)
		}
	} else {
		lb := models.LayoutBlock{
			Slug:         def.Slug,
			Name:         def.Name,
			LanguageID:   nil,
			TemplateCode: code,
			FieldSchema:  fieldSchema,
			Source:       source,
			ThemeName:    sourceNamePtr,
			ContentHash:  contentHash,
		}
		if err := db.Create(&lb).Error; err != nil {
			return fmt.Errorf("failed to create partial %s: %w", def.Slug, err)
		}
	}
	return nil
}

// blockManifest is the structure of a block's block.json file.
type blockManifest struct {
	Slug        string          `json:"slug"`
	Label       string          `json:"label"`
	Icon        string          `json:"icon"`
	Description string          `json:"description"`
	FieldSchema json.RawMessage `json:"field_schema"`
	TestData    json.RawMessage `json:"test_data"`
}

// RegisterBlockFromDir reads block files from blockDir and upserts the block type.
// This is a shared helper used by both ThemeLoader and ExtensionLoader.
func RegisterBlockFromDir(db *gorm.DB, registry *ThemeAssetRegistry, blockDir string, slug string, source string, sourceName string) error {
	// Read block.json.
	bjData, err := os.ReadFile(filepath.Join(blockDir, "block.json"))
	if err != nil {
		return fmt.Errorf("block.json not found for block %s: %w", slug, err)
	}

	var bm blockManifest
	if err := json.Unmarshal(bjData, &bm); err != nil {
		return fmt.Errorf("failed to parse block.json for %s: %w", slug, err)
	}

	// Read view.html (the HTML template for the block).
	viewData, err := os.ReadFile(filepath.Join(blockDir, "view.html"))
	if err != nil {
		return fmt.Errorf("view.html not found for block %s: %w", slug, err)
	}

	// Read optional style.css and script.js.
	var blockCSS, blockJS string
	if cssData, err := os.ReadFile(filepath.Join(blockDir, "style.css")); err == nil {
		blockCSS = string(cssData)
	}
	if jsData, err := os.ReadFile(filepath.Join(blockDir, "script.js")); err == nil {
		blockJS = string(jsData)
	}

	// Prepare field_schema and test_data as JSONB.
	fieldSchema := models.JSONB("[]")
	if len(bm.FieldSchema) > 0 {
		fieldSchema = models.JSONB(bm.FieldSchema)
	}
	testData := models.JSONB("{}")
	if len(bm.TestData) > 0 {
		testData = models.JSONB(bm.TestData)
	}

	// Set defaults.
	label := bm.Label
	if label == "" {
		label = slug
	}
	icon := bm.Icon
	if icon == "" {
		icon = "square"
	}

	// Source name pointer (nil for "custom").
	var sourceNamePtr *string
	if sourceName != "" {
		sourceNamePtr = &sourceName
	}

	// Compute content hash from all block files.
	h := sha256.New()
	h.Write(bjData)
	h.Write(viewData)
	h.Write([]byte(blockCSS))
	h.Write([]byte(blockJS))
	contentHash := hex.EncodeToString(h.Sum(nil))

	// Upsert block type.
	var existing models.BlockType
	result := db.Where("slug = ?", slug).First(&existing)

	viewFile := filepath.Join("blocks", filepath.Base(blockDir), "view.html")

	if result.Error == nil {
		// Skip update if block is detached (custom) — user owns it now.
		if existing.Source == "custom" {
			return nil
		}

		// Skip update if content hasn't changed.
		if existing.ContentHash == contentHash && existing.Source == source {
			return nil
		}

		existing.Label = label
		existing.Icon = icon
		existing.Description = bm.Description
		existing.FieldSchema = fieldSchema
		existing.HTMLTemplate = string(viewData)
		existing.TestData = testData
		existing.Source = source
		existing.ThemeName = sourceNamePtr
		existing.ViewFile = viewFile
		existing.BlockCSS = blockCSS
		existing.BlockJS = blockJS
		existing.ContentHash = contentHash
		if err := db.Save(&existing).Error; err != nil {
			return fmt.Errorf("failed to update block_type %s: %w", slug, err)
		}
	} else {
		bt := models.BlockType{
			Slug:         slug,
			Label:        label,
			Icon:         icon,
			Description:  bm.Description,
			FieldSchema:  fieldSchema,
			HTMLTemplate: string(viewData),
			TestData:     testData,
			Source:       source,
			ThemeName:    sourceNamePtr,
			ViewFile:     viewFile,
			BlockCSS:     blockCSS,
			BlockJS:      blockJS,
			ContentHash:  contentHash,
		}
		if err := db.Create(&bt).Error; err != nil {
			return fmt.Errorf("failed to create block_type %s: %w", slug, err)
		}
	}

	// Store block assets in registry.
	if blockCSS != "" || blockJS != "" {
		registry.mu.Lock()
		registry.blockAssets[slug] = &BlockAsset{
			CSS: blockCSS,
			JS:  blockJS,
		}
		registry.mu.Unlock()
	}

	return nil
}

// registerBlock reads block files and upserts the block type (theme source).
func (tl *ThemeLoader) registerBlock(themeName string, def ThemeBlockDef, themeDir string) {
	blockDir := filepath.Join(themeDir, "blocks", def.Dir)
	if err := RegisterBlockFromDir(tl.db, tl.registry, blockDir, def.Slug, "theme", themeName); err != nil {
		log.Printf("WARN: %v", err)
	}
}

// DeregisterTheme removes all DB records and registry entries that were
// registered by the named theme (source='theme', theme_name=themeName).
// Call this when a theme is deactivated so its blocks/layouts/partials/templates
// are no longer visible to the rest of the system.
//
// Emits theme.deactivated (sync) FIRST so subscribing extensions can clean up
// their own theme-owned data (e.g. media-manager deleting imported theme
// assets from media_files). Core-owned records are deleted afterwards.
func (tl *ThemeLoader) DeregisterTheme(themeName string) error {
	// Give extensions a chance to clean up their theme-owned data before we
	// delete core-owned records. Sync so handlers complete before we proceed.
	if tl.eventBus != nil {
		tl.eventBus.PublishSync("theme.deactivated", events.Payload{
			"name": themeName,
		})
	}

	// Collect block slugs before deleting so we can purge the asset registry.
	var blockSlugs []string
	if err := tl.db.Model(&models.BlockType{}).
		Where("source = 'theme' AND theme_name = ?", themeName).
		Pluck("slug", &blockSlugs).Error; err != nil {
		return fmt.Errorf("deregister theme %q: collect block slugs: %w", themeName, err)
	}

	// Delete theme-owned records from all relevant tables.
	cond := "source = 'theme' AND theme_name = ?"
	for _, q := range []interface{}{
		&models.BlockType{},
		&models.Layout{},
		&models.LayoutBlock{},
		&models.Template{},
	} {
		if err := tl.db.Where(cond, themeName).Delete(q).Error; err != nil {
			return fmt.Errorf("deregister theme %q: delete %T: %w", themeName, q, err)
		}
	}

	// Purge block assets from the in-memory registry.
	if len(blockSlugs) > 0 {
		tl.registry.mu.Lock()
		for _, slug := range blockSlugs {
			delete(tl.registry.blockAssets, slug)
		}
		tl.registry.mu.Unlock()
	}

	log.Printf("theme deregistered: %s (%d blocks removed)", themeName, len(blockSlugs))
	return nil
}

// themeAssetsPayload converts a manifest's asset definitions into the shape
// carried by the theme.activated event — adding an absolute path so
// extensions running in-process (or out-of-process with the same CWD) can
// read the file reliably regardless of working directory.
func themeAssetsPayload(themeDir string, defs []ThemeMediaAssetDef) []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(defs))
	for _, d := range defs {
		rel := filepath.Join(themeDir, "assets", d.Src)
		abs, err := filepath.Abs(rel)
		if err != nil {
			abs = rel
		}
		out = append(out, map[string]interface{}{
			"key":      d.Key,
			"src":      d.Src,
			"alt":      d.Alt,
			"width":    d.Width,
			"height":   d.Height,
			"abs_path": abs,
		})
	}
	return out
}

// PurgeInactiveThemes deregisters theme-owned assets (blocks, layouts,
// layout_blocks, templates) for every theme currently marked is_active=false.
// Runs at boot to heal DB state for themes that were deactivated before the
// deregistration hook existed, or where cleanup failed.
// Detached records (source='custom') are untouched — the user owns them.
func (tl *ThemeLoader) PurgeInactiveThemes() error {
	var inactive []models.Theme
	if err := tl.db.Where("is_active = ?", false).Find(&inactive).Error; err != nil {
		return fmt.Errorf("purge inactive themes: list: %w", err)
	}
	for _, t := range inactive {
		if err := tl.DeregisterTheme(t.Name); err != nil {
			log.Printf("WARN: purge inactive theme %q: %v", t.Name, err)
		}
	}
	return nil
}

// resolveDeps performs a topological sort of scripts by their deps field.
func (tl *ThemeLoader) resolveDeps(scripts []ThemeAssetDef) []ThemeAssetDef {
	if len(scripts) == 0 {
		return scripts
	}

	// Build handle -> script map.
	byHandle := make(map[string]ThemeAssetDef)
	for _, s := range scripts {
		byHandle[s.Handle] = s
	}

	// Kahn's algorithm for topological sort.
	visited := make(map[string]bool)
	visiting := make(map[string]bool)
	var sorted []ThemeAssetDef

	var visit func(handle string)
	visit = func(handle string) {
		if visited[handle] {
			return
		}
		if visiting[handle] {
			log.Printf("WARN: circular dependency detected for script handle %s", handle)
			return
		}
		visiting[handle] = true

		s, ok := byHandle[handle]
		if !ok {
			visiting[handle] = false
			return
		}

		for _, dep := range s.Deps {
			visit(dep)
		}

		visiting[handle] = false
		visited[handle] = true
		sorted = append(sorted, s)
	}

	for _, s := range scripts {
		visit(s.Handle)
	}

	return sorted
}
