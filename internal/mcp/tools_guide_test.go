package mcp

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

// TestBuildGuide_CompactMenuFitsBudget pins the token-shaped budget for the
// default core.guide() call. The compact menu MUST stay small enough that
// 125k–250k context models can call it as their first tool without triggering
// per-tool token dumps. Hard ceiling: 32 KB JSON ≈ ~8k tokens. Today's payload
// runs ~6 KB; the headroom soaks up tool growth before we have to revisit the
// shape.
func TestBuildGuide_CompactMenuFitsBudget(t *testing.T) {
	s := &Server{toolCatalog: synthCatalog()}
	out, err := s.buildGuide(context.Background(), "", false)
	if err != nil {
		t.Fatalf("buildGuide: %v", err)
	}
	if got, want := out["mode"], "menu"; got != want {
		t.Errorf("mode = %q, want %q", got, want)
	}
	requireKeys(t, out, "available_topics", "recipe_index", "gotcha_topics",
		"data_shapes", "conventions", "snapshot", "tools_by_domain", "next_step")
	requireAbsent(t, out, "recipes", "gotchas", "editing_playbook", "tool_index")

	size := jsonSize(t, out)
	const ceiling = 32 * 1024
	if size > ceiling {
		t.Errorf("compact menu = %d bytes, ceiling %d (~8k tokens). Trim recipe goals / gotcha topics / tool names.", size, ceiling)
	}
	t.Logf("compact menu: %d bytes", size)
}

// TestBuildGuide_TopicMode pins the focused-topic budget. Topic mode includes
// full recipes for that topic + relevant gotchas + tools for the topic's
// domains. Should fit comfortably under 64 KB ≈ ~16k tokens.
func TestBuildGuide_TopicMode(t *testing.T) {
	s := &Server{toolCatalog: synthCatalog()}
	for _, topic := range []string{"pages", "editing", "themes", "extensions", "media", "blocks", "taxonomies"} {
		out, err := s.buildGuide(context.Background(), topic, false)
		if err != nil {
			t.Fatalf("buildGuide(%q): %v", topic, err)
		}
		if got := out["mode"]; got != "topic" {
			t.Errorf("topic %q: mode = %q, want topic", topic, got)
		}
		if got := out["topic"]; got != topic {
			t.Errorf("topic field round-trip = %v, want %q", got, topic)
		}
		requireKeys(t, out, "recipes", "gotchas", "tools")
		size := jsonSize(t, out)
		const ceiling = 64 * 1024
		if size > ceiling {
			t.Errorf("topic %q = %d bytes, ceiling %d (~16k tokens)", topic, size, ceiling)
		}
		t.Logf("topic %q: %d bytes", topic, size)
	}
}

// TestBuildGuide_VerboseEscapeHatch confirms verbose=true returns the full
// reference and is the only mode that does. Size is unbudgeted on purpose —
// verbose is the explicit "give me everything" path.
func TestBuildGuide_VerboseEscapeHatch(t *testing.T) {
	s := &Server{toolCatalog: synthCatalog()}
	out, err := s.buildGuide(context.Background(), "", true)
	if err != nil {
		t.Fatalf("buildGuide: %v", err)
	}
	if got := out["mode"]; got != "verbose" {
		t.Errorf("mode = %q, want verbose", got)
	}
	requireKeys(t, out, "recipes", "gotchas", "editing_playbook", "tool_index")
	t.Logf("verbose: %d bytes", jsonSize(t, out))
}

