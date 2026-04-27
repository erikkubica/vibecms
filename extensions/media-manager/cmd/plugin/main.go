package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"log"
	"math"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	goplugin "github.com/hashicorp/go-plugin"
	"golang.org/x/image/draw"
	"google.golang.org/grpc"

	// Register WebP decoder so image.Decode can read WebP files.
	_ "golang.org/x/image/webp"

	// Pure Go WebP encoder via WebAssembly (no CGO required).
	gowebp "github.com/gen2brain/webp"

	"vibecms/internal/coreapi"
	vibeplugin "vibecms/pkg/plugin"
	coreapipb "vibecms/pkg/plugin/coreapipb"
	pb "vibecms/pkg/plugin/proto"
)

const tableName = "media_files"
const sizesTable = "media_image_sizes"

// bulkJobProgress tracks progress of a long-running bulk operation.
type bulkJobProgress struct {
	mu        sync.Mutex
	Running   bool   `json:"running"`
	Total     int    `json:"total"`
	Processed int    `json:"processed"`
	Failed    int    `json:"failed"`
	Savings   int64  `json:"total_saved"`
	Status    string `json:"status"` // "idle", "running", "done", "error"
}

func (p *bulkJobProgress) reset(total int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.Running = true
	p.Total = total
	p.Processed = 0
	p.Failed = 0
	p.Savings = 0
	p.Status = "running"
}

func (p *bulkJobProgress) advance(saved int64, failed bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if failed {
		p.Failed++
	} else {
		p.Processed++
		p.Savings += saved
	}
}

func (p *bulkJobProgress) finish() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.Running = false
	p.Status = "done"
}

func (p *bulkJobProgress) snapshot() map[string]any {
	p.mu.Lock()
	defer p.mu.Unlock()
	return map[string]any{
		"running":     p.Running,
		"total":       p.Total,
		"processed":   p.Processed,
		"failed":      p.Failed,
		"total_saved": p.Savings,
		"status":      p.Status,
	}
}

// MediaManagerPlugin implements the ExtensionPlugin interface.
type MediaManagerPlugin struct {
	host               *coreapi.GRPCHostClient
	storageDir         string   // base storage path (e.g. "storage")
	cacheLocks         sync.Map // per-path mutexes to prevent thundering herd
	reoptimizeProgress bulkJobProgress
	restoreProgress    bulkJobProgress
}

func (p *MediaManagerPlugin) GetSubscriptions() ([]*pb.Subscription, error) {
	return []*pb.Subscription{
		{EventName: "theme.activated", Priority: 50},
		{EventName: "theme.deactivated", Priority: 50},
		{EventName: "extension.activated", Priority: 50},
		{EventName: "extension.deactivated", Priority: 50},
	}, nil
}

func (p *MediaManagerPlugin) HandleEvent(action string, payload []byte) (*pb.EventResponse, error) {
	switch action {
	case "theme.activated":
		if err := p.handleOwnedAssetsActivated(payload, ownerTheme); err != nil {
			return &pb.EventResponse{Handled: true, Error: err.Error()}, nil
		}
		return &pb.EventResponse{Handled: true}, nil
	case "theme.deactivated":
		if err := p.handleOwnedAssetsDeactivated(payload, ownerTheme); err != nil {
			return &pb.EventResponse{Handled: true, Error: err.Error()}, nil
		}
		return &pb.EventResponse{Handled: true}, nil
	case "extension.activated":
		if err := p.handleOwnedAssetsActivated(payload, ownerExtension); err != nil {
			return &pb.EventResponse{Handled: true, Error: err.Error()}, nil
		}
		return &pb.EventResponse{Handled: true}, nil
	case "extension.deactivated":
		if err := p.handleOwnedAssetsDeactivated(payload, ownerExtension); err != nil {
			return &pb.EventResponse{Handled: true, Error: err.Error()}, nil
		}
		return &pb.EventResponse{Handled: true}, nil
	default:
		return &pb.EventResponse{Handled: false}, nil
	}
}

// ---------------------------------------------------------------------------
// Theme & extension asset import / purge
// (theme.activated, theme.deactivated, extension.activated, extension.deactivated)
// ---------------------------------------------------------------------------

// ownerKind distinguishes theme-owned vs extension-owned asset rows in
// media_files. The column names, storage prefix, and payload field that
// carries the owner identifier all vary per kind.
type ownerKind int

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
		Name   string                   `json:"name"`
		Slug   string                   `json:"slug"`
		Path   string                   `json:"path"`
		Assets []map[string]interface{} `json:"assets"`
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
	return nil
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
func toInt(v any) (int, bool) {
	switch x := v.(type) {
	case int:
		return x, true
	case int32:
		return int(x), true
	case int64:
		return int(x), true
	case float64:
		return int(x), true
	case float32:
		return int(x), true
	case string:
		if n, err := strconv.Atoi(x); err == nil {
			return n, true
		}
	}
	return 0, false
}

// toUint coerces a row column (int/int64/float64) to uint for ID parameters.
func toUint(v any) (uint, bool) {
	switch x := v.(type) {
	case uint:
		return x, true
	case int:
		return uint(x), true
	case int32:
		return uint(x), true
	case int64:
		return uint(x), true
	case float64:
		return uint(x), true
	case string:
		if n, err := strconv.ParseUint(x, 10, 64); err == nil {
			return uint(n), true
		}
	}
	return 0, false
}

// slugify is a minimal theme-name to path-safe slug (spaces, slashes → dash,
// ASCII-lowercase). Used for the storage path `media/theme/<slug>/<key>.ext`.
func slugify(name string) string {
	s := strings.ToLower(name)
	s = strings.ReplaceAll(s, " ", "-")
	s = strings.ReplaceAll(s, "/", "-")
	s = strings.ReplaceAll(s, "\\", "-")
	return s
}

func (p *MediaManagerPlugin) Shutdown() error {
	return nil
}

func (p *MediaManagerPlugin) Initialize(hostConn *grpc.ClientConn) error {
	p.host = coreapi.NewGRPCHostClient(coreapipb.NewVibeCMSHostClient(hostConn))
	p.storageDir = "storage"
	if dir := os.Getenv("STORAGE_DIR"); dir != "" {
		p.storageDir = dir
	}

	// Seed default image sizes if the table is empty.
	p.seedDefaultSizes()

	// Seed optimizer settings so keys exist on fresh installs — keeps the
	// UI's initial read consistent and avoids "record not found" log noise
	// from GetSetting probes across the rest of the plugin.
	p.seedOptimizerDefaults()

	return nil
}

// seedOptimizerDefaults writes default values for any optimizer setting
// keys that aren't yet present. Idempotent — existing keys are left
// untouched so user-set values are preserved across plugin restarts.
func (p *MediaManagerPlugin) seedOptimizerDefaults() {
	ctx := context.Background()
	for key, defaultVal := range optimizerSettingDefaults {
		existing, err := p.host.GetSetting(ctx, key)
		if err == nil && existing != "" {
			continue
		}
		if err := p.host.SetSetting(ctx, key, defaultVal); err != nil {
			continue
		}
	}
}

// seedDefaultSizes ensures the default image sizes exist in the database.
// This runs on plugin startup so sizes are available even on fresh installs.
func (p *MediaManagerPlugin) seedDefaultSizes() {
	ctx := context.Background()

	// Check if sizes already exist.
	result, err := p.host.DataQuery(ctx, sizesTable, coreapi.DataStoreQuery{Limit: 1})
	if err != nil {
		return
	}
	if result.Total > 0 {
		return // Already seeded.
	}

	defaults := []map[string]any{
		{"name": "thumbnail", "width": 150, "height": 150, "mode": "crop", "source": "default", "quality": 0},
		{"name": "medium", "width": 250, "height": 250, "mode": "fit", "source": "default", "quality": 0},
		{"name": "large", "width": 500, "height": 500, "mode": "fit", "source": "default", "quality": 0},
	}

	for _, size := range defaults {
		if _, err := p.host.DataCreate(ctx, sizesTable, size); err != nil {
			// Ignore duplicate errors (race condition with another instance).
			continue
		}
	}

	// Notify core to refresh its in-memory registry.
	_ = p.host.Emit(ctx, "media:sizes_changed", map[string]any{"action": "seeded"})
}

