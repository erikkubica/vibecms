package cms

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"squilla/internal/models"
	"squilla/internal/secrets"

	"gorm.io/gorm"
)

// themeMgmtManifest is the theme.json structure used by the management service.
// It extends ThemeManifest with a Slug field for installation purposes.
type themeMgmtManifest struct {
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Version     string `json:"version"`
	Author      string `json:"author"`
	Description string `json:"description"`
}

// ThemeMgmtService manages theme installation, activation, and lifecycle.
//
// Themes live in two parallel directories:
//   - bundledDir (e.g. "themes")        — shipped with the image, read-only intent.
//   - dataDir    (e.g. "data/themes")   — operator-installed, persistent across
//     container restarts when mounted as a volume.
//
// Both are scanned at boot. Writes (git install, zip upload, deploy, delete)
// always target dataDir so the image's bundled themes stay intact. On slug
// collision the dataDir copy wins so an operator can override a bundled
// theme by deploying a same-slug replacement.
type ThemeMgmtService struct {
	db          *gorm.DB
	themeLoader *ThemeLoader
	bundledDir  string           // image-bundled themes (read-only intent)
	dataDir     string           // user-installed themes (volume-backed)
	secrets     *secrets.Service // may be nil; transparently encrypts/decrypts git tokens

	// Callbacks for loading/unloading Tengo scripts. Set via SetScriptLoader
	// to avoid an import cycle on the scripting package.
	loadThemeScripts   func(themeDir string) error
	unloadThemeScripts func()
}

// NewThemeMgmtService creates a new ThemeMgmtService. Pass a non-nil
// *secrets.Service to encrypt git tokens at rest; nil keeps tokens
// plaintext (legacy behaviour).
//
// dataDir is auto-created if missing so a fresh container without the
// volume populated can still scan/install without an explicit mkdir.
func NewThemeMgmtService(db *gorm.DB, themeLoader *ThemeLoader, bundledDir, dataDir string, secretsSvc *secrets.Service) *ThemeMgmtService {
	if dataDir != "" {
		_ = os.MkdirAll(dataDir, 0o755)
	}
	return &ThemeMgmtService{
		db:          db,
		themeLoader: themeLoader,
		bundledDir:  bundledDir,
		dataDir:     dataDir,
		secrets:     secretsSvc,
	}
}

// resolveGitToken returns the plaintext token from a stored model, decrypting
// the envelope when present. Legacy plaintext rows pass through unchanged.
func (s *ThemeMgmtService) resolveGitToken(stored *string) (string, error) {
	if stored == nil || *stored == "" {
		return "", nil
	}
	if s.secrets == nil {
		return *stored, nil
	}
	return s.secrets.Decrypt(*stored)
}

// wrapGitToken encrypts a plaintext token for storage if the secrets service
// is configured. Empty input returns empty (no zero-length token wrapping).
func (s *ThemeMgmtService) wrapGitToken(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}
	if s.secrets == nil {
		return plaintext, nil
	}
	return s.secrets.Encrypt(plaintext)
}

// SetScriptLoader wires callbacks for loading/unloading Tengo scripts during
// theme activation. Accepts function values to avoid an import cycle on the
// scripting package. load runs theme.tengo (registers node types, taxonomies,
// seeds content, event handlers, filters, routes). unload tears those down.
func (s *ThemeMgmtService) SetScriptLoader(load func(string) error, unload func()) {
	s.loadThemeScripts = load
	s.unloadThemeScripts = unload
}

// ScanAndRegister walks both the bundled and data theme directories and
// upserts a Theme row for every subdirectory that has a valid theme.json.
// Data dir wins on slug collision so an operator override replaces the
// image-bundled copy without renaming.
//
// Called at startup so on-disk themes don't have to be manually uploaded after
// a fresh install or DB reset.
func (s *ThemeMgmtService) ScanAndRegister() {
	// Order matters: scan dataDir first to claim slugs, then bundled. The
	// per-entry registerScannedTheme helper detects an existing row with the
	// same slug and skips when the registered row already lives in dataDir
	// (the override wins).
	dataCount := s.scanDir(s.dataDir, true)
	bundledCount := s.scanDir(s.bundledDir, false)
	log.Printf("[themes] scanned %d themes (%d data, %d bundled) from %s + %s",
		dataCount+bundledCount, dataCount, bundledCount, s.dataDir, s.bundledDir)
}

