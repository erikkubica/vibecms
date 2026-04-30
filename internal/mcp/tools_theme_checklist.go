package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"squilla/internal/cms"

	"github.com/mark3labs/mcp-go/mcp"
)

// summarizeActivatedThemeChecklist runs the structural checklist against a
// just-activated theme and returns a small map suitable for embedding in
// the core.theme.activate response. Goal: AI agents see checklist failures
// the moment they activate, so they can't accidentally declare the work
// done while screenshots silently pass over hardcoded fallbacks or empty
// test_data. Returns nil on any lookup failure rather than blocking the
// activation result — the checklist is a hint, not a gate.
func summarizeActivatedThemeChecklist(mgmt *cms.ThemeMgmtService, id int) map[string]any {
	if mgmt == nil {
		return nil
	}
	theme, err := mgmt.GetByID(id)
	if err != nil || theme == nil {
		return nil
	}
	slug := theme.Slug
	if slug == "" {
		slug = theme.Name
	}
	if slug == "" {
		return nil
	}
	themeDir := filepath.Join(mgmt.ThemesDir(), slug)
	if _, err := os.Stat(themeDir); err != nil {
		return nil
	}
	report := runThemeChecklist(slug, themeDir)
	summary, _ := report["summary"].(map[string]int)
	if summary == nil {
		return nil
	}
	out := map[string]any{
		"passed":   summary["passed"],
		"failed":   summary["failed"],
		"warnings": summary["warnings"],
		"total":    summary["total"],
		"call":     "core.theme.checklist({slug:\"" + slug + "\"}) for the full list",
	}
	// Surface the first few failure messages inline so the agent has
	// something concrete without a follow-up call.
	if checks, ok := report["checks"].([]checklistItem); ok {
		var top []string
		for _, c := range checks {
			if c.Severity == "fail" {
				top = append(top, c.ID+": "+c.Message)
				if len(top) >= 5 {
					break
				}
			}
		}
		if len(top) > 0 {
			out["sample_failures"] = top
		}
	}
	return out
}

// registerThemeChecklistTool exposes core.theme.checklist — the
// production-readiness loop. Walks the on-disk theme directory and reports
// pass/fail per check so an AI agent can self-verify before declaring done.
//
// On-demand only: never gates activation. Pairs with docs/theme-checklist.md
// (which covers the things only a human or agentic browser loop can verify).
func (s *Server) registerThemeChecklistTool() {
	s.addTool(mcp.NewTool("core.theme.checklist",
		mcp.WithDescription("Run automated production-readiness checks on a theme. Walks themes/<slug>/ on disk, validates theme.json (including any declared settings_pages[] — referenced files must exist, parse, and declare at least one valid field), every block.json field schema and test_data completeness, every block view.html for hardcoded fallback strings (`{{ or .x \"Default\" }}` patterns that fake a completed render and defeat screenshot verification), seed-script Tengo gotchas (top-level `fields:` typos, `log.error(` usage, missing `term_node_type`, object-shaped select options). Layouts and partials are also scanned for fallback strings. Returns {checks:[{id,severity,pass,message,file?}], summary:{passed,failed,warnings}}. Pair with docs/theme-checklist.md for the manual checks (admin UX, public render visual sanity)."),
		mcp.WithString("slug", mcp.Description("Theme slug (directory name). Defaults to the active theme.")),
	), "read", func(ctx context.Context, args map[string]any) (any, error) {
		slug := stringArg(args, "slug")
		mgmt := s.deps.ThemeMgmtSvc
		if mgmt == nil {
			return nil, fmt.Errorf("theme management service not wired")
		}
		if slug == "" {
			active, err := mgmt.GetActive()
			if err != nil || active == nil {
				return nil, fmt.Errorf("no slug provided and no active theme found")
			}
			slug = active.Slug
			if slug == "" {
				slug = active.Name
			}
		}
		themeDir := filepath.Join(mgmt.ThemesDir(), slug)
		if _, err := os.Stat(themeDir); err != nil {
			return nil, fmt.Errorf("theme directory not found: %s", themeDir)
		}
		return runThemeChecklist(slug, themeDir), nil
	})
}