func (p *MediaManagerPlugin) HandleHTTPRequest(req *pb.PluginHTTPRequest) (*pb.PluginHTTPResponse, error) {
	path := strings.TrimSuffix(req.GetPath(), "/")
	method := strings.ToUpper(req.GetMethod())
	ctx := context.Background()

	// Public route: GET /media/cache/{size}/{path...}
	// This comes from the public extension proxy with the full URL path.
	if method == "GET" && strings.HasPrefix(path, "/media/cache/") {
		return p.handlePublicCacheRequest(ctx, req)
	}

	// Public route: GET /media/{path...} — auto WebP conversion
	// Intercepts all media file requests; serves WebP when browser accepts it.
	if method == "GET" && strings.HasPrefix(path, "/media/") && !strings.HasPrefix(path, "/media/cache/") {
		return p.handlePublicMediaRequest(ctx, req)
	}

	// Route: POST /upload
	if method == "POST" && (path == "/upload" || path == "upload") {
		return p.handleUpload(ctx, req)
	}

	// Route: GET / (list)
	if method == "GET" && (path == "" || path == "/") {
		return p.handleList(ctx, req)
	}

	// --- Optimizer routes ---

	// GET /optimizer/settings
	if method == "GET" && path == "/optimizer/settings" {
		return p.handleGetOptimizerSettings(ctx)
	}
	// PUT /optimizer/settings
	if method == "PUT" && path == "/optimizer/settings" {
		return p.handleUpdateOptimizerSettings(ctx, req.GetBody())
	}
	// GET /optimizer/sizes
	if method == "GET" && path == "/optimizer/sizes" {
		return p.handleListSizes(ctx)
	}
	// POST /optimizer/sizes
	if method == "POST" && path == "/optimizer/sizes" {
		return p.handleCreateSize(ctx, req.GetBody())
	}
	// DELETE /optimizer/sizes/:name
	if method == "DELETE" && strings.HasPrefix(path, "/optimizer/sizes/") {
		name := strings.TrimPrefix(path, "/optimizer/sizes/")
		if name == "" {
			return jsonError(400, "MISSING_NAME", "Size name is required"), nil
		}
		return p.handleDeleteSize(ctx, name)
	}
	// POST /optimizer/cache/clear
	if method == "POST" && path == "/optimizer/cache/clear" {
		return p.handleClearAllCache(ctx)
	}
	// POST /optimizer/cache/clear/:size
	if method == "POST" && strings.HasPrefix(path, "/optimizer/cache/clear/") {
		sizeName := strings.TrimPrefix(path, "/optimizer/cache/clear/")
		if sizeName == "" {
			return jsonError(400, "MISSING_SIZE", "Size name is required"), nil
		}
		return p.handleClearSizeCache(ctx, sizeName)
	}
	// GET /optimizer/stats — optimization statistics
	if method == "GET" && path == "/optimizer/stats" {
		return p.handleOptimizerStats(ctx)
	}
	// POST /optimizer/reoptimize-all — re-optimize all images with current settings (async)
	if method == "POST" && path == "/optimizer/reoptimize-all" {
		return p.handleReoptimizeAll(ctx, false)
	}
	// POST /optimizer/optimize-pending — optimize only images that haven't been optimized yet (async)
	if method == "POST" && path == "/optimizer/optimize-pending" {
		return p.handleReoptimizeAll(ctx, true)
	}
	// GET /optimizer/reoptimize-progress — poll progress of bulk re-optimize
	if method == "GET" && path == "/optimizer/reoptimize-progress" {
		return jsonResponse(200, map[string]any{"data": p.reoptimizeProgress.snapshot()}), nil
	}
	// POST /optimizer/restore-all — restore all images to originals (async)
	if method == "POST" && path == "/optimizer/restore-all" {
		return p.handleRestoreAll(ctx)
	}
	// GET /optimizer/restore-progress — poll progress of bulk restore
	if method == "GET" && path == "/optimizer/restore-progress" {
		return jsonResponse(200, map[string]any{"data": p.restoreProgress.snapshot()}), nil
	}

	// --- Media routes with ID ---

	// POST /:id/restore — restore original image
	if method == "POST" && strings.HasSuffix(path, "/restore") {
		idPart := strings.TrimSuffix(strings.TrimPrefix(path, "/"), "/restore")
		if rid, err := strconv.ParseUint(idPart, 10, 64); err == nil && rid > 0 {
			return p.handleRestoreOriginal(ctx, uint(rid))
		}
	}
	// POST /:id/reoptimize — re-optimize single image
	if method == "POST" && strings.HasSuffix(path, "/reoptimize") {
		idPart := strings.TrimSuffix(strings.TrimPrefix(path, "/"), "/reoptimize")
		if rid, err := strconv.ParseUint(idPart, 10, 64); err == nil && rid > 0 {
			return p.handleReoptimize(ctx, uint(rid))
		}
	}

	// --- Media CRUD routes with ID ---

	// Route with ID: GET /:id, PUT /:id, DELETE /:id
	id := extractID(path, req.GetPathParams())
	if id == 0 {
		return jsonError(404, "NOT_FOUND", "Route not found"), nil
	}

	switch method {
	case "GET":
		return p.handleGet(ctx, id)
	case "PUT":
		return p.handleUpdate(ctx, id, req.GetBody())
	case "DELETE":
		return p.handleDelete(ctx, id)
	default:
		return jsonError(405, "METHOD_NOT_ALLOWED", "Method not allowed"), nil
	}
}

// handleList handles GET / — returns paginated media files.
func (p *MediaManagerPlugin) handleList(ctx context.Context, req *pb.PluginHTTPRequest) (*pb.PluginHTTPResponse, error) {
	params := req.GetQueryParams()
	page, _ := strconv.Atoi(paramOr(params, "page", "1"))
	perPage, _ := strconv.Atoi(paramOr(params, "per_page", "20"))
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}

	mimeType := params["mime_type"]
	search := params["search"]
	sortBy := params["sort_by"]

	orderBy := "created_at DESC"
	switch sortBy {
	case "name_asc":
		orderBy = "original_name ASC"
	case "name_desc":
		orderBy = "original_name DESC"
	case "size_asc":
		orderBy = "size ASC"
	case "size_desc":
		orderBy = "size DESC"
	case "date_asc":
		orderBy = "created_at ASC"
	case "date_desc":
		orderBy = "created_at DESC"
	}

	query := coreapi.DataStoreQuery{
		OrderBy: orderBy,
		Limit:   perPage,
		Offset:  (page - 1) * perPage,
	}

	where := make(map[string]any)
	if mimeType != "" {
		if strings.Contains(mimeType, "/") {
			where["mime_type"] = mimeType
		} else {
			query.Raw = "mime_type LIKE ?"
			query.Args = []any{mimeType + "/%"}
		}
	}
	if search != "" {
		if query.Raw != "" {
			query.Raw += " AND original_name ILIKE ?"
			query.Args = append(query.Args, "%"+search+"%")
		} else {
			query.Raw = "original_name ILIKE ?"
			query.Args = []any{"%" + search + "%"}
		}
	}
	if len(where) > 0 {
		query.Where = where
	}

	result, err := p.host.DataQuery(ctx, tableName, query)
	if err != nil {
		return jsonError(500, "LIST_FAILED", "Failed to list media files"), nil
	}

	totalPages := int(math.Ceil(float64(result.Total) / float64(perPage)))
	resp := map[string]any{
		"data": result.Rows,
		"meta": map[string]any{
			"total":       result.Total,
			"page":        page,
			"per_page":    perPage,
			"total_pages": totalPages,
		},
	}

	return jsonResponse(200, resp), nil
}

// handleGet handles GET /:id — returns a single media file.
func (p *MediaManagerPlugin) handleGet(ctx context.Context, id uint) (*pb.PluginHTTPResponse, error) {
	row, err := p.host.DataGet(ctx, tableName, id)
	if err != nil {
		if isNotFound(err) {
			return jsonError(404, "NOT_FOUND", "Media file not found"), nil
		}
		return jsonError(500, "FETCH_FAILED", "Failed to fetch media file"), nil
	}

	return jsonResponse(200, map[string]any{"data": row}), nil
}

