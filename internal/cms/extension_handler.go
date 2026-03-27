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

	"vibecms/internal/api"
	"vibecms/internal/auth"
	"vibecms/internal/models"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// ExtensionHandler provides HTTP handlers for extension management.
type ExtensionHandler struct {
	db     *gorm.DB
	loader *ExtensionLoader
}

// NewExtensionHandler creates a new ExtensionHandler.
func NewExtensionHandler(db *gorm.DB, loader *ExtensionLoader) *ExtensionHandler {
	return &ExtensionHandler{db: db, loader: loader}
}

// RegisterRoutes registers all admin API extension routes on the provided router group.
func (h *ExtensionHandler) RegisterRoutes(router fiber.Router) {
	g := router.Group("/extensions", auth.CapabilityRequired("manage_settings"))
	g.Get("/", h.List)
	g.Get("/:slug/files", h.BrowseFiles)
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

// Activate handles POST /extensions/:slug/activate.
func (h *ExtensionHandler) Activate(c *fiber.Ctx) error {
	slug := c.Params("slug")
	if err := h.loader.Activate(slug); err != nil {
		if err.Error() == "extension not found: "+slug {
			return api.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Extension not found")
		}
		return api.Error(c, fiber.StatusInternalServerError, "ACTIVATE_FAILED", "Failed to activate extension")
	}
	return api.Success(c, fiber.Map{"message": "Extension activated"})
}

// Deactivate handles POST /extensions/:slug/deactivate.
func (h *ExtensionHandler) Deactivate(c *fiber.Ctx) error {
	slug := c.Params("slug")
	if err := h.loader.Deactivate(slug); err != nil {
		if err.Error() == "extension not found: "+slug {
			return api.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Extension not found")
		}
		return api.Error(c, fiber.StatusInternalServerError, "DEACTIVATE_FAILED", "Failed to deactivate extension")
	}
	return api.Success(c, fiber.Map{"message": "Extension deactivated"})
}

// Upload handles POST /extensions/upload — uploads a ZIP containing an extension.
func (h *ExtensionHandler) Upload(c *fiber.Ctx) error {
	file, err := c.FormFile("file")
	if err != nil {
		return api.Error(c, fiber.StatusBadRequest, "NO_FILE", "No file uploaded")
	}
	if !strings.HasSuffix(file.Filename, ".zip") {
		return api.Error(c, fiber.StatusBadRequest, "INVALID_FORMAT", "File must be a .zip archive")
	}

	// Read ZIP into memory
	f, err := file.Open()
	if err != nil {
		return api.Error(c, fiber.StatusInternalServerError, "READ_FAILED", "Failed to read uploaded file")
	}
	defer f.Close()

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, f); err != nil {
		return api.Error(c, fiber.StatusInternalServerError, "READ_FAILED", "Failed to read uploaded file")
	}

	// Open as ZIP
	reader, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		return api.Error(c, fiber.StatusBadRequest, "INVALID_ZIP", "Invalid ZIP archive")
	}

	// Find extension.json to determine slug
	var manifest struct {
		Name        string `json:"name"`
		Slug        string `json:"slug"`
		Version     string `json:"version"`
		Author      string `json:"author"`
		Description string `json:"description"`
		Priority    int    `json:"priority"`
	}
	foundManifest := false
	manifestPrefix := "" // directory prefix inside ZIP

	for _, zf := range reader.File {
		name := zf.Name
		// Handle extension.json at root or inside a single directory
		base := filepath.Base(name)
		if base == "extension.json" && !zf.FileInfo().IsDir() {
			rc, err := zf.Open()
			if err != nil {
				continue
			}
			data, err := io.ReadAll(rc)
			rc.Close()
			if err != nil {
				continue
			}
			if err := json.Unmarshal(data, &manifest); err != nil {
				return api.Error(c, fiber.StatusBadRequest, "INVALID_MANIFEST", "Invalid extension.json: "+err.Error())
			}
			foundManifest = true
			manifestPrefix = strings.TrimSuffix(name, "extension.json")
			break
		}
	}

	if !foundManifest || manifest.Slug == "" {
		return api.Error(c, fiber.StatusBadRequest, "NO_MANIFEST", "ZIP must contain extension.json with a slug field")
	}

	// Extract to extensions/{slug}/
	destDir := filepath.Join(h.loader.extensionsDir, manifest.Slug)
	if err := os.RemoveAll(destDir); err != nil {
		log.Printf("WARN: failed to clean extension dir %s: %v", destDir, err)
	}

	for _, zf := range reader.File {
		name := zf.Name
		// Strip the manifest prefix (in case ZIP has a wrapper directory)
		if manifestPrefix != "" {
			if !strings.HasPrefix(name, manifestPrefix) {
				continue
			}
			name = strings.TrimPrefix(name, manifestPrefix)
		}
		if name == "" {
			continue
		}

		destPath := filepath.Join(destDir, name)
		// Prevent zip slip
		if !strings.HasPrefix(filepath.Clean(destPath), filepath.Clean(destDir)+string(os.PathSeparator)) {
			continue
		}

		if zf.FileInfo().IsDir() {
			os.MkdirAll(destPath, 0755)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return api.Error(c, fiber.StatusInternalServerError, "EXTRACT_FAILED", fmt.Sprintf("Failed to create directory: %v", err))
		}

		rc, err := zf.Open()
		if err != nil {
			continue
		}
		outFile, err := os.Create(destPath)
		if err != nil {
			rc.Close()
			continue
		}
		io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()
	}

	// Register in DB
	h.loader.ScanAndRegister()

	return api.Success(c, fiber.Map{"message": "Extension uploaded", "slug": manifest.Slug})
}

// Delete handles DELETE /extensions/:slug — removes extension files and DB record.
func (h *ExtensionHandler) Delete(c *fiber.Ctx) error {
	slug := c.Params("slug")

	// Check if extension exists and is not active
	var ext models.Extension
	if err := h.db.Where("slug = ?", slug).First(&ext).Error; err != nil {
		return api.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Extension not found")
	}
	if ext.IsActive {
		return api.Error(c, fiber.StatusBadRequest, "STILL_ACTIVE", "Deactivate extension before deleting")
	}

	// Remove files
	extDir := filepath.Join(h.loader.extensionsDir, slug)
	if err := os.RemoveAll(extDir); err != nil {
		log.Printf("WARN: failed to remove extension dir %s: %v", extDir, err)
	}

	// Remove from DB
	h.db.Where("slug = ?", slug).Delete(&models.Extension{})

	return api.Success(c, fiber.Map{"message": "Extension deleted"})
}
