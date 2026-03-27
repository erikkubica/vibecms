package scripting

import (
	"fmt"
	"html/template"
	"log"
	"sort"
	"strings"

	"vibecms/internal/events"

	"github.com/d5/tengo/v2"
)

// scriptHandler is a registered event handler with priority.
type scriptHandler struct {
	scriptPath string
	priority   int
	baseDir    string // scripts directory for this handler (extension or theme)
}

// eventsModule returns the cms/events built-in module.
//
// Unified event system — handles both lifecycle events (node.created, etc.)
// and render-time events (before_main_content, etc.) through one API.
//
// Usage:
//
//	events := import("cms/events")
//	events.on("node.created", "handlers/on_created")           // default priority 50
//	events.on("before_main_content", "hooks/banner", 10)       // priority 10 (runs first)
//	events.on("before_main_content", "hooks/hello_world", 20)  // priority 20 (runs second)
//	events.emit("custom.action", {key: "value"})               // fire custom event
func (e *ScriptEngine) eventsModule() map[string]tengo.Object {
	return map[string]tengo.Object{
		"on":   &tengo.UserFunction{Name: "on", Value: e.eventsOn},
		"emit": &tengo.UserFunction{Name: "emit", Value: e.eventsEmit},
	}
}

// eventsOn handles events.on(name, script_path[, priority])
// Registers a handler script for the given event name.
// Priority is optional (default 50). Lower number = runs first.
func (e *ScriptEngine) eventsOn(args ...tengo.Object) (tengo.Object, error) {
	if len(args) < 2 {
		return tengo.UndefinedValue, fmt.Errorf("events.on: requires name and script_path arguments")
	}

	name := getString(args[0])
	scriptPath := getString(args[1])

	if name == "" {
		return tengo.UndefinedValue, fmt.Errorf("events.on: name cannot be empty")
	}
	if scriptPath == "" {
		return tengo.UndefinedValue, fmt.Errorf("events.on: script_path cannot be empty")
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
	e.eventHandlers[name] = append(e.eventHandlers[name], handler)
	// Keep sorted by priority
	sort.Slice(e.eventHandlers[name], func(i, j int) bool {
		return e.eventHandlers[name][i].priority < e.eventHandlers[name][j].priority
	})
	e.mu.Unlock()

	log.Printf("[script] registered event: %s -> %s (priority %d)", name, scriptPath, priority)
	return tengo.UndefinedValue, nil
}

// eventsEmit handles events.emit(name, payload)
// Publishes a custom event to the EventBus, triggering any registered handlers
// (both script-based and Go-based like email rules).
func (e *ScriptEngine) eventsEmit(args ...tengo.Object) (tengo.Object, error) {
	if len(args) < 1 {
		return tengo.UndefinedValue, fmt.Errorf("events.emit: requires name argument")
	}

	name := getString(args[0])
	if name == "" {
		return tengo.UndefinedValue, fmt.Errorf("events.emit: name cannot be empty")
	}

	payload := events.Payload{}
	if len(args) > 1 {
		if m := getMap(args[1]); m != nil {
			for k, v := range m {
				payload[k] = tengoToGo(v)
			}
		}
	}

	// Support optional third argument: an array of extra args
	if len(args) > 2 {
		if arr, ok := args[2].(*tengo.Array); ok {
			payload["_args"] = tengoArrayToGo(arr.Value)
		} else if arr, ok := args[2].(*tengo.ImmutableArray); ok {
			payload["_args"] = tengoArrayToGo(arr.Value)
		}
	}

	e.eventBus.Publish(name, payload)
	return tengo.UndefinedValue, nil
}

// subscribeEventHandlers wires registered event handlers to the Go EventBus.
// This connects script handlers to lifecycle events fired by Go services
// (node.created, node.updated, etc.). Called after theme.tengo finishes.
func (e *ScriptEngine) subscribeEventHandlers() {
	if e.eventBus == nil {
		return
	}

	e.mu.RLock()
	defer e.mu.RUnlock()

	for name, handlers := range e.eventHandlers {
		for _, h := range handlers {
			n := name          // capture for closure
			sp := h.scriptPath // capture for closure
			bd := h.baseDir    // capture for closure
			e.eventBus.Subscribe(n, func(act string, payload events.Payload) {
				payloadMap := make(map[string]interface{}, len(payload))
				for k, v := range payload {
					payloadMap[k] = v
				}

				vars := map[string]interface{}{
					"event": map[string]interface{}{
						"action":  act,
						"payload": payloadMap,
					},
				}

				if _, err := e.runScript(sp, vars, nil, bd); err != nil {
					log.Printf("[script] event handler error: %s (%s): %v", n, sp, err)
				}
			})
		}
	}
}

// RunEvent executes all script handlers registered for the named event,
// sorted by priority, and returns concatenated HTML output.
// Called by {{event "name" .}} in templates. Also works for non-render events
// (handlers that don't set response are simply ignored).
func (e *ScriptEngine) RunEvent(name string, ctx interface{}, args []interface{}) template.HTML {
	e.mu.RLock()
	handlers, ok := e.eventHandlers[name]
	if !ok || len(handlers) == 0 {
		e.mu.RUnlock()
		return ""
	}
	// Copy to release lock during execution
	sorted := make([]scriptHandler, len(handlers))
	copy(sorted, handlers)
	e.mu.RUnlock()

	renderCtx := normalizeForTengo(ctx)

	// Pass args as a script variable so handler scripts can inspect them
	var vars map[string]interface{}
	if len(args) > 0 {
		vars = map[string]interface{}{
			"args": args,
		}
	}

	var sb strings.Builder
	for _, h := range sorted {
		result, err := e.runScript(h.scriptPath, vars, renderCtx, h.baseDir)
		if err != nil {
			log.Printf("[script] event error: %s (%s): %v", name, h.scriptPath, err)
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