type checklistItem struct {
	ID       string `json:"id"`
	Severity string `json:"severity"` // "fail" | "warn" | "pass"
	Pass     bool   `json:"pass"`
	Message  string `json:"message"`
	File     string `json:"file,omitempty"`
}

func runThemeChecklist(slug, themeDir string) map[string]any {
	checks := []checklistItem{}

	// 1. theme.json structural checks ---------------------------------------
	themeJSONPath := filepath.Join(themeDir, "theme.json")
	manifestRaw, err := os.ReadFile(themeJSONPath)
	if err != nil {
		checks = append(checks, checklistItem{
			ID: "theme.json.exists", Severity: "fail", Pass: false,
			Message: "theme.json not found at " + themeJSONPath,
			File:    themeJSONPath,
		})
		return summarize(slug, themeDir, checks)
	}
	checks = append(checks, checklistItem{
		ID: "theme.json.exists", Severity: "pass", Pass: true,
		Message: "theme.json present", File: themeJSONPath,
	})

	var manifest map[string]any
	if err := json.Unmarshal(manifestRaw, &manifest); err != nil {
		checks = append(checks, checklistItem{
			ID: "theme.json.parses", Severity: "fail", Pass: false,
			Message: "theme.json failed to parse: " + err.Error(),
			File:    themeJSONPath,
		})
		return summarize(slug, themeDir, checks)
	}
	checks = append(checks, checklistItem{
		ID: "theme.json.parses", Severity: "pass", Pass: true, Message: "theme.json parses",
	})

	manifestSlug, _ := manifest["slug"].(string)
	if manifestSlug == "" {
		checks = append(checks, checklistItem{
			ID: "theme.json.slug", Severity: "fail", Pass: false,
			Message: "theme.json is missing required `slug` field — core.theme.deploy will reject this archive",
			File:    themeJSONPath,
		})
	} else if !slugRegex.MatchString(manifestSlug) {
		checks = append(checks, checklistItem{
			ID: "theme.json.slug", Severity: "fail", Pass: false,
			Message: fmt.Sprintf("theme.json slug %q must match [A-Za-z0-9_-]+", manifestSlug),
			File:    themeJSONPath,
		})
	} else if manifestSlug != slug {
		checks = append(checks, checklistItem{
			ID: "theme.json.slug", Severity: "warn", Pass: false,
			Message: fmt.Sprintf("theme.json slug %q does not match directory name %q", manifestSlug, slug),
			File:    themeJSONPath,
		})
	} else {
		checks = append(checks, checklistItem{
			ID: "theme.json.slug", Severity: "pass", Pass: true,
			Message: "slug present and matches directory",
		})
	}

	// 1a. at least one default layout
	layouts, _ := manifest["layouts"].([]any)
	hasDefault := false
	for _, l := range layouts {
		if lm, ok := l.(map[string]any); ok {
			if d, _ := lm["is_default"].(bool); d {
				hasDefault = true
				break
			}
		}
	}
	if hasDefault {
		checks = append(checks, checklistItem{
			ID: "theme.json.default_layout", Severity: "pass", Pass: true,
			Message: "at least one layout has is_default: true",
		})
	} else {
		checks = append(checks, checklistItem{
			ID: "theme.json.default_layout", Severity: "fail", Pass: false,
			Message: "no layout has is_default: true — nodes without explicit layout_slug will fail to render",
			File:    themeJSONPath,
		})
	}

	// 1a-bis. theme should ship a 404 / error layout. Core no longer renders
	// a hardcoded fallback (would clash with theme styling) — when the theme
	// owns no 404 layout, missing pages render through the default layout
	// with empty blocks_html, which usually shows as a blank main area.
	// Surface a warning so theme devs make a conscious choice.
	hasNotFoundLayout := false
	for _, l := range layouts {
		lm, ok := l.(map[string]any)
		if !ok {
			continue
		}
		slug, _ := lm["slug"].(string)
		if slug == "404" || slug == "error" {
			hasNotFoundLayout = true
			break
		}
	}
	if hasNotFoundLayout {
		checks = append(checks, checklistItem{
			ID: "theme.json.notfound_layout", Severity: "pass", Pass: true,
			Message: "404/error layout registered — theme owns the missing-page UI",
		})
	} else {
		checks = append(checks, checklistItem{
			ID: "theme.json.notfound_layout", Severity: "warning", Pass: false,
			Message: "no layout with slug \"404\" or \"error\" — missing pages fall back to the default layout with empty blocks_html (usually a blank main area). Register `{ \"slug\": \"404\", \"file\": \"404.html\" }` and ship a themed 404 page.",
			File:    themeJSONPath,
		})
	}

	// 1b. settings_pages[] declared files exist + parse + carry valid fields.
	// Soft-fail at runtime (the loader logs & skips bad pages), but for an
	// authoring-time check we want to surface every issue so the agent fixes
	// them before shipping. Themes without settings_pages skip this block.
	if pages, ok := manifest["settings_pages"].([]any); ok && len(pages) > 0 {
		settingsIssues := validateSettingsPages(themeDir, pages)
		if len(settingsIssues) == 0 {
			checks = append(checks, checklistItem{
				ID: "theme.json.settings_pages", Severity: "pass", Pass: true,
				Message: fmt.Sprintf("settings_pages: %d page(s) declared, all files present with valid fields", len(pages)),
			})
		} else {
			for _, it := range settingsIssues {
				checks = append(checks, it)
			}
		}
	}

	// 2. blocks/*/block.json field-schema checks ----------------------------
	blocksDir := filepath.Join(themeDir, "blocks")
	if entries, err := os.ReadDir(blocksDir); err == nil {
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			bjPath := filepath.Join(blocksDir, e.Name(), "block.json")
			data, err := os.ReadFile(bjPath)
			if err != nil {
				checks = append(checks, checklistItem{
					ID: "block." + e.Name() + ".json.exists", Severity: "fail", Pass: false,
					Message: "block.json missing", File: bjPath,
				})
				continue
			}
			var bj map[string]any
			if err := json.Unmarshal(data, &bj); err != nil {
				checks = append(checks, checklistItem{
					ID: "block." + e.Name() + ".json.parses", Severity: "fail", Pass: false,
					Message: "block.json parse error: " + err.Error(),
					File:    bjPath,
				})
				continue
			}
			fs, _ := bj["field_schema"].([]any)
			schemaErrors := walkBlockSchema(e.Name(), "", fs)
			if len(schemaErrors) == 0 {
				checks = append(checks, checklistItem{
					ID: "block." + e.Name() + ".schema", Severity: "pass", Pass: true,
					Message: "field_schema clean",
				})
			} else {
				for _, msg := range schemaErrors {
					checks = append(checks, checklistItem{
						ID: "block." + e.Name() + ".schema", Severity: "fail", Pass: false,
						Message: msg, File: bjPath,
					})
				}
			}
			// Slug prefix hint
			bjSlug, _ := bj["slug"].(string)
			if bjSlug != "" && !strings.ContainsAny(bjSlug, "-_") {
				checks = append(checks, checklistItem{
					ID: "block." + e.Name() + ".slug_prefix", Severity: "warn", Pass: false,
					Message: fmt.Sprintf("block slug %q is unprefixed — consider <theme>-%s to avoid collisions with extension blocks", bjSlug, bjSlug),
					File:    bjPath,
				})
			}
			// test_data completeness — every schema field needs a value so
			// the admin preview and screenshot tests aren't fooled by an
			// empty render that "looks" filled in.
			td, _ := bj["test_data"].(map[string]any)
			tdIssues := validateTestData(e.Name(), fs, td)
			if len(tdIssues) == 0 {
				checks = append(checks, checklistItem{
					ID: "block." + e.Name() + ".test_data", Severity: "pass", Pass: true,
					Message: "test_data covers every schema field",
				})
			} else {
				for _, msg := range tdIssues {
					checks = append(checks, checklistItem{
						ID: "block." + e.Name() + ".test_data", Severity: "fail", Pass: false,
						Message: msg, File: bjPath,
					})
				}
			}
			// view.html fallback scan — hardcoded strings inside {{ or .x "…" }}
			// produce a render that LOOKS complete even when seed data is
			// missing. That breaks playwright screenshot verification, since
			// the page passes visual inspection while masking real data bugs.
			viewPath := filepath.Join(blocksDir, e.Name(), "view.html")
			if viewIssues := scanViewFallbacks(viewPath); len(viewIssues) > 0 {
				for _, it := range viewIssues {
					checks = append(checks, it)
				}
			} else if _, err := os.Stat(viewPath); err == nil {
				checks = append(checks, checklistItem{
					ID: "block." + e.Name() + ".view_no_fallbacks", Severity: "pass", Pass: true,
					Message: "view.html has no hardcoded fallback strings",
				})
			}
		}
	}

	// Layouts and partials get the same fallback scan — they bind page
	// chrome (titles, nav copy) where hardcoded defaults are equally
	// dangerous for screenshot-driven verification.
	for _, sub := range []string{"layouts", "partials"} {
		dir := filepath.Join(themeDir, sub)
		_ = filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() || filepath.Ext(p) != ".html" {
				return nil
			}
			if issues := scanViewFallbacks(p); len(issues) > 0 {
				for _, it := range issues {
					checks = append(checks, it)
				}
			}
			return nil
		})
	}

	// 3. Tengo seed gotchas (textual scan; complements runtime warnings)
	scriptsDir := filepath.Join(themeDir, "scripts")
	tengoIssues := scanTengoScripts(scriptsDir)
	if len(tengoIssues) == 0 {
		checks = append(checks, checklistItem{
			ID: "scripts.tengo", Severity: "pass", Pass: true,
			Message: "no obvious Tengo gotchas in seed scripts",
		})
	} else {
		for _, it := range tengoIssues {
			checks = append(checks, it)
		}
	}

	return summarize(slug, themeDir, checks)
}

