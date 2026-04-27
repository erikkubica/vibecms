package cms

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"gorm.io/gorm"

	"vibecms/internal/models"
)

// pluginPinSettingKey is the site_settings key holding the trusted
// SHA-256 for a plugin binary. Format:
//
//	plugin.<slug>.<binary>.sha256 = <hex digest>
//
// where <binary> is the basename of the binary path declared in the
// extension manifest. Storing pins in site_settings keeps them under
// the kernel's audit trail (capability gates, settings handler) and
// out of the on-disk manifest — an attacker who can rewrite the bin/
// directory typically can't also rewrite settings rows.
func pluginPinSettingKey(slug, binary string) string {
	// Use just the basename so pins survive an extension being moved to
	// a new path (e.g. when reinstalling from a different upload).
	binBase := binary
	if i := strings.LastIndex(binBase, "/"); i >= 0 {
		binBase = binBase[i+1:]
	}
	return fmt.Sprintf("plugin.%s.%s.sha256", slug, binBase)
}

// hashFile returns the lowercase hex SHA-256 of the file at path.
func hashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open: %w", err)
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("read: %w", err)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// ErrPluginHashMismatch indicates that a plugin binary's SHA-256 did
// not match the operator's pinned value. The kernel refuses to start
// such a plugin.
var ErrPluginHashMismatch = errors.New("plugin binary SHA-256 mismatch")

// verifyPluginBinary computes the SHA-256 of the binary at path and
// compares it against an operator-pinned hash in site_settings. The
// behaviour is defense-in-depth, not gatekeeping:
//
//   - When a pin exists: a mismatch returns ErrPluginHashMismatch and
//     the kernel refuses to spawn the plugin. This is the protection
//     against a swapped bin/ directory.
//   - When no pin exists: the actual digest is logged so the operator
//     can pin it via the admin Settings UI. The plugin still loads —
//     refusing all unpinned plugins would break first-boot installs.
//
// The DB lookup is "best effort": a transient DB failure logs a warning
// and falls back to no-pin behaviour rather than block plugin startup.
// (We bias toward availability here because plugin manager runs after
// migrations have already passed health checks.)
func verifyPluginBinary(db *gorm.DB, slug, binary, path string) error {
	if db == nil {
		// No DB context — skip pin verification (used in tests and
		// during early bootstrap before the DB is wired).
		return nil
	}
	digest, err := hashFile(path)
	if err != nil {
		return err
	}

	key := pluginPinSettingKey(slug, binary)
	var rows []models.SiteSetting
	if dbErr := db.Where("\"key\" = ?", key).Limit(1).Find(&rows).Error; dbErr != nil {
		log.Printf("[plugins] WARN: pin lookup failed for %s/%s: %v (loading unpinned)", slug, binary, dbErr)
		return nil
	}

	if len(rows) == 0 || rows[0].Value == nil || strings.TrimSpace(*rows[0].Value) == "" {
		// Unpinned — log digest so an operator can pin it.
		log.Printf("[plugins] %s/%s unpinned (sha256=%s) — set %s in settings to enforce",
			slug, binary, digest, key)
		return nil
	}

	pinned := strings.ToLower(strings.TrimSpace(*rows[0].Value))
	if subtle.ConstantTimeCompare([]byte(pinned), []byte(digest)) != 1 {
		return fmt.Errorf("%w: %s/%s expected=%s actual=%s",
			ErrPluginHashMismatch, slug, binary, pinned, digest)
	}
	return nil
}

// HashPluginBinary is exported so the `vibecms verify-plugin` CLI
// helper can print the digest of a binary the operator wants to pin.
func HashPluginBinary(path string) (string, error) {
	return hashFile(path)
}

// PluginPinSettingKey is exported so the same CLI helper can tell the
// operator which setting key to write to.
func PluginPinSettingKey(slug, binary string) string {
	return pluginPinSettingKey(slug, binary)
}
