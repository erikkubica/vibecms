package scripting

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"sort"
	"strings"

	"vibecms/internal/events"

	"github.com/gofiber/fiber/v2"
)

// scriptHandler is a registered event/filter handler with priority.
type scriptHandler struct {
	scriptPath string
	priority   int
	baseDir    string // scripts directory for this handler (extension or theme)
}

// httpRoute represents a registered script HTTP endpoint.
type httpRoute struct {
	method     string
	path       string
	scriptPath string
	baseDir    string // scripts directory for this route (extension or theme)
}

// wellKnownRoute represents a script-backed /.well-known/* endpoint.
type wellKnownRoute struct {
	path       string // suffix after "/.well-known/", may end with "*"
	scriptPath string
	baseDir    string
}

// ---------------------------------------------------------------------------
// Event handler registration (used during theme/extension script loading)
// ---------------------------------------------------------------------------

// EventsOn handles events.on(name, script_path[, priority])
// Registers a handler script for the given event name.
// This is called from theme.tengo / extension.tengo entry scripts.
func (e *ScriptEngine) EventsOn(name, scriptPath string, priority int) {
	if name == "" || scriptPath == "" {
		return
	}
	// Strip leading "./"
	if len(scriptPath) > 2 && scriptPath[:2] == "./" {
		scriptPath = scriptPath[2:]
	}

	handler := scriptHandler{
		scriptPath: scriptPath,
		priority:   priority,
		baseDir:    e.activeScriptsDir,
	}

	e.mu.Lock()
	e.eventHandlers[name] = append(e.eventHandlers[name], handler)
	sort.Slice(e.eventHandlers[name], func(i, j int) bool {
		return e.eventHandlers[name][i].priority < e.eventHandlers[name][j].priority
	})
	e.mu.Unlock()

	log.Printf("[script] registered event: %s -> %s (priority %d)", name, scriptPath, priority)
}

// ---------------------------------------------------------------------------
// Filter handler registration
// ---------------------------------------------------------------------------

// FiltersAdd registers a filter handler script.
func (e *ScriptEngine) FiltersAdd(name, scriptPath string, priority int) {
	if name == "" || scriptPath == "" {
		return
	}
	if len(scriptPath) > 2 && scriptPath[:2] == "./" {
		scriptPath = scriptPath[2:]
	}

	handler := scriptHandler{
		scriptPath: scriptPath,
		priority:   priority,
		baseDir:    e.activeScriptsDir,
	}

	e.mu.Lock()
	e.filterHandlers[name] = append(e.filterHandlers[name], handler)
	sort.Slice(e.filterHandlers[name], func(i, j int) bool {
		return e.filterHandlers[name][i].priority < e.filterHandlers[name][j].priority
	})
	e.mu.Unlock()

	log.Printf("[script] registered filter: %s -> %s (priority %d)", name, scriptPath, priority)
}

// ApplyFilter runs the filter chain for the named filter, passing the value
// through each registered filter script in priority order.
func (e *ScriptEngine) ApplyFilter(name string, value interface{}, renderCtx interface{}) interface{} {
	e.mu.RLock()
	handlers, ok := e.filterHandlers[name]
	if !ok || len(handlers) == 0 {
		e.mu.RUnlock()
		return value
	}
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
			continue
		}
		if result != nil {
			currentValue = result
		}
	}
	return currentValue
}

// ---------------------------------------------------------------------------
// Event bus wiring
// ---------------------------------------------------------------------------

