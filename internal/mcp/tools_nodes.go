package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"

	"squilla/internal/coreapi"
)

func (s *Server) registerNodeTools() {
	api := s.deps.CoreAPI

	s.addTool(mcp.NewTool("core.node.get",
		mcp.WithDescription("Fetch ONE content node by numeric ID. Returns full node with blocks_data, fields_data, taxonomies, seo_settings, translations.\n\nUse when: you already have the ID and need the full record.\nDO NOT use when: searching by slug/title/type — use core.node.query. Reading a node type schema — use core.nodetype.get. Previewing rendered HTML — use core.render.node_preview."),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("Node ID")),
	), "read", func(ctx context.Context, args map[string]any) (any, error) {
		return api.GetNode(ctx, uintArg(args, "id"))
	})

	s.addTool(mcp.NewTool("core.node.query",
		mcp.WithDescription("Search/list content nodes with filters. Returns {nodes, total}. Always paginate: default limit 25, max 200."),
		mcp.WithString("node_type", mcp.Description("Filter by node type slug (e.g. 'blog_post')")),
		mcp.WithString("status", mcp.Description("Filter by status: 'draft' | 'published'")),
		mcp.WithString("language_code", mcp.Description("Filter by language (e.g. 'en')")),
		mcp.WithString("slug"),
		mcp.WithString("search", mcp.Description("Full-text search across title/content")),
		mcp.WithString("order_by", mcp.Description("e.g. 'created_at DESC'")),
		mcp.WithNumber("limit", mcp.Description("Default 25, max 200")),
		mcp.WithNumber("offset"),
	), "read", func(ctx context.Context, args map[string]any) (any, error) {
		q := coreapi.NodeQuery{
			NodeType:     stringArg(args, "node_type"),
			Status:       stringArg(args, "status"),
			LanguageCode: stringArg(args, "language_code"),
			Slug:         stringArg(args, "slug"),
			Search:       stringArg(args, "search"),
			OrderBy:      stringArg(args, "order_by"),
			Limit:        clampLimit(intArg(args, "limit")),
			Offset:       intArg(args, "offset"),
		}
		return api.QueryNodes(ctx, q)
	})

	s.addTool(mcp.NewTool("core.node.create",
		mcp.WithDescription("Create a new content node (an instance of a node type — a page, post, trip, etc.).\n\nUse when: you're authoring actual content.\nDO NOT use when: defining a NEW post type — use core.nodetype.create. Uploading a file — use core.media.upload first, then reference the returned media object here as featured_image.\n\nRequired: node_type, language_code, title, status.\nShapes: blocks_data=[{type,fields},...]; fields_data={<field_key>:<value>,...}; featured_image is an object {url,alt,...}, never a bare string."),
		mcp.WithString("node_type", mcp.Required()),
		mcp.WithString("language_code", mcp.Required(), mcp.Description("e.g. 'en'")),
		mcp.WithString("title", mcp.Required()),
		mcp.WithString("slug", mcp.Description("Auto-generated if omitted")),
		mcp.WithString("status", mcp.DefaultString("draft"), mcp.Enum("draft", "published")),
		mcp.WithString("excerpt"),
		mcp.WithString("layout_slug", mcp.Description("Theme layout slug to render this node with (e.g. 'docs', 'default'). Omit to use the active theme's default layout. Discoverable via core.layout.list. NOTE: node-type-specific defaults are not auto-applied — set explicitly when authoring sections that need a non-default layout (docs, landing pages, etc.).")),
		mcp.WithArray("blocks_data", mcp.Description("Array of {type, fields} blocks")),
		mcp.WithObject("fields_data"),
		mcp.WithObject("seo_settings"),
		mcp.WithObject("featured_image"),
	), "content", func(ctx context.Context, args map[string]any) (any, error) {
		input := nodeInputFromArgs(args)
		node, err := api.CreateNode(ctx, input)
		if err != nil {
			return nil, err
		}
		return wrapNodeResult(node, input), nil
	})

	s.addTool(mcp.NewTool("core.node.update",
		mcp.WithDescription("Update an existing node by ID. Provide only the fields you want to change; omitted fields keep their current values."),
		mcp.WithNumber("id", mcp.Required()),
		mcp.WithString("title"),
		mcp.WithString("slug"),
		mcp.WithString("status", mcp.Enum("draft", "published")),
		mcp.WithString("excerpt"),
		mcp.WithString("layout_slug", mcp.Description("Theme layout slug. Pass empty string to leave unchanged; pass a real slug to switch layouts.")),
		mcp.WithArray("blocks_data", mcp.Description("Array of {type, fields} blocks")),
		mcp.WithObject("fields_data"),
		mcp.WithObject("seo_settings"),
		mcp.WithObject("featured_image"),
	), "content", func(ctx context.Context, args map[string]any) (any, error) {
		id := uintArg(args, "id")
		if id == 0 {
			return nil, fmt.Errorf("id is required")
		}
		input := nodeInputFromArgs(args)
		node, err := api.UpdateNode(ctx, id, input)
		if err != nil {
			return nil, err
		}
		return wrapNodeResult(node, input), nil
	})

	s.addTool(mcp.NewTool("core.node.delete",
		mcp.WithDescription("Permanently delete a node by ID. Use core.node.update with status='draft' if you want to unpublish without deleting."),
		mcp.WithNumber("id", mcp.Required()),
	), "content", func(ctx context.Context, args map[string]any) (any, error) {
		id := uintArg(args, "id")
		if err := api.DeleteNode(ctx, id); err != nil {
			return nil, err
		}
		return map[string]any{"ok": true, "id": id}, nil
	})
}

