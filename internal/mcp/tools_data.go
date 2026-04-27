package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"

	"vibecms/internal/coreapi"
)

// guardKernelTable refuses access to kernel-private tables for tokens
// that aren't scope=full. dispatch.go runs every tool as
// coreapi.InternalCaller — handy for typed APIs but it makes the
// per-table data guard a no-op. Without this gate, a scope=read token
// could `core.data.query` the `users` table and dump every
// password_hash; scope=content could overwrite sessions or
// mcp_tokens. scope=full is admin-equivalent by design and stays
// permissive (operator opted in when issuing the token).
func guardKernelTable(ctx context.Context, table string) error {
	if !coreapi.IsKernelPrivateTable(table) {
		return nil
	}
	tok := tokenFromCtx(ctx)
	if tok != nil && tok.Scope == ScopeFull {
		return nil
	}
	return fmt.Errorf("table %q is kernel-private; scope=full required for direct access", table)
}

func (s *Server) registerDataTools() {
	api := s.deps.CoreAPI

	s.addTool(mcp.NewTool("core.data.get",
		mcp.WithDescription("Low-level: read a row from any table by ID. Prefer domain tools (core.node.*, etc.) where available."),
		mcp.WithString("table", mcp.Required()),
		mcp.WithNumber("id", mcp.Required()),
	), "read", func(ctx context.Context, args map[string]any) (any, error) {
		table := stringArg(args, "table")
		if err := guardKernelTable(ctx, table); err != nil {
			return nil, err
		}
		return api.DataGet(ctx, table, uintArg(args, "id"))
	})

	s.addTool(mcp.NewTool("core.data.query",
		mcp.WithDescription("Low-level: query rows from any table with where/search/order_by. 'raw' is a WHERE-fragment with ? placeholders bound from 'args'."),
		mcp.WithString("table", mcp.Required()),
		mcp.WithObject("where"),
		mcp.WithString("search"),
		mcp.WithString("order_by"),
		mcp.WithNumber("limit"),
		mcp.WithNumber("offset"),
		mcp.WithString("raw"),
		mcp.WithArray("args"),
	), "read", func(ctx context.Context, args map[string]any) (any, error) {
		table := stringArg(args, "table")
		if err := guardKernelTable(ctx, table); err != nil {
			return nil, err
		}
		q := coreapi.DataStoreQuery{
			Where:   mapArg(args, "where"),
			Search:  stringArg(args, "search"),
			OrderBy: stringArg(args, "order_by"),
			Limit:   clampLimit(intArg(args, "limit")),
			Offset:  intArg(args, "offset"),
			Raw:     stringArg(args, "raw"),
		}
		if raw, ok := args["args"]; ok {
			b, _ := json.Marshal(raw)
			_ = json.Unmarshal(b, &q.Args)
		}
		return api.DataQuery(ctx, table, q)
	})

	s.addTool(mcp.NewTool("core.data.create",
		mcp.WithDescription("Low-level: insert a row into any table. Returns the new row."),
		mcp.WithString("table", mcp.Required()),
		mcp.WithObject("data", mcp.Required()),
	), "content", func(ctx context.Context, args map[string]any) (any, error) {
		table := stringArg(args, "table")
		if err := guardKernelTable(ctx, table); err != nil {
			return nil, err
		}
		return api.DataCreate(ctx, table, mapArg(args, "data"))
	})

	s.addTool(mcp.NewTool("core.data.update",
		mcp.WithDescription("Low-level: update a row by ID."),
		mcp.WithString("table", mcp.Required()),
		mcp.WithNumber("id", mcp.Required()),
		mcp.WithObject("data", mcp.Required()),
	), "content", func(ctx context.Context, args map[string]any) (any, error) {
		table := stringArg(args, "table")
		if err := guardKernelTable(ctx, table); err != nil {
			return nil, err
		}
		err := api.DataUpdate(ctx, table, uintArg(args, "id"), mapArg(args, "data"))
		return map[string]any{"ok": err == nil}, err
	})

	s.addTool(mcp.NewTool("core.data.delete",
		mcp.WithDescription("Low-level: delete a row by ID."),
		mcp.WithString("table", mcp.Required()),
		mcp.WithNumber("id", mcp.Required()),
	), "content", func(ctx context.Context, args map[string]any) (any, error) {
		table := stringArg(args, "table")
		if err := guardKernelTable(ctx, table); err != nil {
			return nil, err
		}
		err := api.DataDelete(ctx, table, uintArg(args, "id"))
		return map[string]any{"ok": err == nil}, err
	})

	// Raw SQL is gated: scope=full AND the env flag must be set.
	if s.allowRawSQL {
		s.addTool(mcp.NewTool("core.data.exec",
			mcp.WithDescription("Execute a raw parameterized SQL statement. Gated behind scope=full and VIBECMS_MCP_ALLOW_RAW_SQL=true. Returns rows_affected. Use with extreme care — this bypasses all model-level validation and events."),
			mcp.WithString("sql", mcp.Required()),
			mcp.WithArray("args"),
		), "full", func(ctx context.Context, args map[string]any) (any, error) {
			sql := stringArg(args, "sql")
			if sql == "" {
				return nil, fmt.Errorf("sql is required")
			}
			var bound []any
			if raw, ok := args["args"]; ok {
				b, _ := json.Marshal(raw)
				_ = json.Unmarshal(b, &bound)
			}
			n, err := api.DataExec(ctx, sql, bound...)
			return map[string]any{"rows_affected": n}, err
		})
	}
}
