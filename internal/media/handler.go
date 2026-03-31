package media

import (
	"bytes"
	"fmt"
	"log"
	"mime"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/gofiber/fiber/v2"
)

// NewCacheHandler returns a Fiber handler for /media/cache/{size}/{path...}.
// settingsGet is a function that retrieves a setting value by key (e.g. from CoreAPI).
func NewCacheHandler(registry *SizeRegistry, cache *CacheManager, storageDir string, settingsGet func(key string) string) fiber.Handler {
	// Per-path mutexes to prevent thundering herd.
	var locks sync.Map

	return func(c *fiber.Ctx) error {
		// Parse size name and original path from URL.
		// URL pattern: /media/cache/:size/*
		sizeName := c.Params("size")
		originalPath := c.Params("*")

		if sizeName == "" || originalPath == "" {
			return c.SendStatus(fiber.StatusNotFound)
		}

		// Validate path — prevent directory traversal.
		if !safeStoragePath(originalPath) {
			return c.SendStatus(fiber.StatusNotFound)
		}

		// Look up size in registry.
		size, ok := registry.Get(sizeName)
		if !ok {
			return c.SendStatus(fiber.StatusNotFound)
		}

		// Determine cache path (always original format — WebP encoding not available).
		cachePath := cache.GetPath(sizeName, originalPath)

		// Check cache — serve immediately if exists.
		if cache.Exists(cachePath) {
			return serveCachedFile(c, cachePath, false)
		}

		// Acquire per-path mutex to prevent thundering herd.
		mu := getPathMutex(&locks, cachePath)
		mu.Lock()
		defer mu.Unlock()

		// Double-check after acquiring lock (another goroutine may have generated it).
		if cache.Exists(cachePath) {
			return serveCachedFile(c, cachePath, false)
		}

		// Read original file.
		originalFile := filepath.Join(storageDir, "media", originalPath)
		originalData, err := os.ReadFile(originalFile)
		if err != nil {
			if os.IsNotExist(err) {
				return c.SendStatus(fiber.StatusNotFound)
			}
			log.Printf("ERROR: read original %s: %v", originalFile, err)
			return c.SendStatus(fiber.StatusInternalServerError)
		}

		// Determine MIME type from extension.
		mimeType := mimeFromPath(originalPath)
		if !isImageMime(mimeType) {
			return c.SendStatus(fiber.StatusNotFound)
		}

		// Determine quality.
		quality := size.Quality
		if quality <= 0 {
			// Use global default from settings.
			if q := settingsGet("media:optimizer:jpeg_quality"); q != "" {
				fmt.Sscanf(q, "%d", &quality)
			}
			if quality <= 0 {
				quality = 80
			}
		}

		// Resize the image.
		resized, outMime, err := Resize(bytes.NewReader(originalData), mimeType, size.Width, size.Height, size.Mode, quality)
		if err != nil {
			log.Printf("ERROR: resize %s/%s: %v", sizeName, originalPath, err)
			return c.SendStatus(fiber.StatusInternalServerError)
		}

		outputData := resized

		// Write to cache.
		if err := cache.Write(cachePath, outputData); err != nil {
			log.Printf("WARN: cache write %s: %v", cachePath, err)
			// Still serve the image even if caching fails.
		}

		// Serve the response.
		c.Set("Content-Type", outMime)
		c.Set("Cache-Control", "public, max-age=31536000")
		return c.Send(outputData)
	}
}

// serveCachedFile serves a cached file with appropriate headers.
func serveCachedFile(c *fiber.Ctx, path string, isWebP bool) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return c.SendStatus(fiber.StatusInternalServerError)
	}

	contentType := mimeFromPath(path)
	if isWebP {
		contentType = "image/webp"
	}
	c.Set("Content-Type", contentType)
	c.Set("Cache-Control", "public, max-age=31536000")
	return c.Send(data)
}

// getPathMutex returns a mutex for the given cache path, creating one if needed.
func getPathMutex(locks *sync.Map, path string) *sync.Mutex {
	val, _ := locks.LoadOrStore(path, &sync.Mutex{})
	return val.(*sync.Mutex)
}

// safeStoragePath validates a path to prevent directory traversal.
func safeStoragePath(p string) bool {
	if strings.Contains(p, "..") {
		return false
	}
	if strings.HasPrefix(p, "/") {
		return false
	}
	// No null bytes.
	if strings.ContainsRune(p, 0) {
		return false
	}
	return true
}

// mimeFromPath returns the MIME type for a file based on its extension.
func mimeFromPath(path string) string {
	ext := filepath.Ext(path)
	if ext == "" {
		return "application/octet-stream"
	}
	mimeType := mime.TypeByExtension(ext)
	if mimeType == "" {
		// Fallback for common image extensions.
		switch strings.ToLower(ext) {
		case ".jpg", ".jpeg":
			return "image/jpeg"
		case ".png":
			return "image/png"
		case ".gif":
			return "image/gif"
		case ".webp":
			return "image/webp"
		default:
			return "application/octet-stream"
		}
	}
	return mimeType
}

// isImageMime returns true if the MIME type is a supported image format.
func isImageMime(mimeType string) bool {
	switch mimeType {
	case "image/jpeg", "image/png", "image/gif", "image/webp":
		return true
	default:
		return false
	}
}
