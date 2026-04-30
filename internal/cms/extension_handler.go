package cms

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"squilla/internal/api"
	"squilla/internal/auth"
	"squilla/internal/events"
	"squilla/internal/models"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// ExtensionScriptLoader is an interface for loading/unloading extension scripts at runtime.
type ExtensionScriptLoader interface {
	LoadExtensionScripts(extDir string, slug string, capabilities ...map[string]bool) error
	UnloadExtensionScripts(extDir string, slug string)
}

// ExtensionHandler provides HTTP handlers for extension management.
type ExtensionHandler struct {
	db            *gorm.DB
	loader        *ExtensionLoader
	scriptLoader  ExtensionScriptLoader
	pluginManager *PluginManager
	assetRegistry *ThemeAssetRegistry
	eventBus      *events.EventBus
	themeLoader   *ThemeLoader
}

// NewExtensionHandler creates a new ExtensionHandler.
func NewExtensionHandler(db *gorm.DB, loader *ExtensionLoader) *ExtensionHandler {
	return &ExtensionHandler{db: db, loader: loader}
}

// SetScriptLoader sets the script engine for hot-reloading extension scripts.
func (h *ExtensionHandler) SetScriptLoader(sl ExtensionScriptLoader) {
	h.scriptLoader = sl
}

// SetPluginManager sets the plugin manager for starting/stopping gRPC plugins.
func (h *ExtensionHandler) SetPluginManager(pm *PluginManager) {
	h.pluginManager = pm
}

// SetAssetRegistry sets the theme asset registry for extension block loading.
func (h *ExtensionHandler) SetAssetRegistry(r *ThemeAssetRegistry) {
	h.assetRegistry = r
}

// SetEventBus sets the event bus used to publish extension lifecycle events.
func (h *ExtensionHandler) SetEventBus(b *events.EventBus) {
	h.eventBus = b
}

// SetThemeLoader sets the theme loader, used to replay theme.activated events
// when extensions are activated at runtime (so they can import theme assets
// without requiring a server restart).
func (h *ExtensionHandler) SetThemeLoader(tl *ThemeLoader) {
	h.themeLoader = tl
}

// RegisterRoutes registers all admin API extension routes on the provided router group.
func (h *ExtensionHandler) RegisterRoutes(router fiber.Router) {
	g := router.Group("/extensions", auth.CapabilityRequired("manage_settings"))
	g.Get("/manifests", h.Manifests)
	g.Get("/", h.List)
	g.Get("/:slug/files", h.BrowseFiles)
	g.Get("/:slug/settings", h.GetSettings)
	g.Put("/:slug/settings", h.UpdateSettings)
	g.Get("/:slug/preview", h.ServePreview)
	g.Get("/:slug/assets/*", h.ServeAsset)
	g.Get("/:slug", h.Get)
	g.Post("/:slug/activate", h.Activate)
	g.Post("/:slug/deactivate", h.Deactivate)
	g.Post("/upload", h.Upload)
	g.Delete("/:slug", h.Delete)
}

// List handles GET /extensions — returns all extensions.
func (h *ExtensionHandler) List(c *fiber.Ctx) error {
	exts, err := h.loader.List()
	if err != nil {
		return api.Error(c, fiber.StatusInternalServerError, "LIST_FAILED", "Failed to list extensions")
	}
	return api.Success(c, exts)
}

// Get handles GET /extensions/:slug — returns a single extension.
func (h *ExtensionHandler) Get(c *fiber.Ctx) error {
	slug := c.Params("slug")
	ext, err := h.loader.GetBySlug(slug)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return api.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Extension not found")
		}
		return api.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", "Failed to fetch extension")
	}
	return api.Success(c, ext)
}

// Manifests handles GET /extensions/manifests — returns admin_ui manifests for all active extensions.
func (h *ExtensionHandler) Manifests(c *fiber.Ctx) error {
	exts, err := h.loader.GetActive()
	if err != nil {
		return api.Error(c, fiber.StatusInternalServerError, "LIST_FAILED", "Failed to list extensions")
	}

	type manifestEntry struct {
		Slug     string          `json:"slug"`
		Name     string          `json:"name"`
		Manifest json.RawMessage `json:"manifest"`
	}

	entries := make([]manifestEntry, 0, len(exts))
	for _, ext := range exts {
		entries = append(entries, manifestEntry{
			Slug:     ext.Slug,
			Name:     ext.Name,
			Manifest: json.RawMessage(ext.Manifest),
		})
	}
	return api.Success(c, entries)
}