func nodeInputFromArgs(args map[string]any) coreapi.NodeInput {
	input := coreapi.NodeInput{
		NodeType:     stringArg(args, "node_type"),
		LanguageCode: stringArg(args, "language_code"),
		Slug:         stringArg(args, "slug"),
		Status:       stringArg(args, "status"),
		Title:        stringArg(args, "title"),
		Excerpt:      stringArg(args, "excerpt"),
		LayoutSlug:   stringArg(args, "layout_slug"),
	}
	// Structured fields: accept either a decoded value (array/object) or a
	// JSON-encoded string (some MCP clients stringify nested JSON).
	if v, ok := args["blocks_data"]; ok {
		input.BlocksData = jsonFieldDecode(v)
	}
	if v, ok := args["featured_image"]; ok {
		input.FeaturedImage = jsonFieldDecode(v)
	}
	if v, ok := args["fields_data"]; ok {
		if m, okm := jsonFieldDecode(v).(map[string]any); okm && len(m) > 0 {
			input.FieldsData = m
		}
	}
	if v, ok := args["seo_settings"]; ok {
		if m, okm := jsonFieldDecode(v).(map[string]any); okm {
			seo := make(map[string]string, len(m))
			for k, vv := range m {
				if s, okS := vv.(string); okS {
					seo[k] = s
				}
			}
			input.SeoSettings = seo
		}
	}
	if raw, ok := args["taxonomies"]; ok {
		b, _ := json.Marshal(jsonFieldDecode(raw))
		var tax map[string][]string
		_ = json.Unmarshal(b, &tax)
		input.Taxonomies = tax
	}
	return input
}

// jsonFieldBytes returns a canonical JSON byte slice for a value that may come
// in as a decoded array/object, a JSON-encoded string, or be missing. Falls
// back to fallback when the value cannot be encoded.
func jsonFieldBytes(v any, fallback string) []byte {
	decoded := jsonFieldDecode(v)
	b, err := json.Marshal(decoded)
	if err != nil || len(b) == 0 {
		return []byte(fallback)
	}
	return b
}

// nodeWriteResult embeds a Node so all node fields stay top-level (existing
// callers reading result.id / result.full_url keep working) while also
// surfacing AI-authoring hints — preview tooling and soft validation
// warnings — that would otherwise require a follow-up round trip.
type nodeWriteResult struct {
	*coreapi.Node
	PreviewTool string              `json:"preview_tool,omitempty"`
	Warnings    []map[string]string `json:"warnings,omitempty"`
}

// wrapNodeResult composes a node-write response with preview hint + soft
// validation warnings. Warnings are advisory (no field is rejected); they
// surface common authoring oversights flagged in the editing playbook
// (missing layout for non-default sections, empty or over-long SEO meta).
func wrapNodeResult(node *coreapi.Node, input coreapi.NodeInput) *nodeWriteResult {
	out := &nodeWriteResult{Node: node}
	if node != nil {
		out.PreviewTool = fmt.Sprintf("core.render.node_preview(id=%d) — returns rendered HTML; or open %s in a browser once status='published'", node.ID, node.FullURL)
	}
	out.Warnings = nodeAuthoringWarnings(node, input)
	return out
}

// nodeAuthoringWarnings emits soft validation hints. Returns nil when the
// node looks clean. Length thresholds match common SEO guidance (Yoast):
// meta_title up to 60 chars, meta_description up to 160 chars.
func nodeAuthoringWarnings(node *coreapi.Node, input coreapi.NodeInput) []map[string]string {
	var w []map[string]string
	add := func(field, level, msg string) {
		w = append(w, map[string]string{"field": field, "level": level, "message": msg})
	}

	seo := input.SeoSettings
	if seo == nil && node != nil {
		seo = node.SeoSettings
	}

	title := seo["meta_title"]
	desc := seo["meta_description"]

	if strings.TrimSpace(title) == "" {
		add("seo_settings.meta_title", "info", "missing — search/social cards will fall back to node title. Set explicitly for better SEO.")
	} else if n := len(title); n > 60 {
		add("seo_settings.meta_title", "warn", fmt.Sprintf("length %d exceeds the 60-char recommendation; engines may truncate.", n))
	}

	if strings.TrimSpace(desc) == "" {
		add("seo_settings.meta_description", "info", "missing — page ships with empty meta description. Set explicitly or it will fall back to excerpt/empty.")
	} else if n := len(desc); n > 160 {
		add("seo_settings.meta_description", "warn", fmt.Sprintf("length %d exceeds the 160-char recommendation; engines may truncate.", n))
	}

	if node != nil && strings.TrimSpace(node.Excerpt) == "" {
		add("excerpt", "info", "missing — list/index views will show no summary. Consider deriving from the first paragraph.")
	}

	return w
}

// jsonFieldDecode unwraps a JSON-encoded string back into its decoded value.
// Pass-through for any non-string input. Used because some MCP clients
// stringify nested objects/arrays when the schema type is object/array.
func jsonFieldDecode(v any) any {
	s, ok := v.(string)
	if !ok {
		return v
	}
	trimmed := strings.TrimSpace(s)
	if trimmed == "" {
		return v
	}
	if trimmed[0] != '{' && trimmed[0] != '[' {
		return v
	}
	var out any
	if err := json.Unmarshal([]byte(trimmed), &out); err != nil {
		return v
	}
	return out
}
