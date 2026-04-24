package scripting

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"vibecms/internal/coreapi"
	"vibecms/internal/events"

	"github.com/d5/tengo/v2"
	"github.com/gofiber/fiber/v2"
)

// ScriptEngine manages the Tengo scripting runtime for themes.
// It loads theme.tengo as the entry script, provides API modules for
// CMS operations, and manages event handler and HTTP route registrations.
type ScriptEngine struct {
	eventBus *events.EventBus
	coreAPI  coreapi.CoreAPI

	themeDir   string // path to active theme root
	scriptsDir string // themeDir + "/scripts"

	// activeScriptsDir is set during LoadThemeScripts/LoadExtensionScripts
	// and used by registration functions to set baseDir on handlers.
	activeScriptsDir string

	// activeSlug tracks the current extension/theme slug being loaded
	activeSlug string

	// Registered handlers populated during theme.tengo execution
	mu             sync.RWMutex
	eventHandlers  map[string][]scriptHandler // event name -> handlers sorted by priority
	filterHandlers map[string][]scriptHandler // filter name -> handlers sorted by priority
	httpRoutes     []httpRoute
	wellKnown      []wellKnownRoute
}

// WellKnownRegistrar is satisfied by cms.WellKnownRegistry. Declared here
// to avoid a scripting → cms import cycle.
type WellKnownRegistrar interface {
	Register(path string, handler fiber.Handler)
}

// NewScriptEngine creates a new ScriptEngine.
func NewScriptEngine(
	eventBus *events.EventBus,
	coreAPI coreapi.CoreAPI,
) *ScriptEngine {
	return &ScriptEngine{
		eventBus:       eventBus,
		coreAPI:        coreAPI,
		eventHandlers:  make(map[string][]scriptHandler),
		filterHandlers: make(map[string][]scriptHandler),
	}
}

// scriptCallbacks returns a ScriptCallbacks that wires into the engine's
// registration methods (EventsOn, FiltersAdd, HTTPRegister).
func (e *ScriptEngine) scriptCallbacks() *coreapi.ScriptCallbacks {
	return &coreapi.ScriptCallbacks{
		OnEvent:     e.EventsOn,
		OnFilter:    e.FiltersAdd,
		OnRoute:     e.HTTPRegister,
		OnWellKnown: e.WellKnownRegister,
	}
}

// LoadThemeScripts loads and executes the theme's entry script (scripts/theme.tengo).
// This populates event handler and HTTP route registrations.
// Returns nil if no scripts directory or entry script exists (scripts are optional).
func (e *ScriptEngine) LoadThemeScripts(themeDir string) error {
	e.themeDir = themeDir
	e.scriptsDir = filepath.Join(themeDir, "scripts")

	entryScript := filepath.Join(e.scriptsDir, "theme.tengo")

	// Check if scripts directory and entry script exist
	if _, err := os.Stat(entryScript); os.IsNotExist(err) {
		log.Printf("[script] no theme.tengo found at %s, scripting disabled", entryScript)
		return nil
	}

	src, err := os.ReadFile(entryScript)
	if err != nil {
		return fmt.Errorf("reading theme.tengo: %w", err)
	}

	log.Printf("[script] loading theme scripts from %s", e.scriptsDir)

	// Set activeScriptsDir so registration functions capture it on handlers.
	e.activeScriptsDir = e.scriptsDir
	e.activeSlug = "theme"

	// Compile and execute the entry script
	script := tengo.NewScript(src)
	caller := coreapi.CallerInfo{Slug: "theme", Type: "tengo", Capabilities: nil}
	script.SetImports(coreapi.BuildTengoModules(e.coreAPI, caller, nil, e.scriptsDir, e.scriptCallbacks()))

	// Set a max execution time for safety
	script.SetMaxAllocs(50000)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err = script.RunContext(ctx)
	if err != nil {
		e.activeScriptsDir = ""
		e.activeSlug = ""
		return fmt.Errorf("executing theme.tengo: %w", err)
	}

	e.activeScriptsDir = ""
	e.activeSlug = ""

	// Wire event handlers to the EventBus
	e.subscribeEventHandlers()

	e.mu.RLock()
	numEvents := 0
	for _, handlers := range e.eventHandlers {
		numEvents += len(handlers)
	}
	numFilters := 0
	for _, handlers := range e.filterHandlers {
		numFilters += len(handlers)
	}
	numRoutes := len(e.httpRoutes)
	e.mu.RUnlock()

	log.Printf("[script] theme scripts loaded: %d event handlers, %d filters, %d HTTP routes", numEvents, numFilters, numRoutes)

	return nil
}

// MountHTTPRoutes registers all script-defined HTTP endpoints on the Fiber app.
// Must be called after LoadThemeScripts.
func (e *ScriptEngine) MountHTTPRoutes(app *fiber.App) {
	e.MountRoutes(app)
}

