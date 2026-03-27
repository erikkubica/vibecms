package scripting

import (
	"fmt"
	"log"
	"sort"

	"github.com/d5/tengo/v2"
)

// filtersModule returns the cms/filters built-in module.
//
// Filters transform values through a priority-ordered chain of scripts.
// Each filter script receives a `value` variable and sets `response` to the modified value.
// The output of one filter becomes the input of the next (in priority order).
//
// Usage:
//
//	filters := import("cms/filters")
//	filters.add("node.title", "filters/uppercase_titles")           // default priority 50
//	filters.add("node.title", "filters/site_title_suffix", 90)      // priority 90 (runs later)
func (e *ScriptEngine) filtersModule() map[string]tengo.Object {
	return map[string]tengo.Object{
		"add": &tengo.UserFunction{Name: "add", Value: e.filtersAdd},
	}
}

// filtersAdd handles filters.add(name, script_path[, priority])
// Registers a filter script for the given filter name.
// Priority is optional (default 50). Lower number = runs first.
func (e *ScriptEngine) filtersAdd(args ...tengo.Object) (tengo.Object, error) {
	if len(args) < 2 {
		return tengo.UndefinedValue, fmt.Errorf("filters.add: requires name and script_path arguments")
	}

	name := getString(args[0])
	scriptPath := getString(args[1])

	if name == "" {
		return tengo.UndefinedValue, fmt.Errorf("filters.add: name cannot be empty")
	}
	if scriptPath == "" {
		return tengo.UndefinedValue, fmt.Errorf("filters.add: script_path cannot be empty")
	}

	// Strip leading "./"
	if len(scriptPath) > 2 && scriptPath[:2] == "./" {
		scriptPath = scriptPath[2:]
	}

	priority := 50
	if len(args) > 2 {
		if p := getInt(args[2]); p > 0 {
			priority = p
		}
	}

	handler := scriptHandler{
		scriptPath: scriptPath,
		priority:   priority,
		baseDir:    e.activeScriptsDir,
	}

	e.mu.Lock()
	e.filterHandlers[name] = append(e.filterHandlers[name], handler)
	// Keep sorted by priority
	sort.Slice(e.filterHandlers[name], func(i, j int) bool {
		return e.filterHandlers[name][i].priority < e.filterHandlers[name][j].priority
	})
	e.mu.Unlock()

	log.Printf("[script] registered filter: %s -> %s (priority %d)", name, scriptPath, priority)
	return tengo.UndefinedValue, nil
}

// ApplyFilter runs the filter chain for the named filter, passing the value
// through each registered filter script in priority order. Returns the final value.
// renderCtx is the optional template render context; pass nil if not in a render.
func (e *ScriptEngine) ApplyFilter(name string, value interface{}, renderCtx interface{}) interface{} {
	e.mu.RLock()
	handlers, ok := e.filterHandlers[name]
	if !ok || len(handlers) == 0 {
		e.mu.RUnlock()
		return value
	}
	// Copy to release lock during execution
	sorted := make([]scriptHandler, len(handlers))
	copy(sorted, handlers)
	e.mu.RUnlock()

	currentValue := value
	for _, h := range sorted {
		vars := map[string]interface{}{
			"value": currentValue,
		}

		result, err := e.runScript(h.scriptPath, vars, renderCtx, h.baseDir)
		if err != nil {
			log.Printf("[script] filter error: %s (%s): %v", name, h.scriptPath, err)
			continue // pass value through unchanged on error
		}

		if result != nil {
			currentValue = result
		}
	}

	return currentValue
}