// TestThemeStandards_CompactDropsExamples confirms the compact mode strips the
// embedded source examples (~2k tokens) and replaces them with a pointer.
// Compact must fit under 32 KB ≈ ~8k tokens to stay below per-tool dump
// thresholds; verbose is unbudgeted on purpose.
func TestThemeStandards_CompactDropsExamples(t *testing.T) {
	out := themeStandards(false)
	if _, has := out["examples"]; has {
		t.Error("compact themeStandards should not include `examples`")
	}
	if _, has := out["seeding_patterns"]; has {
		t.Error("compact themeStandards should not include `seeding_patterns`")
	}
	if _, has := out["examples_pointer"]; !has {
		t.Error("compact themeStandards should include an examples_pointer breadcrumb")
	}
	if size := jsonSize(t, out); size > 32*1024 {
		t.Errorf("compact themeStandards = %d bytes, ceiling 32 KB", size)
	} else {
		t.Logf("compact themeStandards: %d bytes", size)
	}

	verbose := themeStandards(true)
	if _, has := verbose["examples"]; !has {
		t.Error("verbose themeStandards must include `examples`")
	}
	if _, has := verbose["seeding_patterns"]; !has {
		t.Error("verbose themeStandards must include `seeding_patterns`")
	}
	t.Logf("verbose themeStandards: %d bytes", jsonSize(t, verbose))
}

// TestExtensionStandards_CompactDropsHeavySections — symmetric with theme
// standards. Compact omits event_modes detail, sdui_reactivity, hot_deploy.
func TestExtensionStandards_CompactDropsHeavySections(t *testing.T) {
	out := extensionStandards(false)
	for _, k := range []string{"event_modes", "sdui_reactivity", "hot_deploy"} {
		if _, has := out[k]; has {
			t.Errorf("compact extensionStandards leaks `%s`", k)
		}
	}
	if _, has := out["verbose_pointer"]; !has {
		t.Error("compact extensionStandards should include a verbose_pointer breadcrumb")
	}
	if size := jsonSize(t, out); size > 32*1024 {
		t.Errorf("compact extensionStandards = %d bytes, ceiling 32 KB", size)
	} else {
		t.Logf("compact extensionStandards: %d bytes", size)
	}

	verbose := extensionStandards(true)
	for _, k := range []string{"event_modes", "sdui_reactivity", "hot_deploy"} {
		if _, has := verbose[k]; !has {
			t.Errorf("verbose extensionStandards missing `%s`", k)
		}
	}
	t.Logf("verbose extensionStandards: %d bytes", jsonSize(t, verbose))
}

func TestToolDomain(t *testing.T) {
	cases := map[string]string{
		"core.node.create":           "node",
		"core.media.upload_init":     "media",
		"core.theme.standards":       "theme",
		"core.guide":                 "guide",
		"missing.namespace":          "",
		"":                           "",
		"core":                       "",
	}
	for in, want := range cases {
		if got := toolDomain(in); got != want {
			t.Errorf("toolDomain(%q) = %q, want %q", in, got, want)
		}
	}
}

// synthCatalog mirrors the rough shape of the real tool catalog so the size
// budgets test what production agents will actually see. Keep this in sync if
// the real registration count grows by more than ~30 tools.
func synthCatalog() []toolCatalogEntry {
	domains := []string{"node", "nodetype", "media", "theme", "extension",
		"taxonomy", "term", "block_types", "layout", "menu", "settings",
		"data", "files", "events", "filters", "render", "users", "http",
		"field_types", "guide", "email"}
	out := make([]toolCatalogEntry, 0, 96)
	for _, d := range domains {
		for i := 0; i < 5; i++ {
			out = append(out, toolCatalogEntry{
				Name:        "core." + d + "." + verb(i),
				Description: strings.Repeat("describe-this-tool. ", 12),
				Class:       "read",
			})
		}
	}
	return out
}

func verb(i int) string {
	v := []string{"get", "list", "create", "update", "delete"}
	return v[i%len(v)]
}

func jsonSize(t *testing.T, v any) int {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	return len(b)
}

func requireKeys(t *testing.T, m map[string]any, keys ...string) {
	t.Helper()
	for _, k := range keys {
		if _, ok := m[k]; !ok {
			t.Errorf("missing key: %q", k)
		}
	}
}

func requireAbsent(t *testing.T, m map[string]any, keys ...string) {
	t.Helper()
	for _, k := range keys {
		if _, ok := m[k]; ok {
			t.Errorf("unexpected key in compact menu: %q", k)
		}
	}
}
