# MCP / CMS Workarounds Needed During Doc Editing

Captured while authoring the expanded extensions / themes / scripting / api-reference
documentation pages on 2026-04-30. Each entry documents an actual obstacle the
edit hit, with a workaround used and a suggested capability improvement.

## 1. `core.guide` and `core.node.query` results exceed the MCP token cap

**Hit:** A bare `core.guide()` call returned ~90 KB; `core.node.query({node_type: "documentation", limit: 100})` returned ~170 KB. Both exceeded the tool-result token cap and were dumped to a temp file the editor had to re-parse with `jq`.

**Workaround:** Read the dumped file via `Read` / `jq` instead of the original tool result.

**Suggested improvements:**
- `core.guide` should support `topic` filtering that narrows the **whole** payload (recipes, snapshot, tool_index) rather than only narrowing recipes.
- `core.node.query` should support a `select` parameter that lets the caller request specific fields (e.g. `["id", "slug", "title"]`) — most discovery calls don't need `blocks_data`.
- `core.node.query` should support a `fields_only` boolean that omits `blocks_data` and `seo_settings`.
- Pagination defaults should be sane (limit=20) rather than returning everything when `limit` is missing/large.

## 2. `core.node.update` echoes the entire updated node back

**Hit:** Each update of a doc page sent ~7 KB in and got ~7 KB back, doubling token cost on what should be a write-only operation. Across 15 page updates this added up.

**Workaround:** Switched mid-batch to `core.data.update` on `content_nodes`, which returns `{"ok": true}` — same effect, ~99% smaller response.

**Suggested improvements:**
- Add a `return` or `return_node` parameter (default `true` for back-compat) so callers can opt into `{"ok": true}` responses.
- For bulk doc-style writes, consider `core.node.upsert_many` that accepts an array and returns counts.

## 3. No `core.node.upsert` (insert-or-update by slug)

**Hit:** Pushing 15 docs required logic to query each by slug, branch on `total > 0`, and dispatch either `update(id, ...)` or `create(...)`. The seed Tengo file already does this with helper functions. Two tool calls per doc.

**Workaround:** Pre-built per-doc payloads with `op` and `id` fields, then dispatched the right MCP tool per file.

**Suggested improvement:** `core.node.upsert({slug, node_type, ...})` — returns `{node, created: bool}`. Idempotent on (`node_type`, `slug`, `language_code`).

## 4. No `core.extension.delete` MCP tool

**Hit:** `MCP Tool Catalog` doc had to call this gap out — the HTTP admin handler supports deletion (`extensions/<slug>` DELETE) but the MCP surface does not. An AI agent cannot fully manage the extensions table via MCP.

**Workaround:** Document the gap, suggest manual `extensions` table edits.

**Suggested improvement:** Add `core.extension.delete` (full scope) that wraps the existing HTTP handler. Require the extension to be inactive first; same precondition as the HTTP handler.

## 5. Capability guard is dead code from the MCP path

**Hit:** Documenting the capability matrix conflicted with reality — every MCP tool runs as `InternalCaller` (`internal/mcp/dispatch.go:91`), bypassing every per-method capability check on `CoreAPI` (`internal/coreapi/capability.go:19-20`). The guard works for direct-extension and Tengo-via-CoreAPI calls, but token-scoped enforcement happens only via the dispatch-level scope gate (`read | content | full`).

**Workaround:** Documented this explicitly in `api-coreapi` and `api-mcp` so doc readers don't expect fine-grained MCP enforcement.

**Suggested improvements:**
- Either bind tokens to a capability set and propagate that into the caller, or remove the per-method guard from CoreAPI to reduce confusion.
- Audit log should record which capabilities a token's tool call would have needed, even if the guard isn't enforced — useful for future migration to fine-grained scoping.

## 6. `data:delete` capability silently falls back to `data:write`

**Hit:** Documented in `internal/coreapi/capability.go:408-413`. An extension that declares only `data:write` can still call `DataDelete`. The fallback is undocumented in the manifest schema and surprises auditors.

**Workaround:** Documented the fallback explicitly in the extensions intro and api-coreapi.

**Suggested improvement:** Either drop the fallback (clean break: extensions that need delete must declare `data:delete`), or surface the fallback in `core.extension.standards` output.

## 7. There's no admin UI for the audit log

**Hit:** Both `/admin/audit` and audit-browsing tools were initially documented as available; reality (`internal/mcp/audit.go`) is just a `mcp_audit_log` table + a cleanup loop.

**Workaround:** Documented "query the table directly via psql or `core.data.query`."

**Suggested improvement:** Ship a minimal audit page in the admin SPA shell; or expose a dedicated `core.audit.query` MCP tool with structured filters (token, tool, time range, status).

## 8. `image`, `media`, `file`, `gallery` field types require an extension

**Hit:** Documenting field types — half the catalogue (the visual half) is not in the kernel. They come from `media-manager`. A theme that lists `type: "image"` in a block will silently render a broken admin form if `media-manager` is deactivated.

