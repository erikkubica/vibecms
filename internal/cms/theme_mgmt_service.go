package cms

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"vibecms/internal/models"

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
type ThemeMgmtService struct {
	db          *gorm.DB
	themeLoader *ThemeLoader
	themesDir   string // base directory e.g. "themes"

	// Callbacks for loading/unloading Tengo scripts. Set via SetScriptLoader
	// to avoid an import cycle on the scripting package.
	loadThemeScripts   func(themeDir string) error
	unloadThemeScripts func()
}

// NewThemeMgmtService creates a new ThemeMgmtService.
func NewThemeMgmtService(db *gorm.DB, themeLoader *ThemeLoader, themesDir string) *ThemeMgmtService {
	return &ThemeMgmtService{
		db:          db,
		themeLoader: themeLoader,
		themesDir:   themesDir,
	}
}

// SetScriptLoader wires callbacks for loading/unloading Tengo scripts during
// theme activation. Accepts function values to avoid an import cycle on the
// scripting package. load runs theme.tengo (registers node types, taxonomies,
// seeds content, event handlers, filters, routes). unload tears those down.
func (s *ThemeMgmtService) SetScriptLoader(load func(string) error, unload func()) {
	s.loadThemeScripts = load
	s.unloadThemeScripts = unload
}

// ScanAndRegister walks themesDir and upserts a Theme row for every
// subdirectory that has a valid theme.json. Mirrors the extension loader's
// scan behaviour: new themes are registered with is_active=false; existing
// rows have their metadata (name/version/description/author/path) refreshed
// from disk, is_active is left untouched.
//
// Called at startup so on-disk themes don't have to be manually uploaded after
// a fresh install or DB reset.
func (s *ThemeMgmtService) ScanAndRegister() {
	entries, err := os.ReadDir(s.themesDir)
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("[themes] no themes directory found at %s", s.themesDir)
			return
		}
		log.Printf("[themes] error reading themes directory: %v", err)
		return
	}

	registered := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		themeDir := filepath.Join(s.themesDir, entry.Name())
		manifest, _, err := findAndParseManifest(themeDir)
		if err != nil {
			// Missing/invalid theme.json — ignore this directory.
			continue
		}
		if manifest.Slug == "" {
			// Derive slug from name the same way ThemeLoader.upsertThemeRecord
			// does, so both registration paths agree on the row identity.
			if manifest.Name != "" {
				manifest.Slug = strings.ToLower(strings.ReplaceAll(manifest.Name, " ", "-"))
			} else {
				manifest.Slug = entry.Name()
			}
		}

		// Look up by path first (stable across slug derivation changes) then
		// fall back to slug.
		var existing models.Theme
		err = s.db.Where("path = ?", themeDir).First(&existing).Error
		if err == gorm.ErrRecordNotFound {
			err = s.db.Where("slug = ?", manifest.Slug).First(&existing).Error
		}
		if err == nil {
			// Refresh metadata + path; leave IsActive and Source alone.
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

	log.Printf("[themes] scanned %d themes from %s", registered, s.themesDir)
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

// GetActive returns the currently active theme.
func (s *ThemeMgmtService) GetActive() (*models.Theme, error) {
	var theme models.Theme
	if err := s.db.Where("is_active = ?", true).First(&theme).Error; err != nil {
		return nil, err
	}
	return &theme, nil
}

// InstallFromZip extracts a ZIP archive and installs the theme.
func (s *ThemeMgmtService) InstallFromZip(file io.Reader, filename string) (*models.Theme, error) {
	// Read entire ZIP into memory so we can use zip.NewReader.
	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read zip data: %w", err)
	}

	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("failed to open zip archive: %w", err)
	}

	// Create a temp directory for extraction.
	tmpDir, err := os.MkdirTemp("", "vibecms-theme-*")
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

	// Copy extracted theme to final destination.
	destDir := filepath.Join(s.themesDir, manifest.Slug)
	if err := os.RemoveAll(destDir); err != nil {
		return nil, fmt.Errorf("failed to clean destination dir: %w", err)
	}
	if err := copyDir(manifestDir, destDir); err != nil {
		return nil, fmt.Errorf("failed to copy theme to %s: %w", destDir, err)
	}

	// Create DB record.
	theme := models.Theme{
		Slug:        manifest.Slug,
		Name:        manifest.Name,
		Description: manifest.Description,
		Version:     manifest.Version,
		Author:      manifest.Author,
		Source:      "upload",
		Path:        destDir,
	}
	if err := s.db.Create(&theme).Error; err != nil {
		// Clean up on DB failure.
		os.RemoveAll(destDir)
		return nil, fmt.Errorf("failed to create theme record: %w", err)
	}

	return &theme, nil
}

