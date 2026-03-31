# Image Optimizer — Design Spec

## Overview

On-the-fly image resizing, cropping, and WebP conversion for VibeCMS media files. Images are generated on first request and cached to disk. Inspired by WordPress fly-images + WebP Express.

## Upload Normalization

Every image uploaded through the media system is automatically normalized before storage:

1. **Max dimensions:** Downscale to 5000×5000px if either dimension exceeds it (preserve aspect ratio, fit inside)
2. **Strip metadata:** Remove EXIF, ICC profiles, GPS data, camera info — reduces file size significantly
3. **Normalize color depth:** Convert to standard 8-bit sRGB — no 16-bit, ProPhoto, or exotic color spaces
4. **Compress losslessly:** Re-encode with optimal compression settings:
   - JPEG: progressive encoding, quality 92 (visually lossless, ~60-80% size reduction from camera originals)
   - PNG: maximum zlib compression
   - Keep original format — no format conversion at this stage
5. **Update metadata:** Write the new dimensions and file size back to the `media_files` DB record

This runs synchronously during upload (before the response is sent). A 10MB camera JPEG typically normalizes to 500KB-1.5MB with no visible quality loss.

**Settings (admin UI):**
- Max dimension: number input (default: 5000)
- Upload JPEG quality: slider 1-100 (default: 92)
- Normalize enabled: toggle (default: on) — escape hatch if someone needs raw uploads

## URL Scheme

```
/media/cache/{size_name}/{original_path}
```

Examples:
- `/media/cache/thumbnail/2026/03/photo.jpg` → 150x150 crop, JPEG
- `/media/cache/medium/2026/03/photo.jpg` → 250x250 fit, JPEG

WebP auto-negotiation: if the browser sends `Accept: image/webp`, the handler returns a WebP variant instead of the original format. No URL change needed — the handler checks the Accept header.

## Architecture

### Where things live

| Component | Location | Why |
|-----------|----------|-----|
| Cache route handler | `cmd/vibecms/main.go` (before static `/media`) | Must be public, not behind admin auth |
| Image processing logic | `internal/media/optimizer.go` | Core package, reusable |
| Image size registry | `internal/media/sizes.go` | In-memory registry with DB backing |
| Template functions | `internal/rendering/template_renderer.go` | Added to FuncMap |
| Size registration API | Media extension gRPC plugin | Extensions/themes call `DataCreate` |
| Admin UI settings | Media extension React micro-frontend | Settings tab |
| DB migration | Media extension SQL migration | `media_image_sizes` table |
| Cache storage | `storage/cache/images/{size}/{path}.{ext}` | On disk, gitignored |

### Flow: On-the-fly generation

```
Browser requests /media/cache/thumbnail/2026/03/photo.jpg
  → Core route handler (before static /media)
  → Parse size name + original path
  → Look up size in registry → { width: 150, height: 150, mode: "crop" }
  → Check disk cache: storage/cache/images/thumbnail/2026/03/photo.jpg
    → If exists: serve file, done
    → If not:
      → Read original: storage/media/2026/03/photo.jpg
      → Resize/crop per size definition
      → If Accept: image/webp → encode as WebP
      → Else → encode in original format (JPEG quality from settings)
      → Write to cache path
      → Serve file
```

### Image Size Registry

**In-memory map** loaded on startup from DB, refreshed on change.

```go
type ImageSize struct {
    Name     string // "thumbnail", "medium", "large", "hero"
    Width    int    // Target width in px
    Height   int    // Target height in px
    Mode     string // "crop" | "fit" | "width"
    Source   string // "default" | "theme" | extension slug
    Quality  int    // 0 = use global default
}
```

**Resize modes:**
- `crop` — Resize to cover target dimensions, then center-crop to exact size (like WordPress thumbnail)
- `fit` — Resize to fit inside target dimensions, preserving aspect ratio (like WordPress medium/large)
- `width` — Resize to target width, auto-calculate height

**Default sizes** (registered by media extension on activation):
- `thumbnail` — 150×150, crop
- `medium` — 250×250, fit
- `large` — 500×500, fit

### Size Registration by Extensions/Themes

Extensions and themes register sizes via CoreAPI `DataCreate` on the `media_image_sizes` table:

**From Tengo (theme script.tgo):**
```javascript
core := import("core/data")

core.create("media_image_sizes", {
    "name": "hero",
    "width": 1200,
    "height": 600,
    "mode": "crop",
    "source": "theme"
})
```

**From gRPC plugin (Go extension):**
```go
host.DataCreate(ctx, "media_image_sizes", map[string]any{
    "name":   "card",
    "width":  400,
    "height": 300,
    "mode":   "crop",
    "source": "my-extension",
})
```

Media extension loads all sizes from `media_image_sizes` table on startup and caches in memory. A `media:sizes_changed` event triggers a reload.

### Template Functions

Added to Go template FuncMap:

