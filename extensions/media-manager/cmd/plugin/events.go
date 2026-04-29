package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"squilla/internal/coreapi"
)

// This file owns the event-handling path: dispatching theme.activated /
// extension.activated / deactivated to the per-owner asset import logic.

const (
	ownerTheme ownerKind = iota
	ownerExtension
)

// ownerSpec bundles the column + payload lookup details for one owner kind.
type ownerSpec struct {
	source      string // media_files.source value
	ownerColumn string // media_files column storing the owner identifier
	payloadKey  string // payload field that carries the owner identifier
	storageRoot string // storage path prefix
	label       string // human-readable label for log messages
}

func (k ownerKind) spec() ownerSpec {
	if k == ownerExtension {
		return ownerSpec{
			source:      "extension",
			ownerColumn: "extension_slug",
			payloadKey:  "slug",
			storageRoot: "media/extension",
			label:       "extension",
		}
	}
	return ownerSpec{
		source:      "theme",
		ownerColumn: "theme_name",
		payloadKey:  "name",
		storageRoot: "media/theme",
		label:       "theme",
	}
}

// handleOwnedAssetsActivated imports the assets[] array carried by
// theme.activated or extension.activated into media_files, tagged with the
// matching source + owner identifier + asset_key + content_hash. Idempotent
// — unchanged hashes skip, changed hashes overwrite in place, removed keys
// are purged (row + file).
func (p *MediaManagerPlugin) handleOwnedAssetsActivated(payload []byte, kind ownerKind) error {
	spec := kind.spec()

	var evt struct {
		Name       string                   `json:"name"`
		Slug       string                   `json:"slug"`
		Path       string                   `json:"path"`
		Assets     []map[string]interface{} `json:"assets"`
		ImageSizes []map[string]interface{} `json:"image_sizes"`
	}
	if err := json.Unmarshal(payload, &evt); err != nil {
		return fmt.Errorf("%s.activated: decode payload: %w", spec.label, err)
	}
	ownerID := evt.Name
	if spec.payloadKey == "slug" {
		ownerID = evt.Slug
	}
	if ownerID == "" {
		return fmt.Errorf("%s.activated: missing %s", spec.label, spec.payloadKey)
	}

	ctx := context.Background()

	existing, err := p.host.DataQuery(ctx, tableName, coreapi.DataStoreQuery{
		Where: map[string]any{"source": spec.source, spec.ownerColumn: ownerID},
		Limit: 1000,
	})
	if err != nil {
		// Ownership columns may not exist yet (fresh install before migration);
		// skip gracefully with a warning.
		_ = p.host.Log(ctx, "warn", fmt.Sprintf("%s.activated %q: query existing failed: %v — skipping import", spec.label, ownerID, err), nil)
		return nil
	}
	byKey := map[string]map[string]any{}
	if existing != nil {
		for _, row := range existing.Rows {
			if k, ok := row["asset_key"].(string); ok && k != "" {
				byKey[k] = row
			}
		}
	}

	seen := map[string]bool{}
	for _, a := range evt.Assets {
		key, _ := a["key"].(string)
		if key == "" {
			continue
		}
		seen[key] = true
		absPath, _ := a["abs_path"].(string)
		if absPath == "" {
			continue
		}
		alt, _ := a["alt"].(string)
		width, _ := toInt(a["width"])
		height, _ := toInt(a["height"])

		data, err := os.ReadFile(absPath)
		if err != nil {
			_ = p.host.Log(ctx, "warn", fmt.Sprintf("%s.activated %q: read %s: %v", spec.label, ownerID, absPath, err), nil)
			continue
		}
		sum := sha256.Sum256(data)
		hash := hex.EncodeToString(sum[:])

		detectedType := http.DetectContentType(data)
		ext := safeExtension(detectedType, absPath)

		if (width == 0 || height == 0) && strings.HasPrefix(detectedType, "image/") && detectedType != "image/svg+xml" {
			if cfg, _, dErr := image.DecodeConfig(bytes.NewReader(data)); dErr == nil {
				if width == 0 {
					width = cfg.Width
				}
				if height == 0 {
					height = cfg.Height
				}
			}
		}

		storagePath := fmt.Sprintf("%s/%s/%s%s", spec.storageRoot, slugify(ownerID), key, ext)

		if prev, ok := byKey[key]; ok {
			if prevHash, _ := prev["content_hash"].(string); prevHash == hash {
				continue
			}
			publicURL, err := p.host.StoreFile(ctx, storagePath, data)
			if err != nil {
				_ = p.host.Log(ctx, "warn", fmt.Sprintf("%s.activated %q: store %s: %v", spec.label, ownerID, storagePath, err), nil)
				continue
			}
			idU, _ := toUint(prev["id"])
			updates := map[string]any{
				"path":         storagePath,
				"url":          publicURL,
				"mime_type":    detectedType,
				"size":         len(data),
				"width":        width,
				"height":       height,
				"alt":          alt,
				"content_hash": hash,
				"filename":     filepath.Base(storagePath),
			}
			if err := p.host.DataUpdate(ctx, tableName, idU, updates); err != nil {
				_ = p.host.Log(ctx, "warn", fmt.Sprintf("%s.activated %q: update row: %v", spec.label, ownerID, err), nil)
			}
			continue
		}

		publicURL, err := p.host.StoreFile(ctx, storagePath, data)
		if err != nil {
			_ = p.host.Log(ctx, "warn", fmt.Sprintf("%s.activated %q: store %s: %v", spec.label, ownerID, storagePath, err), nil)
			continue
		}
		record := map[string]any{
			"filename":       filepath.Base(storagePath),
			"original_name":  filepath.Base(absPath),
			"mime_type":      detectedType,
			"size":           len(data),
			"path":           storagePath,
			"url":            publicURL,
			"alt":            alt,
			"width":          width,
			"height":         height,
			"source":         spec.source,
			spec.ownerColumn: ownerID,
			"content_hash":   hash,
			"asset_key":      key,
		}
		if _, err := p.host.DataCreate(ctx, tableName, record); err != nil {
			// Duplicate key or other constraint violation — try to overwrite
			// the existing row instead of losing the imported asset.
			_ = p.host.Log(ctx, "info", fmt.Sprintf("%s.activated %q: create row for key %s failed (%v), falling back to update", spec.label, ownerID, key, err), nil)
			if fallbackRows, qErr := p.host.DataQuery(ctx, tableName, coreapi.DataStoreQuery{
				Where: map[string]any{"source": spec.source, spec.ownerColumn: ownerID, "asset_key": key},
				Limit: 1,
			}); qErr == nil && len(fallbackRows.Rows) == 1 {
				fallbackID, _ := toUint(fallbackRows.Rows[0]["id"])
				updates := map[string]any{
					"path":         storagePath,
					"url":          publicURL,
					"mime_type":    detectedType,
					"size":         len(data),
					"width":        width,
					"height":       height,
					"alt":          alt,
					"content_hash": hash,
					"filename":     filepath.Base(storagePath),
				}
				if uErr := p.host.DataUpdate(ctx, tableName, fallbackID, updates); uErr != nil {
					_ = p.host.Log(ctx, "warn", fmt.Sprintf("%s.activated %q: fallback update key %s: %v", spec.label, ownerID, key, uErr), nil)
					_ = p.host.DeleteFile(ctx, storagePath)
				}
			} else {
				_ = p.host.Log(ctx, "warn", fmt.Sprintf("%s.activated %q: fallback query for key %s failed: %v", spec.label, ownerID, key, qErr), nil)
				_ = p.host.DeleteFile(ctx, storagePath)
			}
		}
	}

	// Reconcile: delete rows whose key is no longer in the manifest.
	for key, row := range byKey {
		if seen[key] {
			continue
		}
		if path, _ := row["path"].(string); path != "" {
			_ = p.host.DeleteFile(ctx, path)
		}
		idU, _ := toUint(row["id"])
		if err := p.host.DataDelete(ctx, tableName, idU); err != nil {
			_ = p.host.Log(ctx, "warn", fmt.Sprintf("%s.activated %q: delete stale %s: %v", spec.label, ownerID, key, err), nil)
		}
	}

	_ = p.host.Log(ctx, "info", fmt.Sprintf("%s assets synced: %s (%d declared)", spec.label, ownerID, len(evt.Assets)), nil)

	// Image sizes — only themes ship these in the manifest. Idempotent upsert
	// by name. Source is "<kind>:<owner>" so admin UI can show provenance.
	if len(evt.ImageSizes) > 0 {
		p.upsertManifestImageSizes(ctx, evt.ImageSizes, fmt.Sprintf("%s:%s", spec.label, ownerID))
	}

	return nil
}

