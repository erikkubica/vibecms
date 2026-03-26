package scripting

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/d5/tengo/v2"
	"github.com/gofiber/fiber/v2"
)

// httpRoute represents a registered script HTTP endpoint.
type httpRoute struct {
	method     string
	path       string
	scriptPath string
}

// httpModule returns the cms/http built-in module.
func (e *ScriptEngine) httpModule() map[string]tengo.Object {
	return map[string]tengo.Object{
		"get":    &tengo.UserFunction{Name: "get", Value: e.httpRegister("GET")},
		"post":   &tengo.UserFunction{Name: "post", Value: e.httpRegister("POST")},
		"put":    &tengo.UserFunction{Name: "put", Value: e.httpRegister("PUT")},
		"patch":  &tengo.UserFunction{Name: "patch", Value: e.httpRegister("PATCH")},
		"delete": &tengo.UserFunction{Name: "delete", Value: e.httpRegister("DELETE")},
	}
}

// httpRegister returns a Tengo function that registers an HTTP route for the given method.
func (e *ScriptEngine) httpRegister(method string) tengo.CallableFunc {
	return func(args ...tengo.Object) (tengo.Object, error) {
		if len(args) < 2 {
			return tengo.UndefinedValue, fmt.Errorf("http.%s: requires path and script_path arguments", strings.ToLower(method))
		}

		path := getString(args[0])
		scriptPath := getString(args[1])

		if path == "" || scriptPath == "" {
			return tengo.UndefinedValue, fmt.Errorf("http.%s: path and script_path cannot be empty", strings.ToLower(method))
		}

		// Normalize path: ensure it starts with /
		if !strings.HasPrefix(path, "/") {
			path = "/" + path
		}

		// Strip leading "./" from script path
		if len(scriptPath) > 2 && scriptPath[:2] == "./" {
			scriptPath = scriptPath[2:]
		}

		e.mu.Lock()
		e.httpRoutes = append(e.httpRoutes, httpRoute{
			method:     method,
			path:       path,
			scriptPath: scriptPath,
		})
		e.mu.Unlock()

		log.Printf("[script] registered HTTP route: %s /api/theme%s -> %s", method, path, scriptPath)
		return tengo.UndefinedValue, nil
	}
}

// MountRoutes registers all script HTTP handlers on the Fiber app.
// Routes are mounted under /api/theme/* prefix.
func (e *ScriptEngine) MountRoutes(app *fiber.App) {
	e.mu.RLock()
	routes := make([]httpRoute, len(e.httpRoutes))
	copy(routes, e.httpRoutes)
	e.mu.RUnlock()

	if len(routes) == 0 {
		return
	}

	themeAPI := app.Group("/api/theme")

	for _, route := range routes {
		handler := e.makeHTTPHandler(route.scriptPath)
		fullPath := route.path

		switch route.method {
		case "GET":
			themeAPI.Get(fullPath, handler)
		case "POST":
			themeAPI.Post(fullPath, handler)
		case "PUT":
			themeAPI.Put(fullPath, handler)
		case "PATCH":
			themeAPI.Patch(fullPath, handler)
		case "DELETE":
			themeAPI.Delete(fullPath, handler)
		}
	}

	log.Printf("[script] mounted %d HTTP routes under /api/theme", len(routes))
}

// makeHTTPHandler creates a Fiber handler that runs a Tengo script.
func (e *ScriptEngine) makeHTTPHandler(scriptPath string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Build request object for the script
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

		// Parse body for POST/PUT/PATCH
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

		result, err := e.runScript(scriptPath, vars)
		if err != nil {
			log.Printf("[script] HTTP handler error: %s: %v", scriptPath, err)
			return c.Status(500).JSON(fiber.Map{"error": "script execution error"})
		}

		// Parse response from script
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

			// Set custom headers
			if h, ok := resp["headers"].(map[string]interface{}); ok {
				for k, v := range h {
					c.Set(k, fmt.Sprintf("%v", v))
				}
			}

			if body, ok := resp["body"]; ok {
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

		// If result is not a map, return it as JSON
		if result != nil {
			return c.JSON(result)
		}

		return c.Status(204).SendString("")
	}
}
