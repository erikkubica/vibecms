package scripting

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"vibecms/internal/cms"
	"vibecms/internal/events"

	"github.com/d5/tengo/v2"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// ScriptEngine manages the Tengo scripting runtime for themes.
// It loads theme.tengo as the entry script, provides API modules for
// CMS operations, and manages event handler and HTTP route registrations.
type ScriptEngine struct {
	db         *gorm.DB
	eventBus   *events.EventBus
	contentSvc *cms.ContentService
	menuSvc    *cms.MenuService

	themeDir   string // path to active theme root
	scriptsDir string // themeDir + "/scripts"

	// Registered handlers populated during theme.tengo execution
	mu            sync.RWMutex
	eventHandlers map[string][]scriptHandler // event name -> handlers sorted by priority
	httpRoutes    []httpRoute
}

// NewScriptEngine creates a new ScriptEngine.
func NewScriptEngine(
	db *gorm.DB,
	eventBus *events.EventBus,
	contentSvc *cms.ContentService,
	menuSvc *cms.MenuService,
) *ScriptEngine {
	return &ScriptEngine{
		db:            db,
		eventBus:      eventBus,
		contentSvc:    contentSvc,
		menuSvc:       menuSvc,
		eventHandlers: make(map[string][]scriptHandler),
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

	// Compile and execute the entry script
	script := tengo.NewScript(src)
	script.SetImports(e.buildModuleMap(nil))

	// Set a max execution time for safety
	script.SetMaxAllocs(50000)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err = script.RunContext(ctx)
	if err != nil {
		return fmt.Errorf("executing theme.tengo: %w", err)
	}

	// Wire event handlers to the EventBus
	e.subscribeEventHandlers()

	e.mu.RLock()
	numEvents := 0
	for _, handlers := range e.eventHandlers {
		numEvents += len(handlers)
	}
	numRoutes := len(e.httpRoutes)
	e.mu.RUnlock()

	log.Printf("[script] theme scripts loaded: %d event handlers, %d HTTP routes", numEvents, numRoutes)

	return nil
}

// MountHTTPRoutes registers all script-defined HTTP endpoints on the Fiber app.
// Must be called after LoadThemeScripts.
func (e *ScriptEngine) MountHTTPRoutes(app *fiber.App) {
	e.MountRoutes(app)
}

// runScript compiles and executes a Tengo script file with the given context variables.
// Script path is relative to the theme's scripts/ directory (without .tengo extension).
// renderCtx is the optional template render context (for cms/routing module); pass nil if not in a render.
// Returns the value of the "response" variable after execution.
func (e *ScriptEngine) runScript(scriptPath string, vars map[string]interface{}, renderCtx interface{}) (interface{}, error) {
	fullPath := filepath.Join(e.scriptsDir, scriptPath+".tengo")

	src, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, fmt.Errorf("script not found: %s: %w", scriptPath, err)
	}

	script := tengo.NewScript(src)
	script.SetImports(e.buildModuleMap(renderCtx))
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
