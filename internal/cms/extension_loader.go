package cms

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"squilla/internal/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ThemeBlockDef is defined in theme_loader.go — reused here for extension block definitions.

// builtinActiveExtensions lists extension slugs that should be auto-activated
// on fresh install. On existing installations the is_active column is not in
// the ON CONFLICT update set, so re-scanning won't override user choices.
var builtinActiveExtensions = map[string]bool{
	"media-manager":     true,
	"email-manager":     true,
	"sitemap-generator": true,
	"smtp-provider":     true,
	"resend-provider":   true,
	"forms":             true,
	"content-blocks":    true,
}

// AdminUISlot describes a component injected into a named slot.
type AdminUISlot struct {
	Component string `json:"component"`
	Label     string `json:"label"`
}

// AdminUIRoute describes a route registered by an extension.
type AdminUIRoute struct {
	Path      string `json:"path"`
	Component string `json:"component"`
}

// AdminUIMenuItem describes a menu child item.
type AdminUIMenuItem struct {
	Label string `json:"label"`
	Route string `json:"route"`
}

// AdminUIMenu describes a sidebar menu group registered by an extension.
type AdminUIMenu struct {
	Label    string `json:"label"`
	Icon     string `json:"icon"`
	Position string `json:"position"`
	// Section routes the menu into a sidebar section: "content" (default),
	// "design", "development", or "settings".
	Section  string            `json:"section,omitempty"`
	Children []AdminUIMenuItem `json:"children"`
}

// AdminUIFieldType describes a custom field type registered by an extension.
type AdminUIFieldType struct {
	Type        string   `json:"type"`
	Label       string   `json:"label"`
	Description string   `json:"description"`
	HowTo       string   `json:"how_to,omitempty"`
	Icon        string   `json:"icon"`
	Group       string   `json:"group"`
	Component   string   `json:"component"`
	Supports    []string `json:"supports,omitempty"`
}

// AdminUIManifest describes the admin UI components an extension provides.
type AdminUIManifest struct {
	Entry      string                 `json:"entry"`
	Slots      map[string]AdminUISlot `json:"slots"`
	Routes     []AdminUIRoute         `json:"routes"`
	Menu       *AdminUIMenu           `json:"menu"`
	FieldTypes []AdminUIFieldType     `json:"field_types,omitempty"`
}

// SettingsField describes a single setting field in the schema.
type SettingsField struct {
	Type      string   `json:"type"`
	Label     string   `json:"label"`
	Required  bool     `json:"required"`
	Default   any      `json:"default"`
	Sensitive bool     `json:"sensitive"`
	Enum      []string `json:"enum"`
}

// ExtensionManifest represents the extension.json manifest file.
// PublicRouteEntry declares a public route that should be proxied to an extension plugin.
type PublicRouteEntry struct {
	Method string `json:"method"`
	Path   string `json:"path"`
}

type ExtensionManifest struct {
	Name           string                   `json:"name"`
	Slug           string                   `json:"slug"`
	Version        string                   `json:"version"`
	Author         string                   `json:"author"`
	Description    string                   `json:"description"`
	Priority       int                      `json:"priority"`
	Provides       []string                 `json:"provides"`
	Capabilities   []string                 `json:"capabilities"`
	// DataOwnedTables enumerates the database tables this extension
	// is allowed to read/write through the Data* CoreAPI methods. The
	// kernel default-denies any table not on this list (see
	// coreapi.checkTableAccess) so an extension granted "data:read"
	// can't pivot from its own data into users / sessions / etc.
	DataOwnedTables []string                 `json:"data_owned_tables"`
	Plugins        []PluginManifestEntry    `json:"plugins"`
	AdminUI        *AdminUIManifest         `json:"admin_ui"`
	SettingsSchema map[string]SettingsField `json:"settings_schema"`
	// Settings is the rich, schema-driven settings declaration. Each entry
	// is a complete settings.Schema (sections + fields with translatable
	// flags). Registered into the in-process settings registry on
	// activation, unregistered on deactivation. The schema ID is namespaced
	// to "ext.<slug>.<id>" by the activation bridge so two extensions can
	// declare schemas with the same local ID without collision.
	Settings []json.RawMessage `json:"settings"`
	Blocks         []ThemeBlockDef          `json:"blocks"`
	Templates      []ThemeTemplateDef       `json:"templates"`
	Layouts        []ThemeLayoutDef         `json:"layouts"`
	Partials       []ThemePartialDef        `json:"partials"`
	PublicRoutes   []PublicRouteEntry       `json:"public_routes"`
	Assets         []ThemeMediaAssetDef     `json:"assets"`
}

