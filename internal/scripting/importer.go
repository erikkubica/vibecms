package scripting

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/d5/tengo/v2"
	"github.com/d5/tengo/v2/stdlib"
)

// buildModuleMap creates a ModuleMap with all API modules and source modules.
// renderCtx is the optional template render context passed to cms/routing; nil outside renders.
// scriptsDir is an optional override for the directory from which source modules are loaded;
// if not provided, e.scriptsDir (theme default) is used.
func (e *ScriptEngine) buildModuleMap(renderCtx interface{}, scriptsDir ...string) *tengo.ModuleMap {
	modules := tengo.NewModuleMap()

	// Register built-in API modules
	modules.AddBuiltinModule("cms/nodes", e.nodesModule())
	modules.AddBuiltinModule("cms/settings", e.settingsModule())
	modules.AddBuiltinModule("cms/events", e.eventsModule())
	modules.AddBuiltinModule("cms/filters", e.filtersModule())
	modules.AddBuiltinModule("cms/http", e.httpModule())
	modules.AddBuiltinModule("cms/email", e.emailModule())
	modules.AddBuiltinModule("cms/menus", e.menusModule())
	modules.AddBuiltinModule("cms/log", logModule())
	modules.AddBuiltinModule("cms/routing", e.routingModule(renderCtx))
	modules.AddBuiltinModule("cms/helpers", helpersModule())
	modules.AddBuiltinModule("cms/fetch",   e.fetchModule())

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

	// Load source modules from scripts/ directory
	if len(scriptsDir) > 0 && scriptsDir[0] != "" {
		e.loadSourceModules(modules, scriptsDir[0])
	} else {
		e.loadSourceModules(modules, "")
	}

	return modules
}

// loadSourceModules recursively scans the scripts directory and adds all .tengo
// files as source modules. Module names use paths relative to scripts/ with a
// "./" prefix (e.g., "./lib/helpers" for scripts/lib/helpers.tengo).
// If dir is empty, e.scriptsDir (theme default) is used.
func (e *ScriptEngine) loadSourceModules(modules *tengo.ModuleMap, dir string) {
	scriptsDir := dir
	if scriptsDir == "" {
		scriptsDir = e.scriptsDir
	}
	if scriptsDir == "" {
		return
	}

	filepath.Walk(scriptsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".tengo") {
			return nil
		}

		// Skip entry scripts — they're not importable
		base := filepath.Base(path)
		if (base == "theme.tengo" || base == "extension.tengo") && filepath.Dir(path) == scriptsDir {
			return nil
		}

		// Calculate relative path from scripts dir
		relPath, err := filepath.Rel(scriptsDir, path)
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