var slugRegex = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

// looksLikeMediaKey reports whether a field key is shaped like it should
// hold media. Used to flag text/textarea fields that almost certainly
// should be image/media/file/gallery — the #1 way AI authors lose data
// (storing a media object as a string of "[object Object]").
func looksLikeMediaKey(key string) bool {
	k := strings.ToLower(key)
	for _, suffix := range []string{"photo", "image", "img", "media", "gallery", "thumbnail", "thumb", "avatar", "logo", "icon", "picture", "banner"} {
		if k == suffix || strings.HasSuffix(k, "_"+suffix) || strings.HasPrefix(k, suffix+"_") {
			return true
		}
	}
	return false
}

func walkBlockSchema(blockSlug, parentPath string, fields []any) []string {
	out := []string{}
	for _, raw := range fields {
		f, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		key, _ := f["key"].(string)
		if key == "" {
			if name, ok := f["name"].(string); ok && name != "" {
				out = append(out, fmt.Sprintf("block %q field %q uses `name:` — block.json field_schema must use `key:` (admin will render empty inputs)", blockSlug, name))
				continue
			}
		}
		path := key
		if parentPath != "" {
			path = parentPath + "." + key
		}
		typ, _ := f["type"].(string)
		// Heuristic: media-shaped key paired with a plain-text type is
		// almost always a mistake. The author (often AI) declared the
		// wrong type, then dumped a JSON object into it, which the admin
		// stringifies to "[object Object]". Surface as a warn so it's
		// visible without blocking the build.
		if (typ == "text" || typ == "textarea") && looksLikeMediaKey(key) {
			out = append(out, fmt.Sprintf("block %q field %q has media-shaped key but type=%q — almost certainly a mistake. Use type=image / media / file / gallery so the admin renders the right input and templates can read .url/.alt.", blockSlug, path, typ))
		}
		switch typ {
		case "select", "radio":
			if opts, ok := f["options"].([]any); ok {
				for _, o := range opts {
					if _, isObj := o.(map[string]any); isObj {
						out = append(out, fmt.Sprintf("block %q field %q (type=%q) has object options — must be plain strings (admin crashes with React #31)", blockSlug, path, typ))
						break
					}
				}
			}
		case "term":
			if tnt, _ := f["term_node_type"].(string); tnt == "" {
				out = append(out, fmt.Sprintf("block %q field %q is type=term but term_node_type is empty — hydration will not match any term row", blockSlug, path))
			}
			if tax, _ := f["taxonomy"].(string); tax == "" {
				out = append(out, fmt.Sprintf("block %q field %q is type=term but taxonomy is empty", blockSlug, path))
			}
		}
		if sub, ok := f["sub_fields"].([]any); ok {
			out = append(out, walkBlockSchema(blockSlug, path, sub)...)
		}
	}
	return out
}