// CapabilityMap returns the Capabilities slice as a map for quick lookup.
func (m *ExtensionManifest) CapabilityMap() map[string]bool {
	caps := make(map[string]bool, len(m.Capabilities))
	for _, c := range m.Capabilities {
		caps[c] = true
	}
	return caps
}

// OwnedTablesMap returns DataOwnedTables as a lookup map. Returns an
// empty (non-nil) map for extensions that didn't declare any tables —
// which is the safe default: no entries means no Data* access.
func (m *ExtensionManifest) OwnedTablesMap() map[string]bool {
	owned := make(map[string]bool, len(m.DataOwnedTables))
	for _, t := range m.DataOwnedTables {
		owned[t] = true
	}
	return owned
}

// ExtensionLoader handles scanning, registering, and managing extensions.
//
// Extensions live in two parallel directories:
//   - bundledDir   (e.g. "extensions")        — image-shipped, read-only intent.
//   - dataDir      (e.g. "data/extensions")   — operator-installed, persistent
//     across container restarts when mounted as a volume.
//
// Both are scanned at boot. Writes (zip upload, deploy) target dataDir so
// bundled extensions stay intact. On slug collision the data copy wins so
// an operator override replaces a bundled extension transparently.
type ExtensionLoader struct {
	db            *gorm.DB
	extensionsDir string // bundled (image)
	dataDir       string // operator-installed (volume)
}

// NewExtensionLoader creates a new ExtensionLoader. dataDir is auto-created
// if missing so a fresh container can scan/install without an explicit mkdir.
func NewExtensionLoader(db *gorm.DB, bundledDir, dataDir string) *ExtensionLoader {
	if dataDir != "" {
		_ = os.MkdirAll(dataDir, 0o755)
	}
	return &ExtensionLoader{
		db:            db,
		extensionsDir: bundledDir,
		dataDir:       dataDir,
	}
}

// DataDir returns the writable extensions directory. Install paths target
// it; bundled stays untouched.
func (l *ExtensionLoader) DataDir() string { return l.dataDir }

// ScanAndRegister walks both the data and bundled extension directories
// and upserts a row per valid extension.json. Data wins on slug collision
// because the upsert order claims dataDir slugs first; the bundled scan
// then skips any slug whose stored path already lives under dataDir.
func (l *ExtensionLoader) ScanAndRegister() {
	dataCount := l.scanDir(l.dataDir, true)
	bundledCount := l.scanDir(l.extensionsDir, false)
	log.Printf("[extensions] scanned %d extensions (%d data, %d bundled) from %s + %s",
		dataCount+bundledCount, dataCount, bundledCount, l.dataDir, l.extensionsDir)
}

