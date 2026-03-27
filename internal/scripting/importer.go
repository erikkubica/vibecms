package scripting

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/d5/tengo/v2"
	"github.com/d5/tengo/v2/stdlib"
)

// buildModuleMap creates a ModuleMap with all API modules and theme source modules.
// renderCtx is the optional template render context passed to cms/routing; nil outside renders.
func (e *ScriptEngine) buildModuleMap(renderCtx interface{}) *tengo.ModuleMap {
	modules := tengo.NewModuleMap()

	// Register built-in API modules
	modules.AddBuiltinModule("cms/nodes", e.nodesModule())
	modules.AddBuiltinModule("cms/settings", e.settingsModule())
	modules.AddBuiltinModule("cms/events", e.eventsModule())
	modules.AddBuiltinModule("cms/http", e.httpModule())
	modules.AddBuiltinModule("cms/email", e.emailModule())
	modules.AddBuiltinModule("cms/menus", e.menusModule())
	modules.AddBuiltinModule("cms/log", logModule())
	modules.AddBuiltinModule("cms/routing", e.routingModule(renderCtx))
	modules.AddBuiltinModule("cms/helpers", helpersModule())

	// Register standard Tengo stdlib modules (safe subset — no OS/file access)
	safeModules := []string{"fmt", "math", "text", "times", "rand", "json", "base64", "hex", "enum"}
	for _, name := range safeModules {
		if mod := stdlib.BuiltinModules[name]; mod != nil {
			modules.AddBuiltinModule(name, mod)
		}
		if mod := stdlib.SourceModules[name]; mod != "" {
			modules.AddSourceModule(name, []byte(mod))
		}
	}

	// Load theme source modules from scripts/ directory
	e.loadSourceModules(modules)

	return modules
}

// loadSourceModules recursively scans the scripts directory and adds all .tengo
// files as source modules. Module names use paths relative to scripts/ with a
// "./" prefix (e.g., "./lib/helpers" for scripts/lib/helpers.tengo).
func (e *ScriptEngine) loadSourceModules(modules *tengo.ModuleMap) {
	if e.scriptsDir == "" {
		return
	}

	filepath.Walk(e.scriptsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".tengo") {
			return nil
		}

		// Skip the entry script itself — it's not importable
		if filepath.Base(path) == "theme.tengo" && filepath.Dir(path) == e.scriptsDir {
			return nil
		}

		// Calculate relative path from scripts dir
		relPath, err := filepath.Rel(e.scriptsDir, path)
		if err != nil {
			return nil
		}

		// Remove .tengo extension and add "./" prefix
		moduleName := "./" + strings.TrimSuffix(relPath, ".tengo")
		// Normalize path separators
		moduleName = strings.ReplaceAll(moduleName, string(filepath.Separator), "/")

		src, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		modules.AddSourceModule(moduleName, src)
		return nil
	})
}
