package mcp

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"

	"squilla/internal/coreapi"
	"squilla/internal/uploads"
)

// uploadInitTTL is the lifetime of issued tokens. Short enough that abandoned
// tokens cannot accumulate, long enough for a slow client on a flaky network
// to upload a couple-hundred-MB theme.
const uploadInitTTL = 15 * time.Minute

// uploadBaseURL builds the absolute URL prefix for issued upload URLs.
// Falls back to env so deployments behind a proxy can override (Coolify
// SERVICE_FQDN_*, Cloudflare-fronted hosts, etc.). Empty string yields a
// path-only URL — fine for clients that already know the host.
func (s *Server) uploadBaseURL() string {
	if s.deps.UploadBaseURL != "" {
		return strings.TrimRight(s.deps.UploadBaseURL, "/")
	}
	if v := os.Getenv("SQUILLA_PUBLIC_URL"); v != "" {
		return strings.TrimRight(v, "/")
	}
	return ""
}

// finalizeRow validates the token + sha and returns the row + open temp file.
// Caller must Close() the file and call MarkFinalized on success.
func (s *Server) finalizeRow(token, sha string, kind uploads.Kind) (*uploads.PendingUpload, *os.File, error) {
	if s.deps.UploadStore == nil {
		return nil, nil, fmt.Errorf("upload store not wired")
	}
	row, err := s.deps.UploadStore.Lookup(token)
	if err != nil {
		return nil, nil, err
	}
	if err := s.deps.UploadStore.ValidateForFinalize(row, kind, sha); err != nil {
		return nil, nil, err
	}
	if row.TempPath == nil || *row.TempPath == "" {
		return nil, nil, fmt.Errorf("temp file missing for token")
	}
	f, err := os.Open(*row.TempPath)
	if err != nil {
		return nil, nil, fmt.Errorf("open temp file: %w", err)
	}
	return row, f, nil
}

