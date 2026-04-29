package cms

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"squilla/internal/models"
)

// This file owns InstallFromGit and PullUpdate. Both rely on the
// hardening helpers in theme_git_safety.go (URL allowlist, askpass
// token plumbing, hostile-config defenses) and the secrets-service
// envelope helpers in theme_mgmt_service.go.

//   - Token encrypted at rest if a *secrets.Service is configured.
func (s *ThemeMgmtService) InstallFromGit(gitURL, branch, token string) (*models.Theme, error) {
	if branch == "" {
		branch = "main"
	}

	if err := validateGitURL(gitURL); err != nil {
		return nil, err
	}

	// Clone to a temp directory first to parse theme.json for the slug.
	tmpDir, err := os.MkdirTemp("", "squilla-git-theme-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	cmd, cleanup, err := buildSafeGitClone(gitURL, branch, token, tmpDir)
	if err != nil {
		return nil, err
	}
	defer cleanup()
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

	// Move cloned directory into the writable data dir. Bundled themes in
	// the image stay untouched; an operator can override a bundled slug by
	// installing a same-slug theme here, and ScanAndRegister will prefer
	// the data copy.
	destDir := filepath.Join(s.dataDir, manifest.Slug)
	if err := os.RemoveAll(destDir); err != nil {
		return nil, fmt.Errorf("failed to clean destination dir: %w", err)
	}
	if err := os.Rename(tmpDir, destDir); err != nil {
		// Rename may fail across filesystems; fall back to copy.
		if err := copyDir(tmpDir, destDir); err != nil {
			return nil, fmt.Errorf("failed to move theme to %s: %w", destDir, err)
		}
	}

	// Encrypt git token at rest before persisting.
	var gitToken *string
	if token != "" {
		wrapped, werr := s.wrapGitToken(token)
		if werr != nil {
			os.RemoveAll(destDir)
			return nil, fmt.Errorf("failed to encrypt git token: %w", werr)
		}
		gitToken = &wrapped
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

	// Decrypt token before passing to git askpass helper.
	plainToken, terr := s.resolveGitToken(theme.GitToken)
	if terr != nil {
		return nil, fmt.Errorf("failed to decrypt git token: %w", terr)
	}

	// Run git pull with safe flags + askpass token plumbing.
	cmd, cleanup, err := buildSafeGitPull(theme.Path, theme.GitBranch, plainToken)
	if err != nil {
		return nil, err
	}
	defer cleanup()
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
