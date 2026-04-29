package mcp

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"

	"squilla/internal/models"
)

func (s *Server) registerSystemTools() {
	s.registerThemeTools()
	s.registerExtensionTools()
	s.registerBlockTypeTools()
	s.registerLayoutTools()
}

func (s *Server) registerThemeTools() {
	mgmt := s.deps.ThemeMgmtSvc

	s.addTool(mcp.NewTool("core.theme.list",
		mcp.WithDescription("List all installed themes with their metadata (name, version, is_active, etc.)."),
	), "read", func(ctx context.Context, args map[string]any) (any, error) {
		if mgmt == nil {
			return nil, fmt.Errorf("theme management service not wired")
		}
		return mgmt.List()
	})

	s.addTool(mcp.NewTool("core.theme.active",
		mcp.WithDescription("Return the currently active theme, or an error if no theme is active."),
	), "read", func(ctx context.Context, args map[string]any) (any, error) {
		if mgmt == nil {
			return nil, fmt.Errorf("theme management service not wired")
		}
		return mgmt.GetActive()
	})

	s.addTool(mcp.NewTool("core.theme.get",
		mcp.WithDescription("Fetch a single theme by ID."),
		mcp.WithNumber("id", mcp.Required()),
	), "read", func(ctx context.Context, args map[string]any) (any, error) {
		if mgmt == nil {
			return nil, fmt.Errorf("theme management service not wired")
		}
		return mgmt.GetByID(intArg(args, "id"))
	})

	s.addTool(mcp.NewTool("core.theme.activate",
		mcp.WithDescription("Activate a theme by ID. Emits theme.activated; layouts and blocks from the theme become live. Theme activation does NOT require a server restart. The response embeds a summary of core.theme.checklist for the activated theme — when checklist.failed > 0 you MUST call core.theme.checklist for details and address the failures BEFORE declaring the work done."),
		mcp.WithNumber("id", mcp.Required()),
	), "full", func(ctx context.Context, args map[string]any) (any, error) {
		if mgmt == nil {
			return nil, fmt.Errorf("theme management service not wired")
		}
		id := intArg(args, "id")
		err := mgmt.Activate(id)
		out := map[string]any{"ok": err == nil, "restart_required": false}
		if cl := summarizeActivatedThemeChecklist(mgmt, id); cl != nil {
			out["checklist"] = cl
			if failed, _ := cl["failed"].(int); failed > 0 {
				out["next_step"] = "core.theme.checklist returned failures — call it for the full list and fix every fail before claiming done. Hardcoded fallbacks and missing test_data make screenshots fake-pass."
			}
		}
		return out, err
	})

	s.addTool(mcp.NewTool("core.theme.deactivate",
		mcp.WithDescription("Deactivate a theme by ID. The site falls back to whatever layouts remain registered."),
		mcp.WithNumber("id", mcp.Required()),
	), "full", func(ctx context.Context, args map[string]any) (any, error) {
		if mgmt == nil {
			return nil, fmt.Errorf("theme management service not wired")
		}
		err := mgmt.Deactivate(intArg(args, "id"))
		return map[string]any{"ok": err == nil, "restart_required": false}, err
	})

	s.addTool(mcp.NewTool("core.theme.rescan",
		mcp.WithDescription("Re-scan the themes/ directory and upsert a Theme row for every subdirectory containing a valid theme.json. Idempotent. Use after dropping a theme onto the filesystem out-of-band (docker cp, volume mount); the runtime fs watcher already does this automatically, this tool is the explicit trigger for ops scripts and CI. Does not activate anything."),
	), "full", func(ctx context.Context, args map[string]any) (any, error) {
		if mgmt == nil {
			return nil, fmt.Errorf("theme management service not wired")
		}
		mgmt.ScanAndRegister()
		return map[string]any{"ok": true}, nil
	})
}

