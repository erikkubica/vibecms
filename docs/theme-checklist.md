# Theme production-readiness checklist

A self-verification loop for AI agents (and humans) building a Squilla theme.
Each item is concrete: there's a command to run, a query to inspect, or a URL
to hit. **Don't claim done until every check passes.**

For an automated pass against an active theme, call MCP
`core.theme.checklist` — it walks the theme directory, runs the structural
checks, and reports `pass | fail` per item with file paths and line hints.

> **Convention:** `<base>` is your local origin (`http://localhost:8080` by
> default). `<slug>` is the theme directory name and the value of `slug:` in
> `theme.json`.

---

## 0. Pre-flight (file-level structure)

- [ ] `themes/<slug>/theme.json` exists and parses as JSON.
- [ ] `theme.json` has a `slug` field — required by `core.theme.deploy` —
      and it matches the directory name (regex `[A-Za-z0-9_-]+`).
- [ ] `theme.json` declares at least one layout with `is_default: true`.
- [ ] Every block declared in `theme.json.blocks[]` has a directory at
      `themes/<slug>/blocks/<dir>/` containing `block.json` and `view.html`.

## 1. Block schemas (silent-data-loss prevention)

- [ ] Every `block.json` `field_schema` entry uses `"key":` (not `"name":`).
- [ ] Every `field_schema` entry of type `select` or `radio` has `options:`
      as **plain strings** — never `[{value,label}]`. Object options crash
      the admin with React error #31; the theme loader rejects them.
- [ ] Every term-typed field (`type: "term"`) has both `taxonomy:` and
      `term_node_type:` — without `term_node_type`, hydration won't match
      any term row.
- [ ] Every block slug in `theme.json` is **prefixed** (e.g. `sq-hero`,
      `<theme>-feature`) to avoid collisions with extensions like
      content-blocks (`cb-*`).
- [ ] Every block.json `test_data` covers **every** field declared in
      `field_schema` with realistic, on-brand values — not lorem ipsum,
      not empty strings. Missing test_data entries leave the admin
      preview empty and silently pass screenshot tests because the
      surrounding chrome looks fine.
- [ ] Repeater fields in `test_data` have at least one populated entry,
      so the admin preview shows the layout instead of an empty void.
- [ ] No view.html contains a hardcoded fallback like
      `{{ or .heading "Welcome" }}` or `{{ else }}Default{{ end }}` —
      these render fake-complete pages when seed data is missing,
      defeating playwright screenshot verification. Replace with
      `{{ with .heading }}{{ . }}{{ end }}` so empty data renders empty
      (loud).

## 2. Tengo seeds (`scripts/theme.tengo` + modules)

- [ ] No call to `log.error("…")` in any `.tengo` file (Tengo parser rejects
      `error` as a selector). Use `log.warn(…)`, `log.info(…)`, or the
      alias `log.err(…)`.
- [ ] Every `nodes.create` / `nodes.update` call uses `fields_data:` at the
      top level — never `fields:`.
- [ ] Every block inside a `blocks_data:` array uses `fields:` — never
      `fields_data:`. (Yes, the asymmetry is real and intentional.)
- [ ] Every term-typed field value is stored as an object
      (`{slug: "…", name: "…"}`), not a bare slug string. Admin requires
      object form to pre-select; templates handle both shapes.
- [ ] Real taxonomies (visible in admin "Taxonomies" tab, queryable via
      `tax_query`) are written under `taxonomies:` — never `fields_data`.
- [ ] Every `terms.create` call passes `node_type:` so the term hydrates
      correctly when used in template fields.
- [ ] Every theme block slug declared in `theme.json` is the same slug used
      inside seed `blocks_data: [{type: "<slug>", ...}]`.
- [ ] Seeds are **idempotent in production**: existence checks before
      creating nodes/terms, upserts for menus and settings — except for
      kernel pointers (e.g. `homepage_node_id`) which must always overwrite
      so they don't go stale after a re-seed.
- [ ] Seeds branch on `dev_mode` (top-level Tengo global) where the AI
      development loop benefits from overwriting prior content. In
      production (`SQUILLA_DEV_MODE` unset) the branch becomes a no-op.

## 3. Templates (`layouts/`, `partials/`, `blocks/*/view.html`)

- [ ] No hardcoded fallback strings: `{{ or .x "Default" }}` is forbidden —
      it masks data bugs upstream. Use `{{ with .x }}{{ . }}{{ end }}` so
      missing data renders empty (loud) rather than fake (silent).