// validateTestData walks every field in the block's field_schema and
// confirms test_data carries a corresponding non-empty value. This
// matters because admin previews and playwright screenshots both use
// test_data — a block whose test_data omits "heading" will render with
// an empty H1 in the preview, looking broken in admin and silently
// passing screenshot tests because the surrounding chrome rendered fine.
func validateTestData(blockSlug string, fields []any, td map[string]any) []string {
	out := []string{}
	if td == nil {
		out = append(out, fmt.Sprintf("block %q has no test_data — admin preview will render empty; declare a realistic value for every field_schema entry", blockSlug))
		// Still walk fields so the report lists what's missing.
	}
	for _, raw := range fields {
		f, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		key, _ := f["key"].(string)
		if key == "" {
			continue
		}
		typ, _ := f["type"].(string)
		v, present := td[key]
		if !present {
			out = append(out, fmt.Sprintf("block %q test_data is missing key %q (type=%q) — admin preview will render this field empty", blockSlug, key, typ))
			continue
		}
		if !testDataValueLooksRealistic(typ, v) {
			out = append(out, fmt.Sprintf("block %q test_data[%q] looks empty/placeholder for type=%q — supply on-brand demo data so preview reflects real usage", blockSlug, key, typ))
		}
		// Recurse into repeater sub_fields if both schema and data agree.
		if typ == "repeater" {
			if sub, ok := f["sub_fields"].([]any); ok {
				if arr, ok := v.([]any); ok && len(arr) > 0 {
					if first, ok := arr[0].(map[string]any); ok {
						out = append(out, validateTestData(blockSlug+"["+key+"][0]", sub, first)...)
					}
				} else {
					out = append(out, fmt.Sprintf("block %q repeater test_data[%q] is empty — supply at least one demo entry so admin preview shows the layout", blockSlug, key))
				}
			}
		}
	}
	return out
}

