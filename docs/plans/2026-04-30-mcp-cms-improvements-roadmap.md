# MCP / CMS Improvements — Roadmap

Source: `docs/workarounds-needed-during-editing.md` (2026-04-30).

This roadmap captures the **larger / architectural** items deferred from the
autopilot pass on the same day. Quick wins and medium-scope items have been
implemented in-tree; what remains here needs proper design discussion before
implementation.

## Deferred items

### #5 — Capability guard is dead code from the MCP path

Every MCP tool runs as `InternalCaller`, bypassing per-method capability
checks on `CoreAPI`. Token-scoped enforcement happens only at dispatch level
(`read | content | full`).

**Options:**
- Bind tokens to a capability set and propagate that into the caller.
- Remove the per-method guard from CoreAPI to reduce confusion.
- Audit log records which capabilities a token's tool call would have needed,
  even if the guard isn't enforced.

**Why deferred:** Cross-cuts auth, dispatch, audit, and the CoreAPI guard
package. Needs an explicit decision on whether MCP tokens become first-class
capability principals.

---

### #7 — No admin UI for the audit log

`internal/mcp/audit.go` is just a `mcp_audit_log` table + cleanup loop. No
admin page, no `core.audit.query` MCP tool.

**Options:**
- Ship a minimal audit page in the admin SPA shell.
- Expose `core.audit.query` MCP tool with structured filters
  (token, tool, time range, status).

**Why deferred:** A page touches admin-ui shell + a new HTTP route + likely
a new extension. A tool requires schema design (filter shape, pagination,
PII handling). Both deserve design before code.

---

### #8 — Field-type provider fallback rendering + checklist flagging

`image`, `media`, `file`, `gallery` field types come from `media-manager`. A
theme that lists `type: "image"` while the extension is inactive renders a
broken admin form.

**Options:**
- Admin renders a clear "field type 'image' provided by extension
  media-manager (inactive)" placeholder.
- Theme/extension activation warns (not fails) on missing providers.
- `core.theme.checklist` flags missing field-type providers.

**Why deferred:** Touches admin-ui field renderer registry + checklist
output + activation pipeline. Lots of small touches across boundaries; needs
a coordinated change rather than one-shot patches.

---

### #11 — Theme settings field-type changes don't migrate values

Changing a settings field from `text` to `select` leaves the old text value
in `site_settings`; admin renders an empty select with no warning.

**Options:**
- Detect type changes between snapshot and new schema on
  `core.theme.activate` / `rescan`.
- Coerce where possible, clear with audit-log entry, or raise a checklist
  warning.

**Why deferred:** A real migration story (snapshot diff, coercion table,
rollback). Out of scope for a single pass.

---

### #12 — `core.data.query` blocks `mcp_audit_log` for non-full tokens

Kernel-private tables are blocked unless `scope=full`. Read tokens cannot
inspect their own audit history.

**Options:**
- Allow read with a `mcp_audit:read` capability rather than gating it
  behind full scope.
- Couples to #5 (capability principals) and #7 (dedicated tool).

**Why deferred:** Best resolved alongside #5 / #7 as part of a coherent
audit story.

---

## Done in this pass

The following are implemented, see commits referencing
`docs/workarounds-needed-during-editing.md`:

- #1 — `core.node.query` `select` / `fields_only` + saner default limit
- #2 — `core.node.update` `return_node` parameter
- #3 — `core.node.upsert` by slug
- #4 — `core.extension.delete` MCP tool
- #6 — `data:delete` fallback dropped (with deprecation period)
- #9 — Field schema accepts both `key` and `name`
- #10 — Menu render shape includes `label`
- #13 — Tengo `core/log.caller_info()`
- #14 — Validate `admin_ui.entry` resolves to a file on activation
- #15 — `core.guide` snapshot includes block_types + layouts
- #16 — `image_url` returns `template.URL` and resolves `theme-asset:`
- #17 — Public render emits HTML comment for per-block panics
- #18 — Admin rejects media drops on non-media fields
- #19 — `core.media.upload` MCP path debugged + fixed
- #19a — `field_schema[].type` validated against type registry