// upsertManifestImageSizes inserts or updates rows in media_image_sizes from
// the theme/extension manifest's image_sizes[] entries. Idempotent: rows are
// keyed by name and re-applied per activation. Existing rows owned by a
// different source are NOT clobbered (admin-created sizes win).
func (p *MediaManagerPlugin) upsertManifestImageSizes(ctx context.Context, defs []map[string]interface{}, source string) {
	emitChange := false
	for _, d := range defs {
		name, _ := d["name"].(string)
		if name == "" {
			continue
		}
		width, _ := toInt(d["width"])
		height, _ := toInt(d["height"])
		if width <= 0 || height <= 0 {
			_ = p.host.Log(ctx, "warn", fmt.Sprintf("image_sizes %q: invalid dimensions (%dx%d), skipped", name, width, height), nil)
			continue
		}
		mode, _ := d["mode"].(string)
		if mode == "" {
			mode = "fit"
		}
		quality, _ := toInt(d["quality"])

		row := map[string]any{
			"name":    name,
			"width":   width,
			"height":  height,
			"mode":    mode,
			"source":  source,
			"quality": quality,
		}

		existing, err := p.host.DataQuery(ctx, sizesTable, coreapi.DataStoreQuery{
			Where: map[string]any{"name": name}, Limit: 1,
		})
		if err != nil {
			_ = p.host.Log(ctx, "warn", fmt.Sprintf("image_sizes %q: query: %v", name, err), nil)
			continue
		}
		if existing != nil && len(existing.Rows) == 1 {
			prevSource, _ := existing.Rows[0]["source"].(string)
			// Don't clobber admin-created or differently-owned sizes.
			if prevSource != "" && prevSource != source && !strings.HasPrefix(prevSource, "default") {
				continue
			}
			idU, _ := toUint(existing.Rows[0]["id"])
			if err := p.host.DataUpdate(ctx, sizesTable, idU, row); err != nil {
				_ = p.host.Log(ctx, "warn", fmt.Sprintf("image_sizes %q: update: %v", name, err), nil)
				continue
			}
			emitChange = true
			continue
		}
		if _, err := p.host.DataCreate(ctx, sizesTable, row); err != nil {
			_ = p.host.Log(ctx, "warn", fmt.Sprintf("image_sizes %q: create: %v", name, err), nil)
			continue
		}
		emitChange = true
	}
	if emitChange {
		_ = p.host.Emit(ctx, "media:sizes_changed", map[string]any{"action": "manifest", "source": source})
	}
}