// handleUpload handles POST /upload — multipart file upload.
func (p *MediaManagerPlugin) handleUpload(ctx context.Context, req *pb.PluginHTTPRequest) (*pb.PluginHTTPResponse, error) {
	contentType := ""
	for k, v := range req.GetHeaders() {
		if strings.EqualFold(k, "content-type") {
			contentType = v
			break
		}
	}

	if contentType == "" {
		return jsonError(400, "NO_CONTENT_TYPE", "Missing Content-Type header"), nil
	}

	mediaType, params, err := mime.ParseMediaType(contentType)
	if err != nil || !strings.HasPrefix(mediaType, "multipart/") {
		return jsonError(400, "INVALID_CONTENT_TYPE", "Expected multipart form data"), nil
	}

	boundary := params["boundary"]
	if boundary == "" {
		return jsonError(400, "NO_BOUNDARY", "Missing multipart boundary"), nil
	}

	reader := multipart.NewReader(bytes.NewReader(req.GetBody()), boundary)

	var fileData []byte
	var originalName string
	var fileMimeType string

	// Maximum upload size: 50 MB.
	const maxUploadSize = 50 * 1024 * 1024

	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return jsonError(400, "PARSE_FAILED", "Failed to parse multipart data"), nil
		}

		if part.FormName() == "file" {
			originalName = part.FileName()
			fileMimeType = part.Header.Get("Content-Type")
			fileData, err = io.ReadAll(io.LimitReader(part, maxUploadSize+1))
			if err != nil {
				return jsonError(500, "READ_FAILED", "Failed to read uploaded file"), nil
			}
			if len(fileData) > maxUploadSize {
				return jsonError(400, "FILE_TOO_LARGE", "File exceeds maximum upload size of 50 MB"), nil
			}
			break
		}
		part.Close()
	}

	if fileData == nil || originalName == "" {
		return jsonError(400, "NO_FILE", "No file uploaded"), nil
	}

	// Validate MIME type: detect from content and enforce allowlist.
	detectedType := http.DetectContentType(fileData)
	if fileMimeType == "" {
		fileMimeType = detectedType
	}
	if !isAllowedMimeType(fileMimeType) {
		return jsonError(400, "INVALID_FILE_TYPE", fmt.Sprintf("File type %s is not allowed", fileMimeType)), nil
	}

	// Generate unique filename with safe extension derived from MIME type.
	now := time.Now()
	dateDir := fmt.Sprintf("%04d/%02d", now.Year(), now.Month())
	ext := safeExtension(fileMimeType, originalName)
	storedName := fmt.Sprintf("%d%s", now.UnixNano(), ext)
	storagePath := fmt.Sprintf("media/%s/%s", dateDir, storedName)

	// Track original dimensions before any processing.
	var origW, origH int
	if strings.HasPrefix(fileMimeType, "image/") && fileMimeType != "image/svg+xml" {
		if img, _, err := image.Decode(bytes.NewReader(fileData)); err == nil {
			origW = img.Bounds().Dx()
			origH = img.Bounds().Dy()
		}
	}

	// Prepare optimization tracking fields.
	originalSize := len(fileData)
	isOptimized := false
	originalBackupPath := ""
	optimizationSavings := 0

	// Normalize image before storage if enabled — but save original first.
	if strings.HasPrefix(fileMimeType, "image/") && fileMimeType != "image/svg+xml" {
		normalizedData, normalizedMime := p.normalizeImage(ctx, fileData, fileMimeType)
		// Only swap in the normalized bytes if they're smaller or the
		// encoder changed mime type (e.g. format conversion). Otherwise
		// keep the original — but still mark as optimized so the file
		// isn't re-processed forever.
		changedMime := normalizedMime != fileMimeType
		smaller := len(normalizedData) < len(fileData)
		if changedMime || smaller {
			originalBackupPath = fmt.Sprintf("media/originals/%s/%s", dateDir, storedName)
			if _, storeErr := p.host.StoreFile(ctx, originalBackupPath, fileData); storeErr != nil {
				log.Printf("[upload] failed to store original backup: %v", storeErr)
				originalBackupPath = ""
			}
			if smaller {
				optimizationSavings = len(fileData) - len(normalizedData)
			}
			fileData = normalizedData
			fileMimeType = normalizedMime
		}
		isOptimized = true
	}

	// Store the file via CoreAPI.
	publicURL, err := p.host.StoreFile(ctx, storagePath, fileData)
	if err != nil {
		return jsonError(500, "STORE_FAILED", "Failed to store file"), nil
	}

	// Create the database record.
	record := map[string]any{
		"filename":             storedName,
		"original_name":        originalName,
		"mime_type":            fileMimeType,
		"size":                 len(fileData),
		"path":                 storagePath,
		"url":                  publicURL,
		"alt":                  "",
		"is_optimized":         isOptimized,
		"original_size":        originalSize,
		"original_path":        originalBackupPath,
		"original_width":       origW,
		"original_height":      origH,
		"optimization_savings": optimizationSavings,
	}

	created, err := p.host.DataCreate(ctx, tableName, record)
	if err != nil {
		// Try to clean up the stored file and backup.
		_ = p.host.DeleteFile(ctx, storagePath)
		if originalBackupPath != "" {
			_ = p.host.DeleteFile(ctx, originalBackupPath)
		}
		return jsonError(500, "CREATE_FAILED", "Failed to create media record"), nil
	}

	return jsonResponse(201, map[string]any{"data": created}), nil
}

// handleUpdate handles PUT /:id — update alt text.
func (p *MediaManagerPlugin) handleUpdate(ctx context.Context, id uint, body []byte) (*pb.PluginHTTPResponse, error) {
	// Verify exists.
	_, err := p.host.DataGet(ctx, tableName, id)
	if err != nil {
		if isNotFound(err) {
			return jsonError(404, "NOT_FOUND", "Media file not found"), nil
		}
		return jsonError(500, "FETCH_FAILED", "Failed to fetch media file"), nil
	}

	var input struct {
		Alt          *string `json:"alt"`
		OriginalName *string `json:"original_name"`
	}
	if err := json.Unmarshal(body, &input); err != nil {
		return jsonError(400, "INVALID_BODY", "Invalid request body"), nil
	}

	updateData := map[string]any{
		"updated_at": time.Now().Format(time.RFC3339),
	}
	if input.Alt != nil {
		updateData["alt"] = *input.Alt
	}
	if input.OriginalName != nil && *input.OriginalName != "" {
		updateData["original_name"] = *input.OriginalName
	}

	if err := p.host.DataUpdate(ctx, tableName, id, updateData); err != nil {
		return jsonError(500, "UPDATE_FAILED", "Failed to update media file"), nil
	}

	// Fetch updated record.
	row, err := p.host.DataGet(ctx, tableName, id)
	if err != nil {
		return jsonError(500, "FETCH_FAILED", "Failed to fetch updated media file"), nil
	}

	return jsonResponse(200, map[string]any{"data": row}), nil
}

// handleDelete handles DELETE /:id — delete a media file.
func (p *MediaManagerPlugin) handleDelete(ctx context.Context, id uint) (*pb.PluginHTTPResponse, error) {
	row, err := p.host.DataGet(ctx, tableName, id)
	if err != nil {
		if isNotFound(err) {
			return jsonError(404, "NOT_FOUND", "Media file not found"), nil
		}
		return jsonError(500, "FETCH_FAILED", "Failed to fetch media file"), nil
	}

	// Delete the file from storage.
	if path, ok := row["path"].(string); ok && path != "" {
		_ = p.host.DeleteFile(ctx, path)

		// Clear cached image variants for this file.
		p.clearCacheForOriginal(ctx, path)
	}

	// Delete the original backup if it exists.
	if origPath, ok := row["original_path"].(string); ok && origPath != "" {
		_ = p.host.DeleteFile(ctx, origPath)
	}

	// Delete the database record.
	if err := p.host.DataDelete(ctx, tableName, id); err != nil {
		return jsonError(500, "DELETE_FAILED", "Failed to delete media file"), nil
	}

	return jsonResponse(200, map[string]any{"data": map[string]any{"message": "Media file deleted"}}), nil
}

// --- Optimizer Settings ---

// optimizerSettingKeys lists all optimizer setting keys with their default values.
var optimizerSettingDefaults = map[string]string{
	"media:optimizer:jpeg_quality":            "80",
	"media:optimizer:webp_enabled":            "true",
	"media:optimizer:webp_quality":            "75",
	"media:optimizer:normalize_enabled":       "true",
	"media:optimizer:normalize_max_dimension": "5000",
	"media:optimizer:upload_quality":          "100",
}

// handleGetOptimizerSettings returns all optimizer settings.
func (p *MediaManagerPlugin) handleGetOptimizerSettings(ctx context.Context) (*pb.PluginHTTPResponse, error) {
	settings := make(map[string]string)
	for key, defaultVal := range optimizerSettingDefaults {
		val, err := p.host.GetSetting(ctx, key)
		if err != nil || val == "" {
			val = defaultVal
		}
		// Strip the prefix for cleaner JSON keys.
		shortKey := strings.TrimPrefix(key, "media:optimizer:")
		settings[shortKey] = val
	}
	return jsonResponse(200, map[string]any{"data": settings}), nil
}

// handleUpdateOptimizerSettings updates optimizer settings.
func (p *MediaManagerPlugin) handleUpdateOptimizerSettings(ctx context.Context, body []byte) (*pb.PluginHTTPResponse, error) {
	var input map[string]any
	if err := json.Unmarshal(body, &input); err != nil {
		return jsonError(400, "INVALID_BODY", "Invalid request body"), nil
	}

	updated := make(map[string]string)
	for shortKey, rawValue := range input {
		fullKey := "media:optimizer:" + shortKey
		if _, ok := optimizerSettingDefaults[fullKey]; !ok {
			continue // Skip unknown settings.
		}
		value := fmt.Sprintf("%v", rawValue)
		if err := p.host.SetSetting(ctx, fullKey, value); err != nil {
			return jsonError(500, "SETTINGS_FAILED", fmt.Sprintf("Failed to save setting %s", shortKey)), nil
		}
		updated[shortKey] = value
	}

	return jsonResponse(200, map[string]any{"data": updated}), nil
}

// --- Image Size Management ---

// handleListSizes returns all registered image sizes with cache stats.
func (p *MediaManagerPlugin) handleListSizes(ctx context.Context) (*pb.PluginHTTPResponse, error) {
	result, err := p.host.DataQuery(ctx, sizesTable, coreapi.DataStoreQuery{
		OrderBy: "name ASC",
		Limit:   100,
	})
	if err != nil {
		return jsonError(500, "LIST_FAILED", "Failed to list image sizes"), nil
	}

	// Enrich each size with cache stats.
	for i, row := range result.Rows {
		name, _ := row["name"].(string)
		if name == "" {
			continue
		}
		sizeDir := filepath.Join(p.cacheBaseDir(), name)
		fileCount, totalSize := dirStats(sizeDir)
		result.Rows[i]["cached_files"] = fileCount
		result.Rows[i]["cache_size"] = totalSize
	}

	// Also count _webp cache.
	webpDir := filepath.Join(p.cacheBaseDir(), "_webp")
	webpFiles, webpSize := dirStats(webpDir)

	return jsonResponse(200, map[string]any{
		"data": result.Rows,
		"meta": map[string]any{
			"total":      result.Total,
			"webp_files": webpFiles,
			"webp_size":  webpSize,
		},
	}), nil
}

