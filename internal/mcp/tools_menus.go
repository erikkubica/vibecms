package mcp

import (
	"context"
	"encoding/json"

	"github.com/mark3labs/mcp-go/mcp"

	"vibecms/internal/coreapi"
)

func (s *Server) registerMenuTools() {
	api := s.deps.CoreAPI

	s.addTool(mcp.NewTool("core.menu.list",
		mcp.WithDescription("List all menus with their items."),
	), "read", func(ctx context.Context, args map[string]any) (any, error) {
		return api.GetMenus(ctx)
	})

	s.addTool(mcp.NewTool("core.menu.get",
		mcp.WithDescription("Fetch a menu by slug."),
		mcp.WithString("slug", mcp.Required()),
	), "read", func(ctx context.Context, args map[string]any) (any, error) {
		return api.GetMenu(ctx, stringArg(args, "slug"))
	})

	s.addTool(mcp.NewTool("core.menu.create",
		mcp.WithDescription("Create a new menu. items is an array of {label,url,target?,parent_id?,position?,children?}."),
		mcp.WithString("name", mcp.Required()),
		mcp.WithString("slug"),
		mcp.WithArray("items"),
	), "content", func(ctx context.Context, args map[string]any) (any, error) {
		return api.CreateMenu(ctx, menuInputFromArgs(args))
	})

	s.addTool(mcp.NewTool("core.menu.update",
		mcp.WithDescription("Update a menu by slug."),
		mcp.WithString("slug", mcp.Required()),
		mcp.WithString("name"),
		mcp.WithArray("items"),
	), "content", func(ctx context.Context, args map[string]any) (any, error) {
		return api.UpdateMenu(ctx, stringArg(args, "slug"), menuInputFromArgs(args))
	})

	s.addTool(mcp.NewTool("core.menu.delete",
		mcp.WithDescription("Delete a menu by slug."),
		mcp.WithString("slug", mcp.Required()),
	), "content", func(ctx context.Context, args map[string]any) (any, error) {
		err := api.DeleteMenu(ctx, stringArg(args, "slug"))
		return map[string]any{"ok": err == nil}, err
	})

	s.addTool(mcp.NewTool("core.menu.upsert",
		mcp.WithDescription("Create-or-replace a menu by slug. Items support {label, url, target?, children?} for static links and {label, page:'<node-slug>', target?, children?} for node-linked items — page slugs resolve to NodeIDs at upsert so renaming the target page doesn't break the menu. Existing menu's items are fully replaced (this matches the Tengo menus.upsert semantics from core/menus)."),
		mcp.WithString("name", mcp.Required()),
		mcp.WithString("slug"),
		mcp.WithArray("items"),
	), "content", func(ctx context.Context, args map[string]any) (any, error) {
		input, err := menuUpsertInputFromArgs(ctx, api, args)
		if err != nil {
			return nil, err
		}
		return api.UpsertMenu(ctx, input)
	})
}

// menuUpsertInputFromArgs builds a MenuInput from the loose JSON the MCP
// client sent, resolving any {page:"<slug>"} entries to {item_type:"node",
// node_id:<id>} so the menu renderer derives the URL from the node's
// current full_url at render time. Unresolvable slugs are dropped from the
// tree with the failure surfaced in the response so the caller can react.
func menuUpsertInputFromArgs(ctx context.Context, api coreapi.CoreAPI, args map[string]any) (coreapi.MenuInput, error) {
	in := coreapi.MenuInput{
		Name: stringArg(args, "name"),
		Slug: stringArg(args, "slug"),
	}
	rawItems, ok := args["items"]
	if !ok {
		return in, nil
	}
	b, err := json.Marshal(rawItems)
	if err != nil {
		return in, err
	}
	var loose []menuUpsertItem
	if uerr := json.Unmarshal(b, &loose); uerr != nil {
		return in, uerr
	}
	resolved, rerr := resolveMenuUpsertItems(ctx, api, loose)
	if rerr != nil {
		return in, rerr
	}
	in.Items = resolved
	return in, nil
}

type menuUpsertItem struct {
	Label    string           `json:"label"`
	URL      string           `json:"url,omitempty"`
	Target   string           `json:"target,omitempty"`
	Page     string           `json:"page,omitempty"`
	Node     string           `json:"node,omitempty"`
	NodeID   *uint            `json:"node_id,omitempty"`
	ItemType string           `json:"item_type,omitempty"`
	Children []menuUpsertItem `json:"children,omitempty"`
}

func resolveMenuUpsertItems(ctx context.Context, api coreapi.CoreAPI, items []menuUpsertItem) ([]coreapi.MenuItem, error) {
	out := make([]coreapi.MenuItem, 0, len(items))
	for _, it := range items {
		mi := coreapi.MenuItem{
			Label:    it.Label,
			URL:      it.URL,
			Target:   it.Target,
			NodeID:   it.NodeID,
			ItemType: it.ItemType,
		}
		// page / node accept a slug; translate to NodeID.
		slug := it.Page
		if slug == "" {
			slug = it.Node
		}
		if slug != "" && mi.NodeID == nil {
			q, err := api.QueryNodes(ctx, coreapi.NodeQuery{Slug: slug, Limit: 1})
			if err != nil {
				return nil, err
			}
			if q != nil && len(q.Nodes) > 0 {
				id := q.Nodes[0].ID
				mi.NodeID = &id
				mi.ItemType = "node"
			} else {
				// Skip unresolvable items rather than failing the whole upsert.
				continue
			}
		}
		if len(it.Children) > 0 {
			children, cerr := resolveMenuUpsertItems(ctx, api, it.Children)
			if cerr != nil {
				return nil, cerr
			}
			mi.Children = children
		}
		out = append(out, mi)
	}
	return out, nil
}

func menuInputFromArgs(args map[string]any) coreapi.MenuInput {
	in := coreapi.MenuInput{
		Name: stringArg(args, "name"),
		Slug: stringArg(args, "slug"),
	}
	if raw, ok := args["items"]; ok {
		b, _ := json.Marshal(raw)
		var items []coreapi.MenuItem
		_ = json.Unmarshal(b, &items)
		in.Items = items
	}
	return in
}
