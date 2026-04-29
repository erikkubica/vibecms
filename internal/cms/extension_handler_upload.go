package cms

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"squilla/internal/api"
	"squilla/internal/models"

	"github.com/gofiber/fiber/v2"
)

// This file owns the extension upload/install path: zip parsing, manifest
// validation, atomic directory swap, plugin-binary chmod, and triggering a
// rescan. The Fiber Upload handler is a thin wrapper around InstallFromZip
// so the same install logic backs both HTTP uploads and the MCP
// core.extension.deploy tool.

// MaxExtensionUploadSize caps the size of an extension archive accepted by
// either the Fiber Upload or InstallFromZip path. 50 MB is large enough for
// every shipped extension (admin-ui builds + plugin binary) with headroom
// for vendor JS, but small enough that a malicious caller cannot exhaust
// memory.
const MaxExtensionUploadSize = 50 * 1024 * 1024

// InstallFromZip extracts a ZIP archive into extensions/<slug>/ and registers
// the extension with the database. Returns the registered Extension row.
//
// Safety:
//   - data must be the full archive bytes; the caller is responsible for any
//     transport-level size enforcement, but this function still rejects > 50 MB
//     as a belt-and-braces check.
//   - The slug declared in extension.json is validated against
//     isValidSettingsKey before any path is constructed, so a hostile manifest
//     cannot escape extensions/.
//   - Each entry's destination is checked for zip-slip before write.
//   - Extraction lands in extensions/<slug>.tmp first; the swap into the final
//     name is a single os.Rename so the watcher and loader never observe a
//     partial directory.
//   - Plugin binaries declared in manifest.plugins[].binary are chmod'd to 0755
//     so the host can exec them without a separate post-deploy step.
//
// On success the loader's ScanAndRegister has been invoked and a fresh row
// for the extension is returned. The extension's is_active flag is left
// untouched — callers wanting the new code live should follow up with
// ExtensionHandler.HotActivate.
func (h *ExtensionHandler) InstallFromZip(data []byte) (*models.Extension, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty archive")
	}
	if len(data) > MaxExtensionUploadSize {
		return nil, fmt.Errorf("archive exceeds maximum size of %d bytes", MaxExtensionUploadSize)
	}

	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("invalid zip archive: %w", err)
	}

	manifest, manifestPrefix, err := findExtensionManifestInZip(reader)
	if err != nil {
		return nil, err
	}
	if manifest.Slug == "" {
		return nil, fmt.Errorf("extension.json missing required 'slug' field")
	}
	if !isValidSettingsKey(manifest.Slug) {
		return nil, fmt.Errorf("extension slug %q contains invalid characters", manifest.Slug)
	}

	// Always deploy into the writable data dir; bundled extensions in the
	// image stay untouched. ScanAndRegister prefers data on slug collision
	// so a same-slug deploy overrides the bundled copy without renaming.
	finalDir := filepath.Join(h.loader.dataDir, manifest.Slug)
	stagingDir := finalDir + ".deploy.tmp"
	backupDir := finalDir + ".deploy.old"

	// Clean up any stale staging/backup left by a previous failed deploy.
	_ = os.RemoveAll(stagingDir)
	_ = os.RemoveAll(backupDir)

	if err := extractZipToDir(reader, stagingDir, manifestPrefix); err != nil {
		_ = os.RemoveAll(stagingDir)
		return nil, err
	}

	// Mark plugin binaries executable so the host can spawn them without an
	// out-of-band chmod. Best-effort: a missing binary just means the manifest
	// references it but the archive didn't ship it — the plugin manager will
	// surface that at activation time.
	for _, p := range manifest.Plugins {
		if p.Binary == "" {
			continue
		}
		bin := filepath.Join(stagingDir, filepath.FromSlash(p.Binary))
		if _, statErr := os.Stat(bin); statErr == nil {
			if chmodErr := os.Chmod(bin, 0o755); chmodErr != nil {
				log.Printf("[extensions] chmod %s: %v", bin, chmodErr)
			}
		}
	}

	// Atomic swap: move existing dir aside, rename staging into place, drop
	// backup. os.Rename is atomic on the same filesystem; finalDir, stagingDir,
	// and backupDir all live under extensions/, so the rename never crosses a
	// device boundary.
	hadExisting := false
	if _, statErr := os.Stat(finalDir); statErr == nil {
		hadExisting = true
		if err := os.Rename(finalDir, backupDir); err != nil {
			_ = os.RemoveAll(stagingDir)
			return nil, fmt.Errorf("backup existing extension dir: %w", err)
		}
	}
	if err := os.Rename(stagingDir, finalDir); err != nil {
		// Roll back the backup if we moved one aside.
		if hadExisting {
			_ = os.Rename(backupDir, finalDir)
		}
		_ = os.RemoveAll(stagingDir)
		return nil, fmt.Errorf("swap in deployed dir: %w", err)
	}
	if hadExisting {
		_ = os.RemoveAll(backupDir)
	}

	// Register / refresh the row. Idempotent — ScanAndRegister upserts on slug.
	h.loader.ScanAndRegister()

	ext, err := h.loader.GetBySlug(manifest.Slug)
	if err != nil {
		return nil, fmt.Errorf("extension registered but lookup failed: %w", err)
	}
	return ext, nil
}