// scanDir walks one root and registers each theme it finds. Returns the
// number of rows successfully upserted. Missing dir is not an error — both
// roots are optional (image with no bundled themes, fresh volume).
func (s *ThemeMgmtService) scanDir(root string, isData bool) int {
	if root == "" {
		return 0
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("[themes] error reading %s: %v", root, err)
		}
		return 0
	}

	registered := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		themeDir := filepath.Join(root, entry.Name())
		manifest, _, err := findAndParseManifest(themeDir)
		if err != nil {
			continue
		}
		if manifest.Slug == "" {
			if manifest.Name != "" {
				manifest.Slug = strings.ToLower(strings.ReplaceAll(manifest.Name, " ", "-"))
			} else {
				manifest.Slug = entry.Name()
			}
		}

		// Look up by path first (stable identifier) then by slug.
		var existing models.Theme
		err = s.db.Where("path = ?", themeDir).First(&existing).Error
		if err == gorm.ErrRecordNotFound {
			err = s.db.Where("slug = ?", manifest.Slug).First(&existing).Error
		}
		if err == nil {
			// Bundled scan must NOT overwrite a data-backed registration —
			// the operator's override wins. Detect by comparing the
			// existing path against the dataDir prefix.
			if !isData && s.dataDir != "" && strings.HasPrefix(existing.Path, s.dataDir+string(os.PathSeparator)) {
				continue
			}
			existing.Name = manifest.Name
			existing.Description = manifest.Description
			existing.Version = manifest.Version
			existing.Author = manifest.Author
			existing.Path = themeDir
			if err := s.db.Save(&existing).Error; err != nil {
				log.Printf("[themes] refresh %s failed: %v", manifest.Slug, err)
				continue
			}
			registered++
			continue
		}
		if err != gorm.ErrRecordNotFound {
			log.Printf("[themes] lookup %s failed: %v", manifest.Slug, err)
			continue
		}

		theme := models.Theme{
			Slug:        manifest.Slug,
			Name:        manifest.Name,
			Description: manifest.Description,
			Version:     manifest.Version,
			Author:      manifest.Author,
			Source:      "local",
			Path:        themeDir,
		}
		if err := s.db.Create(&theme).Error; err != nil {
			log.Printf("[themes] register %s failed: %v", manifest.Slug, err)
			continue
		}
		registered++
	}
	return registered
}

// List returns all installed themes ordered by name.
func (s *ThemeMgmtService) List() ([]models.Theme, error) {
	var themes []models.Theme
	if err := s.db.Order("name ASC").Find(&themes).Error; err != nil {
		return nil, fmt.Errorf("failed to list themes: %w", err)
	}
	return themes, nil
}

// GetByID retrieves a single theme by its ID.
func (s *ThemeMgmtService) GetByID(id int) (*models.Theme, error) {
	var theme models.Theme
	if err := s.db.First(&theme, id).Error; err != nil {
		return nil, err
	}
	return &theme, nil
}

// ThemesDir returns the writable directory where new themes are installed.
// Exposed so the MCP checklist tool and install paths can target a single
// canonical location. The bundled directory is intentionally not surfaced
// here because nothing should ever write into it.
func (s *ThemeMgmtService) ThemesDir() string { return s.dataDir }

// BundledThemesDir returns the read-only image-bundled themes directory.
// Useful for diagnostics; not used by write paths.
func (s *ThemeMgmtService) BundledThemesDir() string { return s.bundledDir }