// runScript compiles and executes a Tengo script file with the given context variables.
// Script path is relative to the scripts/ directory (without .tengo extension).
// renderCtx is the optional template render context (for cms/routing module); pass nil if not in a render.
// baseDir is an optional scripts directory override; if empty or not provided, e.scriptsDir (theme default) is used.
// Returns the value of the "response" variable after execution.
func (e *ScriptEngine) runScript(scriptPath string, vars map[string]interface{}, renderCtx interface{}, baseDir ...string) (interface{}, error) {
	scriptsDir := e.scriptsDir
	if len(baseDir) > 0 && baseDir[0] != "" {
		scriptsDir = baseDir[0]
	}

	fullPath := filepath.Join(scriptsDir, scriptPath+".tengo")

	src, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, fmt.Errorf("script not found: %s: %w", scriptPath, err)
	}

	script := tengo.NewScript(src)
	caller := coreapi.CallerInfo{Slug: e.activeSlug, Type: "tengo", Capabilities: nil}
	script.SetImports(coreapi.BuildTengoModules(e.coreAPI, caller, renderCtx, scriptsDir, e.scriptCallbacks()))
	script.SetMaxAllocs(50000)

	// Inject context variables — convert to Tengo objects first to handle
	// types that Tengo's FromInterface can't auto-convert (e.g. map[string]string).
	for k, v := range vars {
		obj := goToTengo(normalizeForTengo(v))
		if err := script.Add(k, obj); err != nil {
			log.Printf("[script] warning: failed to inject variable %q: %v", k, err)
		}
	}

	// Add response placeholder
	_ = script.Add("response", nil)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	compiled, err := script.RunContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("script execution error (%s): %w", scriptPath, err)
	}

	// Read back the response variable
	respVar := compiled.Get("response")
	if respVar == nil {
		return nil, nil
	}

	obj := respVar.Object()
	if obj == nil || obj == tengo.UndefinedValue {
		return nil, nil
	}

	return tengoToGo(obj), nil
}

// LoadExtensionScripts loads and executes an extension's entry script (scripts/extension.tengo).
// This works like LoadThemeScripts but uses the extension's scripts/ directory for module resolution.
// capabilities is the set of permissions declared in the extension manifest.
// Returns nil if no scripts directory or entry script exists.
func (e *ScriptEngine) LoadExtensionScripts(extDir string, slug string, capabilities ...map[string]bool) error {
	extScriptsDir := filepath.Join(extDir, "scripts")
	entryScript := filepath.Join(extScriptsDir, "extension.tengo")

	if _, err := os.Stat(entryScript); os.IsNotExist(err) {
		log.Printf("[script] no extension.tengo found for %s at %s", slug, entryScript)
		return nil
	}

	src, err := os.ReadFile(entryScript)
	if err != nil {
		return fmt.Errorf("reading extension.tengo for %s: %w", slug, err)
	}

	log.Printf("[script] loading extension scripts for %s from %s", slug, extScriptsDir)

	// Set activeScriptsDir so registration functions capture it on handlers.
	e.activeScriptsDir = extScriptsDir
	e.activeSlug = slug

	var caps map[string]bool
	if len(capabilities) > 0 {
		caps = capabilities[0]
	}

	caller := coreapi.CallerInfo{Slug: slug, Type: "tengo", Capabilities: caps}
	script := tengo.NewScript(src)
	script.SetImports(coreapi.BuildTengoModules(e.coreAPI, caller, nil, extScriptsDir, e.scriptCallbacks()))
	script.SetMaxAllocs(50000)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err = script.RunContext(ctx)
	if err != nil {
		e.activeScriptsDir = ""
		e.activeSlug = ""
		return fmt.Errorf("executing extension.tengo for %s: %w", slug, err)
	}

	e.activeScriptsDir = ""
	e.activeSlug = ""

	// Wire any new event handlers to the EventBus
	e.subscribeEventHandlers()

	log.Printf("[script] extension %s scripts loaded", slug)
	return nil
}

// UnloadExtensionScripts removes all event and filter handlers registered by an extension.
func (e *ScriptEngine) UnloadExtensionScripts(extDir string, slug string) {
	extScriptsDir := filepath.Join(extDir, "scripts")

	e.mu.Lock()
	defer e.mu.Unlock()

	// Remove event handlers from this extension
	for name, handlers := range e.eventHandlers {
		filtered := make([]scriptHandler, 0, len(handlers))
		for _, h := range handlers {
			if h.baseDir != extScriptsDir {
				filtered = append(filtered, h)
			}
		}
		e.eventHandlers[name] = filtered
	}

	// Remove filter handlers from this extension
	for name, handlers := range e.filterHandlers {
		filtered := make([]scriptHandler, 0, len(handlers))
		for _, h := range handlers {
			if h.baseDir != extScriptsDir {
				filtered = append(filtered, h)
			}
		}
		e.filterHandlers[name] = filtered
	}

	log.Printf("[script] extension %s scripts unloaded", slug)
}

// UnloadThemeScripts removes all event handlers, filter handlers, and HTTP
// routes that were registered by the currently loaded theme's theme.tengo.
// Call this before loading a new theme so stale handlers don't leak.
func (e *ScriptEngine) UnloadThemeScripts() {
	e.mu.Lock()
	defer e.mu.Unlock()

	scriptsDir := e.scriptsDir
	if scriptsDir == "" {
		return
	}

	// Remove event handlers from the theme
	for name, handlers := range e.eventHandlers {
		filtered := make([]scriptHandler, 0, len(handlers))
		for _, h := range handlers {
			if h.baseDir != scriptsDir {
				filtered = append(filtered, h)
			}
		}
		e.eventHandlers[name] = filtered
	}

	// Remove filter handlers from the theme
	for name, handlers := range e.filterHandlers {
		filtered := make([]scriptHandler, 0, len(handlers))
		for _, h := range handlers {
			if h.baseDir != scriptsDir {
				filtered = append(filtered, h)
			}
		}
		e.filterHandlers[name] = filtered
	}

	// Remove HTTP routes from the theme
	filteredRoutes := make([]httpRoute, 0, len(e.httpRoutes))
	for _, r := range e.httpRoutes {
		if r.baseDir != scriptsDir {
			filteredRoutes = append(filteredRoutes, r)
		}
	}
	e.httpRoutes = filteredRoutes

	e.themeDir = ""
	e.scriptsDir = ""

	log.Printf("[script] theme scripts unloaded")
}