// scanDir walks one root and registers each extension. Missing dir is fine.
func (l *ExtensionLoader) scanDir(root string, isData bool) int {
	if root == "" {
		return 0
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("[extensions] error reading %s: %v", root, err)
		}
		return 0
	}

	registered := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		extDir := filepath.Join(root, entry.Name())
		manifestPath := filepath.Join(extDir, "extension.json")

		data, err := os.ReadFile(manifestPath)
		if err != nil {
			continue
		}

		var manifest ExtensionManifest
		if err := json.Unmarshal(data, &manifest); err != nil {
			log.Printf("[extensions] invalid extension.json in %s: %v", entry.Name(), err)
			continue
		}

		if manifest.Slug == "" {
			manifest.Slug = entry.Name()
		}
		if manifest.Priority == 0 {
			manifest.Priority = 50
		}

		// Bundled scan must not overwrite a data-backed registration.
		if !isData && l.dataDir != "" {
			var existing models.Extension
			if err := l.db.Where("slug = ?", manifest.Slug).First(&existing).Error; err == nil {
				if strings.HasPrefix(existing.Path, l.dataDir+string(os.PathSeparator)) {
					continue
				}
			}
		}

		ext := models.Extension{
			Slug:        manifest.Slug,
			Name:        manifest.Name,
			Version:     manifest.Version,
			Description: manifest.Description,
			Author:      manifest.Author,
			Path:        extDir,
			Priority:    manifest.Priority,
			Manifest:    models.JSONB(data),
			IsActive:    builtinActiveExtensions[manifest.Slug],
		}

		result := l.db.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "slug"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"name", "version", "description", "author", "path", "priority", "manifest",
			}),
		}).Create(&ext)

		if result.Error != nil {
			log.Printf("[extensions] error registering %s: %v", manifest.Slug, result.Error)
			continue
		}
		registered++
	}
	return registered
}

// GetActive returns all active extensions sorted by priority (ascending).
func (l *ExtensionLoader) GetActive() ([]models.Extension, error) {
	var exts []models.Extension
	err := l.db.Where("is_active = ?", true).Order("priority ASC, slug ASC").Find(&exts).Error
	return exts, err
}

// Activate sets is_active=true for the extension with the given slug.
func (l *ExtensionLoader) Activate(slug string) error {
	result := l.db.Model(&models.Extension{}).Where("slug = ?", slug).Update("is_active", true)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("extension not found: %s", slug)
	}
	return nil
}

// Deactivate sets is_active=false for the extension with the given slug.
func (l *ExtensionLoader) Deactivate(slug string) error {
	result := l.db.Model(&models.Extension{}).Where("slug = ?", slug).Update("is_active", false)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("extension not found: %s", slug)
	}
	return nil
}

// GetBySlug returns a single extension by slug.
func (l *ExtensionLoader) GetBySlug(slug string) (*models.Extension, error) {
	var ext models.Extension
	err := l.db.Where("slug = ?", slug).First(&ext).Error
	if err != nil {
		return nil, err
	}
	return &ext, nil
}

// List returns all extensions ordered by priority.
func (l *ExtensionLoader) List() ([]models.Extension, error) {
	var exts []models.Extension
	err := l.db.Order("priority ASC, slug ASC").Find(&exts).Error
	return exts, err
}

// LoadBlocksForActiveExtensions loads block types from all active extensions
// that declare blocks in their manifest. This mirrors how ThemeLoader registers
// theme blocks but uses source="extension" and stores the extension slug in theme_name.
func (l *ExtensionLoader) LoadBlocksForActiveExtensions(registry *ThemeAssetRegistry) {
	exts, err := l.GetActive()
	if err != nil {
		log.Printf("[extensions] failed to get active extensions for block loading: %v", err)
		return
	}

	total := 0
	for _, ext := range exts {
		n := l.loadExtensionBlocks(ext, registry)
		total += n
	}
	if total > 0 {
		log.Printf("[extensions] loaded %d block types from active extensions", total)
	}
}

// LoadBlocksForExtension loads block types for a single extension.
func (l *ExtensionLoader) LoadBlocksForExtension(slug string, registry *ThemeAssetRegistry) {
	ext, err := l.GetBySlug(slug)
	if err != nil {
		log.Printf("[extensions] cannot load blocks for %s: %v", slug, err)
		return
	}
	n := l.loadExtensionBlocks(*ext, registry)
	if n > 0 {
		log.Printf("[extensions] loaded %d block types from extension %s", n, slug)
	}
}

