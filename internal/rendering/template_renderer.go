package rendering

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"path/filepath"
	"sync"
)

// TemplateRenderer handles parsing and rendering of Go html/template files.
// It supports caching in production and always re-parses in dev mode.
type TemplateRenderer struct {
	templateDir string
	cache       map[string]*template.Template
	mu          sync.RWMutex
	isDev        bool
	funcMap      template.FuncMap
	actionRunner func(string) template.HTML
}

// SetActionRunner sets the function called by {{action "name"}} in templates.
// This connects the template engine to the scripting hook system.
func (r *TemplateRenderer) SetActionRunner(fn func(string) template.HTML) {
	r.actionRunner = fn
}

// NewTemplateRenderer creates a new TemplateRenderer.
// templateDir is the root directory containing template files.
// isDev controls whether templates are cached (false) or re-parsed on every request (true).
func NewTemplateRenderer(templateDir string, isDev bool) *TemplateRenderer {
	r := &TemplateRenderer{
		templateDir: templateDir,
		cache:       make(map[string]*template.Template),
		isDev:       isDev,
	}
	// Default no-op action runner (replaced when scripting engine is loaded)
	r.actionRunner = func(name string) template.HTML { return "" }

	r.funcMap = template.FuncMap{
		"action": func(name string) template.HTML {
			return r.actionRunner(name)
		},
		"deref": func(v interface{}) interface{} {
			if v == nil {
				return ""
			}
			switch p := v.(type) {
			case *string:
				if p == nil {
					return ""
				}
				return *p
			case *int:
				if p == nil {
					return 0
				}
				return *p
			default:
				return v
			}
		},
		"safeHTML": func(s interface{}) template.HTML {
			switch v := s.(type) {
			case string:
				return template.HTML(v)
			case template.HTML:
				return v
			default:
				return template.HTML(fmt.Sprintf("%v", v))
			}
		},
		"json": func(v interface{}) string {
			b, err := json.MarshalIndent(v, "", "  ")
			if err != nil {
				return fmt.Sprintf("{\"error\": %q}", err.Error())
			}
			return string(b)
		},
	}
	return r
}

// Render parses and executes a layout + page template combination.
// layoutName is relative to templateDir, e.g. "layouts/base.html".
// pageName is relative to templateDir, e.g. "public/home.html".
// The page template must define blocks expected by the layout (e.g. "title", "content").
func (r *TemplateRenderer) Render(w io.Writer, layoutName, pageName string, data interface{}) error {
	cacheKey := layoutName + ":" + pageName

	if !r.isDev {
		r.mu.RLock()
		tmpl, ok := r.cache[cacheKey]
		r.mu.RUnlock()
		if ok {
			return tmpl.Execute(w, data)
		}
	}

	layoutPath := filepath.Join(r.templateDir, layoutName)
	pagePath := filepath.Join(r.templateDir, pageName)

	tmpl, err := template.New(filepath.Base(layoutName)).Funcs(r.funcMap).ParseFiles(layoutPath, pagePath)
	if err != nil {
		return fmt.Errorf("template parse error [%s + %s]: %w", layoutName, pageName, err)
	}

	if !r.isDev {
		r.mu.Lock()
		r.cache[cacheKey] = tmpl
		r.mu.Unlock()
	}

	return tmpl.Execute(w, data)
}

// RenderPage is a convenience method that renders a page template with the
// default base layout ("layouts/base.html").
func (r *TemplateRenderer) RenderPage(w io.Writer, pageName string, data interface{}) error {
	return r.Render(w, "layouts/base.html", pageName, data)
}

// RenderFragment renders only the "content" block from a page template,
// without wrapping in any layout. Used when the site layout engine handles the wrapper.
func (r *TemplateRenderer) RenderFragment(w io.Writer, pageName string, data interface{}) error {
	pagePath := filepath.Join(r.templateDir, pageName)

	// Parse with a minimal base that just executes the content block
	base := `{{template "content" .}}`
	tmpl, err := template.New("fragment").Funcs(r.funcMap).Parse(base)
	if err != nil {
		return fmt.Errorf("fragment base parse error: %w", err)
	}
	tmpl, err = tmpl.ParseFiles(pagePath)
	if err != nil {
		return fmt.Errorf("fragment parse error [%s]: %w", pageName, err)
	}
	return tmpl.Execute(w, data)
}

// RenderLayout renders a layout template_code string (from the DB) with a
// blockResolver that supports the renderLayoutBlock template function.
// The blockResolver returns the template_code for a given layout block slug.
// Recursion is guarded to a maximum depth of 5.
func (r *TemplateRenderer) RenderLayout(w io.Writer, templateCode string, data interface{}, blockResolver func(slug string) (string, error)) error {
	depth := 0
	maxDepth := 5

	var renderBlock func(slug string) template.HTML
	renderBlock = func(slug string) template.HTML {
		depth++
		if depth > maxDepth {
			log.Printf("WARN: renderLayoutBlock recursion limit reached for '%s'", slug)
			depth--
			return ""
		}
		defer func() { depth-- }()

		code, err := blockResolver(slug)
		if err != nil {
			log.Printf("WARN: layout block '%s' not found: %v", slug, err)
			return ""
		}

		funcMap := template.FuncMap{}
		for k, v := range r.funcMap {
			funcMap[k] = v
		}
		funcMap["renderLayoutBlock"] = renderBlock

		tmpl, err := template.New("partial-" + slug).Funcs(funcMap).Parse(code)
		if err != nil {
			log.Printf("WARN: template parse error in '%s': %v", slug, err)
			return ""
		}
		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, data); err != nil {
			log.Printf("WARN: template execute error in '%s': %v", slug, err)
			return ""
		}
		return template.HTML(buf.String())
	}

	funcMap := template.FuncMap{}
	for k, v := range r.funcMap {
		funcMap[k] = v
	}
	funcMap["renderLayoutBlock"] = renderBlock

	tmpl, err := template.New("layout").Funcs(funcMap).Parse(templateCode)
	if err != nil {
		return fmt.Errorf("layout template parse error: %w", err)
	}
	return tmpl.Execute(w, data)
}
