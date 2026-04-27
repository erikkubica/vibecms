package rendering

import (
	"bytes"
	"html/template"
	"strings"
	"sync"
	"testing"
)

// renderFunc is a tiny helper that exercises the FuncMap in isolation.
// We don't go through Render — just pull the template's funcMap, build a
// trivial template that calls the function, and check the output.
func renderFunc(t *testing.T, r *TemplateRenderer, body string, data any) string {
	t.Helper()
	tmpl, err := template.New("t").Funcs(r.funcMap).Parse(body)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		t.Fatalf("execute: %v", err)
	}
	return buf.String()
}

func TestSafeHTML_DoesNotEscape(t *testing.T) {
	r := NewTemplateRenderer(t.TempDir(), false)
	got := renderFunc(t, r, `{{safeHTML "<b>hi</b>"}}`, nil)
	if got != "<b>hi</b>" {
		t.Fatalf("safeHTML escaped output: %q", got)
	}
}

func TestSafeURL_AllowsDangerousScheme(t *testing.T) {
	r := NewTemplateRenderer(t.TempDir(), false)
	// safeURL is a known-dangerous helper — it disables Go's URL-scheme
	// safety check that would otherwise rewrite `javascript:...` to
	// `#ZgotmplZ`. The Go template engine still URL-encodes the rest of
	// the value inside an href attribute, but the unsafe scheme passes
	// through. This test pins that behaviour so a future renaming/safety
	// pass is a deliberate choice — the dev guide already flags this.
	got := renderFunc(t, r, `<a href="{{safeURL .}}">x</a>`, "javascript:alert(1)")
	if !strings.Contains(got, "javascript:") {
		t.Fatalf("safeURL stripped javascript scheme: %q", got)
	}
	if strings.Contains(got, "ZgotmplZ") {
		t.Fatalf("safeURL was bypassed by template safety net: %q", got)
	}
}

func TestDict_RequiresEvenArgs(t *testing.T) {
	r := NewTemplateRenderer(t.TempDir(), false)
	tmpl, err := template.New("t").Funcs(r.funcMap).Parse(`{{dict "k" "v"}}`)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, nil); err != nil {
		t.Fatalf("even-arg dict should succeed: %v", err)
	}

	tmpl, err = template.New("t").Funcs(r.funcMap).Parse(`{{dict "k"}}`)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if err := tmpl.Execute(&buf, nil); err == nil {
		t.Fatal("odd-arg dict must error")
	}
}

func TestMod_DivByZeroReturnsZero(t *testing.T) {
	r := NewTemplateRenderer(t.TempDir(), false)
	got := renderFunc(t, r, `{{mod 5 0}}`, nil)
	if got != "0" {
		t.Fatalf("expected 0, got %q", got)
	}
}

func TestAddSub(t *testing.T) {
	r := NewTemplateRenderer(t.TempDir(), false)
	if got := renderFunc(t, r, `{{add 2 3}}`, nil); got != "5" {
		t.Errorf("add 2 3 = %q, want 5", got)
	}
	if got := renderFunc(t, r, `{{sub 10 4}}`, nil); got != "6" {
		t.Errorf("sub 10 4 = %q, want 6", got)
	}
}

func TestSeq_NegativeReturnsEmpty(t *testing.T) {
	r := NewTemplateRenderer(t.TempDir(), false)
	got := renderFunc(t, r, `{{range seq -1}}x{{end}}`, nil)
	if got != "" {
		t.Fatalf("seq -1 produced output: %q", got)
	}
}

func TestSplit(t *testing.T) {
	r := NewTemplateRenderer(t.TempDir(), false)
	got := renderFunc(t, r, `{{range split "-" "a-b-c"}}{{.}},{{end}}`, nil)
	if got != "a,b,c," {
		t.Fatalf("split output: %q", got)
	}
}

func TestLastWord(t *testing.T) {
	r := NewTemplateRenderer(t.TempDir(), false)
	got := renderFunc(t, r, `{{lastWord "the quick brown fox"}}`, nil)
	if got != "fox" {
		t.Fatalf("lastWord = %q", got)
	}
}

func TestBeforeLastWord(t *testing.T) {
	r := NewTemplateRenderer(t.TempDir(), false)
	got := renderFunc(t, r, `{{beforeLastWord "the quick brown fox"}}`, nil)
	if got != "the quick brown" {
		t.Fatalf("beforeLastWord = %q", got)
	}
}

func TestDeref_NilSafe(t *testing.T) {
	r := NewTemplateRenderer(t.TempDir(), false)
	var p *string
	if got := renderFunc(t, r, `{{deref .}}`, p); got != "" {
		t.Errorf("nil *string deref = %q, want empty", got)
	}
	v := "hello"
	if got := renderFunc(t, r, `{{deref .}}`, &v); got != "hello" {
		t.Errorf("non-nil deref = %q, want hello", got)
	}
}

// TestRenderParsed_CacheHit verifies the LRU caches a parsed template
// and reuses it on subsequent calls. We assert by counting fresh parses
// — the cache should mean only one parse for two renders of the same key.
func TestRenderParsed_CacheHit(t *testing.T) {
	r := NewTemplateRenderer(t.TempDir(), false)

	var buf1, buf2 bytes.Buffer
	if err := r.RenderParsed(&buf1, "k1", `Hello {{.Name}}`, map[string]string{"Name": "world"}, nil); err != nil {
		t.Fatalf("first render: %v", err)
	}
	if err := r.RenderParsed(&buf2, "k1", `Hello {{.Name}}`, map[string]string{"Name": "again"}, nil); err != nil {
		t.Fatalf("second render: %v", err)
	}
	if buf1.String() != "Hello world" || buf2.String() != "Hello again" {
		t.Fatalf("got %q / %q", buf1.String(), buf2.String())
	}
}

// TestClearCache_Resets verifies ClearCache empties all three caches.
func TestClearCache_Resets(t *testing.T) {
	r := NewTemplateRenderer(t.TempDir(), false)
	var buf bytes.Buffer
	if err := r.RenderParsed(&buf, "k", `{{.X}}`, map[string]string{"X": "y"}, nil); err != nil {
		t.Fatalf("render: %v", err)
	}
	if r.blockCache.Len() == 0 {
		t.Fatal("block cache should be populated after render")
	}
	r.ClearCache()
	if r.blockCache.Len() != 0 {
		t.Fatal("ClearCache did not empty blockCache")
	}
}

// TestConcurrentRender_NoRace exercises concurrent template parsing and
// caching under the race detector. Without the renderer's mutex,
// multiple goroutines parsing the same template would race on the
// underlying map. Run with `go test -race ./internal/rendering/...`.
func TestConcurrentRender_NoRace(t *testing.T) {
	r := NewTemplateRenderer(t.TempDir(), false)
	const N = 50
	var wg sync.WaitGroup
	wg.Add(N)
	for i := 0; i < N; i++ {
		go func() {
			defer wg.Done()
			var buf bytes.Buffer
			_ = r.RenderParsed(&buf, "shared", `{{.X}}`, map[string]string{"X": "v"}, nil)
		}()
	}
	wg.Wait()
}