// GetSettings handles GET /extensions/:slug/settings — returns extension settings.
func (h *ExtensionHandler) GetSettings(c *fiber.Ctx) error {
	slug := c.Params("slug")

	if _, err := h.loader.GetBySlug(slug); err != nil {
		return api.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Extension not found")
	}

	prefix := "ext." + slug + "."
	var settings []models.SiteSetting
	h.db.Where("key LIKE ?", prefix+"%").Find(&settings)

	result := make(map[string]string)
	for _, s := range settings {
		key := strings.TrimPrefix(s.Key, prefix)
		if s.Value != nil {
			result[key] = *s.Value
		}
	}
	return api.Success(c, result)
}

// UpdateSettings handles PUT /extensions/:slug/settings — updates extension settings.
func (h *ExtensionHandler) UpdateSettings(c *fiber.Ctx) error {
	slug := c.Params("slug")

	if _, err := h.loader.GetBySlug(slug); err != nil {
		return api.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Extension not found")
	}

	var body map[string]string
	if err := c.BodyParser(&body); err != nil {
		return api.Error(c, fiber.StatusBadRequest, "INVALID_BODY", "Invalid request body")
	}

	prefix := "ext." + slug + "."
	for key, value := range body {
		// Validate setting key to prevent namespace collisions.
		if !isValidSettingsKey(key) {
			return api.Error(c, fiber.StatusBadRequest, "INVALID_KEY", fmt.Sprintf("Invalid settings key: %s", key))
		}
		v := value
		setting := models.SiteSetting{
			Key:   prefix + key,
			Value: &v,
		}
		h.db.Where("key = ?", setting.Key).Assign(models.SiteSetting{Value: &v}).FirstOrCreate(&setting)
	}

	return api.Success(c, fiber.Map{"message": "Settings saved"})
}

// ServePreview handles GET /extensions/:slug/preview — serves preview image from extension directory.
// Looks for preview.svg, preview.png, preview.jpg in the extension root. Falls back to default.
func (h *ExtensionHandler) ServePreview(c *fiber.Ctx) error {
	slug := c.Params("slug")
	ext, err := h.loader.GetBySlug(slug)
	if err != nil {
		return c.Redirect("/admin/previews/default-extension.svg")
	}

	for _, name := range []string{"preview.svg", "preview.png", "preview.jpg", "preview.webp"} {
		path := filepath.Join(ext.Path, name)
		if _, err := os.Stat(path); err == nil {
			c.Set("Cache-Control", "public, max-age=3600")
			return c.SendFile(path)
		}
	}

	return c.Redirect("/admin/previews/default-extension.svg")
}

// ServeAsset handles GET /extensions/:slug/assets/* — serves static files from extension admin-ui/dist/.
func (h *ExtensionHandler) ServeAsset(c *fiber.Ctx) error {
	slug := c.Params("slug")
	filePath := c.Params("*")

	ext, err := h.loader.GetBySlug(slug)
	if err != nil {
		return api.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Extension not found")
	}

	fullPath := filepath.Join(ext.Path, "admin-ui", "dist", filePath)
	cleanPath := filepath.Clean(fullPath)
	basePath := filepath.Clean(filepath.Join(ext.Path, "admin-ui", "dist"))

	// Prefix check MUST end at a path separator — otherwise a sibling
	// directory whose name starts with "dist" (e.g. "dist-evil/") would
	// pass containment when filePath crafts a "../dist-evil/x" jump.
	// Equality covers the basePath-itself case (file inside dist root).
	if cleanPath != basePath && !strings.HasPrefix(cleanPath, basePath+string(os.PathSeparator)) {
		return api.Error(c, fiber.StatusBadRequest, "INVALID_PATH", "Path traversal not allowed")
	}

	c.Set("Cache-Control", "no-cache, no-store, must-revalidate")
	return c.SendFile(cleanPath)
}

