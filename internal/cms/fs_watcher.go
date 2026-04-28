package cms

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// readDirNames returns the names of the immediate children (files and dirs) of
// dir. Used by the watcher to enumerate child manifests at startup and after
// each rescan.
func readDirNames(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	return names, nil
}

// DropInWatcher watches a parent directory (themes/ or extensions/) for new
// subdirectories or manifest writes, and triggers a debounced rescan callback.
//
// Use case: an operator does `docker cp my-theme/. squilla:/app/themes/my-theme/`
// or mounts a host volume, and the new package becomes visible to the loader
// without an app restart. fsnotify observes the FS events; the callback wraps
// the existing ScanAndRegister and is idempotent (upsert), so repeated calls
// during a slow upload are safe.
//
// The watcher only observes the parent directory and the manifest file inside
// each child directory — it does NOT recursively watch every file under a
// theme/extension. That keeps inotify watches O(top-level dirs).
type DropInWatcher struct {
	root       string // absolute or relative path being watched
	manifest   string // manifest filename to also watch (e.g. "theme.json")
	rescan     func()
	debounce   time.Duration
	w          *fsnotify.Watcher
	mu         sync.Mutex
	pending    *time.Timer
	cancelOnce sync.Once
}

// NewDropInWatcher constructs a watcher. Call Start to begin observing.
//
// rescan is invoked from a single goroutine after a debounce window has
// elapsed since the last relevant FS event — never concurrently with itself.
func NewDropInWatcher(root, manifest string, rescan func()) *DropInWatcher {
	return &DropInWatcher{
		root:     root,
		manifest: manifest,
		rescan:   rescan,
		debounce: 750 * time.Millisecond,
	}
}

// Start begins watching. Cancel ctx to stop. Errors creating the underlying
// watcher are logged and the function returns nil — drop-in is a quality-of-
// life feature, not a hard dependency, and we'd rather boot without it than
// crash the whole app.
func (d *DropInWatcher) Start(ctx context.Context) error {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		log.Printf("[fs-watcher] %s: failed to create watcher: %v", d.root, err)
		return nil
	}
	d.w = w

	// Watch the root for new subdirectory creation.
	if err := w.Add(d.root); err != nil {
		log.Printf("[fs-watcher] %s: failed to add root: %v", d.root, err)
		_ = w.Close()
		return nil
	}

	// Also watch any manifest file already present, so editing an existing
	// theme/extension's manifest in place (rare, but useful for dev) triggers
	// a refresh too.
	if entries, err := readDirNames(d.root); err == nil {
		for _, name := range entries {
			d.watchChild(filepath.Join(d.root, name))
		}
	}

	go d.loop(ctx)
	return nil
}

func (d *DropInWatcher) loop(ctx context.Context) {
	defer func() {
		d.cancelOnce.Do(func() {
			if d.w != nil {
				_ = d.w.Close()
			}
		})
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case err, ok := <-d.w.Errors:
			if !ok {
				return
			}
			log.Printf("[fs-watcher] %s: error: %v", d.root, err)
		case ev, ok := <-d.w.Events:
			if !ok {
				return
			}
			d.handle(ev)
		}
	}
}

// handle decides whether a single event is relevant and schedules a debounced
// rescan if so. We treat any of the following as worth rescanning:
//   - a new top-level subdirectory under root
//   - any write/create of <root>/<child>/<manifest>
//   - removal of a top-level subdirectory (so deregistration logic, when added
//     later, can react)
func (d *DropInWatcher) handle(ev fsnotify.Event) {
	rel, err := filepath.Rel(d.root, ev.Name)
	if err != nil {
		return
	}
	rel = filepath.ToSlash(rel)

	// Ignore hidden files and editor turds.
	base := filepath.Base(ev.Name)
	if strings.HasPrefix(base, ".") || strings.HasSuffix(base, "~") {
		return
	}

	// Top-level dir Create: start watching its manifest, and rescan.
	if !strings.Contains(rel, "/") && ev.Op&(fsnotify.Create) != 0 {
		d.watchChild(ev.Name)
		d.scheduleRescan()
		return
	}

	// Manifest writes inside a child dir: rescan.
	if filepath.Base(ev.Name) == d.manifest {
		d.scheduleRescan()
		return
	}

	// Top-level removal: still useful to schedule, in case scan ever learns to
	// deregister missing rows.
	if !strings.Contains(rel, "/") && ev.Op&(fsnotify.Remove|fsnotify.Rename) != 0 {
		d.scheduleRescan()
	}
}

// watchChild adds a watch on <child>/<manifest> if it exists, so editing the
// manifest in place fires Write events. Failure is non-fatal; the new-dir
// Create event already covers the common path.
func (d *DropInWatcher) watchChild(childDir string) {
	if d.w == nil {
		return
	}
	manifestPath := filepath.Join(childDir, d.manifest)
	if err := d.w.Add(manifestPath); err != nil {
		// File may not exist yet — that's fine, the directory Create event
		// already triggered scheduleRescan and the next manifest write will
		// be picked up after the rescan re-walks.
		return
	}
}

// scheduleRescan coalesces bursts of FS events (a typical `cp -r` fires dozens)
// into a single rescan call after the debounce window elapses.
func (d *DropInWatcher) scheduleRescan() {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.pending != nil {
		d.pending.Stop()
	}
	d.pending = time.AfterFunc(d.debounce, func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[fs-watcher] %s: rescan panicked: %v", d.root, r)
			}
		}()
		d.rescan()

		// Re-walk children so any directory created since the last walk has
		// its manifest watched too.
		if entries, err := readDirNames(d.root); err == nil {
			for _, name := range entries {
				d.watchChild(filepath.Join(d.root, name))
			}
		}
	})
}