// UnloadExtensionBlocks removes all block types, templates, layouts, and partials
// owned by an extension and clears their assets from the registry.
func (l *ExtensionLoader) UnloadExtensionBlocks(slug string, registry *ThemeAssetRegistry) {
	// Find all block slugs for this extension before deleting.
	var blockTypes []models.BlockType
	l.db.Where("source = ? AND theme_name = ?", "extension", slug).Find(&blockTypes)

	// Delete blocks from DB.
	result := l.db.Where("source = ? AND theme_name = ?", "extension", slug).Delete(&models.BlockType{})
	if result.RowsAffected > 0 {
		log.Printf("[extensions] removed %d block types from extension %s", result.RowsAffected, slug)
	}

	// Delete templates from DB.
	tResult := l.db.Where("source = ? AND theme_name = ?", "extension", slug).Delete(&models.Template{})
	if tResult.RowsAffected > 0 {
		log.Printf("[extensions] removed %d templates from extension %s", tResult.RowsAffected, slug)
	}

	// Delete layouts from DB.
	lResult := l.db.Where("source = ? AND theme_name = ?", "extension", slug).Delete(&models.Layout{})
	if lResult.RowsAffected > 0 {
		log.Printf("[extensions] removed %d layouts from extension %s", lResult.RowsAffected, slug)
	}

	// Delete partials from DB.
	pResult := l.db.Where("source = ? AND theme_name = ?", "extension", slug).Delete(&models.LayoutBlock{})
	if pResult.RowsAffected > 0 {
		log.Printf("[extensions] removed %d partials from extension %s", pResult.RowsAffected, slug)
	}

	// Remove from asset registry.
	registry.mu.Lock()
	for _, bt := range blockTypes {
		delete(registry.blockAssets, bt.Slug)
	}
	registry.mu.Unlock()
}

// loadExtensionBlocks loads blocks, templates, layouts, and partials for a single extension.
// Returns count of blocks loaded.
func (l *ExtensionLoader) loadExtensionBlocks(ext models.Extension, registry *ThemeAssetRegistry) int {
	var manifest ExtensionManifest
	if err := json.Unmarshal(ext.Manifest, &manifest); err != nil {
		return 0
	}

	loaded := 0

	// Load blocks.
	for _, def := range manifest.Blocks {
		blockDir := filepath.Join(ext.Path, "blocks", def.Dir)
		if err := RegisterBlockFromDir(l.db, registry, blockDir, def.Slug, "extension", ext.Slug); err != nil {
			log.Printf("[extensions] block %s from %s: %v", def.Slug, ext.Slug, err)
			continue
		}
		loaded++
	}

	// Load templates.
	for _, def := range manifest.Templates {
		filePath := filepath.Join(ext.Path, "templates", def.File)
		if err := RegisterTemplateFromFile(l.db, filePath, def.Slug, "extension", ext.Slug); err != nil {
			log.Printf("[extensions] template %s from %s: %v", def.Slug, ext.Slug, err)
			continue
		}
	}

	// Load layouts.
	for _, def := range manifest.Layouts {
		filePath := filepath.Join(ext.Path, "layouts", def.File)
		code, err := os.ReadFile(filePath)
		if err != nil {
			log.Printf("[extensions] layout %s from %s: %v", def.Slug, ext.Slug, err)
			continue
		}
		if err := RegisterLayoutFromFile(l.db, def, string(code), "extension", ext.Slug); err != nil {
			log.Printf("[extensions] layout %s from %s: %v", def.Slug, ext.Slug, err)
		}
	}

	// Load partials.
	for _, def := range manifest.Partials {
		filePath := filepath.Join(ext.Path, "partials", def.File)
		code, err := os.ReadFile(filePath)
		if err != nil {
			log.Printf("[extensions] partial %s from %s: %v", def.Slug, ext.Slug, err)
			continue
		}
		if err := RegisterPartialFromFile(l.db, def, string(code), "extension", ext.Slug); err != nil {
			log.Printf("[extensions] partial %s from %s: %v", def.Slug, ext.Slug, err)
		}
	}

	return loaded
}