// findExtensionManifestInZip locates extension.json at the archive root or one
// level deep. Returns the parsed manifest and the prefix to strip from each
// entry so the on-disk layout is always extensions/<slug>/extension.json
// regardless of whether the archive was created with or without a wrapper
// directory.
func findExtensionManifestInZip(r *zip.Reader) (*ExtensionManifest, string, error) {
	for _, zf := range r.File {
		if zf.FileInfo().IsDir() {
			continue
		}
		if filepath.Base(zf.Name) != "extension.json" {
			continue
		}
		// Only accept root or one-level-deep manifests.
		dir := strings.TrimSuffix(zf.Name, "extension.json")
		if dir != "" && strings.Count(strings.TrimSuffix(dir, "/"), "/") > 0 {
			continue
		}

		rc, err := zf.Open()
		if err != nil {
			return nil, "", fmt.Errorf("open extension.json: %w", err)
		}
		raw, err := io.ReadAll(io.LimitReader(rc, 1<<20))
		rc.Close()
		if err != nil {
			return nil, "", fmt.Errorf("read extension.json: %w", err)
		}
		var m ExtensionManifest
		if err := json.Unmarshal(raw, &m); err != nil {
			return nil, "", fmt.Errorf("parse extension.json: %w", err)
		}
		return &m, dir, nil
	}
	return nil, "", fmt.Errorf("extension.json not found at archive root or one level deep")
}

// extractZipToDir writes every entry from r into destDir, stripping the given
// archive-internal prefix and rejecting zip-slip attempts. Files are limited
// to 256 MB each (zip-bomb cap shared with the theme installer).
func extractZipToDir(r *zip.Reader, destDir, prefix string) error {
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return fmt.Errorf("create dest dir: %w", err)
	}
	for _, zf := range r.File {
		name := zf.Name
		if prefix != "" {
			if !strings.HasPrefix(name, prefix) {
				continue
			}
			name = strings.TrimPrefix(name, prefix)
		}
		if name == "" {
			continue
		}

		destPath := filepath.Join(destDir, filepath.FromSlash(name))
		// Zip-slip: ensure the cleaned destination stays inside destDir.
		cleanedDest := filepath.Clean(destPath)
		if !strings.HasPrefix(cleanedDest, filepath.Clean(destDir)+string(os.PathSeparator)) && cleanedDest != filepath.Clean(destDir) {
			return fmt.Errorf("zip slip detected: %s", zf.Name)
		}

		if zf.FileInfo().IsDir() {
			if err := os.MkdirAll(destPath, 0o755); err != nil {
				return fmt.Errorf("mkdir %s: %w", destPath, err)
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
			return fmt.Errorf("mkdir parent %s: %w", destPath, err)
		}
		if err := extractZipFile(zf, destPath); err != nil {
			return fmt.Errorf("extract %s: %w", zf.Name, err)
		}
	}
	return nil
}

// Upload is the Fiber multipart upload handler. It is a thin wrapper around
// InstallFromZip so HTTP and MCP share one install path.
func (h *ExtensionHandler) Upload(c *fiber.Ctx) error {
	file, err := c.FormFile("file")
	if err != nil {
		return api.Error(c, fiber.StatusBadRequest, "NO_FILE", "No file uploaded")
	}
	if !strings.HasSuffix(strings.ToLower(file.Filename), ".zip") {
		return api.Error(c, fiber.StatusBadRequest, "INVALID_FORMAT", "File must be a .zip archive")
	}

	f, err := file.Open()
	if err != nil {
		return api.Error(c, fiber.StatusInternalServerError, "READ_FAILED", "Failed to read uploaded file")
	}
	defer f.Close()

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, io.LimitReader(f, MaxExtensionUploadSize+1)); err != nil {
		return api.Error(c, fiber.StatusInternalServerError, "READ_FAILED", "Failed to read uploaded file")
	}
	if buf.Len() > MaxExtensionUploadSize {
		return api.Error(c, fiber.StatusBadRequest, "FILE_TOO_LARGE", fmt.Sprintf("Upload exceeds maximum size of %d bytes", MaxExtensionUploadSize))
	}

	ext, err := h.InstallFromZip(buf.Bytes())
	if err != nil {
		return api.Error(c, fiber.StatusBadRequest, "INSTALL_FAILED", err.Error())
	}
	return api.Success(c, fiber.Map{"message": "Extension uploaded", "slug": ext.Slug, "extension": ext})
}