// InstallFromGit clones a git repository and installs the theme.
func (s *ThemeMgmtService) InstallFromGit(gitURL, branch, token string) (*models.Theme, error) {
	if branch == "" {
		branch = "main"
	}

	// Build the clone URL, injecting token for HTTPS if provided.
	cloneURL := gitURL
	if token != "" && strings.HasPrefix(gitURL, "https://") {
		// Inject oauth2 token: https://oauth2:{token}@host/path
		cloneURL = strings.Replace(gitURL, "https://", fmt.Sprintf("https://oauth2:%s@", token), 1)
	}

	// Clone to a temp directory first to parse theme.json for the slug.
	tmpDir, err := os.MkdirTemp("", "vibecms-git-theme-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	cmd := exec.Command("git", "clone", "--branch", branch, "--single-branch", "--depth", "1", cloneURL, tmpDir)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("git clone failed: %s: %w", strings.TrimSpace(string(output)), err)
	}

	// Parse theme.json from cloned directory.
	manifest, _, err := findAndParseManifest(tmpDir)
	if err != nil {
		return nil, err
	}

	if manifest.Slug == "" {
		return nil, fmt.Errorf("theme.json missing required 'slug' field")
	}

	// Move cloned directory to final destination.
	destDir := filepath.Join(s.themesDir, manifest.Slug)
	if err := os.RemoveAll(destDir); err != nil {
		return nil, fmt.Errorf("failed to clean destination dir: %w", err)
	}
	if err := os.Rename(tmpDir, destDir); err != nil {
		// Rename may fail across filesystems; fall back to copy.
		if err := copyDir(tmpDir, destDir); err != nil {
			return nil, fmt.Errorf("failed to move theme to %s: %w", destDir, err)
		}
	}

	// Create DB record.
	var gitToken *string
	if token != "" {
		gitToken = &token
	}
	theme := models.Theme{
		Slug:        manifest.Slug,
		Name:        manifest.Name,
		Description: manifest.Description,
		Version:     manifest.Version,
		Author:      manifest.Author,
		Source:      "git",
		GitURL:      &gitURL,
		GitBranch:   branch,
		GitToken:    gitToken,
		Path:        destDir,
	}
	if err := s.db.Create(&theme).Error; err != nil {
		os.RemoveAll(destDir)
		return nil, fmt.Errorf("failed to create theme record: %w", err)
	}

	return &theme, nil
}

// PullUpdate pulls the latest changes for a git-sourced theme.
func (s *ThemeMgmtService) PullUpdate(id int) (*models.Theme, error) {
	theme, err := s.GetByID(id)
	if err != nil {
		return nil, err
	}

	if theme.Source != "git" {
		return nil, fmt.Errorf("theme %q is not git-sourced (source=%s)", theme.Slug, theme.Source)
	}

	// Run git pull.
	cmd := exec.Command("git", "-C", theme.Path, "pull", "origin", theme.GitBranch)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("git pull failed: %s: %w", strings.TrimSpace(string(output)), err)
	}

	// Re-parse theme.json to pick up version changes.
	manifest, _, err := findAndParseManifest(theme.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to re-parse theme.json after pull: %w", err)
	}

	// Update version (and other metadata) in DB.
	updates := map[string]interface{}{
		"version":     manifest.Version,
		"name":        manifest.Name,
		"description": manifest.Description,
		"author":      manifest.Author,
	}
	if err := s.db.Model(theme).Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("failed to update theme record: %w", err)
	}

	// If theme is active, reload it.
	if theme.IsActive {
		if err := s.Reload(theme.Path); err != nil {
			log.Printf("WARN: failed to reload active theme after pull: %v", err)
		}
	}

	// Re-fetch from DB.
	return s.GetByID(id)
}

// Activate sets the given theme as the active theme (deactivating all others).
// The previously active theme is fully deregistered (emitting theme.deactivated
// so extensions like media-manager can clean up) before the new theme is loaded.
func (s *ThemeMgmtService) Activate(id int) error {
	theme, err := s.GetByID(id)
	if err != nil {
		return err
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

	// Remove filesystem directory.
	if theme.Path != "" {
		if err := os.RemoveAll(theme.Path); err != nil {
			return fmt.Errorf("failed to remove theme directory %s: %w", theme.Path, err)
		}
	}

	// Delete DB record.
	if err := s.db.Delete(&models.Theme{}, id).Error; err != nil {
		return fmt.Errorf("failed to delete theme record: %w", err)
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
func extractZipFile(f *zip.File, destPath string) error {
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	out, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, f.Mode())
	if err != nil {
		return err
	}
	defer out.Close()

	// Limit copy size to prevent zip bombs (256 MB per file).
	_, err = io.Copy(out, io.LimitReader(rc, 256<<20))
	return err
}

// findAndParseManifest looks for theme.json in dir or one level deep.
func findAndParseManifest(dir string) (*themeMgmtManifest, string, error) {
	// Check root.
	rootManifest := filepath.Join(dir, "theme.json")
	if data, err := os.ReadFile(rootManifest); err == nil {
		var m themeMgmtManifest
		if err := json.Unmarshal(data, &m); err != nil {
			return nil, "", fmt.Errorf("failed to parse theme.json: %w", err)
		}
		return &m, dir, nil
	}

	// Check one level deep.
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read temp dir: %w", err)
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		subManifest := filepath.Join(dir, entry.Name(), "theme.json")
		if data, err := os.ReadFile(subManifest); err == nil {
			var m themeMgmtManifest
			if err := json.Unmarshal(data, &m); err != nil {
				return nil, "", fmt.Errorf("failed to parse theme.json: %w", err)
			}
			return &m, filepath.Join(dir, entry.Name()), nil
		}
	}

	return nil, "", fmt.Errorf("theme.json not found in archive (checked root and one level deep)")
}

// copyDir recursively copies a directory tree from src to dst.
func copyDir(src, dst string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dst, srcInfo.Mode()); err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			if err := copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}

	return nil
}

// copyFile copies a single file from src to dst.
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	info, err := in.Stat()
	if err != nil {
		return err
	}

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode())
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}
