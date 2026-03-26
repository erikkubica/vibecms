package scripting

import (
	"fmt"
	"html/template"
	"log"
	"strings"

	"github.com/d5/tengo/v2"
)

// hooksModule returns the cms/hooks built-in module.
// Hooks allow scripts to inject content at named template action points.
//
// Usage in theme.tengo:
//
//	hooks := import("cms/hooks")
//	hooks.add("before_main_content", "hooks/hello_world")
//
// Usage in layout templates:
//
//	{{action "before_main_content"}}
func (e *ScriptEngine) hooksModule() map[string]tengo.Object {
	return map[string]tengo.Object{
		"add": &tengo.UserFunction{Name: "add", Value: e.hooksAdd},
	}
}

// hooksAdd handles hooks.add(hook_name, script_path)
func (e *ScriptEngine) hooksAdd(args ...tengo.Object) (tengo.Object, error) {
	if len(args) < 2 {
		return tengo.UndefinedValue, fmt.Errorf("hooks.add: requires hook_name and script_path arguments")
	}

	hookName := getString(args[0])
	scriptPath := getString(args[1])

	if hookName == "" || scriptPath == "" {
		return tengo.UndefinedValue, fmt.Errorf("hooks.add: hook_name and script_path cannot be empty")
	}

	// Strip leading "./"
	if len(scriptPath) > 2 && scriptPath[:2] == "./" {
		scriptPath = scriptPath[2:]
	}

	e.mu.Lock()
	e.hookHandlers[hookName] = append(e.hookHandlers[hookName], scriptPath)
	e.mu.Unlock()

	log.Printf("[script] registered hook: %s -> %s", hookName, scriptPath)
	return tengo.UndefinedValue, nil
}

// RunAction executes all hook scripts registered for the named action
// and returns their concatenated HTML output. This is called by the
// {{action "name" .}} template function during layout rendering.
// The ctx parameter receives the current template data (node, app, user).
func (e *ScriptEngine) RunAction(name string, ctx interface{}) template.HTML {
	e.mu.RLock()
	scripts, ok := e.hookHandlers[name]
	if !ok || len(scripts) == 0 {
		e.mu.RUnlock()
		return ""
	}
	// Copy to release lock during execution
	paths := make([]string, len(scripts))
	copy(paths, scripts)
	e.mu.RUnlock()

	// Normalize the render context for the routing module
	renderCtx := normalizeForTengo(ctx)

	var sb strings.Builder
	for _, scriptPath := range paths {
		result, err := e.runScript(scriptPath, nil, renderCtx)
		if err != nil {
			log.Printf("[script] hook error: %s (%s): %v", name, scriptPath, err)
			continue
		}
		if result == nil {
			continue
		}
		if resp, ok := result.(map[string]interface{}); ok {
			if html, ok := resp["html"].(string); ok {
				sb.WriteString(html)
			}
		} else if s, ok := result.(string); ok {
			sb.WriteString(s)
		}
	}

	return template.HTML(sb.String())
}
