package scripting

import (
	"fmt"
	"log"

	"vibecms/internal/events"

	"github.com/d5/tengo/v2"
)

// eventsModule returns the cms/events built-in module.
// During theme.tengo execution, on() registers handler script paths.
// After initialization, the engine subscribes to the EventBus for each registered action.
func (e *ScriptEngine) eventsModule() map[string]tengo.Object {
	return map[string]tengo.Object{
		"on":   &tengo.UserFunction{Name: "on", Value: e.eventsOn},
		"emit": &tengo.UserFunction{Name: "emit", Value: e.eventsEmit},
	}
}

// eventsOn handles events.on(action, script_path)
// Registers a handler script to run when the given event action fires.
// script_path is relative to the theme's scripts/ directory (without .tengo extension).
func (e *ScriptEngine) eventsOn(args ...tengo.Object) (tengo.Object, error) {
	if len(args) < 2 {
		return tengo.UndefinedValue, fmt.Errorf("events.on: requires action and script_path arguments")
	}

	action := getString(args[0])
	scriptPath := getString(args[1])

	if action == "" {
		return tengo.UndefinedValue, fmt.Errorf("events.on: action cannot be empty")
	}
	if scriptPath == "" {
		return tengo.UndefinedValue, fmt.Errorf("events.on: script_path cannot be empty")
	}

	// Strip leading "./" for consistency
	if len(scriptPath) > 2 && scriptPath[:2] == "./" {
		scriptPath = scriptPath[2:]
	}

	e.mu.Lock()
	e.eventHandlers[action] = append(e.eventHandlers[action], scriptPath)
	e.mu.Unlock()

	log.Printf("[script] registered event handler: %s -> %s", action, scriptPath)
	return tengo.UndefinedValue, nil
}

// eventsEmit handles events.emit(action, payload)
// Publishes a custom event to the EventBus, triggering any registered handlers
// (both script-based and Go-based like email rules).
func (e *ScriptEngine) eventsEmit(args ...tengo.Object) (tengo.Object, error) {
	if len(args) < 1 {
		return tengo.UndefinedValue, fmt.Errorf("events.emit: requires action argument")
	}

	action := getString(args[0])
	if action == "" {
		return tengo.UndefinedValue, fmt.Errorf("events.emit: action cannot be empty")
	}

	payload := events.Payload{}
	if len(args) > 1 {
		if m := getMap(args[1]); m != nil {
			for k, v := range m {
				payload[k] = tengoToGo(v)
			}
		}
	}

	e.eventBus.Publish(action, payload)
	return tengo.UndefinedValue, nil
}

// subscribeEventHandlers wires all registered event handlers to the EventBus.
// Called after theme.tengo finishes executing.
func (e *ScriptEngine) subscribeEventHandlers() {
	if e.eventBus == nil {
		return
	}

	e.mu.RLock()
	defer e.mu.RUnlock()

	for action, scripts := range e.eventHandlers {
		for _, scriptPath := range scripts {
			a := action     // capture for closure
			sp := scriptPath // capture for closure
			e.eventBus.Subscribe(a, func(act string, payload events.Payload) {
				// Convert payload to map for script injection
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

				if _, err := e.runScript(sp, vars, nil); err != nil {
					log.Printf("[script] event handler error: %s (%s): %v", a, sp, err)
				}
			})
		}
	}
}
