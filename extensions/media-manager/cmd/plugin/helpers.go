package main

import (
	"encoding/json"
	"mime"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	pb "squilla/pkg/plugin/proto"
)

// This file owns the small response/parsing/mime helpers used across
// the handlers.

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
	// Office / iWork document mime-types — added in the editing audit
	// follow-up so common downloadable assets (specs, contracts,
	// spreadsheets) can live alongside images.
	"application/msword",
	"application/vnd.openxmlformats-officedocument.wordprocessingml.document",
	"application/vnd.ms-excel",
	"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
	"application/vnd.ms-powerpoint",
	"application/vnd.openxmlformats-officedocument.presentationml.presentation",
	"application/vnd.oasis.opendocument.text",
	"application/vnd.oasis.opendocument.spreadsheet",
	"application/vnd.oasis.opendocument.presentation",
	"text/plain",
	"text/csv",
	"text/markdown",
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

func binaryError(status int) *pb.PluginHTTPResponse {
	return &pb.PluginHTTPResponse{
		StatusCode: int32(status),
		Headers:    map[string]string{"Content-Type": "text/plain"},
		Body:       []byte(http.StatusText(status)),
	}
}

// --- Image Processing ---

// resizeImage resizes an image to given dimensions with the specified mode.

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
	case ".avif":
		return "image/avif"
	case ".svg":
		return "image/svg+xml"
	// Video
	case ".mp4":
		return "video/mp4"
	case ".webm":
		return "video/webm"
	case ".mov":
		return "video/quicktime"
	case ".m4v":
		return "video/x-m4v"
	case ".mkv":
		return "video/x-matroska"
	// Audio
	case ".mp3":
		return "audio/mpeg"
	case ".wav":
		return "audio/wav"
	case ".ogg", ".oga":
		return "audio/ogg"
	case ".m4a":
		return "audio/mp4"
	case ".flac":
		return "audio/flac"
	// Documents
	case ".pdf":
		return "application/pdf"
	case ".zip":
		return "application/zip"
	case ".doc":
		return "application/msword"
	case ".docx":
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	case ".xls":
		return "application/vnd.ms-excel"
	case ".xlsx":
		return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	case ".ppt":
		return "application/vnd.ms-powerpoint"
	case ".pptx":
		return "application/vnd.openxmlformats-officedocument.presentationml.presentation"
	case ".odt":
		return "application/vnd.oasis.opendocument.text"
	case ".ods":
		return "application/vnd.oasis.opendocument.spreadsheet"
	case ".odp":
		return "application/vnd.oasis.opendocument.presentation"
	case ".txt":
		return "text/plain; charset=utf-8"
	case ".csv":
		return "text/csv; charset=utf-8"
	case ".md":
		return "text/markdown; charset=utf-8"
	default:
		return "application/octet-stream"
	}
}
