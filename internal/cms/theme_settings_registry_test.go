package cms

import (
	"sync"
	"testing"
)

func samplePages() []ThemeSettingsPage {
	return []ThemeSettingsPage{
		{Slug: "general", Name: "General"},
		{Slug: "social", Name: "Social"},
	}
}

func TestRegistry_SetActiveAndActivePages(t *testing.T) {
	r := NewThemeSettingsRegistry()
	r.SetActive("squilla-default", samplePages())

	if got := r.ActiveSlug(); got != "squilla-default" {
		t.Fatalf("ActiveSlug = %q, want %q", got, "squilla-default")
	}
	pages := r.ActivePages()
	if len(pages) != 2 {
		t.Fatalf("len(ActivePages) = %d, want 2", len(pages))
	}
	if pages[0].Slug != "general" || pages[1].Slug != "social" {
		t.Fatalf("page order wrong: %+v", pages)
	}
}

func TestRegistry_ActivePagesReturnsCopy(t *testing.T) {
	r := NewThemeSettingsRegistry()
	r.SetActive("t", samplePages())

	got := r.ActivePages()
	got[0].Slug = "mutated"

	again := r.ActivePages()
	if again[0].Slug != "general" {
		t.Fatalf("registry state mutated through returned slice: %q", again[0].Slug)
	}
}

func TestRegistry_SetActiveCopiesInput(t *testing.T) {
	r := NewThemeSettingsRegistry()
	in := samplePages()
	r.SetActive("t", in)

	in[0].Slug = "mutated"

	got := r.ActivePages()
	if got[0].Slug != "general" {
		t.Fatalf("registry state mutated through caller's input slice: %q", got[0].Slug)
	}
}

func TestRegistry_ActivePage_FoundAndNotFound(t *testing.T) {
	r := NewThemeSettingsRegistry()
	r.SetActive("t", samplePages())

	page, ok := r.ActivePage("social")
	if !ok {
		t.Fatalf("ActivePage(social) not found")
	}
	if page.Name != "Social" {
		t.Fatalf("ActivePage(social).Name = %q, want %q", page.Name, "Social")
	}

	zero, ok := r.ActivePage("missing")
	if ok {
		t.Fatalf("ActivePage(missing) returned ok=true")
	}
	if zero.Slug != "" || zero.Name != "" {
		t.Fatalf("ActivePage(missing) returned non-zero value: %+v", zero)
	}
}

func TestRegistry_Clear(t *testing.T) {
	r := NewThemeSettingsRegistry()
	r.SetActive("t", samplePages())
	r.Clear()

	if r.ActiveSlug() != "" {
		t.Fatalf("ActiveSlug after Clear = %q, want empty", r.ActiveSlug())
	}
	if pages := r.ActivePages(); len(pages) != 0 {
		t.Fatalf("ActivePages after Clear has %d entries, want 0", len(pages))
	}
}

func TestRegistry_ConcurrentReadsAreSafe(t *testing.T) {
	r := NewThemeSettingsRegistry()
	r.SetActive("t", samplePages())

	var wg sync.WaitGroup
	stop := make(chan struct{})

	// Writer goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			select {
			case <-stop:
				return
			default:
			}
			r.SetActive("t", samplePages())
		}
	}()

	// 50 reader goroutines
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				_ = r.ActivePages()
				_, _ = r.ActivePage("general")
				_ = r.ActiveSlug()
			}
		}()
	}

	wg.Wait()
	close(stop)
}