// dirStats returns the file count and total size of all files in a directory (recursive).
func dirStats(dir string) (int, int64) {
	var count int
	var size int64
	filepath.Walk(dir, func(_ string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		count++
		size += info.Size()
		return nil
	})
	return count, size
}

// handleCreateSize registers a new image size.
func (p *MediaManagerPlugin) handleCreateSize(ctx context.Context, body []byte) (*pb.PluginHTTPResponse, error) {
	var input struct {
		Name    string `json:"name"`
		Width   int    `json:"width"`
		Height  int    `json:"height"`
		Mode    string `json:"mode"`
		Source  string `json:"source"`
		Quality int    `json:"quality"`
	}
	if err := json.Unmarshal(body, &input); err != nil {
		return jsonError(400, "INVALID_BODY", "Invalid request body"), nil
	}

	if input.Name == "" || input.Width <= 0 || input.Height <= 0 {
		return jsonError(400, "VALIDATION_FAILED", "Name, width, and height are required"), nil
	}
	if input.Mode == "" {
		input.Mode = "fit"
	}
	if input.Mode != "crop" && input.Mode != "fit" && input.Mode != "width" {
		return jsonError(400, "INVALID_MODE", "Mode must be crop, fit, or width"), nil
	}
	if input.Source == "" {
		input.Source = "default"
	}

	record := map[string]any{
		"name":    input.Name,
		"width":   input.Width,
		"height":  input.Height,
		"mode":    input.Mode,
		"source":  input.Source,
		"quality": input.Quality,
	}

	created, err := p.host.DataCreate(ctx, sizesTable, record)
	if err != nil {
		if strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "duplicate") {
			return jsonError(409, "DUPLICATE_NAME", fmt.Sprintf("Size %q already exists", input.Name)), nil
		}
		return jsonError(500, "CREATE_FAILED", "Failed to create image size"), nil
	}

	// Notify that sizes changed so core can refresh its in-memory registry.
	_ = p.host.Emit(ctx, "media:sizes_changed", map[string]any{"action": "created", "name": input.Name})

	return jsonResponse(201, map[string]any{"data": created}), nil
}

// handleDeleteSize deletes an image size by name.
func (p *MediaManagerPlugin) handleDeleteSize(ctx context.Context, name string) (*pb.PluginHTTPResponse, error) {
	// Find the size by name.
	result, err := p.host.DataQuery(ctx, sizesTable, coreapi.DataStoreQuery{
		Where: map[string]any{"name": name},
		Limit: 1,
	})
	if err != nil {
		return jsonError(500, "QUERY_FAILED", "Failed to query image sizes"), nil
	}
	if result.Total == 0 || len(result.Rows) == 0 {
		return jsonError(404, "NOT_FOUND", fmt.Sprintf("Size %q not found", name)), nil
	}

	// Extract the ID from the first row.
	row := result.Rows[0]
	var sizeID uint
	switch v := row["id"].(type) {
	case float64:
		sizeID = uint(v)
	case json.Number:
		n, _ := v.Int64()
		sizeID = uint(n)
	}
	if sizeID == 0 {
		return jsonError(500, "PARSE_FAILED", "Failed to parse size ID"), nil
	}

	if err := p.host.DataDelete(ctx, sizesTable, sizeID); err != nil {
		return jsonError(500, "DELETE_FAILED", "Failed to delete image size"), nil
	}

	// Clear cache for this size.
	sizeDir := filepath.Join(p.cacheBaseDir(), name)
	_ = os.RemoveAll(sizeDir)

	return jsonResponse(200, map[string]any{"data": map[string]any{"message": fmt.Sprintf("Size %q deleted", name)}}), nil
}

// --- Cache Management ---

// cacheBaseDir returns the base directory for the image cache.
func (p *MediaManagerPlugin) cacheBaseDir() string {
	return filepath.Join(p.storageDir, "cache", "images")
}

// cachePath returns the cache file path for a given size and original path.
func (p *MediaManagerPlugin) cachePath(sizeName, originalPath string) string {
	return filepath.Join(p.cacheBaseDir(), sizeName, originalPath)
}

// cacheWebPPath returns the WebP cache file path for a given size and original path.
func (p *MediaManagerPlugin) cacheWebPPath(sizeName, originalPath string) string {
	ext := filepath.Ext(originalPath)
	base := strings.TrimSuffix(originalPath, ext)
	return filepath.Join(p.cacheBaseDir(), sizeName, base+".webp")
}

// cacheExists checks whether a cached file exists at the given path.
func (p *MediaManagerPlugin) cacheExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// cacheWrite writes data to the cache path, creating directories as needed.
func (p *MediaManagerPlugin) cacheWrite(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create cache dir: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}

// getPathMutex returns a mutex for the given cache path, creating one if needed.
func (p *MediaManagerPlugin) getPathMutex(path string) *sync.Mutex {
	val, _ := p.cacheLocks.LoadOrStore(path, &sync.Mutex{})
	return val.(*sync.Mutex)
}

// handleClearAllCache clears all cached image variants.
func (p *MediaManagerPlugin) handleClearAllCache(ctx context.Context) (*pb.PluginHTTPResponse, error) {
	if err := os.RemoveAll(p.cacheBaseDir()); err != nil {
		return jsonError(500, "CACHE_CLEAR_FAILED", "Failed to clear image cache"), nil
	}
	return jsonResponse(200, map[string]any{"data": map[string]any{"message": "Image cache cleared"}}), nil
}

// handleClearSizeCache clears cached images for a specific size.
func (p *MediaManagerPlugin) handleClearSizeCache(ctx context.Context, sizeName string) (*pb.PluginHTTPResponse, error) {
	dir := filepath.Join(p.cacheBaseDir(), sizeName)
	if err := os.RemoveAll(dir); err != nil {
		return jsonError(500, "CACHE_CLEAR_FAILED", fmt.Sprintf("Failed to clear cache for size %q", sizeName)), nil
	}
	return jsonResponse(200, map[string]any{"data": map[string]any{"message": fmt.Sprintf("Cache cleared for size %q", sizeName)}}), nil
}

// clearCacheForOriginal deletes all cached variants of an original file across all sizes.
func (p *MediaManagerPlugin) clearCacheForOriginal(ctx context.Context, originalPath string) {
	// Strip "media/" prefix if present to get the relative path.
	relPath := strings.TrimPrefix(originalPath, "media/")

	// Get all sizes to know which cache dirs to check.
	result, err := p.host.DataQuery(ctx, sizesTable, coreapi.DataStoreQuery{Limit: 100})
	if err != nil {
		return
	}
	for _, row := range result.Rows {
		name, _ := row["name"].(string)
		if name == "" {
			continue
		}
		_ = os.Remove(p.cachePath(name, relPath))
		_ = os.Remove(p.cacheWebPPath(name, relPath))
	}
}

// --- Upload Normalization ---