// GetActive returns the currently active theme.
func (s *ThemeMgmtService) GetActive() (*models.Theme, error) {
	var theme models.Theme
	if err := s.db.Where("is_active = ?", true).First(&theme).Error; err != nil {
		return nil, err
	}
	return &theme, nil
}

// MaxThemeUploadSize caps the size of a theme archive accepted by
// InstallFromZip. 50 MB matches the extension limit; themes are
// asset-heavier than extensions but still fit comfortably.
const MaxThemeUploadSize = 50 * 1024 * 1024

// InstallFromZip extracts a ZIP archive and installs the theme.
//
// Safety:
//   - The archive is bounded to MaxThemeUploadSize. The Reader is wrapped in
//     io.LimitReader so a streaming caller cannot exhaust memory.
//   - The slug declared in theme.json must satisfy isValidSettingsKey before
//     any path is constructed, blocking ../-style escapes via a hostile
//     manifest.
//   - Each entry is checked for zip-slip during extraction.
//   - Extraction lands in themes/<slug>.deploy.tmp; the final swap into
//     themes/<slug> is a single os.Rename so the watcher and theme loader
//     never observe a partial directory.
func (s *ThemeMgmtService) InstallFromZip(file io.Reader, filename string) (*models.Theme, error) {
	// Read entire ZIP into memory, capping at MaxThemeUploadSize+1 so we can
	// distinguish "exactly the cap" from "over the cap".
	data, err := io.ReadAll(io.LimitReader(file, MaxThemeUploadSize+1))
	if err != nil {
		return nil, fmt.Errorf("failed to read zip data: %w", err)
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("empty archive")
	}
	if len(data) > MaxThemeUploadSize {
		return nil, fmt.Errorf("archive exceeds maximum size of %d bytes", MaxThemeUploadSize)
	}

	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("failed to open zip archive: %w", err)
	}

	// Create a temp directory for extraction.
	tmpDir, err := os.MkdirTemp("", "squilla-theme-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Extract all files, guarding against zip-slip.
	for _, f := range zr.File {
		destPath := filepath.Join(tmpDir, f.Name)
		// Zip-slip protection: ensure extracted path stays within tmpDir.
		if !strings.HasPrefix(filepath.Clean(destPath), filepath.Clean(tmpDir)+string(os.PathSeparator)) {
			return nil, fmt.Errorf("zip slip detected: %s", f.Name)
		}

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(destPath, 0755); err != nil {
				return nil, fmt.Errorf("failed to create directory %s: %w", destPath, err)
			}
			continue
		}

		// Ensure parent directory exists.
		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return nil, fmt.Errorf("failed to create parent dir for %s: %w", destPath, err)
		}

		if err := extractZipFile(f, destPath); err != nil {
			return nil, fmt.Errorf("failed to extract %s: %w", f.Name, err)
		}
	}

	// Find theme.json (root or one level deep).
	manifest, manifestDir, err := findAndParseManifest(tmpDir)
	if err != nil {
		return nil, err
	}

	if manifest.Slug == "" {
		return nil, fmt.Errorf("theme.json missing required 'slug' field")
	}
	if !isValidSettingsKey(manifest.Slug) {
		return nil, fmt.Errorf("theme slug %q contains invalid characters", manifest.Slug)
	}

	// Atomic swap: copy into <dataDir>/<slug>.deploy.tmp, then rename. Both
	// paths live under dataDir so the rename never crosses a filesystem
	// boundary. Writes always target dataDir; the bundled image dir stays
	// untouched.
	destDir := filepath.Join(s.dataDir, manifest.Slug)
	stagingDir := destDir + ".deploy.tmp"
	backupDir := destDir + ".deploy.old"

	_ = os.RemoveAll(stagingDir)
	_ = os.RemoveAll(backupDir)

	if err := copyDir(manifestDir, stagingDir); err != nil {
		_ = os.RemoveAll(stagingDir)
		return nil, fmt.Errorf("failed to copy theme to %s: %w", stagingDir, err)
	}

	hadExisting := false
	if _, statErr := os.Stat(destDir); statErr == nil {
		hadExisting = true
		if err := os.Rename(destDir, backupDir); err != nil {
			_ = os.RemoveAll(stagingDir)
			return nil, fmt.Errorf("backup existing theme dir: %w", err)
		}
	}
	if err := os.Rename(stagingDir, destDir); err != nil {
		if hadExisting {
			_ = os.Rename(backupDir, destDir)
		}
		_ = os.RemoveAll(stagingDir)
		return nil, fmt.Errorf("swap in deployed theme dir: %w", err)
	}
	if hadExisting {
		_ = os.RemoveAll(backupDir)
	}

	// Upsert the DB record. Re-deploying an existing theme refreshes its
	// metadata in place so the theme registry survives across deploys.
	var theme models.Theme
	switch err := s.db.Where("slug = ?", manifest.Slug).First(&theme).Error; err {
	case nil:
		theme.Name = manifest.Name
		theme.Description = manifest.Description
		theme.Version = manifest.Version
		theme.Author = manifest.Author
		theme.Path = destDir
		if err := s.db.Save(&theme).Error; err != nil {
			return nil, fmt.Errorf("failed to refresh theme record: %w", err)
		}
	case gorm.ErrRecordNotFound:
		theme = models.Theme{
			Slug:        manifest.Slug,
			Name:        manifest.Name,
			Description: manifest.Description,
			Version:     manifest.Version,
			Author:      manifest.Author,
			Source:      "upload",
			Path:        destDir,
		}
		if err := s.db.Create(&theme).Error; err != nil {
			os.RemoveAll(destDir)
			return nil, fmt.Errorf("failed to create theme record: %w", err)
		}
	default:
		return nil, fmt.Errorf("failed to look up theme: %w", err)
	}

	return &theme, nil
}