// subscribeEventHandlers wires registered event handlers to the Go EventBus.
func (e *ScriptEngine) subscribeEventHandlers() {
	if e.eventBus == nil {
		return
	}

	e.mu.RLock()
	defer e.mu.RUnlock()

	for name, handlers := range e.eventHandlers {
		for _, h := range handlers {
			n := name
			sp := h.scriptPath
			bd := h.baseDir
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
func (e *ScriptEngine) RunEvent(name string, ctx interface{}, args []interface{}) template.HTML {
	e.mu.RLock()
	handlers, ok := e.eventHandlers[name]
	if !ok || len(handlers) == 0 {
		e.mu.RUnlock()
		return ""
	}
	sorted := make([]scriptHandler, len(handlers))
	copy(sorted, handlers)
	e.mu.RUnlock()

	renderCtx := normalizeForTengo(ctx)

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

// ---------------------------------------------------------------------------
// HTTP route registration & mounting
// ---------------------------------------------------------------------------

// HTTPRegister registers an HTTP route from a script.
func (e *ScriptEngine) HTTPRegister(method, path, scriptPath string) {
	if path == "" || scriptPath == "" {
		return
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	if len(scriptPath) > 2 && scriptPath[:2] == "./" {
		scriptPath = scriptPath[2:]
	}

	e.mu.Lock()
	e.httpRoutes = append(e.httpRoutes, httpRoute{
		method:     method,
		path:       path,
		scriptPath: scriptPath,
		baseDir:    e.activeScriptsDir,
	})
	e.mu.Unlock()

	log.Printf("[script] registered HTTP route: %s /api/theme%s -> %s", method, path, scriptPath)
}

// MountRoutes registers all script HTTP handlers on the Fiber app.
// Routes with paths starting with "/" are mounted at the top level (e.g., /sitemap.xml).
// Other routes are mounted under /api/theme (e.g., /api/theme/search).
func (e *ScriptEngine) MountRoutes(app *fiber.App) {
	e.mu.RLock()
	routes := make([]httpRoute, len(e.httpRoutes))
	copy(routes, e.httpRoutes)
	e.mu.RUnlock()

	if len(routes) == 0 {
		return
	}

	themeAPI := app.Group("/api/theme")
	var topLevel, apiLevel int

	for _, route := range routes {
		handler := e.makeHTTPHandler(route.scriptPath, route.baseDir)

		// Routes with paths like /sitemap.xml mount at the app root.
		// Routes with paths like /search mount under /api/theme.
		isTopLevel := strings.Contains(route.path, ".")
		var target fiber.Router
		if isTopLevel {
			target = app
			topLevel++
		} else {
			target = themeAPI
			apiLevel++
		}

		switch route.method {
		case "GET":
			target.Get(route.path, handler)
		case "POST":
			target.Post(route.path, handler)
		case "PUT":
			target.Put(route.path, handler)
		case "PATCH":
			target.Patch(route.path, handler)
		case "DELETE":
			target.Delete(route.path, handler)
		}
	}

	if topLevel > 0 {
		log.Printf("[script] mounted %d top-level HTTP routes", topLevel)
	}
	if apiLevel > 0 {
		log.Printf("[script] mounted %d HTTP routes under /api/theme", apiLevel)
	}
}

// WellKnownRegister records a .well-known handler from a Tengo script.
func (e *ScriptEngine) WellKnownRegister(path, scriptPath string) {
	if path == "" || scriptPath == "" {
		return
	}
	if len(scriptPath) > 2 && scriptPath[:2] == "./" {
		scriptPath = scriptPath[2:]
	}
	e.mu.Lock()
	e.wellKnown = append(e.wellKnown, wellKnownRoute{
		path:       path,
		scriptPath: scriptPath,
		baseDir:    e.activeScriptsDir,
	})
	e.mu.Unlock()
	log.Printf("[script] registered well-known: /.well-known/%s -> %s", path, scriptPath)
}

// MountWellKnown registers all script-defined .well-known handlers on the
// provided registry. Call after script loading and before the server starts.
func (e *ScriptEngine) MountWellKnown(reg WellKnownRegistrar) {
	if reg == nil {
		return
	}
	e.mu.RLock()
	routes := make([]wellKnownRoute, len(e.wellKnown))
	copy(routes, e.wellKnown)
	e.mu.RUnlock()

	for _, r := range routes {
		reg.Register(r.path, e.makeHTTPHandler(r.scriptPath, r.baseDir))
	}
	if len(routes) > 0 {
		log.Printf("[script] mounted %d .well-known handlers", len(routes))
	}
}

// makeHTTPHandler creates a Fiber handler that runs a Tengo script.
func (e *ScriptEngine) makeHTTPHandler(scriptPath string, baseDir string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		query := make(map[string]interface{})
		c.Request().URI().QueryArgs().VisitAll(func(key, value []byte) {
			query[string(key)] = string(value)
		})

		params := make(map[string]interface{})
		for _, param := range c.Route().Params {
			params[param] = c.Params(param)
		}

		headers := make(map[string]interface{})
		c.Request().Header.VisitAll(func(key, value []byte) {
			headers[string(key)] = string(value)
		})

		var body interface{}
		if c.Method() != "GET" && c.Method() != "DELETE" {
			contentType := string(c.Request().Header.ContentType())
			if strings.Contains(contentType, "application/json") {
				var jsonBody interface{}
				if err := json.Unmarshal(c.Body(), &jsonBody); err == nil {
					body = jsonBody
				}
			} else {
				body = string(c.Body())
			}
		}

		reqMap := map[string]interface{}{
			"method":  c.Method(),
			"path":    c.Path(),
			"query":   query,
			"params":  params,
			"headers": headers,
			"body":    body,
			"ip":      c.IP(),
		}

		vars := map[string]interface{}{
			"request": reqMap,
		}

		result, err := e.runScript(scriptPath, vars, nil, baseDir)
		if err != nil {
			log.Printf("[script] HTTP handler error: %s: %v", scriptPath, err)
			return c.Status(500).JSON(fiber.Map{"error": "script execution error"})
		}

		if resp, ok := result.(map[string]interface{}); ok {
			status := 200
			if s, ok := resp["status"]; ok {
				switch v := s.(type) {
				case int64:
					status = int(v)
				case int:
					status = v
				}
			}

			if h, ok := resp["headers"].(map[string]interface{}); ok {
				for k, v := range h {
					c.Set(k, fmt.Sprintf("%v", v))
				}
			}

			if ct, ok := resp["content_type"].(string); ok && ct != "" {
				c.Set("Content-Type", ct)
			}

			if body, ok := resp["body"]; ok {
				if bodyStr, ok := body.(string); ok {
					return c.Status(status).SendString(bodyStr)
				}
				return c.Status(status).JSON(body)
			}
			if html, ok := resp["html"].(string); ok {
				c.Set("Content-Type", "text/html")
				return c.Status(status).SendString(html)
			}
			if text, ok := resp["text"].(string); ok {
				c.Set("Content-Type", "text/plain")
				return c.Status(status).SendString(text)
			}

			return c.Status(status).JSON(resp)
		}

		if result != nil {
			return c.JSON(result)
		}

		return c.Status(204).SendString("")
	}
}