**Workaround:** Documented the dependency.

**Suggested improvements:**
- The admin should render a clear "field type 'image' provided by extension media-manager (inactive)" placeholder rather than a broken empty input.
- Theme/extension activation should warn (not fail) if a referenced field type is missing.
- `core.theme.checklist` should flag missing field-type providers.

## 9. `theme.json` field-schema vs `core/nodetypes.register` field-schema use different keys

**Hit:** This is the #1 documented gotcha and it really does bite. block.json / theme.json use `"key":`; Tengo `nodetypes.register` uses `"name":`. Mismatching produces empty admin inputs even when `fields_data` is correct.

**Workaround:** Documented in three places (themes-intro, theme-manifest, scripting-modules) and prominent callouts everywhere.

**Suggested improvement:** Normalise to one field name (`key` is more common). Have the parser accept both for back-compat with a deprecation warning.

## 10. Menu items render via `.title`, not `.label`

**Hit:** Tengo input to `menus.upsert` is `label:`, but the rendered menu shape uses `title`. Templates that read `{{.label}}` silently render empty.

**Workaround:** Documented in the layouts page with a HEADS UP callout + working code example.

**Suggested improvement:** Either return `label` in the rendered menu shape, or have `menus.upsert` accept `title:` directly.

## 11. Theme settings field-type changes don't migrate values

**Hit:** Changing a settings field from `text` to `select` leaves the old text value in `site_settings` and the admin renders an empty select. No warning, no migration path.

**Workaround:** Documented "plan field changes carefully."

**Suggested improvement:** On `core.theme.activate` (or `rescan`), detect type changes between the snapshot and the new schema and either (a) coerce where possible, (b) clear the old value with an audit-log entry, or (c) raise a checklist warning.

## 12. `core.data.query` blocks `mcp_audit_log` and other kernel-private tables when scope < full

**Hit:** Documenting the audit log mentioned it's queryable via `core.data.query`. Actually, kernel-private tables are blocked unless `scope=full` (per `tools_data.go:21-30` plus `data_access.go:20-33`). Read tokens cannot inspect their own audit history.

**Workaround:** Documented the scope requirement.

**Suggested improvement:** `mcp_audit_log` is read-mostly diagnostic — consider allowing read with a `mcp_audit:read` capability rather than gating it behind full scope.

## 13. Nothing exposes "current Tengo capabilities" introspection

**Hit:** Hard to write the scripting docs without claiming what the default theme capability set is. The actual list lives in `internal/scripting/capabilities.go:10` and isn't surfaced to scripts at runtime.

**Workaround:** Read the source, list it in the scripting-intro doc.

**Suggested improvement:** Add `core/log.caller_info()` or similar that returns `{slug, kind, capabilities[]}` so a Tengo script can self-introspect.

## 14. Build-time validation gaps in `extension.json` admin_ui.entry

**Hit:** Documented `admin_ui.entry` as required-if-present but **not validated**. A typo silently breaks the UI without any error log.

**Workaround:** Documented the trap.

**Suggested improvement:** Activation should fail (or at least loudly warn) if `admin_ui.entry` does not resolve to an existing file under the extension directory.

## 15. `core.guide` snapshot omits block types and layouts

**Hit:** The agent-friendly snapshot includes `active_theme`, `node_types`, `recent_nodes`, and `sections_by_node_type` — but not block types, layouts, or partials. Discovering block-type slugs requires a separate `core.block_types.list` call.

**Workaround:** Documented the gap, suggested chaining calls.

**Suggested improvement:** Include `block_types` and `layouts` in the snapshot (slug + name only — the heavy fields can stay behind explicit fetches).

## 16. `image_url` template helper silently strips theme-asset URIs to `#ZgotmplZ`

**Hit:** Building an `about-team` block, I bound `<img src="{{ image_url .photo "" }}">` against a value of `theme-asset:team-erik`. Output was `<img src="#ZgotmplZ">` — Go `html/template`'s URL-context sanitiser stripping the unknown scheme. `image_url` had returned the raw input string instead of resolving it.

**Workaround:** Bypass `image_url`, hardcode the runtime path: `<img src="/theme/assets/images/{{.key}}.jpg">`. Loses image_size variants but renders.

**Suggested improvements:**
- `image_url` should always return a `template.URL` (already-trusted) regardless of size argument, so the sanitiser doesn't fire.
- Calling `image_url x ""` (no size) should resolve `theme-asset:KEY` against the active theme's `assets[]` registry — the original-resolution URL — instead of returning the raw URI.
- Document the difference between `theme-asset:` (theme bundle) and media-library URLs; both currently flow through the same field, but only the second has full resolver support.

## 17. Block silently dropped from public render when its template helper panics

**Hit:** While iterating on `about-team`, the public `/about` page rendered without the `about-team` section entirely — but `core.render.node_preview(id=12)` rendered it with the broken `#ZgotmplZ` image. Two different render paths, two different behaviours, no error logged. The block came back only after I rewrote the template to avoid `image_url` and redeployed.