// BrowseFiles handles GET /extensions/:slug/files?path= — browse extension files.
func (h *ExtensionHandler) BrowseFiles(c *fiber.Ctx) error {
	slug := c.Params("slug")

	var ext models.Extension
	if err := h.db.Where("slug = ?", slug).First(&ext).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return api.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Extension not found")
		}
		return api.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", "Failed to fetch extension")
	}

	result, err := BrowseFilesInDir(ext.Path, c.Query("path", ""))
	if err != nil {
		msg := err.Error()
		if msg == "INVALID_PATH" {
			return api.Error(c, fiber.StatusBadRequest, "INVALID_PATH", "Path traversal is not allowed")
		}
		if msg == "NOT_FOUND" {
			return api.Error(c, fiber.StatusNotFound, "PATH_NOT_FOUND", "The requested path was not found")
		}
		return api.Error(c, fiber.StatusInternalServerError, "BROWSE_FAILED", "Failed to browse files")
	}

	return api.Success(c, result)
}

// HotActivate flips is_active=true and performs the full hot-load sequence:
// SQL migrations, Tengo scripts, gRPC plugin start, block registry load, and
// the extension.activated event. Shared between the Fiber handler and the MCP
// adapter — do not duplicate this sequence elsewhere.
func (h *ExtensionHandler) HotActivate(slug string) error {
	if err := h.loader.Activate(slug); err != nil {
		return err
	}
	ext, err := h.loader.GetBySlug(slug)
	if err != nil {
		return nil
	}
	if migErr := RunExtensionMigrations(h.db, ext.Path, slug); migErr != nil {
		log.Printf("[extensions] warning: failed to run migrations for %s: %v", slug, migErr)
	}
	var manifest ExtensionManifest
	_ = json.Unmarshal(ext.Manifest, &manifest)
	// Validate admin_ui.entry resolves to a real file under the extension dir.
	// A typo here silently breaks the SPA shell's import map for this extension
	// — fail loudly so the operator sees the problem at activation time.
	if manifest.AdminUI != nil && manifest.AdminUI.Entry != "" {
		entryPath := filepath.Join(ext.Path, manifest.AdminUI.Entry)
		if _, statErr := os.Stat(entryPath); statErr != nil {
			log.Printf("[extensions] WARNING: admin_ui.entry %q for extension %q does not resolve to a file (%s) — admin UI for this extension will not load. Check extension.json.", manifest.AdminUI.Entry, slug, statErr)
		}
	}
	caps := manifest.CapabilityMap()
	owned := manifest.OwnedTablesMap()
	if h.scriptLoader != nil {
		if loadErr := h.scriptLoader.LoadExtensionScripts(ext.Path, slug, caps); loadErr != nil {
			log.Printf("[extensions] warning: failed to hot-load scripts for %s: %v", slug, loadErr)
		}
	}
	if h.pluginManager != nil {
		if startErr := h.pluginManager.StartPlugins(ext.Path, slug, json.RawMessage(ext.Manifest), caps, owned); startErr != nil {
			log.Printf("[extensions] warning: failed to start plugins for %s: %v", slug, startErr)
		}
	}
	if h.assetRegistry != nil {
		h.loader.LoadBlocksForExtension(slug, h.assetRegistry)
	}
	PublishExtensionActivated(h.eventBus, slug, ext.Path, json.RawMessage(ext.Manifest))

	// Replay theme.activated so the newly-activated extension can react to
	// the current active theme (e.g. media-manager importing theme assets).
	// Without this, extensions activated after boot miss the theme.activated
	// event that fired during startup.
	h.replayThemeActivated()

	return nil
}

// replayThemeActivated re-emits the theme.activated event for the currently
// active theme. This allows extensions activated at runtime (after boot) to
// process theme assets — e.g. media-manager importing theme images into its
// media_files table. No-op if there is no active theme or no event bus.
func (h *ExtensionHandler) replayThemeActivated() {
	if h.eventBus == nil {
		return
	}
	var activeTheme struct {
		Name string
		Path string
	}
	if err := h.db.Table("themes").Where("is_active = ?", true).
		Select("name, path").Take(&activeTheme).Error; err != nil {
		return
	}
	manifestPath := filepath.Join(activeTheme.Path, "theme.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return
	}
	var mf ThemeManifest
	if err := json.Unmarshal(data, &mf); err != nil {
		return
	}
	h.eventBus.PublishSync("theme.activated", events.Payload{
		"name":    mf.Name,
		"path":    activeTheme.Path,
		"version": mf.Version,
		"assets":  themeAssetsPayload(activeTheme.Path, mf.Assets),
	})
}

