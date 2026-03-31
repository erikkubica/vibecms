package media

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// CacheStats holds aggregate cache statistics.
type CacheStats struct {
	TotalSize  int64 `json:"total_size"`
	TotalFiles int   `json:"total_files"`
}

// SizeCacheStats holds cache statistics for a single size.
type SizeCacheStats struct {
	Size      int64 `json:"size"`
	FileCount int   `json:"file_count"`
}

// CacheManager manages on-disk image cache.
type CacheManager struct {
	baseDir string // e.g. "storage/cache/images"
}

// NewCacheManager creates a new CacheManager.
func NewCacheManager(storageDir string) *CacheManager {
	return &CacheManager{
		baseDir: filepath.Join(storageDir, "cache", "images"),
	}
}

// GetPath returns the cache file path for a given size and original path.
func (c *CacheManager) GetPath(sizeName, originalPath string) string {
	return filepath.Join(c.baseDir, sizeName, originalPath)
}

// GetWebPPath returns the WebP cache file path for a given size and original path.
func (c *CacheManager) GetWebPPath(sizeName, originalPath string) string {
	ext := filepath.Ext(originalPath)
	base := strings.TrimSuffix(originalPath, ext)
	return filepath.Join(c.baseDir, sizeName, base+".webp")
}

// Exists checks whether a cached file exists at the given path.
func (c *CacheManager) Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// Write writes data to the cache path, creating directories as needed.
func (c *CacheManager) Write(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create cache dir: %w", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write cache file: %w", err)
	}
	return nil
}

// Read reads data from a cached file.
func (c *CacheManager) Read(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read cache file: %w", err)
	}
	return data, nil
}

// DeleteForOriginal deletes all cached variants of an original file across given sizes.
func (c *CacheManager) DeleteForOriginal(originalPath string, sizes []string) error {
	for _, sizeName := range sizes {
		// Remove the original-format cached file.
		p := c.GetPath(sizeName, originalPath)
		_ = os.Remove(p)

		// Remove the WebP variant.
		wp := c.GetWebPPath(sizeName, originalPath)
		_ = os.Remove(wp)
	}
	return nil
}

// ClearSize deletes all cached files for a specific size.
func (c *CacheManager) ClearSize(sizeName string) error {
	dir := filepath.Join(c.baseDir, sizeName)
	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("clear size cache %s: %w", sizeName, err)
	}
	return nil
}

// ClearAll wipes the entire image cache directory.
func (c *CacheManager) ClearAll() error {
	if err := os.RemoveAll(c.baseDir); err != nil {
		return fmt.Errorf("clear all cache: %w", err)
	}
	return nil
}

// Stats returns aggregate cache statistics.
func (c *CacheManager) Stats() (CacheStats, error) {
	var stats CacheStats
	err := filepath.Walk(c.baseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			// Base dir may not exist yet.
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		if !info.IsDir() {
			stats.TotalSize += info.Size()
			stats.TotalFiles++
		}
		return nil
	})
	if os.IsNotExist(err) {
		return stats, nil
	}
	return stats, err
}

// SizeStats returns cache statistics for a single size.
func (c *CacheManager) SizeStats(sizeName string) (SizeCacheStats, error) {
	var stats SizeCacheStats
	dir := filepath.Join(c.baseDir, sizeName)
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		if !info.IsDir() {
			stats.Size += info.Size()
			stats.FileCount++
		}
		return nil
	})
	if os.IsNotExist(err) {
		return stats, nil
	}
	return stats, err
}