```go
// image_url returns the cache URL for a given size
// Usage: {{ image_url .image.url "thumbnail" }}
// Returns: /media/cache/thumbnail/2026/03/photo.jpg
func imageURL(originalURL string, sizeName string) string

// image_srcset returns a srcset string for responsive images
// Usage: <img src="{{ image_url .url "medium" }}" srcset="{{ image_srcset .url "medium" "large" }}" />
// Returns: /media/cache/medium/... 250w, /media/cache/large/... 500w
func imageSrcset(originalURL string, sizeNames ...string) string
```

These are simple URL rewriters — they don't process images, just transform `/media/2026/03/photo.jpg` → `/media/cache/{size}/2026/03/photo.jpg`. The actual processing happens on HTTP request.

### Image Processing

**Library:** Go stdlib `image` + `golang.org/x/image/draw` (Lanczos resampling) + `github.com/chai2010/webp` for WebP encoding. No CGO required.

**Supported input formats:** JPEG, PNG, GIF, WebP
**Output formats:** JPEG (configurable quality), WebP (configurable quality)

**Upload normalization:** Same library handles the normalize step — decode, downscale if needed, strip metadata (by not copying it), re-encode with optimal settings.

### Cache Management

- **Cache path:** `storage/cache/images/{size_name}/{original_relative_path}`
- **Invalidation on delete:** When a media file is deleted, delete all its cached variants (media extension already handles delete — add cache cleanup)
- **Clear all cache:** Admin button wipes `storage/cache/images/` directory
- **Size change:** When a size definition changes (width/height/mode), invalidate that size's cache directory

### Admin UI — Image Optimizer Settings

New tab/section in Media extension settings page:

**Registered Sizes table:**
| Name | Dimensions | Mode | Source | Cached Files |
|------|-----------|------|--------|-------------|
| thumbnail | 150×150 | crop | default | 234 |
| medium | 250×250 | fit | default | 198 |
| large | 500×500 | fit | default | 156 |
| hero | 1200×600 | crop | theme | 45 |

**Settings:**
- Global JPEG quality: slider 1-100 (default: 80)
- WebP enabled: toggle (default: on)
- WebP quality: slider 1-100 (default: 75)

**Actions:**
- "Clear Cache" button — wipes all cached images, shows freed disk space
- Per-size "Clear" button — wipes cache for one size

**Cache stats:**
- Total cache size on disk
- Number of cached files
- Per-size breakdown

### Database

**Table: `media_image_sizes`** (media extension migration)

```sql
CREATE TABLE IF NOT EXISTS media_image_sizes (
    id SERIAL PRIMARY KEY,
    name VARCHAR(50) UNIQUE NOT NULL,
    width INT NOT NULL,
    height INT NOT NULL,
    mode VARCHAR(20) NOT NULL DEFAULT 'fit',
    source VARCHAR(100) NOT NULL DEFAULT 'default',
    quality INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

**Settings** stored via CoreAPI `SettingsGet/Set`:
- `media:optimizer:jpeg_quality` → 80
- `media:optimizer:webp_enabled` → true
- `media:optimizer:webp_quality` → 75
- `media:optimizer:normalize_enabled` → true
- `media:optimizer:normalize_max_dimension` → 5000
- `media:optimizer:normalize_jpeg_quality` → 92

### Security

- Size name validated against registry — unknown sizes return 404 (prevents arbitrary resize abuse)
- Original path validated with `safeStoragePath()` — no path traversal
- Only image MIME types are processed — non-images return 404
- Rate limit: no special rate limiting needed since cached files are served on repeat requests

### Performance

- First request: ~50-200ms for resize (depending on image size)
- Subsequent requests: direct file serve from cache (sub-ms, same as static)
- Memory: images processed one at a time, decoded → resized → encoded → written. No concurrent processing of same image (mutex per path).

## Files to Create/Modify

### New files:
1. `internal/media/optimizer.go` — Image resize/crop/WebP logic + upload normalization
2. `internal/media/sizes.go` — Size registry (in-memory + DB)
3. `internal/media/cache.go` — Cache management (paths, clear, stats)
4. `internal/media/handler.go` — HTTP handler for `/media/cache/*`
5. `extensions/media-manager/migrations/003_image_sizes.sql` — DB table
6. `extensions/media-manager/admin-ui/src/ImageOptimizerSettings.tsx` — Admin UI

### Modified files:
1. `cmd/vibecms/main.go` — Register cache route before static `/media`
2. `internal/rendering/template_renderer.go` — Add `image_url`, `image_srcset` to FuncMap
3. `extensions/media-manager/cmd/plugin/main.go` — Register default sizes, handle settings API, cache cleanup on delete
4. `extensions/media-manager/admin-ui/src/MediaLibrary.tsx` — Add settings tab link
5. `go.mod` — Add `golang.org/x/image` and `github.com/chai2010/webp`
