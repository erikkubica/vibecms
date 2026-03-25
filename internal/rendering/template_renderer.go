package rendering

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"path/filepath"
	"sync"
)

// TemplateRenderer handles parsing and rendering of Go html/template files.
// It supports caching in production and always re-parses in dev mode.
type TemplateRenderer struct {
	templateDir string
	cache       map[string]*template.Template
	mu          sync.RWMutex
	isDev       bool
	funcMap     template.FuncMap
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
	r.funcMap = template.FuncMap{
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