func (s *Server) registerExtensionTools() {
	loader := s.deps.ExtensionLoader

	s.addTool(mcp.NewTool("core.extension.list",
		mcp.WithDescription("List all installed extensions (active and inactive)."),
	), "read", func(ctx context.Context, args map[string]any) (any, error) {
		if loader == nil {
			return nil, fmt.Errorf("extension loader not wired")
		}
		return loader.List()
	})

	s.addTool(mcp.NewTool("core.extension.get",
		mcp.WithDescription("Fetch an extension by slug."),
		mcp.WithString("slug", mcp.Required()),
	), "read", func(ctx context.Context, args map[string]any) (any, error) {
		if loader == nil {
			return nil, fmt.Errorf("extension loader not wired")
		}
		return loader.GetBySlug(stringArg(args, "slug"))
	})

	s.addTool(mcp.NewTool("core.extension.activate",
		mcp.WithDescription("Activate an extension by slug. Runs pending migrations, hot-loads scripts, starts gRPC plugins, loads block types, and fires extension.activated. Takes effect immediately — no server restart required."),
		mcp.WithString("slug", mcp.Required()),
	), "full", func(ctx context.Context, args map[string]any) (any, error) {
		if s.deps.ExtensionHandler == nil {
			return nil, fmt.Errorf("extension handler not wired")
		}
		if err := s.deps.ExtensionHandler.HotActivate(stringArg(args, "slug")); err != nil {
			return nil, err
		}
		return map[string]any{"ok": true, "restart_required": false}, nil
	})

	s.addTool(mcp.NewTool("core.extension.deactivate",
		mcp.WithDescription("Deactivate an extension by slug. Fires extension.deactivated, hot-unloads scripts, stops gRPC plugins, and unloads block types. Takes effect immediately — no server restart required."),
		mcp.WithString("slug", mcp.Required()),
	), "full", func(ctx context.Context, args map[string]any) (any, error) {
		if s.deps.ExtensionHandler == nil {
			return nil, fmt.Errorf("extension handler not wired")
		}
		if err := s.deps.ExtensionHandler.HotDeactivate(stringArg(args, "slug")); err != nil {
			return nil, err
		}
		return map[string]any{"ok": true, "restart_required": false}, nil
	})

	s.addTool(mcp.NewTool("core.extension.rescan",
		mcp.WithDescription("Re-scan the extensions/ directory and upsert an Extension row for every subdirectory containing a valid extension.json. New extensions default to is_active=false; call core.extension.activate to start them. Idempotent. The runtime fs watcher already triggers this automatically when a directory or manifest appears — use this tool as an explicit trigger for ops scripts/CI."),
	), "full", func(ctx context.Context, args map[string]any) (any, error) {
		if loader == nil {
			return nil, fmt.Errorf("extension loader not wired")
		}
		loader.ScanAndRegister()
		return map[string]any{"ok": true}, nil
	})
}