- [ ] Every settings lookup uses `index` or `mustSetting`, not dot-access.
      Settings keys keep their dots in the DB (`squilla.brand.version`).
- [ ] For settings your theme **requires**, prefer
      `{{ mustSetting $s "squilla.brand.version" }}` — it errors loudly when
      the seed forgets to write the key. Reserve `index` for optional values.
- [ ] Term-typed field templates handle BOTH shapes (string slug OR hydrated
      `{slug,name}` map). The hydrator may return either depending on whether
      a matching term row exists.
- [ ] Partials are referenced via `{{ renderLayoutBlock "<slug>" }}` from
      layouts. Don't share `{{ define "..." }}` blocks across files —
      Go's template engine parses each file separately.
- [ ] Filters that take no input still pass an empty dict:
      `{{ $things := filter "list_things" (dict) }}`. Bare `filter "name"`
      throws "wrong number of args".

## 4. Activation

- [ ] `core.theme.activate` succeeds for `<slug>` with no errors logged.
- [ ] Server logs show no warnings of the form
      `"top-level fields: is ignored — did you mean fields_data:"` or
      `"blocks_data[N] uses fields_data: …"` or
      `"… is type=term but term_node_type is empty"` or
      `"… is type=select with object options"`. (These are A1–A4 fail-loud
      paths.)
- [ ] After activation, hitting `<base>/` returns HTTP 200, **not 500 and not
      empty**. `curl -s -o /dev/null -w '%{http_code}' <base>/`.
- [ ] All declared blocks appear in `core.block_types.list` with
      `source: theme` and `theme_name: <slug>`.

## 5. Public render

- [ ] Every seeded page returns 200 from its public URL. List pages with
      `core.node.query({status:"published"})` and curl each `full_url`.
- [ ] Every theme-asset reference resolves — search the rendered HTML for
      stray `theme-asset:` prefixes (none should leak through). Find with
      `curl <base>/ | grep -c "theme-asset:"` → expect `0`.
- [ ] Every theme stylesheet/script returns 200. Grab them from `theme.json`
      and `curl -sI <base>/theme/assets/<src>`.

## 6. Admin UX (do this in a real browser, e.g. via playwright-cli)

- [ ] Open `/admin` and edit one of every seeded node type. Confirm:
   - Taxonomies appear in the "Taxonomies" tab and the seeded values are
     pre-selected.
   - Term-typed fields show the seeded term as the current selection (NOT
     "-- select --"). If they don't, the value is stored as a bare slug
     string instead of an object.
   - Sub-field inputs in repeater blocks are **populated** (proves
     `block.json` uses `key:`, not `name:`).
- [ ] Open the block editor on a seeded node. Every `select` field shows
      its options. (If options crash the page with React #31, options were
      `{value,label}` objects instead of strings.)

## 7. Idempotency & re-activation

- [ ] Re-activate the theme: `core.theme.activate(<id>)`. No
      duplicate-key errors, no orphaned data, no unexpected mutations to
      already-edited content.
- [ ] Where applicable, the editor's customizations (e.g. a renamed page
      title, edited block copy) are preserved across re-activation. Seed
      writes that are kernel-owned (homepage_node_id) are refreshed; seed
      writes that are editor-owned (page titles, term names) are not
      touched.

## 8. Optional but recommended

- [ ] Vendor JS (highlight.js, etc.) is lazy-loaded inside the theme's own
      `theme.js`, not declared in `theme.json` `scripts[]`. Every page
      paying the byte tax for a feature most pages don't use is a perf bug.
- [ ] All `block.json` files have `description` and every field has `help`
      text — humans editing in admin need this.
- [ ] `test_data` in every block.json is **on-brand**, not lorem ipsum, so
      the admin preview tells the whole story at a glance.

---

## How to use this with `core.theme.checklist`

1. Activate the theme (`core.theme.activate`).
2. Call `core.theme.checklist({ slug: "<slug>" })`.
3. Walk through the returned `failures[]` and fix each one.
4. Re-run until `failures: []` and `warnings: []`.
5. Then walk this document by hand for the items the automated tool
   can't introspect (admin UX, public render visual sanity).

The automated tool catches structural bugs (missing slug, schema
violations, broken activation). This document catches the things only a
human or an agentic loop can verify (does the page actually look right,
does the admin pre-select the right value, does the editor flow feel
correct).
