package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
)

// registerResources exposes CMS entities as MCP resources under the vibecms://
// URI scheme. Clients can list and read these for discovery; mutations always
// go through tools.
func (s *Server) registerResources() {
	api := s.deps.CoreAPI

	// Node resource — dynamic URI template vibecms://nodes/{id}
	s.mcp.AddResourceTemplate(
		mcp.NewResourceTemplate(
			"vibecms://nodes/{id}",
			"VibeCMS node",
			mcp.WithTemplateDescription("A content node (page, post, etc.) by numeric ID. URI form: vibecms://nodes/{id}"),
			mcp.WithTemplateMIMEType("application/json"),
		),
		func(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
			id, err := parseResourceID(req.Params.URI, "vibecms://nodes/")
			if err != nil {
				return nil, err
			}
			node, err := api.GetNode(ctx, id)
			if err != nil {
				return nil, err
			}
			return jsonResource(req.Params.URI, node)
		},
	)

	// Theme resource — vibecms://themes/{slug}
	s.mcp.AddResourceTemplate(
		mcp.NewResourceTemplate(
			"vibecms://themes/{slug}",
			"VibeCMS theme",
			mcp.WithTemplateDescription("A theme by slug."),
			mcp.WithTemplateMIMEType("application/json"),
		),
		func(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
			if s.deps.ThemeMgmtSvc == nil {
				return nil, fmt.Errorf("theme management service not wired")
			}
			slug := strings.TrimPrefix(req.Params.URI, "vibecms://themes/")
			themes, err := s.deps.ThemeMgmtSvc.List()
			if err != nil {
				return nil, err
			}
			for _, t := range themes {
				if t.Slug == slug {
					return jsonResource(req.Params.URI, t)
				}
			}
			return nil, fmt.Errorf("theme %q not found", slug)
		},
	)

	// Extension resource — vibecms://extensions/{slug}
	s.mcp.AddResourceTemplate(
		mcp.NewResourceTemplate(
			"vibecms://extensions/{slug}",
			"VibeCMS extension",
			mcp.WithTemplateDescription("An extension by slug."),
			mcp.WithTemplateMIMEType("application/json"),
		),
		func(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
			if s.deps.ExtensionLoader == nil {
				return nil, fmt.Errorf("extension loader not wired")
			}
			slug := strings.TrimPrefix(req.Params.URI, "vibecms://extensions/")
			ext, err := s.deps.ExtensionLoader.GetBySlug(slug)
			if err != nil {
				return nil, err
			}
			return jsonResource(req.Params.URI, ext)
		},
	)

	// Theme Guidelines resource — vibecms://guidelines/themes
	s.mcp.AddResource(
		mcp.NewResource(
			"vibecms://guidelines/themes",
			"Theme Development Standards",
			mcp.WithResourceDescription("Official VibeCMS theme development guidelines (Rules 1.1 - 1.6)."),
		),
		func(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
			// Return both structured JSON and raw Markdown
			jsonContent, err := json.Marshal(themeStandards())
			if err != nil {
				return nil, err
			}

			// Try to read the README.md for full context
			readmePath := "themes/README.md"
			readmeContent, _ := os.ReadFile(readmePath) // Ignore error, we'll just return JSON if missing

			contents := []mcp.ResourceContents{
				mcp.TextResourceContents{
					URI:      req.Params.URI,
					MIMEType: "application/json",
					Text:     string(jsonContent),
				},
			}

			if len(readmeContent) > 0 {
				contents = append(contents, mcp.TextResourceContents{
					URI:      req.Params.URI + "#markdown",
					MIMEType: "text/markdown",
					Text:     string(readmeContent),
				})
			}

			return contents, nil
		},
	)

	// Extension Guidelines resource — vibecms://guidelines/extensions
	s.mcp.AddResource(
		mcp.NewResource(
			"vibecms://guidelines/extensions",
			"Extension Development Standards",
			mcp.WithResourceDescription("Official VibeCMS extension development guidelines (manifest, capabilities, gRPC plugin lifecycle, admin-UI micro-frontend, list-page primitives, lifecycle events)."),
		),
		func(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
			jsonContent, err := json.Marshal(extensionStandards())
			if err != nil {
				return nil, err
			}
			readmeContent, _ := os.ReadFile("extensions/README.md")
			contents := []mcp.ResourceContents{
				mcp.TextResourceContents{
					URI:      req.Params.URI,
					MIMEType: "application/json",
					Text:     string(jsonContent),
				},
			}
			if len(readmeContent) > 0 {
				contents = append(contents, mcp.TextResourceContents{
					URI:      req.Params.URI + "#markdown",
					MIMEType: "text/markdown",
					Text:     string(readmeContent),
				})
			}
			return contents, nil
		},
	)

	// AI Onboarding resource — vibecms://guidelines/onboarding
	s.mcp.AddResource(
		mcp.NewResource(
			"vibecms://guidelines/onboarding",
			"AI Agent Onboarding Guide",
			mcp.WithResourceDescription("Mandatory protocol for AI agents building VibeCMS themes."),
		),
		func(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
			return []mcp.ResourceContents{
				mcp.TextResourceContents{
					URI:      req.Params.URI,
					MIMEType: "text/markdown",
					Text:     onboardingGuide(),
				},
			}, nil
		},
	)
}

func parseResourceID(uri, prefix string) (uint, error) {
	rest := strings.TrimPrefix(uri, prefix)
	id, err := strconv.ParseUint(rest, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid id in URI %q: %w", uri, err)
	}
	return uint(id), nil
}

func jsonResource(uri string, v any) ([]mcp.ResourceContents, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			URI:      uri,
			MIMEType: "application/json",
			Text:     string(b),
		},
	}, nil
}