func (s *Server) registerBlockTypeTools() {
	svc := s.deps.BlockTypeSvc

	s.addTool(mcp.NewTool("core.block_types.list",
		mcp.WithDescription("List all registered block types. Each includes slug, label, field_schema, and rendered template source — useful before calling core.render.block."),
	), "read", func(ctx context.Context, args map[string]any) (any, error) {
		if svc == nil {
			return nil, fmt.Errorf("block type service not wired")
		}
		return svc.ListAll()
	})

	s.addTool(mcp.NewTool("core.block_types.get",
		mcp.WithDescription("Get one block type by slug."),
		mcp.WithString("slug", mcp.Required()),
	), "read", func(ctx context.Context, args map[string]any) (any, error) {
		if svc == nil {
			return nil, fmt.Errorf("block type service not wired")
		}
		return svc.GetBySlug(stringArg(args, "slug"))
	})

	s.addTool(mcp.NewTool("core.block_types.create",
		mcp.WithDescription("Create a new block type (custom reusable content block). field_schema is an array of field defs; html_template is the Go html/template source rendered with .fields and .node. source='custom' for user-defined blocks."),
		mcp.WithString("slug", mcp.Required(), mcp.Description("Unique snake_case slug, e.g. 'image_carousel_cta'")),
		mcp.WithString("label", mcp.Required()),
		mcp.WithString("icon", mcp.Description("Lucide icon name, default 'square'")),
		mcp.WithString("description"),
		mcp.WithArray("field_schema", mcp.Description("Array of field definitions [{type, name, label, ...}]")),
		mcp.WithString("html_template", mcp.Description("Go html/template source")),
		mcp.WithObject("test_data", mcp.Description("Example fields payload for preview rendering")),
		mcp.WithString("block_css"),
		mcp.WithString("block_js"),
		mcp.WithBoolean("cache_output", mcp.Description("Default true")),
	), "full", func(ctx context.Context, args map[string]any) (any, error) {
		if svc == nil {
			return nil, fmt.Errorf("block type service not wired")
		}
		bt := &models.BlockType{
			Slug:         stringArg(args, "slug"),
			Label:        stringArg(args, "label"),
			Icon:         stringArg(args, "icon"),
			Description:  stringArg(args, "description"),
			HTMLTemplate: stringArg(args, "html_template"),
			BlockCSS:     stringArg(args, "block_css"),
			BlockJS:      stringArg(args, "block_js"),
			Source:       "custom",
			CacheOutput:  true,
		}
		if bt.Icon == "" {
			bt.Icon = "square"
		}
		if _, ok := args["cache_output"]; ok {
			bt.CacheOutput = boolArg(args, "cache_output")
		}
		if v, ok := args["field_schema"]; ok {
			bt.FieldSchema = models.JSONB(jsonFieldBytes(v, "[]"))
		} else {
			bt.FieldSchema = models.JSONB("[]")
		}
		if v, ok := args["test_data"]; ok {
			bt.TestData = models.JSONB(jsonFieldBytes(v, "{}"))
		} else {
			bt.TestData = models.JSONB("{}")
		}
		if err := svc.Create(bt); err != nil {
			return nil, err
		}
		return bt, nil
	})

	s.addTool(mcp.NewTool("core.block_types.update",
		mcp.WithDescription("Update an existing block type. Provide only fields to change. For theme-sourced blocks, consider core.block_types.detach first."),
		mcp.WithNumber("id", mcp.Required()),
		mcp.WithString("slug"),
		mcp.WithString("label"),
		mcp.WithString("icon"),
		mcp.WithString("description"),
		mcp.WithArray("field_schema"),
		mcp.WithString("html_template"),
		mcp.WithObject("test_data"),
		mcp.WithString("block_css"),
		mcp.WithString("block_js"),
		mcp.WithBoolean("cache_output"),
	), "full", func(ctx context.Context, args map[string]any) (any, error) {
		if svc == nil {
			return nil, fmt.Errorf("block type service not wired")
		}
		updates := map[string]any{}
		for _, k := range []string{"slug", "label", "icon", "description", "html_template", "block_css", "block_js"} {
			if v, ok := args[k]; ok {
				if s, ok := v.(string); ok {
					updates[k] = s
				}
			}
		}
		if v, ok := args["field_schema"]; ok {
			updates["field_schema"] = models.JSONB(jsonFieldBytes(v, "[]"))
		}
		if v, ok := args["test_data"]; ok {
			updates["test_data"] = models.JSONB(jsonFieldBytes(v, "{}"))
		}
		if _, ok := args["cache_output"]; ok {
			updates["cache_output"] = boolArg(args, "cache_output")
		}
		return svc.Update(intArg(args, "id"), updates)
	})

	s.addTool(mcp.NewTool("core.block_types.delete",
		mcp.WithDescription("Delete a block type by ID. Theme-sourced blocks cannot be deleted directly — detach first."),
		mcp.WithNumber("id", mcp.Required()),
	), "full", func(ctx context.Context, args map[string]any) (any, error) {
		if svc == nil {
			return nil, fmt.Errorf("block type service not wired")
		}
		if err := svc.Delete(intArg(args, "id")); err != nil {
			return nil, err
		}
		return map[string]any{"ok": true}, nil
	})

	s.addTool(mcp.NewTool("core.block_types.detach",
		mcp.WithDescription("Detach a theme-sourced block type so it can be edited/deleted as a custom block."),
		mcp.WithNumber("id", mcp.Required()),
	), "full", func(ctx context.Context, args map[string]any) (any, error) {
		if svc == nil {
			return nil, fmt.Errorf("block type service not wired")
		}
		return svc.Detach(intArg(args, "id"))
	})

	s.addTool(mcp.NewTool("core.block_types.reattach",
		mcp.WithDescription("Reattach a previously-detached block type to its theme (discards custom edits)."),
		mcp.WithNumber("id", mcp.Required()),
	), "full", func(ctx context.Context, args map[string]any) (any, error) {
		if svc == nil {
			return nil, fmt.Errorf("block type service not wired")
		}
		return svc.Reattach(intArg(args, "id"))
	})
}