func testDataValueLooksRealistic(typ string, v any) bool {
	switch typ {
	case "text", "textarea", "richtext", "select", "radio", "color":
		s, ok := v.(string)
		return ok && strings.TrimSpace(s) != ""
	case "number":
		_, isF := v.(float64)
		_, isI := v.(int)
		return isF || isI
	case "toggle", "checkbox":
		_, ok := v.(bool)
		return ok
	case "image":
		m, ok := v.(map[string]any)
		if !ok {
			return false
		}
		u, _ := m["url"].(string)
		return strings.TrimSpace(u) != ""
	case "gallery":
		arr, ok := v.([]any)
		return ok && len(arr) > 0
	case "link":
		m, ok := v.(map[string]any)
		if !ok {
			return false
		}
		u, _ := m["url"].(string)
		return strings.TrimSpace(u) != ""
	case "term", "node":
		m, ok := v.(map[string]any)
		if !ok {
			return false
		}
		s, _ := m["slug"].(string)
		return strings.TrimSpace(s) != ""
	case "repeater":
		arr, ok := v.([]any)
		return ok && len(arr) > 0
	case "form_selector":
		s, ok := v.(string)
		return ok && strings.TrimSpace(s) != ""
	}
	// Unknown type — accept anything non-nil non-empty.
	if s, ok := v.(string); ok {
		return strings.TrimSpace(s) != ""
	}
	return v != nil
}