// normalizeImage applies upload normalization to image bytes if enabled.
// It downscales oversized images and re-encodes with optimal compression.
// Returns the (possibly modified) image bytes and MIME type.
func (p *MediaManagerPlugin) normalizeImage(ctx context.Context, data []byte, mimeType string) ([]byte, string) {
	log.Printf("[normalize] input: %d bytes, mime: %s", len(data), mimeType)

	// Check if normalization is enabled.
	enabled, err := p.host.GetSetting(ctx, "media:optimizer:normalize_enabled")
	if err == nil && enabled == "false" {
		log.Printf("[normalize] disabled, skipping")
		return data, mimeType
	}

	maxDimStr, _ := p.host.GetSetting(ctx, "media:optimizer:normalize_max_dimension")
	maxDim := 5000
	if maxDimStr != "" {
		if parsed, err := strconv.Atoi(maxDimStr); err == nil && parsed > 0 {
			maxDim = parsed
		}
	}

	// Write to temp file for CLI tools.
	tmpFile, tmpErr := os.CreateTemp("", "normalize-*")
	if tmpErr != nil {
		log.Printf("[normalize] failed to create temp file: %v", tmpErr)
		return data, mimeType
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)
	tmpFile.Write(data)
	tmpFile.Close()

	// Step 1: Downscale if needed using ImageMagick (much better quality than Go).
	img, format, decErr := image.Decode(bytes.NewReader(data))
	if decErr != nil {
		log.Printf("[normalize] decode failed: %v", decErr)
		return data, mimeType
	}
	bounds := img.Bounds()
	origW := bounds.Dx()
	origH := bounds.Dy()
	log.Printf("[normalize] decoded: %dx%d, format: %s, maxDim: %d", origW, origH, format, maxDim)

	if origW > maxDim || origH > maxDim {
		// Use ImageMagick to resize — much better quality than Go's resizer.
		resizeArg := fmt.Sprintf("%dx%d>", maxDim, maxDim)
		cmd := exec.Command("magick", tmpPath, "-resize", resizeArg, "-strip", tmpPath)
		if out, err := cmd.CombinedOutput(); err != nil {
			log.Printf("[normalize] magick resize failed: %v: %s", err, string(out))
			// Fallback: try without magick (older imagemagick uses "convert")
			cmd2 := exec.Command("convert", tmpPath, "-resize", resizeArg, "-strip", tmpPath)
			if out2, err2 := cmd2.CombinedOutput(); err2 != nil {
				log.Printf("[normalize] convert resize failed: %v: %s", err2, string(out2))
				return data, mimeType
			}
		}
		log.Printf("[normalize] resized to fit %dx%d", maxDim, maxDim)
	}

	// Step 2: Get upload quality setting.
	quality := 100 // default: lossless
	qualityStr, _ := p.host.GetSetting(ctx, "media:optimizer:upload_quality")
	if qualityStr != "" {
		if parsed, err := strconv.Atoi(qualityStr); err == nil && parsed > 0 && parsed <= 100 {
			quality = parsed
		}
	}
	log.Printf("[normalize] quality: %d", quality)

	// Step 3: Optimize using native CLI tools.
	switch {
	case mimeType == "image/jpeg" || format == "jpeg":
		if quality < 100 {
			// Lossy: jpegoptim with max quality cap.
			cmd := exec.Command("jpegoptim", "--strip-all", "--all-progressive",
				fmt.Sprintf("--max=%d", quality), tmpPath)
			if out, err := cmd.CombinedOutput(); err != nil {
				log.Printf("[normalize] jpegoptim lossy failed: %v: %s", err, string(out))
			} else {
				log.Printf("[normalize] jpegoptim (q%d): %s", quality, strings.TrimSpace(string(out)))
			}
		} else {
			// Lossless: optimize Huffman tables + strip metadata only.
			cmd := exec.Command("jpegoptim", "--strip-all", "--all-progressive", tmpPath)
			if out, err := cmd.CombinedOutput(); err != nil {
				log.Printf("[normalize] jpegoptim failed: %v: %s", err, string(out))
			} else {
				log.Printf("[normalize] jpegoptim (lossless): %s", strings.TrimSpace(string(out)))
			}
		}

	case mimeType == "image/png" || format == "png":
		if quality < 100 {
			// Lossy: use pngquant for lossy PNG compression, then optipng for final polish.
			pqQuality := fmt.Sprintf("%d-%d", quality-5, quality)
			if quality <= 5 {
				pqQuality = fmt.Sprintf("0-%d", quality)
			}
			cmd := exec.Command("pngquant", "--quality", pqQuality, "--force", "--output", tmpPath, tmpPath)
			if out, err := cmd.CombinedOutput(); err != nil {
				log.Printf("[normalize] pngquant failed (non-fatal): %v: %s", err, string(out))
			} else {
				log.Printf("[normalize] pngquant (q%d): done", quality)
			}
		}
		// Always run optipng for lossless final optimization.
		cmd := exec.Command("optipng", "-strip", "all", "-o2", tmpPath)
		if out, err := cmd.CombinedOutput(); err != nil {
			log.Printf("[normalize] optipng failed: %v: %s", err, string(out))
		} else {
			log.Printf("[normalize] optipng: %s", strings.TrimSpace(string(out)))
		}
	}

	// Read back optimized file.
	optimized, readErr := os.ReadFile(tmpPath)
	if readErr != nil {
		log.Printf("[normalize] failed to read optimized file: %v", readErr)
		return data, mimeType
	}

	log.Printf("[normalize] result: %d bytes (was %d, saved %d%%)",
		len(optimized), len(data), (len(data)-len(optimized))*100/len(data))
	return optimized, mimeType
}

// --- Restore / Re-optimize ---

// handleRestoreOriginal restores a single image to its pre-optimization original.
func (p *MediaManagerPlugin) handleRestoreOriginal(ctx context.Context, id uint) (*pb.PluginHTTPResponse, error) {
	row, err := p.host.DataGet(ctx, tableName, id)
	if err != nil {
		if isNotFound(err) {
			return jsonError(404, "NOT_FOUND", "Media file not found"), nil
		}
		return jsonError(500, "FETCH_FAILED", "Failed to fetch media file"), nil
	}

	origPath, _ := row["original_path"].(string)
	if origPath == "" {
		return jsonError(400, "NO_ORIGINAL", "No original backup exists for this file"), nil
	}
	currentPath, _ := row["path"].(string)

	// Read the original file from disk.
	origFullPath := filepath.Join(p.storageDir, origPath)
	origData, err := os.ReadFile(origFullPath)
	if err != nil {
		return jsonError(404, "ORIGINAL_MISSING", "Original backup file not found on disk"), nil
	}

	// Overwrite the current file with the original.
	if _, err := p.host.StoreFile(ctx, currentPath, origData); err != nil {
		return jsonError(500, "RESTORE_FAILED", "Failed to write restored file"), nil
	}

	// Get original dimensions.
	var newW, newH int
	if img, _, decErr := image.Decode(bytes.NewReader(origData)); decErr == nil {
		newW = img.Bounds().Dx()
		newH = img.Bounds().Dy()
	}

	// Update the database record.
	updateData := map[string]any{
		"size":                 len(origData),
		"is_optimized":         false,
		"optimization_savings": 0,
		"updated_at":           time.Now().Format(time.RFC3339),
	}
	if newW > 0 {
		updateData["width"] = newW
	}
	if newH > 0 {
		updateData["height"] = newH
	}
	if err := p.host.DataUpdate(ctx, tableName, id, updateData); err != nil {
		return jsonError(500, "UPDATE_FAILED", "Failed to update media record"), nil
	}

	// Clear cached variants since the source image changed.
	p.clearCacheForOriginal(ctx, currentPath)

	// Fetch and return the updated record.
	updated, err := p.host.DataGet(ctx, tableName, id)
	if err != nil {
		return jsonError(500, "FETCH_FAILED", "Failed to fetch updated record"), nil
	}
	return jsonResponse(200, map[string]any{"data": updated}), nil
}

// handleReoptimize re-optimizes a single image with current settings.
// It reads from the original backup (if available) and re-applies normalization.
func (p *MediaManagerPlugin) handleReoptimize(ctx context.Context, id uint) (*pb.PluginHTTPResponse, error) {
	row, err := p.host.DataGet(ctx, tableName, id)
	if err != nil {
		if isNotFound(err) {
			return jsonError(404, "NOT_FOUND", "Media file not found"), nil
		}
		return jsonError(500, "FETCH_FAILED", "Failed to fetch media file"), nil
	}

	currentPath, _ := row["path"].(string)
	mimeType, _ := row["mime_type"].(string)

	if !strings.HasPrefix(mimeType, "image/") || mimeType == "image/svg+xml" {
		return jsonError(400, "NOT_IMAGE", "Only images can be re-optimized"), nil
	}

	// Determine the source: use original backup if available, else current file.
	sourcePath := currentPath
	origPath, _ := row["original_path"].(string)
	if origPath != "" {
		sourcePath = origPath
	}

	sourceFullPath := filepath.Join(p.storageDir, sourcePath)
	sourceData, err := os.ReadFile(sourceFullPath)
	if err != nil {
		return jsonError(404, "SOURCE_MISSING", "Source image file not found"), nil
	}

	originalSize := len(sourceData)

	// If no original backup exists yet, create one before optimizing.
	if origPath == "" {
		now := time.Now()
		filename, _ := row["filename"].(string)
		dateDir := fmt.Sprintf("%04d/%02d", now.Year(), now.Month())
		origPath = fmt.Sprintf("media/originals/%s/%s", dateDir, filename)
		if _, storeErr := p.host.StoreFile(ctx, origPath, sourceData); storeErr != nil {
			log.Printf("[reoptimize] failed to create original backup: %v", storeErr)
			origPath = ""
		}
	}

	// Get original dimensions.
	var origW, origH int
	if img, _, decErr := image.Decode(bytes.NewReader(sourceData)); decErr == nil {
		origW = img.Bounds().Dx()
		origH = img.Bounds().Dy()
	}

	// Apply normalization with current settings.
	optimized, optimizedMime := p.normalizeImage(ctx, sourceData, mimeType)

	// Only keep the new bytes when they're actually smaller — if the
	// encoder produced a larger file (can happen on already-compressed
	// inputs), fall back to the source and record zero savings. The
	// image is still marked is_optimized=true so it stops showing up as
	// "needs optimization" — we did process it, the result just wasn't
	// an improvement.
	finalBytes := optimized
	finalMime := optimizedMime
	if len(optimized) >= len(sourceData) && optimizedMime == mimeType {
		finalBytes = sourceData
		finalMime = mimeType
	}

	savings := len(sourceData) - len(finalBytes)
	if savings < 0 {
		savings = 0
	}

	// Write the final file (either the optimized bytes or the original
	// source when optimization didn't help).
	if _, err := p.host.StoreFile(ctx, currentPath, finalBytes); err != nil {
		return jsonError(500, "STORE_FAILED", "Failed to store re-optimized file"), nil
	}

	// Get new dimensions.
	var newW, newH int
	if img, _, decErr := image.Decode(bytes.NewReader(finalBytes)); decErr == nil {
		newW = img.Bounds().Dx()
		newH = img.Bounds().Dy()
	}

	// Update record. is_optimized is always true after a successful
	// run — see the size-guard above for why we do this even when the
	// encoder couldn't beat the original.
	updateData := map[string]any{
		"size":                 len(finalBytes),
		"mime_type":            finalMime,
		"is_optimized":         true,
		"original_size":        originalSize,
		"original_path":        origPath,
		"optimization_savings": savings,
		"updated_at":           time.Now().Format(time.RFC3339),
	}
	if origW > 0 {
		updateData["original_width"] = origW
		updateData["original_height"] = origH
	}
	if newW > 0 {
		updateData["width"] = newW
		updateData["height"] = newH
	}
	if err := p.host.DataUpdate(ctx, tableName, id, updateData); err != nil {
		return jsonError(500, "UPDATE_FAILED", "Failed to update media record"), nil
	}

	// Clear cached variants since the source changed.
	p.clearCacheForOriginal(ctx, currentPath)

	updated, err := p.host.DataGet(ctx, tableName, id)
	if err != nil {
		return jsonError(500, "FETCH_FAILED", "Failed to fetch updated record"), nil
	}
	return jsonResponse(200, map[string]any{"data": updated}), nil
}

