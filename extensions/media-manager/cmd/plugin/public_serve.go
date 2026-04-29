package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"squilla/internal/coreapi"
	pb "squilla/pkg/plugin/proto"
)

// This file owns the public asset-serving endpoints — the resize-on-demand
// cache route and the media-files passthrough.

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

func (p *MediaManagerPlugin) handlePublicMediaRequest(ctx context.Context, req *pb.PluginHTTPRequest) (*pb.PluginHTTPResponse, error) {
	fullPath := req.GetPath()
	// Path: /media/2026/03/photo.jpg
	relPath := strings.TrimPrefix(fullPath, "/media/")
	if relPath == "" {
		return &pb.PluginHTTPResponse{StatusCode: 404}, nil
	}

	// Reject directory traversal — the cache route rejects it too, and
	// this entry would otherwise read arbitrary files via /media/../etc.
	if strings.Contains(relPath, "..") || strings.ContainsRune(relPath, 0) {
		return &pb.PluginHTTPResponse{StatusCode: 404}, nil
	}

	ext := strings.ToLower(filepath.Ext(relPath))
	imageExts := map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".webp": true}
	if !imageExts[ext] {
		// Non-image (video / audio / pdf / docx / etc.) — serve from disk
		// directly. Honors HTTP Range requests so HTML5 <video> / <audio>
		// elements can seek; the plugin proxy returns the full body so we
		// slice it ourselves.
		return p.serveStaticAsset(relPath, ext, req)
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

// serveStaticAsset serves a non-image media file (video, audio, document)
// from storage/media with HTTP Range support so HTML5 <video>/<audio> can
// seek without buffering the full file. The plugin proxy returns whole
// payloads, so we slice the bytes ourselves and emit 206 Partial Content
// when the client asks for a byte range.
func (p *MediaManagerPlugin) serveStaticAsset(relPath, ext string, req *pb.PluginHTTPRequest) (*pb.PluginHTTPResponse, error) {
	originalFile := filepath.Join("storage", "media", relPath)
	data, err := os.ReadFile(originalFile)
	if err != nil {
		return &pb.PluginHTTPResponse{StatusCode: 404}, nil
	}

	mimeType := mimeFromExt(ext)
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}
	total := int64(len(data))

	rangeHeader := ""
	for k, v := range req.GetHeaders() {
		if strings.EqualFold(k, "range") {
			rangeHeader = v
			break
		}
	}

	headers := map[string]string{
		"Content-Type":  mimeType,
		"Cache-Control": "public, max-age=31536000",
		"Accept-Ranges": "bytes",
	}

	// Parse a single 'bytes=start-end' range. Multipart ranges fall back
	// to a full-body response — they're rare in the wild for the formats
	// we serve.
	if start, end, ok := parseSingleByteRange(rangeHeader, total); ok {
		headers["Content-Range"] = fmt.Sprintf("bytes %d-%d/%d", start, end, total)
		headers["Content-Length"] = strconv.FormatInt(end-start+1, 10)
		return &pb.PluginHTTPResponse{
			StatusCode: 206,
			Headers:    headers,
			Body:       data[start : end+1],
		}, nil
	}

	headers["Content-Length"] = strconv.FormatInt(total, 10)
	return &pb.PluginHTTPResponse{
		StatusCode: 200,
		Headers:    headers,
		Body:       data,
	}, nil
}

// parseSingleByteRange parses a single-range Range header of the form
// `bytes=start-end` (either bound optional). Returns the resolved
// inclusive byte indices and ok=true on success. Multi-range, malformed,
// or unsatisfiable ranges return ok=false so the caller falls back to a
// full 200 response.
func parseSingleByteRange(header string, total int64) (int64, int64, bool) {
	if header == "" || total <= 0 {
		return 0, 0, false
	}
	const prefix = "bytes="
	if !strings.HasPrefix(header, prefix) {
		return 0, 0, false
	}
	spec := strings.TrimSpace(header[len(prefix):])
	if strings.Contains(spec, ",") {
		// Multi-range — out of scope; fall back to 200.
		return 0, 0, false
	}
	parts := strings.SplitN(spec, "-", 2)
	if len(parts) != 2 {
		return 0, 0, false
	}
	startStr := strings.TrimSpace(parts[0])
	endStr := strings.TrimSpace(parts[1])

	var start, end int64
	switch {
	case startStr == "" && endStr != "":
		// Suffix range: last N bytes.
		n, err := strconv.ParseInt(endStr, 10, 64)
		if err != nil || n <= 0 {
			return 0, 0, false
		}
		if n > total {
			n = total
		}
		start = total - n
		end = total - 1
	case startStr != "":
		s, err := strconv.ParseInt(startStr, 10, 64)
		if err != nil || s < 0 || s >= total {
			return 0, 0, false
		}
		start = s
		if endStr == "" {
			end = total - 1
		} else {
			e, err := strconv.ParseInt(endStr, 10, 64)
			if err != nil || e < start {
				return 0, 0, false
			}
			if e >= total {
				e = total - 1
			}
			end = e
		}
	default:
		return 0, 0, false
	}
	return start, end, true
}

// mimeFromExt returns the MIME type for a file extension.
