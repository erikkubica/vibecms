package cms

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"

	"gorm.io/gorm"
)

// RunExtensionMigrations runs SQL migrations from an extension's migrations/ directory.
// Called when an extension is activated. Each migration file is tracked in the
// extension_migrations table to ensure it is only applied once.
func RunExtensionMigrations(db *gorm.DB, extDir string, slug string) error {
	migrationsDir := filepath.Join(extDir, "migrations")

	// Check if migrations directory exists
	info, err := os.Stat(migrationsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No migrations directory — nothing to do
		}
		return fmt.Errorf("checking migrations dir: %w", err)
	}
	if !info.IsDir() {
		return nil
	}

	// Read all .sql files
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("reading migrations dir: %w", err)
	}

	var sqlFiles []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if filepath.Ext(entry.Name()) == ".sql" {
			sqlFiles = append(sqlFiles, entry.Name())
		}
	}

	// Sort alphabetically for deterministic ordering
	sort.Strings(sqlFiles)

	for _, filename := range sqlFiles {
		// Check if already applied
		var count int64
		if err := db.Table("extension_migrations").
			Where("extension_slug = ? AND filename = ?", slug, filename).
			Count(&count).Error; err != nil {
			return fmt.Errorf("checking migration status for %s: %w", filename, err)
		}
		if count > 0 {
			continue // Already applied
		}

		// Read and execute migration
		sqlBytes, err := os.ReadFile(filepath.Join(migrationsDir, filename))
		if err != nil {
			return fmt.Errorf("reading migration %s: %w", filename, err)
		}

		log.Printf("[ext-migrations] applying %s/%s", slug, filename)
		if err := db.Exec(string(sqlBytes)).Error; err != nil {
			return fmt.Errorf("executing migration %s: %w", filename, err)
		}

		// Record as applied
		if err := db.Exec(
			"INSERT INTO extension_migrations (extension_slug, filename) VALUES (?, ?)",
			slug, filename,
		).Error; err != nil {
			return fmt.Errorf("recording migration %s: %w", filename, err)
		}
	}

	return nil
}