// handleReoptimizeAll kicks off async (re-)optimization of images. When
// pendingOnly is true, only images where is_optimized is not true are
// processed — useful for catching up on new uploads after a settings
// change without re-encoding previously optimized files.
func (p *MediaManagerPlugin) handleReoptimizeAll(ctx context.Context, pendingOnly bool) (*pb.PluginHTTPResponse, error) {
	if p.reoptimizeProgress.Running {
		return jsonResponse(409, map[string]any{
			"error": map[string]any{"code": "ALREADY_RUNNING", "message": "Re-optimization is already in progress"},
			"data":  p.reoptimizeProgress.snapshot(),
		}), nil
	}

	rawWhere := "mime_type LIKE ?"
	args := []any{"image/%"}
	if pendingOnly {
		rawWhere += " AND (is_optimized IS NULL OR is_optimized = false)"
	}
	result, err := p.host.DataQuery(ctx, tableName, coreapi.DataStoreQuery{
		Raw:   rawWhere,
		Args:  args,
		Limit: 10000,
	})
	if err != nil {
		return jsonError(500, "QUERY_FAILED", "Failed to query media files"), nil
	}

	// Filter to processable images and collect IDs.
	var fileIDs []uint
	for _, row := range result.Rows {
		mimeType, _ := row["mime_type"].(string)
		if mimeType == "image/svg+xml" {
			continue
		}
		var fileID uint
		switch v := row["id"].(type) {
		case float64:
			fileID = uint(v)
		case json.Number:
			n, _ := v.Int64()
			fileID = uint(n)
		}
		if fileID > 0 {
			fileIDs = append(fileIDs, fileID)
		}
	}

	p.reoptimizeProgress.reset(len(fileIDs))

	// Run in background goroutine.
	go func() {
		bgCtx := context.Background()
		for _, fid := range fileIDs {
			resp, err := p.handleReoptimize(bgCtx, fid)
			if err != nil || resp.StatusCode != 200 {
				p.reoptimizeProgress.advance(0, true)
				continue
			}
			var savings int64
			var respBody map[string]any
			if json.Unmarshal(resp.Body, &respBody) == nil {
				if data, ok := respBody["data"].(map[string]any); ok {
					savings = int64(intFromRow(data, "optimization_savings"))
				}
			}
			p.reoptimizeProgress.advance(savings, false)
		}
		p.reoptimizeProgress.finish()
	}()

	return jsonResponse(202, map[string]any{"data": p.reoptimizeProgress.snapshot()}), nil
}

// handleRestoreAll kicks off async restore of all optimized images.
func (p *MediaManagerPlugin) handleRestoreAll(ctx context.Context) (*pb.PluginHTTPResponse, error) {
	if p.restoreProgress.Running {
		return jsonResponse(409, map[string]any{
			"error": map[string]any{"code": "ALREADY_RUNNING", "message": "Restore is already in progress"},
			"data":  p.restoreProgress.snapshot(),
		}), nil
	}

	result, err := p.host.DataQuery(ctx, tableName, coreapi.DataStoreQuery{
		Raw:   "is_optimized = ? AND original_path != ?",
		Args:  []any{true, ""},
		Limit: 10000,
	})
	if err != nil {
		return jsonError(500, "QUERY_FAILED", "Failed to query media files"), nil
	}

	var fileIDs []uint
	for _, row := range result.Rows {
		var fileID uint
		switch v := row["id"].(type) {
		case float64:
			fileID = uint(v)
		case json.Number:
			n, _ := v.Int64()
			fileID = uint(n)
		}
		if fileID > 0 {
			fileIDs = append(fileIDs, fileID)
		}
	}

	p.restoreProgress.reset(len(fileIDs))

	go func() {
		bgCtx := context.Background()
		for _, fid := range fileIDs {
			resp, err := p.handleRestoreOriginal(bgCtx, fid)
			if err != nil || resp.StatusCode != 200 {
				p.restoreProgress.advance(0, true)
				continue
			}
			p.restoreProgress.advance(0, false)
		}
		p.restoreProgress.finish()
	}()

	return jsonResponse(202, map[string]any{"data": p.restoreProgress.snapshot()}), nil
}

// handleOptimizerStats returns aggregate optimization statistics.
func (p *MediaManagerPlugin) handleOptimizerStats(ctx context.Context) (*pb.PluginHTTPResponse, error) {
	// Count all images.
	allResult, err := p.host.DataQuery(ctx, tableName, coreapi.DataStoreQuery{
		Raw:   "mime_type LIKE ? AND mime_type != ?",
		Args:  []any{"image/%", "image/svg+xml"},
		Limit: 10000,
	})
	if err != nil {
		return jsonError(500, "QUERY_FAILED", "Failed to query images"), nil
	}

	totalImages := len(allResult.Rows)
	optimizedCount := 0
	unoptimizedCount := 0
	totalOriginalSize := int64(0)
	totalCurrentSize := int64(0)
	totalSavings := int64(0)
	withBackup := 0

	for _, row := range allResult.Rows {
		isOpt := false
		if v, ok := row["is_optimized"].(bool); ok {
			isOpt = v
		}
		if isOpt {
			optimizedCount++
		} else {
			unoptimizedCount++
		}

		origSize := int64(intFromRow(row, "original_size"))
		currentSize := int64(intFromRow(row, "size"))
		savings := int64(intFromRow(row, "optimization_savings"))

		if origSize > 0 {
			totalOriginalSize += origSize
		} else {
			totalOriginalSize += currentSize
		}
		totalCurrentSize += currentSize
		totalSavings += savings

		origPath, _ := row["original_path"].(string)
		if origPath != "" {
			withBackup++
		}
	}

	return jsonResponse(200, map[string]any{
		"data": map[string]any{
			"total_images":        totalImages,
			"optimized_count":     optimizedCount,
			"unoptimized_count":   unoptimizedCount,
			"with_backup":         withBackup,
			"total_original_size": totalOriginalSize,
			"total_current_size":  totalCurrentSize,
			"total_savings":       totalSavings,
		},
	}), nil
}

// --- Helpers ---

func extractID(path string, pathParams map[string]string) uint {
	// First check path params from proxy.
	if idStr, ok := pathParams["id"]; ok {
		id, _ := strconv.ParseUint(idStr, 10, 64)
		return uint(id)
	}
	// Fallback: parse from path like "/123" or "123".
	path = strings.TrimPrefix(path, "/")
	if path == "" {
		return 0
	}
	// Only use first segment.
	parts := strings.SplitN(path, "/", 2)
	id, _ := strconv.ParseUint(parts[0], 10, 64)
	return uint(id)
}

func paramOr(params map[string]string, key, def string) string {
	if v, ok := params[key]; ok && v != "" {
		return v
	}
	return def
}

func isNotFound(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "NotFound")
}

func jsonResponse(status int, data any) *pb.PluginHTTPResponse {
	body, _ := json.Marshal(data)
	return &pb.PluginHTTPResponse{
		StatusCode: int32(status),
		Headers:    map[string]string{"Content-Type": "application/json"},
		Body:       body,
	}
}

func jsonError(status int, code, message string) *pb.PluginHTTPResponse {
	body, _ := json.Marshal(map[string]any{
		"error": map[string]any{
			"code":    code,
			"message": message,
		},
	})
	return &pb.PluginHTTPResponse{
		StatusCode: int32(status),
		Headers:    map[string]string{"Content-Type": "application/json"},
		Body:       body,
	}
}

// allowedMimePrefixes defines the permitted MIME type prefixes for uploads.
var allowedMimePrefixes = []string{
	"image/",
	"video/",
	"audio/",
	"application/pdf",
	"application/zip",
	"text/plain",
	"text/csv",
}

