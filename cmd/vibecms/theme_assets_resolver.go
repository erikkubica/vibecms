package main

import (
	"path/filepath"
	"sync/atomic"

	"vibecms/internal/events"

	"gorm.io/gorm"
)

// themeAssetsResolver holds the filesystem directory that /theme/assets/* maps
// to. Because the active theme can change at runtime (via the theme activation
// endpoint), we swap the value on theme.activated events instead of binding a
// static route at startup.
type themeAssetsResolver struct {
	dir atomic.Pointer[string]
	db  *gorm.DB
}

func newThemeAssetsResolver(db *gorm.DB, bus *events.EventBus, initial string) *themeAssetsResolver {
	r := &themeAssetsResolver{db: db}
	r.set(filepath.Join(initial, "assets"))
	// Refresh on first miss too, so a fresh DB that becomes populated after
	// startup eventually converges.
	r.refresh()
	bus.Subscribe("theme.activated", func(_ string, payload events.Payload) {
		if path, ok := payload["path"].(string); ok && path != "" {
			r.set(filepath.Join(path, "assets"))
			return
		}
		r.refresh()
	})
	return r
}

// Get returns the current assets directory.
func (r *themeAssetsResolver) Get() string {
	if p := r.dir.Load(); p != nil {
		return *p
	}
	return ""
}

func (r *themeAssetsResolver) set(dir string) {
	r.dir.Store(&dir)
}

// refresh re-reads the active theme from the DB. Called when a theme.activated
// event arrives without a usable path payload.
func (r *themeAssetsResolver) refresh() {
	var row struct{ Path string }
	if err := r.db.Raw("SELECT path FROM themes WHERE is_active = true LIMIT 1").Scan(&row).Error; err == nil && row.Path != "" {
		r.set(filepath.Join(row.Path, "assets"))
	}
}
