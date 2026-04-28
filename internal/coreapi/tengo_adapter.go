package coreapi

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/d5/tengo/v2"
	"github.com/d5/tengo/v2/stdlib"
)

// ScriptCallbacks allows the ScriptEngine to communicate back to the CMS.
type ScriptCallbacks struct {
	OnRoute     func(method, path, scriptPath string)
	OnFilter    func(name, scriptPath string, priority int)
	OnEvent     func(action, scriptPath string, priority int)
	OnWellKnown func(path, scriptPath string)
}

// BuildTengoModules creates a new ModuleMap and registers all Squilla modules.
func BuildTengoModules(api CoreAPI, caller CallerInfo, renderCtx interface{}, scriptsDir string, cb *ScriptCallbacks) *tengo.ModuleMap {
	modules := tengo.NewModuleMap()
	RegisterModules(modules, api, caller, renderCtx, scriptsDir, cb)
	return modules
}

// RegisterModules adds all Squilla core modules to the Tengo module map.
func RegisterModules(modules *tengo.ModuleMap, api CoreAPI, caller CallerInfo, renderCtx interface{}, scriptsDir string, cb *ScriptCallbacks) {
	// Use WithCaller to associate the script caller with the context
	ctx := WithCaller(context.Background(), caller)

	// Standard library
	for name, mod := range stdlib.BuiltinModules {
		modules.AddBuiltinModule(name, mod)
	}

	// Squilla Core modules
	modules.AddBuiltinModule("core/nodes", nodesModule(api, ctx))
	modules.AddBuiltinModule("core/menus", menusModule(api, ctx))
	modules.AddBuiltinModule("core/routes", routesModule(cb))
	modules.AddBuiltinModule("core/filters", filtersModule(cb))
	modules.AddBuiltinModule("core/http", httpFetchModule(api, ctx))
	modules.AddBuiltinModule("core/log", logModule(api, ctx))
	modules.AddBuiltinModule("core/nodetypes", nodeTypesModule(api, ctx))
	modules.AddBuiltinModule("core/taxonomies", taxonomiesModule(api, ctx))
	modules.AddBuiltinModule("core/terms", termsModule(api, ctx))
	modules.AddBuiltinModule("core/helpers", helpersModule())
	modules.AddBuiltinModule("core/events", eventsModule(api, ctx, cb))
	modules.AddBuiltinModule("core/settings", settingsModule(api, ctx))
	modules.AddBuiltinModule("core/wellknown", wellKnownModule(cb))
	modules.AddBuiltinModule("core/assets", assetsModule(scriptsDir))

	if renderCtx != nil {
		modules.AddBuiltinModule("core/routing", routingModule(api, ctx, renderCtx))
	} else {
		modules.AddBuiltinModule("core/routing", routingModulePlaceholder())
	}

	// Load any source modules from the scripts directory
	loadSourceModules(modules, scriptsDir)
}

// ---------------------------------------------------------------------------
// Source module loader
// ---------------------------------------------------------------------------

// loadSourceModules recursively scans the scripts directory and adds all .tengo
// files as source modules with "./" prefix paths.
func loadSourceModules(modules *tengo.ModuleMap, scriptsDir string) {
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

		// Skip entry scripts
		base := filepath.Base(path)
		if (base == "theme.tengo" || base == "extension.tengo") && filepath.Dir(path) == scriptsDir {
			return nil
		}

		relPath, err := filepath.Rel(scriptsDir, path)
		if err != nil {
			return nil
		}

		moduleName := "./" + strings.TrimSuffix(relPath, ".tengo")
		moduleName = strings.ReplaceAll(moduleName, string(filepath.Separator), "/")

		src, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		modules.AddSourceModule(moduleName, src)
		return nil
	})
}