// InstallFromGit clones a git repository and installs the theme.
// Hardening (see theme_git_safety.go for details):
//   - URL validated against scheme allowlist + private-CIDR block
//   - Token never lands in argv (GIT_ASKPASS via temp helper)
//   - Hostile-config defenses (-c core.hooksPath=/dev/null, fsmonitor=false)
func (s *ThemeMgmtService) Activate(id int) error {
	theme, err := s.GetByID(id)
	if err != nil {
		return err
	}

	// Pre-flight: verify the new theme's manifest is actually on disk before
	// we destroy the previous theme's registration. A missing theme.json
	// means the deploy ate the files (e.g. theme.deploy unpacks into a
	// non-persistent container layer that gets wiped on restart). Without
	// this check Activate would deregister the previous theme, then fail
	// silently in LoadTheme (it soft-fails on missing manifest), leaving
	// the site with NO blocks/layouts/templates registered at all.
	manifestPath := filepath.Join(theme.Path, "theme.json")
	if _, err := os.Stat(manifestPath); err != nil {
		return fmt.Errorf(
			"cannot activate theme %q: theme.json missing at %s — the theme directory may have been wiped (non-persistent volume?). Re-deploy the theme before activating",
			theme.Slug, manifestPath,
		)
	}

	// Find the currently active theme so we can deregister it.
	var prevActive models.Theme
	hasPrev := s.db.Where("is_active = ?", true).First(&prevActive).Error == nil

	// Unload previous theme's Tengo scripts (event handlers, filters, routes)
	// before deregistering its DB records.
	if hasPrev && s.unloadThemeScripts != nil {
		s.unloadThemeScripts()
	}

	// Deregister the previous theme (cleans up its blocks, layouts, partials,
	// templates from DB and in-memory registry, emits theme.deactivated).
	// This MUST happen before the new theme loads so that:
	//   1. Extensions (e.g. media-manager) receive theme.deactivated and purge
	//      the old theme's imported assets.
	//   2. Old theme's DB records don't collide with the new theme's.
	if hasPrev {
		if err := s.themeLoader.DeregisterTheme(prevActive.Name); err != nil {
			log.Printf("WARN: deregister previous theme %q: %v", prevActive.Name, err)
		}
	}

	// Activate the target theme in the DB.
	if err := s.db.Model(&models.Theme{}).Where("is_active = ?", true).Update("is_active", false).Error; err != nil {
		return fmt.Errorf("failed to deactivate themes: %w", err)
	}
	if err := s.db.Model(&models.Theme{}).Where("id = ?", id).Update("is_active", true).Error; err != nil {
		return fmt.Errorf("failed to activate theme %d: %w", id, err)
	}

	// Load the new theme (registers blocks/layouts/partials, emits theme.activated).
	return s.Reload(theme.Path)
}