**Workaround:** Tail the server logs while iterating; if the public page is missing a section that preview shows, the helper is the prime suspect.

**Suggested improvements:**
- Public render should NOT swallow per-block panics into "drop the block silently". Render an HTML comment `<!-- block error: about-team — image_url: invalid arg -->` so the gap is visible to the developer (and not to public visitors with debug=off).
- `core.audit.query` (or the future audit page) should record per-block render errors with block slug + helper + arg.
- Block `cache_output: true` should also cache the **error**, with a short TTL, instead of caching empty — so re-renders aren't free of the diagnostic.

## 18. Use proper field types — `text` masquerading as image produces `[object Object]` in admin

**Hit:** I declared `{ "key": "photo", "type": "text" }` for a sub-field that should hold an uploaded image. The admin renders a plain text input. When a user uploads a media object via drag-and-drop or paste, the form coerces the object to a string (`"[object Object]"`) and saves it. Public render fails because the template expects `.photo.url`, not the string.

**Workaround:** Change the field to `"type": "image"` (or `"type": "media"`) and update the template to read `.photo.url` / `.photo.alt`. Update existing stored data to the object shape `{ "url": "...", "alt": "..." }`.

**Suggested improvements:**
- Admin should reject (or coerce + warn) when a media drop happens onto a non-media field.
- `core.theme.checklist` should flag fields whose `key` matches `photo|image|media|gallery|file` but whose `type` is `text`/`textarea` — almost always a mistake.
- Documentation of field types should include a "when to use which" decision tree for media-shaped data, and explicitly call out: "do not store URLs as text; the admin will appear to accept anything but the result is lossy."

## 19a. Boolean field type is named `toggle`, not `boolean`

**Hit:** Declared `{ "key": "is_ai", "type": "boolean" }` based on a guess from convention. Squilla's actual built-in is `toggle` (per `core.field_types.list`). The admin renders an unknown type as a fallback (often empty/text) so the on/off state is unsettable, and on save the value gets coerced to a string.

**Workaround:** Always call `core.field_types.list` before authoring a `field_schema`. Don't guess from convention — the type names don't track other CMSes (`toggle` not `boolean`/`switch`/`checkbox`, `richtext` not `wysiwyg`/`html`, `select` not `dropdown`).

**Suggested improvements:**
- Block / nodetype activation should validate `field_schema[].type` against the known type registry (built-in + extension-contributed) and fail loudly with the list of valid types when an unknown one is used.
- `core.theme.checklist` should flag unknown field types in `theme.json`-registered blocks.
- Document a one-screen "field type cheatsheet" linking common-name → Squilla-name (`boolean → toggle`, `dropdown → select`, `wysiwyg → richtext`).

## 19. `core.media.upload` MCP tool fails ("media service not configured") even when `media-manager` is active

**Hit:** Trying to upload Erik's headshot via `core.media.upload` returned `internal: internal error: media service not configured`. I initially read this as "media-manager isn't active on prod" and routed around it by bundling the photo into the theme assets directory instead.

**User feedback:** `media-manager` IS active on `squilla.app`. The admin's image picker, the featured-image input on nodes, and the full Media Plugins / Optimizer pages all render and work normally. The failure is **MCP-only** — `core.media.upload` is wired to a service registration that the MCP dispatch path doesn't see, even though the same upload path works through the admin UI / extension HTTP routes.

**Workaround:** For theme-bundled images, ship them in the theme zip and reference via `/theme/assets/images/<key>.jpg` (no media library entry needed). For media that genuinely belongs in the library, use the admin UI manually instead of the MCP tool.

**Suggested improvements:**
- Trace why `core.media.upload` resolves `media service` to nil under the InternalCaller used by MCP dispatch (`internal/mcp/dispatch.go:91`) when the same call from an extension HTTP handler succeeds. Likely a service-locator that's per-request-scope and not populated for MCP requests.
- The error message should name the actual resolution failure, not the misleading "not configured" — at minimum, log the caller kind and which service-registry key was empty.
- Add a `core.media.upload` smoke test to the MCP integration suite so this regression has a tripwire next time.

---

## Source coverage notes

These workarounds were derived from cross-checking the live admin (Squilla v0.2.0)
against:

- `internal/coreapi/api.go`
- `internal/coreapi/capability.go`
- `internal/coreapi/tengo_*.go`
- `internal/cms/extension_loader.go`
- `internal/cms/theme_loader.go`
- `internal/scripting/engine.go`
- `internal/scripting/capabilities.go`
- `internal/mcp/dispatch.go`, `tools_*.go`, `audit.go`
- `proto/squilla_plugin.proto`, `proto/squilla_coreapi.proto`
- `themes/README.md`, `extensions/README.md` (treated as code-adjacent reference)

Stale `docs/*.md` files were **not** treated as truth.