func (s *Server) registerUploadTools() {
	api := s.deps.CoreAPI
	mgmt := s.deps.ThemeMgmtSvc
	extHandler := s.deps.ExtensionHandler

	// ─── Media ───────────────────────────────────────────────────────────
	s.addTool(mcp.NewTool("core.media.upload_init",
		mcp.WithDescription("Begin a presigned upload for a media file. Returns {upload_url, upload_token, expires_at, max_bytes}. Two-step alternative to core.media.upload — preferred for files >5 MB because the binary travels over a normal HTTP PUT instead of a base64-inflated JSON-RPC envelope.\n\nFlow: call upload_init → PUT bytes to upload_url (no auth header, the token in the URL IS the auth) → call core.media.upload_finalize with the token. Token is single-use, ~15 min TTL."),
		mcp.WithString("filename"),
		mcp.WithString("mime_type"),
	), "content", func(ctx context.Context, args map[string]any) (any, error) {
		var userID int64
		if tok := tokenFromCtx(ctx); tok != nil {
			userID = int64(tok.UserID)
		}
		store := s.deps.UploadStore
		if store == nil {
			return nil, fmt.Errorf("upload store not wired")
		}
		maxBytes := int64(envIntDefault("SQUILLA_MEDIA_MAX_MB", 50)) * 1024 * 1024
		row, err := store.Issue(uploads.IssueOptions{
			Kind:     uploads.KindMedia,
			UserID:   userID,
			Filename: stringArg(args, "filename"),
			MimeType: stringArg(args, "mime_type"),
			MaxBytes: maxBytes,
			TTL:      uploadInitTTL,
		})
		if err != nil {
			return nil, err
		}
		return map[string]any{
			"upload_url":   s.uploadBaseURL() + "/api/uploads/" + row.Token,
			"upload_token": row.Token,
			"expires_at":   row.ExpiresAt.UTC().Format(time.RFC3339),
			"max_bytes":    row.MaxBytes,
			"kind":         row.Kind,
		}, nil
	})

	s.addTool(mcp.NewTool("core.media.upload_finalize",
		mcp.WithDescription("Finalize a presigned media upload. Pass the upload_token from upload_init after the PUT step. Optional sha256 is verified against the value computed by the PUT route — set it when the client wants end-to-end corruption detection. Returns the same shape as core.media.upload (id, url, slug, ...). When the media-manager extension is active the bytes route through its /upload handler, so the optimisation pipeline (WebP, normalisation, original backup) runs identically to manual uploads."),
		mcp.WithString("upload_token", mcp.Required()),
		mcp.WithString("sha256", mcp.Description("Optional. Reject finalize if it doesn't match the hash computed during PUT.")),
	), "content", func(ctx context.Context, args map[string]any) (any, error) {
		token := stringArg(args, "upload_token")
		sha := stringArg(args, "sha256")
		row, f, err := s.finalizeRow(token, sha, uploads.KindMedia)
		if err != nil {
			return nil, err
		}
		defer f.Close()

		filename := row.Filename
		if filename == "" {
			filename = "upload"
		}
		mimeType := row.MimeType
		if mimeType == "" {
			mimeType = "application/octet-stream"
		}

		// Try the media-manager extension path first (admin-UI-equivalent
		// pipeline), fall back to the kernel media service otherwise.
		buf, readErr := readAllFromFile(f)
		if readErr != nil {
			return nil, readErr
		}
		var out any
		if proxied, perr := s.uploadViaMediaManager(ctx, filename, mimeType, buf); perr != nil {
			return nil, perr
		} else if proxied != nil {
			out = proxied
		} else {
			result, kerr := api.UploadMedia(ctx, coreapi.MediaUploadRequest{
				Filename: filename,
				MimeType: mimeType,
				Body:     bytes.NewReader(buf),
			})
			if kerr != nil {
				return nil, kerr
			}
			out = result
		}
		if err := s.deps.UploadStore.MarkFinalized(token); err != nil {
			return nil, err
		}
		return out, nil
	})

	// ─── Theme ───────────────────────────────────────────────────────────
	s.addTool(mcp.NewTool("core.theme.deploy_init",
		mcp.WithDescription("Begin a presigned upload for a theme ZIP. Returns {upload_url, upload_token, expires_at, max_bytes}. Use when the theme zip is bigger than what fits comfortably in a JSON envelope (>5–10 MB). After upload_init: PUT the ZIP body to upload_url, then call core.theme.deploy_finalize with the token + activate flag. Default cap 200 MB — override via SQUILLA_THEME_MAX_MB."),
	), "full", func(ctx context.Context, args map[string]any) (any, error) {
		var userID int64
		if tok := tokenFromCtx(ctx); tok != nil {
			userID = int64(tok.UserID)
		}
		store := s.deps.UploadStore
		if store == nil {
			return nil, fmt.Errorf("upload store not wired")
		}
		maxBytes := int64(envIntDefault("SQUILLA_THEME_MAX_MB", 200)) * 1024 * 1024
		row, err := store.Issue(uploads.IssueOptions{
			Kind:     uploads.KindTheme,
			UserID:   userID,
			MaxBytes: maxBytes,
			TTL:      uploadInitTTL,
		})
		if err != nil {
			return nil, err
		}
		return map[string]any{
			"upload_url":   s.uploadBaseURL() + "/api/uploads/" + row.Token,
			"upload_token": row.Token,
			"expires_at":   row.ExpiresAt.UTC().Format(time.RFC3339),
			"max_bytes":    row.MaxBytes,
			"kind":         row.Kind,
		}, nil
	})

	s.addTool(mcp.NewTool("core.theme.deploy_finalize",
		mcp.WithDescription("Finalize a presigned theme upload. Same effect as core.theme.deploy with body_base64 — unpack into themes/<slug>/, register, optionally activate. Pass upload_token from deploy_init."),
		mcp.WithString("upload_token", mcp.Required()),
		mcp.WithString("sha256", mcp.Description("Optional. Reject finalize if it doesn't match the hash computed during PUT.")),
		mcp.WithBoolean("activate", mcp.Description("If true, activate the theme immediately after install.")),
	), "full", func(ctx context.Context, args map[string]any) (any, error) {
		if mgmt == nil {
			return nil, fmt.Errorf("theme management service not wired")
		}
		token := stringArg(args, "upload_token")
		sha := stringArg(args, "sha256")
		_, f, err := s.finalizeRow(token, sha, uploads.KindTheme)
		if err != nil {
			return nil, err
		}
		theme, instErr := mgmt.InstallFromZip(f, "deploy.zip")
		_ = f.Close()
		if instErr != nil {
			return nil, instErr
		}
		activated := false
		if boolArg(args, "activate") {
			if err := mgmt.Activate(int(theme.ID)); err != nil {
				_ = s.deps.UploadStore.MarkFinalized(token)
				return map[string]any{
					"theme":            theme,
					"activated":        false,
					"activate_error":   err.Error(),
					"restart_required": false,
				}, nil
			}
			activated = true
			if refreshed, ferr := mgmt.GetByID(int(theme.ID)); ferr == nil {
				theme = refreshed
			}
		}
		if err := s.deps.UploadStore.MarkFinalized(token); err != nil {
			return nil, err
		}
		return map[string]any{
			"theme":            theme,
			"activated":        activated,
			"restart_required": false,
		}, nil
	})

	// ─── Extension ──────────────────────────────────────────────────────
	s.addTool(mcp.NewTool("core.extension.deploy_init",
		mcp.WithDescription("Begin a presigned upload for an extension ZIP. Returns {upload_url, upload_token, expires_at, max_bytes}. Use when the archive is bigger than what fits in a JSON envelope. After upload_init: PUT the ZIP body to upload_url, then call core.extension.deploy_finalize with the token + activate flag. Default cap 200 MB — override via SQUILLA_EXTENSION_MAX_MB."),
	), "full", func(ctx context.Context, args map[string]any) (any, error) {
		var userID int64
		if tok := tokenFromCtx(ctx); tok != nil {
			userID = int64(tok.UserID)
		}
		store := s.deps.UploadStore
		if store == nil {
			return nil, fmt.Errorf("upload store not wired")
		}
		maxBytes := int64(envIntDefault("SQUILLA_EXTENSION_MAX_MB", 200)) * 1024 * 1024
		row, err := store.Issue(uploads.IssueOptions{
			Kind:     uploads.KindExtension,
			UserID:   userID,
			MaxBytes: maxBytes,
			TTL:      uploadInitTTL,
		})
		if err != nil {
			return nil, err
		}
		return map[string]any{
			"upload_url":   s.uploadBaseURL() + "/api/uploads/" + row.Token,
			"upload_token": row.Token,
			"expires_at":   row.ExpiresAt.UTC().Format(time.RFC3339),
			"max_bytes":    row.MaxBytes,
			"kind":         row.Kind,
		}, nil
	})

	s.addTool(mcp.NewTool("core.extension.deploy_finalize",
		mcp.WithDescription("Finalize a presigned extension upload. Same effect as core.extension.deploy with body_base64 — unpack into extensions/<slug>/, register, optionally hot-activate. Pass upload_token from deploy_init."),
		mcp.WithString("upload_token", mcp.Required()),
		mcp.WithString("sha256", mcp.Description("Optional. Reject finalize if it doesn't match the hash computed during PUT.")),
		mcp.WithBoolean("activate", mcp.Description("If true, hot-activate the extension immediately after install.")),
	), "full", func(ctx context.Context, args map[string]any) (any, error) {
		if extHandler == nil {
			return nil, fmt.Errorf("extension handler not wired")
		}
		token := stringArg(args, "upload_token")
		sha := stringArg(args, "sha256")
		_, f, err := s.finalizeRow(token, sha, uploads.KindExtension)
		if err != nil {
			return nil, err
		}
		raw, readErr := readAllFromFile(f)
		_ = f.Close()
		if readErr != nil {
			return nil, readErr
		}
		ext, instErr := extHandler.InstallFromZip(raw)
		if instErr != nil {
			return nil, instErr
		}
		activated := false
		if boolArg(args, "activate") {
			if err := extHandler.HotActivate(ext.Slug); err != nil {
				_ = s.deps.UploadStore.MarkFinalized(token)
				return map[string]any{
					"extension":        ext,
					"activated":        false,
					"activate_error":   err.Error(),
					"restart_required": false,
				}, nil
			}
			activated = true
			if refreshed, ferr := s.deps.ExtensionLoader.GetBySlug(ext.Slug); ferr == nil {
				ext = refreshed
			}
		}
		if err := s.deps.UploadStore.MarkFinalized(token); err != nil {
			return nil, err
		}
		return map[string]any{
			"extension":        ext,
			"activated":        activated,
			"restart_required": false,
		}, nil
	})
}

// readAllFromFile reads the entire file into memory. Used by handlers that
// need a []byte (the media-manager multipart proxy and the extension
// installer both want bytes). Theme install streams from the *os.File
// directly, so it doesn't pay this cost.
func readAllFromFile(f *os.File) ([]byte, error) {
	var buf bytes.Buffer
	_, err := buf.ReadFrom(f)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