// scanViewFallbacks finds hardcoded fallback strings in a Go template
// file. The signature footgun is `{{ or .field "Some literal" }}` —
// the literal renders when .field is empty, so the page LOOKS complete
// even when seed data is missing or the field name is misspelt. This
// kind of fake completeness defeats playwright screenshot verification.
//
// Reports each match as a `fail` with the line number and the offending
// text, so the agent can locate and replace with `{{ with .field }}{{ . }}{{ end }}`.
func scanViewFallbacks(path string) []checklistItem {
	out := []checklistItem{}
	data, err := os.ReadFile(path)
	if err != nil {
		return out
	}
	for i, line := range strings.Split(string(data), "\n") {
		// {{ or .x "literal" }} — the canonical fallback bug.
		if orFallbackRegex.MatchString(line) {
			out = append(out, checklistItem{
				ID: "view.fallback." + filepath.Base(filepath.Dir(path)),
				Severity: "fail", Pass: false,
				Message: fmt.Sprintf("%s:%d hardcoded fallback `{{ or .x \"…\" }}` — replace with `{{ with .x }}{{ . }}{{ end }}` so empty data renders empty (loud), not fake-complete (silent).",
					relTheme(path), i+1),
				File: path,
			})
		}
		// {{ if not .x }}LITERAL{{ end }} or {{ else }}LITERAL{{ end }}
		// pattern is harder to match with one regex; we flag the most
		// common form: an `else` followed by visible text before the
		// closing `{{end}}` on the same line.
		if elseLiteralRegex.MatchString(line) {
			out = append(out, checklistItem{
				ID: "view.fallback." + filepath.Base(filepath.Dir(path)),
				Severity: "warn", Pass: false,
				Message: fmt.Sprintf("%s:%d `{{ else }}LITERAL{{ end }}` — confirm the literal isn't standing in for missing data; prefer empty render so screenshots reflect reality.",
					relTheme(path), i+1),
				File: path,
			})
		}
	}
	return out
}

// orFallbackRegex matches `{{ or .field "literal" }}` (and the no-space
// variant). The literal is the second arg to `or` and is exactly the
// surface that fakes a complete render.
var orFallbackRegex = regexp.MustCompile(`\{\{-?\s*or\s+\.[\w.]+\s+"[^"]*"\s*-?\}\}`)

// elseLiteralRegex matches `{{ else }}<visible text>{{ end }}` on one
// line — a common fallback shape in small templates. Multi-line cases
// are missed; that's acceptable noise.
var elseLiteralRegex = regexp.MustCompile(`\{\{-?\s*else\s*-?\}\}\s*[A-Za-z][^{<]{2,}\s*\{\{-?\s*end\s*-?\}\}`)

func relTheme(p string) string {
	if i := strings.Index(p, "/themes/"); i >= 0 {
		return p[i+1:]
	}
	return p
}

// scanTengoScripts does a lightweight textual scan for the most common
// silent-failure patterns in seed scripts. It's deliberately not a parser;
// false positives are fine because every hit is a real footgun worth a look.
func scanTengoScripts(dir string) []checklistItem {
	out := []checklistItem{}
	if _, err := os.Stat(dir); err != nil {
		return out
	}
	_ = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".tengo" {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		text := string(data)
		if strings.Contains(text, "log.error(") {
			out = append(out, checklistItem{
				ID: "scripts.tengo.log_error", Severity: "fail", Pass: false,
				Message: "`log.error(` is a Tengo parse error (`error` is a reserved selector). Use `log.warn`, `log.info`, or alias `log.err`.",
				File:    path,
			})
		}
		// `fields:` at the top level of a node create/update is the #1
		// silent data drop. Heuristic: a line starts with `fields:` not
		// preceded by `// `, and the file references nodes.create or update.
		if (strings.Contains(text, "nodes.create") || strings.Contains(text, "nodes.update")) &&
			topLevelFieldsRegex.MatchString(text) {
			out = append(out, checklistItem{
				ID: "scripts.tengo.fields_typo", Severity: "warn", Pass: false,
				Message: "file calls nodes.create/update and contains a top-level `fields:` literal — confirm you didn't mean `fields_data:`. (Inside blocks_data, `fields:` is correct.)",
				File:    path,
			})
		}
		return nil
	})
	return out
}

