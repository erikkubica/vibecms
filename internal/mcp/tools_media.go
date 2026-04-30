package mcp

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"mime"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"path"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"

	"squilla/internal/coreapi"
	pb "squilla/pkg/plugin/proto"
)

// mediaManagerSlug is the extension slug we proxy media uploads through so
// MCP and the admin UI share one upload pipeline (validation, normalisation,
// WebP, optimiser settings, owner tagging, etc.). Falls back to the kernel
// cms.MediaService path when the extension isn't active.
const mediaManagerSlug = "media-manager"

// uploadViaMediaManager proxies (filename, mime_type, body) through the
// media-manager extension's POST /upload handler — the same code path the
// admin UI uses. Returns nil, nil when the extension is unavailable so the
// caller can fall through to the kernel fallback.
func (s *Server) uploadViaMediaManager(ctx context.Context, filename, mimeType string, body []byte) (map[string]any, error) {
	if s.deps.PluginManager == nil {
		return nil, nil
	}
	client := s.deps.PluginManager.GetClient(mediaManagerSlug)
	if client == nil {
		return nil, nil
	}

	// Build a multipart body with one "file" field — matches what the
	// admin UI's <input type="file"> form submits.
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	hdr := make(textproto.MIMEHeader)
	hdr.Set("Content-Disposition", fmt.Sprintf(`form-data; name="file"; filename="%s"`, filename))
	if mimeType != "" {
		hdr.Set("Content-Type", mimeType)
	}
	part, err := mw.CreatePart(hdr)
	if err != nil {
		return nil, fmt.Errorf("multipart create: %w", err)
	}
	if _, err := part.Write(body); err != nil {
		return nil, fmt.Errorf("multipart write: %w", err)
	}
	if err := mw.Close(); err != nil {
		return nil, fmt.Errorf("multipart close: %w", err)
	}

	req := &pb.PluginHTTPRequest{
		Method:      "POST",
		Path:        "/upload",
		Headers:     map[string]string{"Content-Type": mw.FormDataContentType()},
		Body:        buf.Bytes(),
		QueryParams: map[string]string{},
		PathParams:  map[string]string{"slug": mediaManagerSlug, "path": "upload"},
	}
	resp, err := client.HandleHTTPRequest(req)
	if err != nil {
		return nil, fmt.Errorf("media-manager upload: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("media-manager returned status %d: %s", resp.StatusCode, string(resp.Body))
	}
	var out map[string]any
	if err := json.Unmarshal(resp.Body, &out); err != nil {
		return nil, fmt.Errorf("media-manager response not JSON: %w", err)
	}
	// media-manager wraps single-object responses as {"data": {...}} — unwrap
	// for the MCP caller so the shape stays {id, url, ...}.
	if data, ok := out["data"].(map[string]any); ok {
		return data, nil
	}
	return out, nil
}

func (s *Server) registerMediaTools() {
	api := s.deps.CoreAPI

	s.addTool(mcp.NewTool("core.media.get",
		mcp.WithDescription("Fetch a single media file record by ID."),
		mcp.WithNumber("id", mcp.Required()),
	), "read", func(ctx context.Context, args map[string]any) (any, error) {
		return api.GetMedia(ctx, uintArg(args, "id"))
	})

	s.addTool(mcp.NewTool("core.media.query",
		mcp.WithDescription("Search media by mime_type or filename substring."),
		mcp.WithString("mime_type"),
		mcp.WithString("search"),
		mcp.WithNumber("limit"),
		mcp.WithNumber("offset"),
	), "read", func(ctx context.Context, args map[string]any) (any, error) {
		return api.QueryMedia(ctx, coreapi.MediaQuery{
			MimeType: stringArg(args, "mime_type"),
			Search:   stringArg(args, "search"),
			Limit:    clampLimit(intArg(args, "limit")),
			Offset:   intArg(args, "offset"),
		})
	})

	s.addTool(mcp.NewTool("core.media.upload",
		mcp.WithDescription("Upload a media file (image, video, doc) and register it in the media library. Returns {id, url, slug, ...} — reference by slug in theme-portable content.\n\nUse when: attaching an image/file to a node, hero, gallery, etc.\nDO NOT use when: storing arbitrary files with no URL/DB record — use core.files.store. Importing a theme-packaged asset — theme activation handles that automatically.\n\nBody must be base64-encoded. When the media-manager extension is active, the upload routes through the same /upload handler as the admin UI (image normalisation, WebP, original backup, optimiser settings) — MCP uploads are functionally identical to manual uploads, not a parallel pipeline."),
		mcp.WithString("filename", mcp.Required()),
		mcp.WithString("mime_type", mcp.Required()),
		mcp.WithString("body_base64", mcp.Required(), mcp.Description("base64-encoded file body")),
	), "content", func(ctx context.Context, args map[string]any) (any, error) {
		raw, err := base64.StdEncoding.DecodeString(stringArg(args, "body_base64"))
		if err != nil {
			return nil, fmt.Errorf("decode body_base64: %w", err)
		}
		filename := stringArg(args, "filename")
		mimeType := stringArg(args, "mime_type")
		// Prefer the extension path so optimiser / WebP / original backup
		// flows fire identically to a manual upload. Fall through to the
		// kernel fallback only when the extension is inactive.
		if out, err := s.uploadViaMediaManager(ctx, filename, mimeType, raw); err != nil {
			return nil, err
		} else if out != nil {
			return out, nil
		}
		return api.UploadMedia(ctx, coreapi.MediaUploadRequest{
			Filename: filename,
			MimeType: mimeType,
			Body:     bytes.NewReader(raw),
		})
	})

	s.addTool(mcp.NewTool("core.media.import_url",
		mcp.WithDescription("Download a file from a public URL and store it in the media library. Returns the stored MediaFile — use its id and url when populating image/file/gallery fields so you reference real assets instead of guessing paths. Filename and mime_type are inferred from the response when not provided."),
		mcp.WithString("url", mcp.Required(), mcp.Description("Absolute http(s) URL of the asset to import")),
		mcp.WithString("filename", mcp.Description("Override filename; defaults to URL basename")),
		mcp.WithString("mime_type", mcp.Description("Override mime; defaults to response Content-Type")),
	), "content", func(ctx context.Context, args map[string]any) (any, error) {
		src := stringArg(args, "url")
		if src == "" {
			return nil, fmt.Errorf("url is required")
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, src, nil)
		if err != nil {
			return nil, fmt.Errorf("build request: %w", err)
		}
		req.Header.Set("User-Agent", "Squilla-MCP/1.0")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("fetch: %w", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, fmt.Errorf("fetch returned status %d", resp.StatusCode)
		}
		buf := new(bytes.Buffer)
		if _, err := buf.ReadFrom(resp.Body); err != nil {
			return nil, fmt.Errorf("read body: %w", err)
		}

		filename := stringArg(args, "filename")
		if filename == "" {
			if idx := strings.Index(src, "?"); idx >= 0 {
				filename = path.Base(src[:idx])
			} else {
				filename = path.Base(src)
			}
			if filename == "" || filename == "." || filename == "/" {
				filename = "import"
			}
		}

		mimeType := stringArg(args, "mime_type")
		if mimeType == "" {
			ct := resp.Header.Get("Content-Type")
			if ct != "" {
				if parsed, _, perr := mime.ParseMediaType(ct); perr == nil {
					mimeType = parsed
				} else {
					mimeType = ct
				}
			}
		}
		if mimeType == "" {
			mimeType = "application/octet-stream"
		}

		// Same proxy strategy as core.media.upload — go through media-manager
		// so the imported URL gets the same normalisation as a manual upload.
		if out, err := s.uploadViaMediaManager(ctx, filename, mimeType, buf.Bytes()); err != nil {
			return nil, err
		} else if out != nil {
			return out, nil
		}
		return api.UploadMedia(ctx, coreapi.MediaUploadRequest{
			Filename: filename,
			MimeType: mimeType,
			Body:     bytes.NewReader(buf.Bytes()),
		})
	})

	s.addTool(mcp.NewTool("core.media.delete",
		mcp.WithDescription("Delete a media file by ID. Does not affect nodes referencing it — they will show broken images."),
		mcp.WithNumber("id", mcp.Required()),
	), "content", func(ctx context.Context, args map[string]any) (any, error) {
		err := api.DeleteMedia(ctx, uintArg(args, "id"))
		return map[string]any{"ok": err == nil}, err
	})
}