// Deactivate sets the given theme as inactive and removes all its registered
// blocks, layouts, partials, and templates from the DB and in-memory registry.
// Idempotent: safe to call on an already-inactive theme to re-run cleanup.
func (s *ThemeMgmtService) Deactivate(id int) error {
	theme, err := s.GetByID(id)
	if err != nil {
		return err
	}

	if err := s.db.Model(&models.Theme{}).Where("id = ?", id).Update("is_active", false).Error; err != nil {
		return fmt.Errorf("failed to deactivate theme %d: %w", id, err)
	}

	// Unload theme Tengo scripts before deregistering DB records.
	if s.unloadThemeScripts != nil {
		s.unloadThemeScripts()
	}

	if err := s.themeLoader.DeregisterTheme(theme.Name); err != nil {
		log.Printf("WARN: deactivate theme %d: %v", id, err)
	}

	return nil
}

// Delete removes a theme from both the filesystem and database.
func (s *ThemeMgmtService) Delete(id int) error {
	theme, err := s.GetByID(id)
	if err != nil {
		return err
	}

	if theme.IsActive {
		return fmt.Errorf("cannot delete the active theme; deactivate it first")
	}

	// Remove filesystem directory. Refuse to wipe anything under the
	// image-bundled root — those are read-only by intent. The DB row goes
	// regardless; the next scan will re-register the bundled theme as a
	// fresh inactive entry, restoring the operator-visible "I can pick this
	// theme" state without ever touching the image files.
	if theme.Path != "" {
		if s.bundledDir != "" && strings.HasPrefix(theme.Path, s.bundledDir+string(os.PathSeparator)) {
			log.Printf("[themes] delete %s: skipping rmdir of bundled path %s (unregistering only)",
				theme.Slug, theme.Path)
		} else if err := os.RemoveAll(theme.Path); err != nil {
			return fmt.Errorf("failed to remove theme directory %s: %w", theme.Path, err)
		}
	}

	// Delete DB record.
	if err := s.db.Delete(&models.Theme{}, id).Error; err != nil {
		return fmt.Errorf("failed to delete theme record: %w", err)
	}

	// Settings rows wiped here only on full deletion; deactivation preserves
	// them so re-activating restores user values.
	if err := DeleteThemeSettings(context.Background(), s.db, theme.Slug); err != nil {
		log.Printf("WARN: failed to delete theme settings for %s: %v", theme.Slug, err)
		// Don't fail the whole Delete on this — theme row is already gone, settings
		// rows are orphaned but harmless and the operator can clean up manually.
	}

	return nil
}

// Reload re-registers layouts, blocks, and assets from a theme directory,
// then reloads its Tengo scripts (node types, taxonomies, seeding, events,
// filters, and routes).
func (s *ThemeMgmtService) Reload(themePath string) error {
	if err := s.themeLoader.LoadTheme(themePath); err != nil {
		return err
	}
	if s.unloadThemeScripts != nil {
		s.unloadThemeScripts()
	}
	if s.loadThemeScripts != nil {
		if err := s.loadThemeScripts(themePath); err != nil {
			log.Printf("WARN: failed to load theme scripts for %s: %v", themePath, err)
		}
	}
	return nil
}

// --- Helper functions ---

// extractZipFile extracts a single file from a zip archive to destPath.
