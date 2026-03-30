package cms

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"vibecms/internal/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ThemeBlockDef is defined in theme_loader.go — reused here for extension block definitions.

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
	Label    string            `json:"label"`
	Icon     string            `json:"icon"`
	Position string            `json:"position"`
	Children []AdminUIMenuItem `json:"children"`
}

// AdminUIManifest describes the admin UI components an extension provides.
type AdminUIManifest struct {
	Entry  string                 `json:"entry"`
	Slots  map[string]AdminUISlot `json:"slots"`
	Routes []AdminUIRoute         `json:"routes"`
	Menu   *AdminUIMenu           `json:"menu"`
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
type ExtensionManifest struct {
	Name           string                  `json:"name"`
	Slug           string                  `json:"slug"`
	Version        string                  `json:"version"`
	Author         string                  `json:"author"`
	Description    string                  `json:"description"`
	Priority       int                     `json:"priority"`
	Provides       []string                `json:"provides"`
	Capabilities   []string                `json:"capabilities"`
	Plugins        []PluginManifestEntry   `json:"plugins"`
	AdminUI        *AdminUIManifest        `json:"admin_ui"`
	SettingsSchema map[string]SettingsField `json:"settings_schema"`
	Blocks         []ThemeBlockDef         `json:"blocks"`
	Templates      []ThemeTemplateDef      `json:"templates"`
}

// CapabilityMap returns the Capabilities slice as a map for quick lookup.
func (m *ExtensionManifest) CapabilityMap() map[string]bool {
	caps := make(map[string]bool, len(m.Capabilities))
	for _, c := range m.Capabilities {
		caps[c] = true
	}
	return caps
}

// ExtensionLoader handles scanning, registering, and managing extensions.
type ExtensionLoader struct {
	db            *gorm.DB
	extensionsDir string
}

// NewExtensionLoader creates a new ExtensionLoader.
func NewExtensionLoader(db *gorm.DB, extensionsDir string) *ExtensionLoader {
	return &ExtensionLoader{
		db:            db,
		extensionsDir: extensionsDir,
	}
}

// ScanAndRegister scans the extensions directory, reads each extension.json,
// and upserts extension records into the database. New extensions default to is_active=false.
func (l *ExtensionLoader) ScanAndRegister() {
	entries, err := os.ReadDir(l.extensionsDir)
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("[extensions] no extensions directory found at %s", l.extensionsDir)
			return
		}
		log.Printf("[extensions] error reading extensions directory: %v", err)
		return
	}

	registered := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		extDir := filepath.Join(l.extensionsDir, entry.Name())
		manifestPath := filepath.Join(extDir, "extension.json")

		data, err := os.ReadFile(manifestPath)
		if err != nil {
			continue // skip directories without a manifest
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

		ext := models.Extension{
			Slug:        manifest.Slug,
			Name:        manifest.Name,
			Version:     manifest.Version,
			Description: manifest.Description,
			Author:      manifest.Author,
			Path:        extDir,
			Priority:    manifest.Priority,
			Manifest:    models.JSONB(data),
		}

		// Upsert: insert if new, update name/version/description/author/path/priority if exists.
		// Do NOT change is_active on existing records.
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

	log.Printf("[extensions] scanned %d extensions from %s", registered, l.extensionsDir)
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

// UnloadExtensionBlocks removes all block types and templates owned by an extension
// and clears their assets from the registry.
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

	// Remove from asset registry.
	registry.mu.Lock()
	for _, bt := range blockTypes {
		delete(registry.blockAssets, bt.Slug)
	}
	registry.mu.Unlock()
}

// loadExtensionBlocks loads blocks and templates for a single extension. Returns count loaded.
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

	return loaded
}