// HotDeactivate flips is_active=false and performs the full hot-unload
// sequence: extension.deactivated event, script unload, plugin stop, block
// registry unload.
func (h *ExtensionHandler) HotDeactivate(slug string) error {
	ext, _ := h.loader.GetBySlug(slug)
	if err := h.loader.Deactivate(slug); err != nil {
		return err
	}
	if ext == nil {
		return nil
	}
	PublishExtensionDeactivated(h.eventBus, slug)
	if h.scriptLoader != nil {
		h.scriptLoader.UnloadExtensionScripts(ext.Path, slug)
	}
	if h.pluginManager != nil {
		h.pluginManager.StopPlugins(slug)
	}
	if h.assetRegistry != nil {
		h.loader.UnloadExtensionBlocks(slug, h.assetRegistry)
	}
	return nil
}

// Activate handles POST /extensions/:slug/activate.
func (h *ExtensionHandler) Activate(c *fiber.Ctx) error {
	slug := strings.Clone(c.Params("slug"))
	if err := h.HotActivate(slug); err != nil {
		if err.Error() == "extension not found: "+slug {
			return api.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Extension not found")
		}
		return api.Error(c, fiber.StatusInternalServerError, "ACTIVATE_FAILED", "Failed to activate extension")
	}
	return api.Success(c, fiber.Map{"message": "Extension activated"})
}

// Deactivate handles POST /extensions/:slug/deactivate.
func (h *ExtensionHandler) Deactivate(c *fiber.Ctx) error {
	slug := strings.Clone(c.Params("slug"))
	if err := h.HotDeactivate(slug); err != nil {
		if err.Error() == "extension not found: "+slug {
			return api.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Extension not found")
		}
		return api.Error(c, fiber.StatusInternalServerError, "DEACTIVATE_FAILED", "Failed to deactivate extension")
	}
	return api.Success(c, fiber.Map{"message": "Extension deactivated"})
}

// Upload handles POST /extensions/upload — uploads a ZIP containing an extension.

// Delete handles DELETE /extensions/:slug — removes extension files and DB record.
func (h *ExtensionHandler) Delete(c *fiber.Ctx) error {
	slug := c.Params("slug")
	if err := h.DeleteBySlug(slug); err != nil {
		switch err.Error() {
		case "NOT_FOUND":
			return api.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Extension not found")
		case "STILL_ACTIVE":
			return api.Error(c, fiber.StatusBadRequest, "STILL_ACTIVE", "Deactivate extension before deleting")
		default:
			return api.Error(c, fiber.StatusInternalServerError, "DELETE_FAILED", err.Error())
		}
	}
	return api.Success(c, fiber.Map{"message": "Extension deleted"})
}

// DeleteBySlug performs the same destructive operation as the HTTP Delete
// handler without a Fiber context. Returns sentinel error strings
// "NOT_FOUND" or "STILL_ACTIVE" so callers (HTTP, MCP) can map to their own
// response shapes. Any other error is a real I/O / DB failure.
//
// Only the data-dir copy is wiped; the bundled image directory is read-only
// by intent. The next ScanAndRegister will re-register the bundled extension
// as a fresh inactive entry, so "delete" effectively means "uninstall the
// operator override and fall back to the bundled version" rather than
// "obliterate forever".
func (h *ExtensionHandler) DeleteBySlug(slug string) error {
	var ext models.Extension
	if err := h.db.Where("slug = ?", slug).First(&ext).Error; err != nil {
		return fmt.Errorf("NOT_FOUND")
	}
	if ext.IsActive {
		return fmt.Errorf("STILL_ACTIVE")
	}

	if h.assetRegistry != nil {
		h.loader.UnloadExtensionBlocks(slug, h.assetRegistry)
	}

	extDir := filepath.Join(h.loader.dataDir, slug)
	if err := os.RemoveAll(extDir); err != nil {
		log.Printf("WARN: failed to remove extension dir %s: %v", extDir, err)
	}

	h.db.Where("slug = ?", slug).Delete(&models.Extension{})
	return nil
}

// isValidSettingsKey checks that a settings key contains only safe characters.
func isValidSettingsKey(key string) bool {
	if key == "" || len(key) > 128 {
		return false
	}
	for _, ch := range key {
		if !((ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '_' || ch == '-') {
			return false
		}
	}
	return true
}
