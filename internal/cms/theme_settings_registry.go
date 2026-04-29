package cms

import "sync"

// ThemeSettingsRegistry holds the in-memory snapshot of the active theme's
// settings pages. Updated atomically on theme activation; read by the admin
// HTTP handler, the render-context builder, and the Tengo bridge. Empty
// when no theme is active or the active theme declares no settings_pages.
type ThemeSettingsRegistry struct {
	mu         sync.RWMutex
	activeSlug string
	pages      []ThemeSettingsPage
}

// NewThemeSettingsRegistry constructs an empty registry.
func NewThemeSettingsRegistry() *ThemeSettingsRegistry { return &ThemeSettingsRegistry{} }

// SetActive replaces the registry contents with the given theme slug + pages.
// Slices are copied so subsequent caller mutations cannot affect registry state.
func (r *ThemeSettingsRegistry) SetActive(themeSlug string, pages []ThemeSettingsPage) {
	cp := make([]ThemeSettingsPage, len(pages))
	copy(cp, pages)
	r.mu.Lock()
	defer r.mu.Unlock()
	r.activeSlug = themeSlug
	r.pages = cp
}

// Clear empties the registry — call on theme deactivation.
func (r *ThemeSettingsRegistry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.activeSlug = ""
	r.pages = nil
}

// ActiveSlug returns the slug of the currently active theme, or "" when none.
func (r *ThemeSettingsRegistry) ActiveSlug() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.activeSlug
}

// ActivePages returns a defensive copy of the active theme's settings pages.
func (r *ThemeSettingsRegistry) ActivePages() []ThemeSettingsPage {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]ThemeSettingsPage, len(r.pages))
	copy(out, r.pages)
	return out
}

// ActivePage returns the page with the given slug, or (zero, false) if missing.
func (r *ThemeSettingsRegistry) ActivePage(slug string) (ThemeSettingsPage, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, p := range r.pages {
		if p.Slug == slug {
			return p, true
		}
	}
	return ThemeSettingsPage{}, false
}