// handleOwnedAssetsDeactivated deletes all rows (and files) tagged with the
// given owner. Used for both theme.deactivated and extension.deactivated.
func (p *MediaManagerPlugin) handleOwnedAssetsDeactivated(payload []byte, kind ownerKind) error {
	spec := kind.spec()

	var evt struct {
		Name string `json:"name"`
		Slug string `json:"slug"`
	}
	if err := json.Unmarshal(payload, &evt); err != nil {
		return fmt.Errorf("%s.deactivated: decode payload: %w", spec.label, err)
	}
	ownerID := evt.Name
	if spec.payloadKey == "slug" {
		ownerID = evt.Slug
	}
	if ownerID == "" {
		return nil
	}

	ctx := context.Background()
	result, err := p.host.DataQuery(ctx, tableName, coreapi.DataStoreQuery{
		Where: map[string]any{"source": spec.source, spec.ownerColumn: ownerID},
		Limit: 10000,
	})
	if err != nil {
		_ = p.host.Log(ctx, "warn", fmt.Sprintf("%s.deactivated %q: query: %v", spec.label, ownerID, err), nil)
		return nil
	}
	if result == nil {
		return nil
	}
	for _, row := range result.Rows {
		if path, _ := row["path"].(string); path != "" {
			_ = p.host.DeleteFile(ctx, path)
		}
		if origPath, _ := row["original_path"].(string); origPath != "" && origPath != row["path"] {
			_ = p.host.DeleteFile(ctx, origPath)
		}
		idU, _ := toUint(row["id"])
		if err := p.host.DataDelete(ctx, tableName, idU); err != nil {
			_ = p.host.Log(ctx, "warn", fmt.Sprintf("%s.deactivated %q: delete row: %v", spec.label, ownerID, err), nil)
		}
	}
	_ = p.host.Log(ctx, "info", fmt.Sprintf("%s media purged: %s (%d rows)", spec.label, ownerID, len(result.Rows)), nil)
	return nil
}

// toInt coerces a JSON numeric (typically float64 after unmarshaling into
// map[string]interface{}) into int.