// topLevelFieldsRegex matches a line whose first non-whitespace token is
// `fields:` (with optional indentation), excluding commented-out lines.
var topLevelFieldsRegex = regexp.MustCompile(`(?m)^\s*fields\s*:`)

// validateSettingsPages walks every entry in theme.json's settings_pages[],
// confirms the referenced JSON file exists, parses, and declares at least one
// well-formed field. Soft-fail at runtime (loader logs & skips), so the
// checklist surfaces these as fail-severity items so the author sees them
// before shipping a theme whose admin sidebar is silently empty.
func validateSettingsPages(themeDir string, pages []any) []checklistItem {
	out := []checklistItem{}
	for i, raw := range pages {
		page, ok := raw.(map[string]any)
		if !ok {
			out = append(out, checklistItem{
				ID: "theme.json.settings_pages", Severity: "fail", Pass: false,
				Message: fmt.Sprintf("settings_pages[%d] is not an object", i),
			})
			continue
		}
		slug, _ := page["slug"].(string)
		file, _ := page["file"].(string)
		if slug == "" {
			out = append(out, checklistItem{
				ID: "theme.json.settings_pages", Severity: "fail", Pass: false,
				Message: fmt.Sprintf("settings_pages[%d] is missing `slug`", i),
			})
			continue
		}
		if file == "" {
			out = append(out, checklistItem{
				ID: "theme.json.settings_pages." + slug, Severity: "fail", Pass: false,
				Message: fmt.Sprintf("settings_pages[%q] is missing `file`", slug),
			})
			continue
		}
		path := filepath.Join(themeDir, file)
		data, err := os.ReadFile(path)
		if err != nil {
			out = append(out, checklistItem{
				ID: "theme.json.settings_pages." + slug, Severity: "fail", Pass: false,
				Message: fmt.Sprintf("settings_pages[%q] file %s not readable: %v — loader will skip this page silently", slug, file, err),
				File:    path,
			})
			continue
		}
		var parsed map[string]any
		if err := json.Unmarshal(data, &parsed); err != nil {
			out = append(out, checklistItem{
				ID: "theme.json.settings_pages." + slug, Severity: "fail", Pass: false,
				Message: fmt.Sprintf("settings_pages[%q] file %s failed to parse: %v", slug, file, err),
				File:    path,
			})
			continue
		}
		fields, _ := parsed["fields"].([]any)
		if len(fields) == 0 {
			out = append(out, checklistItem{
				ID: "theme.json.settings_pages." + slug, Severity: "warn", Pass: false,
				Message: fmt.Sprintf("settings_pages[%q] file %s declares no fields — page will render empty in admin", slug, file),
				File:    path,
			})
			continue
		}
		for j, fraw := range fields {
			f, ok := fraw.(map[string]any)
			if !ok {
				out = append(out, checklistItem{
					ID: "theme.json.settings_pages." + slug, Severity: "fail", Pass: false,
					Message: fmt.Sprintf("settings_pages[%q] field #%d is not an object", slug, j),
					File:    path,
				})
				continue
			}
			key, _ := f["key"].(string)
			typ, _ := f["type"].(string)
			if key == "" || typ == "" {
				out = append(out, checklistItem{
					ID: "theme.json.settings_pages." + slug, Severity: "fail", Pass: false,
					Message: fmt.Sprintf("settings_pages[%q] field #%d missing `key` or `type` — loader will skip", slug, j),
					File:    path,
				})
			}
		}
	}
	return out
}

func summarize(slug, themeDir string, checks []checklistItem) map[string]any {
	passed, failed, warnings := 0, 0, 0
	for _, c := range checks {
		switch c.Severity {
		case "pass":
			passed++
		case "fail":
			failed++
		case "warn":
			warnings++
		}
	}
	return map[string]any{
		"slug":      slug,
		"theme_dir": themeDir,
		"checks":    checks,
		"summary": map[string]int{
			"passed":   passed,
			"failed":   failed,
			"warnings": warnings,
			"total":    len(checks),
		},
		"see_also": "docs/theme-checklist.md (manual checks: admin UX, public render visual sanity, idempotency)",
	}
}