// isAllowedMimeType checks if a MIME type is in the upload allowlist.
func isAllowedMimeType(mimeType string) bool {
	for _, prefix := range allowedMimePrefixes {
		if strings.HasPrefix(mimeType, prefix) || mimeType == prefix {
			return true
		}
	}
	return false
}

// safeExtension returns a file extension derived from the MIME type when possible,
// falling back to the original filename extension only if it's safe.
func safeExtension(mimeType, originalName string) string {
	// Map common MIME types to safe extensions.
	mimeToExt := map[string]string{
		"image/jpeg":      ".jpg",
		"image/png":       ".png",
		"image/gif":       ".gif",
		"image/webp":      ".webp",
		"image/svg+xml":   ".svg",
		"application/pdf": ".pdf",
		"application/zip": ".zip",
		"video/mp4":       ".mp4",
		"audio/mpeg":      ".mp3",
		"text/plain":      ".txt",
		"text/csv":        ".csv",
	}
	if ext, ok := mimeToExt[mimeType]; ok {
		return ext
	}
	// Fallback: use original extension only if alphanumeric.
	ext := strings.ToLower(filepath.Ext(originalName))
	if ext == "" {
		return ".bin"
	}
	for _, ch := range ext[1:] { // skip the dot
		if !((ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9')) {
			return ".bin"
		}
	}
	return ext
}

// --- Public Image Cache Handler ---

// handlePublicCacheRequest handles GET /media/cache/{size}/{path...} from the public proxy.
// It serves cached/resized images, generating them on-demand if not cached.
func (p *MediaManagerPlugin) handlePublicCacheRequest(ctx context.Context, req *pb.PluginHTTPRequest) (*pb.PluginHTTPResponse, error) {
	// Path is the full URL path: /media/cache/{size}/{original_path...}
	fullPath := req.GetPath()
	remainder := strings.TrimPrefix(fullPath, "/media/cache/")
	if remainder == "" || remainder == fullPath {
		return binaryError(404), nil
	}

	// Split into size name and original path.
	slashIdx := strings.Index(remainder, "/")
	if slashIdx <= 0 {
		return binaryError(404), nil
	}
	sizeName := remainder[:slashIdx]
	originalPath := remainder[slashIdx+1:]

	if sizeName == "" || originalPath == "" {
		return binaryError(404), nil
	}

	// Prevent directory traversal.
	if strings.Contains(originalPath, "..") || strings.HasPrefix(originalPath, "/") || strings.ContainsRune(originalPath, 0) {
		return binaryError(404), nil
	}

	// Look up the size definition from the database.
	sizeResult, err := p.host.DataQuery(ctx, sizesTable, coreapi.DataStoreQuery{
		Where: map[string]any{"name": sizeName},
		Limit: 1,
	})
	if err != nil || sizeResult.Total == 0 || len(sizeResult.Rows) == 0 {
		return binaryError(404), nil
	}
	sizeRow := sizeResult.Rows[0]
	width := intFromRow(sizeRow, "width")
	height := intFromRow(sizeRow, "height")
	mode := stringFromRow(sizeRow, "mode", "fit")
	quality := intFromRow(sizeRow, "quality")

	cp := p.cachePath(sizeName, originalPath)

	// Check if browser wants WebP.
	acceptHeader := ""
	for k, v := range req.GetHeaders() {
		if strings.EqualFold(k, "accept") {
			acceptHeader = v
			break
		}
	}
	wantsWebP := strings.Contains(acceptHeader, "image/webp")

	// Check WebP cache first.
	if wantsWebP {
		webpPath := p.cacheWebPPath(sizeName, originalPath)
		if p.cacheExists(webpPath) {
			return p.serveCachedFile(webpPath, true)
		}
	}

	// Check original-format cache.
	if p.cacheExists(cp) {
		return p.serveCachedFile(cp, false)
	}

	// Acquire per-path mutex to prevent thundering herd.
	mu := p.getPathMutex(cp)
	mu.Lock()
	defer mu.Unlock()

	// Double-check after acquiring lock.
	if wantsWebP {
		webpPath := p.cacheWebPPath(sizeName, originalPath)
		if p.cacheExists(webpPath) {
			return p.serveCachedFile(webpPath, true)
		}
	}
	if p.cacheExists(cp) {
		return p.serveCachedFile(cp, false)
	}

	// Read original file from disk.
	originalFile := filepath.Join(p.storageDir, "media", originalPath)
	originalData, err := os.ReadFile(originalFile)
	if err != nil {
		if os.IsNotExist(err) {
			return binaryError(404), nil
		}
		log.Printf("ERROR: read original %s: %v", originalFile, err)
		return binaryError(500), nil
	}

	// Determine MIME type.
	mimeType := mimeFromPath(originalPath)
	if !isImageMime(mimeType) {
		return binaryError(404), nil
	}

	// Determine quality.
	if quality <= 0 {
		if q, err := p.host.GetSetting(ctx, "media:optimizer:jpeg_quality"); err == nil && q != "" {
			fmt.Sscanf(q, "%d", &quality)
		}
		if quality <= 0 {
			quality = 80
		}
	}

	// Determine WebP settings.
	webpEnabledStr, _ := p.host.GetSetting(ctx, "media:optimizer:webp_enabled")
	webpEnabled := webpEnabledStr != "false"
	webpQuality := 75
	if q, err := p.host.GetSetting(ctx, "media:optimizer:webp_quality"); err == nil && q != "" {
		fmt.Sscanf(q, "%d", &webpQuality)
	}

	// Resize the image.
	resized, outMime, err := resizeImage(bytes.NewReader(originalData), mimeType, width, height, mode, quality)
	if err != nil {
		log.Printf("ERROR: resize %s/%s: %v", sizeName, originalPath, err)
		return binaryError(500), nil
	}

	// Cache the resized version.
	if err := p.cacheWrite(cp, resized); err != nil {
		log.Printf("WARN: cache write %s: %v", cp, err)
	}

	outputData := resized
	outputMime := outMime

	// WebP conversion if requested and enabled.
	if wantsWebP && webpEnabled && outMime != "image/webp" {
		webpData, webpErr := convertToWebP(bytes.NewReader(resized), webpQuality)
		if webpErr == nil && len(webpData) < len(resized) {
			// Cache the WebP variant.
			webpCachePath := p.cacheWebPPath(sizeName, originalPath)
			if err := p.cacheWrite(webpCachePath, webpData); err != nil {
				log.Printf("WARN: cache write webp %s: %v", webpCachePath, err)
			}
			outputData = webpData
			outputMime = "image/webp"
		}
	}

	return &pb.PluginHTTPResponse{
		StatusCode: 200,
		Headers: map[string]string{
			"Content-Type":  outputMime,
			"Cache-Control": "public, max-age=31536000",
			"Vary":          "Accept",
		},
		Body: outputData,
	}, nil
}

// serveCachedFile reads a cached file from disk and returns it as an HTTP response.
func (p *MediaManagerPlugin) serveCachedFile(path string, isWebP bool) (*pb.PluginHTTPResponse, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return binaryError(500), nil
	}
	contentType := mimeFromPath(path)
	if isWebP {
		contentType = "image/webp"
	}
	return &pb.PluginHTTPResponse{
		StatusCode: 200,
		Headers: map[string]string{
			"Content-Type":  contentType,
			"Cache-Control": "public, max-age=31536000",
		},
		Body: data,
	}, nil
}

// binaryError returns a simple HTTP error response (non-JSON, for public routes).
func binaryError(status int) *pb.PluginHTTPResponse {
	return &pb.PluginHTTPResponse{
		StatusCode: int32(status),
		Headers:    map[string]string{"Content-Type": "text/plain"},
		Body:       []byte(http.StatusText(status)),
	}
}

// --- Image Processing ---

// resizeImage resizes an image to given dimensions with the specified mode.
func resizeImage(src io.Reader, mimeType string, width, height int, mode string, quality int) ([]byte, string, error) {
	img, err := decodeImage(src, mimeType)
	if err != nil {
		return nil, "", fmt.Errorf("decode: %w", err)
	}
	if quality <= 0 {
		quality = 80
	}

	var resized image.Image
	switch mode {
	case "crop":
		resized = resizeCrop(img, width, height)
	case "fit":
		resized = resizeFit(img, width, height)
	case "width":
		resized = resizeWidth(img, width)
	default:
		return nil, "", fmt.Errorf("unknown resize mode: %s", mode)
	}

	data, err := encodeImage(resized, mimeType, quality)
	if err != nil {
		return nil, "", fmt.Errorf("encode: %w", err)
	}
	return data, mimeType, nil
}

// convertToWebP converts any image to WebP format.
func convertToWebP(src io.Reader, quality int) ([]byte, error) {
	img, _, err := image.Decode(src)
	if err != nil {
		return nil, fmt.Errorf("decode for webp conversion: %w", err)
	}
	var buf bytes.Buffer
	if err := gowebp.Encode(&buf, img, gowebp.Options{Quality: quality}); err != nil {
		return nil, fmt.Errorf("webp encode: %w", err)
	}
	return buf.Bytes(), nil
}

