package mcp

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
)

// registerDeployTools wires up core.theme.deploy and core.extension.deploy.
//
// These two tools let an AI client ship a packaged theme or extension as a
// base64 ZIP payload, have Squilla unpack it into themes/<slug>/ or
// extensions/<slug>/, register it with the database, and (optionally) flip it
// active in one round-trip — no docker cp, no host filesystem access, no git
// clone required.
//
// The install pipelines are shared with the existing HTTP upload handlers so
// the same safety net (zip-slip, slug validation, atomic dir swap, plugin
// chmod, size cap) applies regardless of who is calling.
func (s *Server) registerDeployTools() {
	mgmt := s.deps.ThemeMgmtSvc
	extHandler := s.deps.ExtensionHandler

	s.addTool(mcp.NewTool("core.theme.deploy",
		mcp.WithDescription("Deploy a theme from a base64-encoded ZIP archive. The archive must contain a theme.json (at root or one level deep) declaring a unique 'slug'. The theme is unpacked into themes/<slug>/ via an atomic directory swap, registered with the database, and — if activate=true — activated immediately (no server restart). Use this to ship themes that live outside the primary git repo (local design handoff → MCP deploy → ready). Max archive size: 50 MB. Slug must match [A-Za-z0-9_-]+.\n\nFor archives >5 MB prefer core.theme.deploy_init + core.theme.deploy_finalize — direct binary PUT, no base64 overhead, 200 MB cap."),
		mcp.WithString("body_base64", mcp.Required(), mcp.Description("Base64-encoded ZIP of the theme directory. theme.json may sit at the archive root or in a single wrapper directory; both layouts are normalised on disk.")),
		mcp.WithBoolean("activate", mcp.Description("If true, activate the theme immediately after install. Default false — register only, leaving the previously active theme in place.")),
	), "full", func(ctx context.Context, args map[string]any) (any, error) {
		if mgmt == nil {
			return nil, fmt.Errorf("theme management service not wired")
		}
		raw, err := base64.StdEncoding.DecodeString(stringArg(args, "body_base64"))
		if err != nil {
			return nil, fmt.Errorf("decode body_base64: %w", err)
		}
		theme, err := mgmt.InstallFromZip(bytes.NewReader(raw), "deploy.zip")
		if err != nil {
			return nil, err
		}
		activated := false
		if boolArg(args, "activate") {
			if err := mgmt.Activate(int(theme.ID)); err != nil {
				return map[string]any{
					"theme":            theme,
					"activated":        false,
					"activate_error":   err.Error(),
					"restart_required": false,
				}, nil
			}
			activated = true
			// Re-fetch to surface the is_active flip in the response.
			if refreshed, ferr := mgmt.GetByID(int(theme.ID)); ferr == nil {
				theme = refreshed
			}
		}
		return map[string]any{
			"theme":            theme,
			"activated":        activated,
			"restart_required": false,
		}, nil
	})

	s.addTool(mcp.NewTool("core.extension.deploy",
		mcp.WithDescription("Deploy an extension from a base64-encoded ZIP archive. The archive must contain an extension.json (at root or one level deep) declaring a unique 'slug'. The extension is unpacked into extensions/<slug>/ via an atomic directory swap, plugin binaries declared in manifest.plugins[].binary are made executable, the extension is registered with the database, and — if activate=true — hot-activated immediately (HotActivate runs migrations, starts plugins, loads scripts/blocks; no server restart). Max archive size: 50 MB. Slug must match [A-Za-z0-9_-]+. NOTE: gRPC plugin binaries must be pre-built for the host's OS/arch — Squilla cannot cross-compile.\n\nFor archives >5 MB prefer core.extension.deploy_init + core.extension.deploy_finalize — direct binary PUT, no base64 overhead, 200 MB cap."),
		mcp.WithString("body_base64", mcp.Required(), mcp.Description("Base64-encoded ZIP of the extension directory. extension.json may sit at the archive root or in a single wrapper directory; both layouts are normalised on disk.")),
		mcp.WithBoolean("activate", mcp.Description("If true, hot-activate the extension immediately after install. Default false — register only, leaving the extension inactive.")),
	), "full", func(ctx context.Context, args map[string]any) (any, error) {
		if extHandler == nil {
			return nil, fmt.Errorf("extension handler not wired")
		}
		raw, err := base64.StdEncoding.DecodeString(stringArg(args, "body_base64"))
		if err != nil {
			return nil, fmt.Errorf("decode body_base64: %w", err)
		}
		ext, err := extHandler.InstallFromZip(raw)
		if err != nil {
			return nil, err
		}
		activated := false
		if boolArg(args, "activate") {
			if err := extHandler.HotActivate(ext.Slug); err != nil {
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
		return map[string]any{
			"extension":        ext,
			"activated":        activated,
			"restart_required": false,
		}, nil
	})

	s.addTool(mcp.NewTool("core.extension.delete",
		mcp.WithDescription("Delete an extension by slug. Wipes the data-dir copy (extensions/<slug> on disk) and removes the database row. Bundled extensions in the read-only image dir are not touched — the next scan re-registers them as fresh inactive entries, so this is effectively 'uninstall the operator override'.\n\nPrecondition: the extension MUST be inactive. Call core.extension.deactivate(slug) first; this tool will not auto-deactivate."),
		mcp.WithString("slug", mcp.Required(), mcp.Description("Extension slug to delete")),
	), "full", func(ctx context.Context, args map[string]any) (any, error) {
		if extHandler == nil {
			return nil, fmt.Errorf("extension handler not wired")
		}
		slug := stringArg(args, "slug")
		if slug == "" {
			return nil, fmt.Errorf("slug is required")
		}
		if err := extHandler.DeleteBySlug(slug); err != nil {
			switch err.Error() {
			case "NOT_FOUND":
				return nil, fmt.Errorf("extension %q not found", slug)
			case "STILL_ACTIVE":
				return nil, fmt.Errorf("extension %q is active — call core.extension.deactivate first", slug)
			default:
				return nil, err
			}
		}
		return map[string]any{"ok": true, "slug": slug}, nil
	})
}