func (s *Server) registerLayoutTools() {
	svc := s.deps.LayoutSvc

	s.addTool(mcp.NewTool("core.layout.list",
		mcp.WithDescription("List registered page layouts. Use before core.render.layout to find a valid layout_slug."),
		mcp.WithString("source", mcp.Description("Optional filter: 'theme' | 'extension' | 'user'")),
	), "read", func(ctx context.Context, args map[string]any) (any, error) {
		if svc == nil {
			return nil, fmt.Errorf("layout service not wired")
		}
		rows, _, err := svc.List(nil, stringArg(args, "source"), 1, 200)
		return rows, err
	})

	s.addTool(mcp.NewTool("core.layout.get",
		mcp.WithDescription("Fetch a single layout by ID with its full template_code."),
		mcp.WithNumber("id", mcp.Required()),
	), "read", func(ctx context.Context, args map[string]any) (any, error) {
		if svc == nil {
			return nil, fmt.Errorf("layout service not wired")
		}
		return svc.GetByID(intArg(args, "id"))
	})

	s.addTool(mcp.NewTool("core.layout.create",
		mcp.WithDescription("Create a new page layout. template_code is the Go html/template source with {{block \"content\" .}} etc."),
		mcp.WithString("slug", mcp.Required()),
		mcp.WithString("name", mcp.Required()),
		mcp.WithString("description"),
		mcp.WithString("template_code", mcp.Required()),
		mcp.WithNumber("language_id", mcp.Description("Optional language scoping")),
		mcp.WithBoolean("is_default"),
	), "full", func(ctx context.Context, args map[string]any) (any, error) {
		if svc == nil {
			return nil, fmt.Errorf("layout service not wired")
		}
		l := &models.Layout{
			Slug:         stringArg(args, "slug"),
			Name:         stringArg(args, "name"),
			Description:  stringArg(args, "description"),
			TemplateCode: stringArg(args, "template_code"),
			Source:       "custom",
			IsDefault:    boolArg(args, "is_default"),
		}
		if v := intArg(args, "language_id"); v != 0 {
			l.LanguageID = &v
		}
		if err := svc.Create(l); err != nil {
			return nil, err
		}
		return l, nil
	})

	s.addTool(mcp.NewTool("core.layout.update",
		mcp.WithDescription("Update an existing layout. Theme-sourced layouts must be detached first."),
		mcp.WithNumber("id", mcp.Required()),
		mcp.WithString("slug"),
		mcp.WithString("name"),
		mcp.WithString("description"),
		mcp.WithString("template_code"),
		mcp.WithBoolean("is_default"),
	), "full", func(ctx context.Context, args map[string]any) (any, error) {
		if svc == nil {
			return nil, fmt.Errorf("layout service not wired")
		}
		updates := map[string]any{}
		for _, k := range []string{"slug", "name", "description", "template_code"} {
			if v, ok := args[k]; ok {
				if s, ok := v.(string); ok {
					updates[k] = s
				}
			}
		}
		if _, ok := args["is_default"]; ok {
			updates["is_default"] = boolArg(args, "is_default")
		}
		return svc.Update(intArg(args, "id"), updates)
	})

	s.addTool(mcp.NewTool("core.layout.delete",
		mcp.WithDescription("Delete a layout by ID. Theme-sourced layouts must be detached first."),
		mcp.WithNumber("id", mcp.Required()),
	), "full", func(ctx context.Context, args map[string]any) (any, error) {
		if svc == nil {
			return nil, fmt.Errorf("layout service not wired")
		}
		if err := svc.Delete(intArg(args, "id")); err != nil {
			return nil, err
		}
		return map[string]any{"ok": true}, nil
	})

	s.addTool(mcp.NewTool("core.layout.detach",
		mcp.WithDescription("Detach a theme-sourced layout so it can be edited/deleted as custom."),
		mcp.WithNumber("id", mcp.Required()),
	), "full", func(ctx context.Context, args map[string]any) (any, error) {
		if svc == nil {
			return nil, fmt.Errorf("layout service not wired")
		}
		return svc.Detach(intArg(args, "id"))
	})

	s.addTool(mcp.NewTool("core.layout.reattach",
		mcp.WithDescription("Reattach a previously-detached layout (discards custom edits)."),
		mcp.WithNumber("id", mcp.Required()),
	), "full", func(ctx context.Context, args map[string]any) (any, error) {
		if svc == nil {
			return nil, fmt.Errorf("layout service not wired")
		}
		return svc.Reattach(intArg(args, "id"))
	})
}
