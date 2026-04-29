package cms

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"squilla/internal/events"
	"squilla/internal/models"

	"gorm.io/gorm"
)

// This file owns the ThemeLoader lifecycle: scanning the theme
// directory, parsing theme.json, upserting Theme records, and
// tearing down on deactivation. Sibling theme_loader_register.go
// holds the shared Register* helpers it delegates to.

// NewThemeLoader creates a new ThemeLoader.
func NewThemeLoader(db *gorm.DB, registry *ThemeAssetRegistry, eventBus *events.EventBus) *ThemeLoader {
	return &ThemeLoader{
		db:               db,
		registry:         registry,
		eventBus:         eventBus,
		SettingsRegistry: NewThemeSettingsRegistry(),
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

	// Snapshot settings pages into the in-memory registry. Slug derived
	// the same way as upsertThemeRecord so admin/Tengo lookups by active
	// theme slug match the DB row.
	themeSlug := strings.ToLower(strings.ReplaceAll(manifest.Name, " ", "-"))
	tl.SettingsRegistry.SetActive(themeSlug, LoadSettingsPages(themeDir, manifest))

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
			"name":        manifest.Name,
			"path":        themeDir,
			"version":     manifest.Version,
			"assets":      themeAssetsPayload(themeDir, manifest.Assets),
			"image_sizes": themeImageSizesPayload(manifest.ImageSizes),
		})
	}

	return nil
}

// upsertThemeRecord creates or updates the theme record in the themes table.
func (tl *ThemeLoader) upsertThemeRecord(manifest ThemeManifest, themeDir string) {
	slug := strings.ToLower(strings.ReplaceAll(manifest.Name, " ", "-"))

	var existing models.Theme

	// Look up by path first — the scanner creates records using the
	// directory name as slug (e.g. "default"), but the manifest name
	// may produce a different slug (e.g. "squilla-default").  Path is
	// the stable identifier across both registration paths.
	result := tl.db.Where("path = ?", themeDir).First(&existing)
	if result.Error != nil {
		// Fallback: try slug derived from manifest name.
		result = tl.db.Where("slug = ?", slug).First(&existing)
	}

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
	// EXCEPTION: layouts referenced by content_nodes (FK ON DELETE SET NULL)
	// must not be hard-deleted, otherwise every node using them has its
	// layout_id wiped and pages stop rendering after any theme
	// deactivate/reactivate cycle. Orphan them instead — the next theme
	// registration with the same slug will re-claim the row by upsert.
	cond := "source = 'theme' AND theme_name = ?"
	if err := tl.db.Where(cond, themeName).Delete(&models.BlockType{}).Error; err != nil {
		return fmt.Errorf("deregister theme %q: delete block_types: %w", themeName, err)
	}
	if err := tl.db.Where(cond, themeName).Delete(&models.LayoutBlock{}).Error; err != nil {
		return fmt.Errorf("deregister theme %q: delete layout_blocks: %w", themeName, err)
	}
	if err := tl.db.Where(cond, themeName).Delete(&models.Template{}).Error; err != nil {
		return fmt.Errorf("deregister theme %q: delete templates: %w", themeName, err)
	}
	// Layouts: hard-delete only those with zero content_node references.
	// Referenced layouts survive as orphans so FK cascade doesn't null layout_id.
	var referencedLayoutIDs []int
	if err := tl.db.Raw(`SELECT DISTINCT layout_id FROM content_nodes WHERE layout_id IN (SELECT id FROM layouts WHERE source = 'theme' AND theme_name = ?)`, themeName).Scan(&referencedLayoutIDs).Error; err != nil {
		return fmt.Errorf("deregister theme %q: list referenced layouts: %w", themeName, err)
	}
	if len(referencedLayoutIDs) == 0 {
		if err := tl.db.Where(cond, themeName).Delete(&models.Layout{}).Error; err != nil {
			return fmt.Errorf("deregister theme %q: delete layouts: %w", themeName, err)
		}
	} else {
		if err := tl.db.Where(cond+" AND id NOT IN ?", themeName, referencedLayoutIDs).Delete(&models.Layout{}).Error; err != nil {
			return fmt.Errorf("deregister theme %q: delete unreferenced layouts: %w", themeName, err)
		}
		// Orphan the survivors: source='orphan', theme_name=NULL.
		// Upsert-by-slug on re-activation will re-claim them.
		if err := tl.db.Model(&models.Layout{}).Where("id IN ?", referencedLayoutIDs).Updates(map[string]interface{}{"source": "orphan", "theme_name": nil}).Error; err != nil {
			return fmt.Errorf("deregister theme %q: orphan referenced layouts: %w", themeName, err)
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

	// Clear the in-memory settings snapshot ONLY when the theme being
	// deregistered is the one currently held by the registry. The activation
	// flow deregisters several themes in sequence (the previously-active one
	// plus a sweep of already-inactive ones); without this guard, those
	// follow-up deregisters wipe the just-loaded settings of the NEW active
	// theme. Persisted site_settings rows are preserved either way; only
	// ThemeMgmtService.Delete wipes them.
	if tl.SettingsRegistry != nil {
		deregSlug := strings.ToLower(strings.ReplaceAll(themeName, " ", "-"))
		if tl.SettingsRegistry.ActiveSlug() == deregSlug {
			tl.SettingsRegistry.Clear()
		}
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

// themeImageSizesPayload converts a manifest's image_sizes definitions into
// the shape carried by the theme.activated event so extensions (media-manager)
// can upsert them into the registered-sizes table.
func themeImageSizesPayload(defs []ThemeImageSizeDef) []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(defs))
	for _, d := range defs {
		mode := d.Mode
		if mode == "" {
			mode = "fit"
		}
		out = append(out, map[string]interface{}{
			"name":    d.Name,
			"width":   d.Width,
			"height":  d.Height,
			"mode":    mode,
			"quality": d.Quality,
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
	// Determine the active theme's name so we don't purge stale records
	// that share the same name — the active theme's freshly-imported assets
	// must survive (e.g. cold boot on existing DB after upgrade).
	var active models.Theme
	activeName := ""
	if err := tl.db.Where("is_active = ?", true).First(&active).Error; err == nil {
		activeName = active.Name
	}

	var inactive []models.Theme
	if err := tl.db.Where("is_active = ?", false).Find(&inactive).Error; err != nil {
		return fmt.Errorf("purge inactive themes: list: %w", err)
	}
	for _, t := range inactive {
		if activeName != "" && t.Name == activeName {
			// Delete the stale DB row but do NOT deregister — that would
			// emit theme.deactivated and purge the active theme's assets.
			tl.db.Delete(&t)
			log.Printf("purged stale record for inactive %q (name matches active theme, skipping deregister)", t.Name)
			continue
		}
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