// decodeImage decodes an image from the reader based on MIME type.
func decodeImage(r io.Reader, mimeType string) (image.Image, error) {
	switch mimeType {
	case "image/jpeg":
		return jpeg.Decode(r)
	case "image/png":
		return png.Decode(r)
	case "image/gif":
		return gif.Decode(r)
	case "image/webp":
		img, _, err := image.Decode(r)
		return img, err
	default:
		img, _, err := image.Decode(r)
		return img, err
	}
}

// resizeFit resizes an image to fit inside the given bounds while preserving aspect ratio.
func resizeFit(img image.Image, maxW, maxH int) image.Image {
	bounds := img.Bounds()
	srcW := bounds.Dx()
	srcH := bounds.Dy()

	if srcW <= maxW && srcH <= maxH {
		return img
	}

	ratio := float64(srcW) / float64(srcH)
	newW := maxW
	newH := int(float64(newW) / ratio)

	if newH > maxH {
		newH = maxH
		newW = int(float64(newH) * ratio)
	}

	if newW < 1 {
		newW = 1
	}
	if newH < 1 {
		newH = 1
	}

	dst := image.NewRGBA(image.Rect(0, 0, newW, newH))
	draw.CatmullRom.Scale(dst, dst.Bounds(), img, bounds, draw.Over, nil)
	return dst
}

// resizeCrop resizes an image to cover the target dimensions, then center-crops to exact size.
func resizeCrop(img image.Image, w, h int) image.Image {
	bounds := img.Bounds()
	srcW := bounds.Dx()
	srcH := bounds.Dy()

	scaleW := float64(w) / float64(srcW)
	scaleH := float64(h) / float64(srcH)
	scale := scaleW
	if scaleH > scaleW {
		scale = scaleH
	}

	scaledW := int(float64(srcW) * scale)
	scaledH := int(float64(srcH) * scale)
	if scaledW < 1 {
		scaledW = 1
	}
	if scaledH < 1 {
		scaledH = 1
	}

	scaled := image.NewRGBA(image.Rect(0, 0, scaledW, scaledH))
	draw.CatmullRom.Scale(scaled, scaled.Bounds(), img, bounds, draw.Over, nil)

	offsetX := (scaledW - w) / 2
	offsetY := (scaledH - h) / 2

	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Copy(dst, image.Point{}, scaled, image.Rect(offsetX, offsetY, offsetX+w, offsetY+h), draw.Src, nil)
	return dst
}

// resizeWidth resizes an image to the given width, preserving aspect ratio.
func resizeWidth(img image.Image, w int) image.Image {
	bounds := img.Bounds()
	srcW := bounds.Dx()
	srcH := bounds.Dy()

	if srcW <= w {
		return img
	}

	ratio := float64(srcH) / float64(srcW)
	newH := int(float64(w) * ratio)
	if newH < 1 {
		newH = 1
	}

	dst := image.NewRGBA(image.Rect(0, 0, w, newH))
	draw.CatmullRom.Scale(dst, dst.Bounds(), img, bounds, draw.Over, nil)
	return dst
}

// encodeImage encodes an image to the specified format.
func encodeImage(img image.Image, mimeType string, quality int) ([]byte, error) {
	var buf bytes.Buffer
	switch mimeType {
	case "image/jpeg":
		if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality}); err != nil {
			return nil, err
		}
	case "image/png":
		enc := &png.Encoder{CompressionLevel: png.BestCompression}
		if err := enc.Encode(&buf, img); err != nil {
			return nil, err
		}
	case "image/gif":
		if err := gif.Encode(&buf, img, nil); err != nil {
			return nil, err
		}
	case "image/webp":
		if err := gowebp.Encode(&buf, img, gowebp.Options{Quality: quality}); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported output format: %s", mimeType)
	}
	return buf.Bytes(), nil
}

// mimeFromPath returns the MIME type for a file based on its extension.
func mimeFromPath(path string) string {
	ext := filepath.Ext(path)
	if ext == "" {
		return "application/octet-stream"
	}
	mimeType := mime.TypeByExtension(ext)
	if mimeType == "" {
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

// intFromRow extracts an int from a data store row map.
func intFromRow(row map[string]any, key string) int {
	v, ok := row[key]
	if !ok {
		return 0
	}
	switch n := v.(type) {
	case float64:
		return int(n)
	case json.Number:
		i, _ := n.Int64()
		return int(i)
	case int:
		return n
	case int64:
		return int(n)
	default:
		return 0
	}
}

// stringFromRow extracts a string from a data store row map with a default.
func stringFromRow(row map[string]any, key, def string) string {
	v, ok := row[key]
	if !ok {
		return def
	}
	s, ok := v.(string)
	if !ok || s == "" {
		return def
	}
	return s
}

func main() {
	goplugin.Serve(&goplugin.ServeConfig{
		HandshakeConfig: vibeplugin.Handshake,
		VersionedPlugins: map[int]goplugin.PluginSet{
			2: {"extension": &vibeplugin.ExtensionGRPCPlugin{Impl: &MediaManagerPlugin{}}},
		},
		GRPCServer: goplugin.DefaultGRPCServer,
	})
}

// handlePublicMediaRequest handles GET /media/{path} — serves the original file
// but auto-converts to WebP if the browser accepts it and WebP is enabled.
// This acts like Apache's mod_rewrite + WebP Express.
func (p *MediaManagerPlugin) handlePublicMediaRequest(ctx context.Context, req *pb.PluginHTTPRequest) (*pb.PluginHTTPResponse, error) {
	fullPath := req.GetPath()
	// Path: /media/2026/03/photo.jpg
	relPath := strings.TrimPrefix(fullPath, "/media/")
	if relPath == "" {
		return &pb.PluginHTTPResponse{StatusCode: 404}, nil
	}

	// Only process image files.
	ext := strings.ToLower(filepath.Ext(relPath))
	imageExts := map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".webp": true}
	if !imageExts[ext] {
		// Not an image — let it fall through (return 404 so Fiber tries next handler / static).
		return &pb.PluginHTTPResponse{StatusCode: 404}, nil
	}

	// Check if browser accepts WebP.
	acceptHeader := req.GetHeaders()["Accept"]
	if acceptHeader == "" {
		acceptHeader = req.GetHeaders()["accept"]
	}
	wantsWebP := strings.Contains(acceptHeader, "image/webp")

	// Check if WebP is enabled in settings.
	webpEnabled := true
	if val, err := p.host.GetSetting(ctx, "media:optimizer:webp_enabled"); err == nil && val == "false" {
		webpEnabled = false
	}

	// If no WebP wanted or not enabled, serve original file directly.
	if !wantsWebP || !webpEnabled || ext == ".webp" {
		originalFile := filepath.Join("storage", "media", relPath)
		data, err := os.ReadFile(originalFile)
		if err != nil {
			return &pb.PluginHTTPResponse{StatusCode: 404}, nil
		}
		mimeType := mimeFromExt(ext)
		return &pb.PluginHTTPResponse{
			StatusCode: 200,
			Headers: map[string]string{
				"Content-Type":  mimeType,
				"Cache-Control": "public, max-age=31536000",
				"Vary":          "Accept",
			},
			Body: data,
		}, nil
	}

	// WebP conversion path.
	webpCachePath := filepath.Join(p.cacheBaseDir(), "_webp", relPath+".webp")

	// Check if we have a cached WebP version.
	if data, err := os.ReadFile(webpCachePath); err == nil {
		return &pb.PluginHTTPResponse{
			StatusCode: 200,
			Headers: map[string]string{
				"Content-Type":  "image/webp",
				"Cache-Control": "public, max-age=31536000",
				"Vary":          "Accept",
			},
			Body: data,
		}, nil
	}

	// Read original file.
	originalFile := filepath.Join("storage", "media", relPath)
	originalData, err := os.ReadFile(originalFile)
	if err != nil {
		return &pb.PluginHTTPResponse{StatusCode: 404}, nil
	}

	// Convert to WebP.
	webpQuality := 75
	if q, err := p.host.GetSetting(ctx, "media:optimizer:webp_quality"); err == nil && q != "" {
		if parsed, err := strconv.Atoi(q); err == nil && parsed > 0 {
			webpQuality = parsed
		}
	}

	webpData, convErr := convertToWebP(bytes.NewReader(originalData), webpQuality)
	if convErr != nil {
		// Conversion failed — serve original with Vary header.
		mimeType := mimeFromExt(ext)
		return &pb.PluginHTTPResponse{
			StatusCode: 200,
			Headers: map[string]string{
				"Content-Type":  mimeType,
				"Cache-Control": "public, max-age=31536000",
				"Vary":          "Accept",
			},
			Body: originalData,
		}, nil
	}

	// Cache the WebP version.
	if dir := filepath.Dir(webpCachePath); dir != "" {
		os.MkdirAll(dir, 0o755)
	}
	os.WriteFile(webpCachePath, webpData, 0o644)

	return &pb.PluginHTTPResponse{
		StatusCode: 200,
		Headers: map[string]string{
			"Content-Type":  "image/webp",
			"Cache-Control": "public, max-age=31536000",
			"Vary":          "Accept",
		},
		Body: webpData,
	}, nil
}

// mimeFromExt returns the MIME type for a file extension.
func mimeFromExt(ext string) string {
	switch ext {
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
