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
	Name           string                   `json:"name"`
	Slug           string                   `json:"slug"`
	Version        string                   `json:"version"`
	Author         string                   `json:"author"`
	Description    string                   `json:"description"`
	Priority       int                      `json:"priority"`
	Provides       []string                 `json:"provides"`
	AdminUI        *AdminUIManifest         `json:"admin_ui"`
	SettingsSchema map[string]SettingsField `json:"settings_schema"`
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
