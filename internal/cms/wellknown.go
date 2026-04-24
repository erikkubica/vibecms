package cms

import (
	"strings"
	"sync"

	"github.com/gofiber/fiber/v2"
)

// WellKnownRegistry routes requests under /.well-known/* to registered
// handlers. Unregistered paths return 404 immediately, bypassing the
// content-node resolution pipeline.
//
// Common well-known endpoints that extensions or core features may want
// to implement:
//
//   - security.txt                               (RFC 9116 — security contact)
//   - change-password                            (password-change redirect)
//   - webfinger                                  (ActivityPub / Mastodon discovery)
//   - nodeinfo                                   (Fediverse metadata)
//   - host-meta, host-meta.json                  (XRD / JRD discovery)
//   - openid-configuration                       (OIDC discovery)
//   - oauth-authorization-server                 (OAuth discovery)
//   - apple-app-site-association                 (iOS universal links)
//   - assetlinks.json                            (Android app links)
//   - acme-challenge/<token>                     (Let's Encrypt HTTP-01)
//   - appspecific/com.chrome.devtools.json       (Chrome DevTools workspace)
//   - ai.txt / llms.txt                          (AI crawler directives)
//
// Handlers are keyed by the path suffix after "/.well-known/". Prefix
// handlers are supported by registering a trailing "*" (e.g. "acme-challenge/*").
type WellKnownRegistry struct {
	mu       sync.RWMutex
	exact    map[string]fiber.Handler
	prefixes []wkPrefix
}

type wkPrefix struct {
	prefix  string
	handler fiber.Handler
}

// NewWellKnownRegistry returns an empty registry.
func NewWellKnownRegistry() *WellKnownRegistry {
	return &WellKnownRegistry{exact: make(map[string]fiber.Handler)}
}

// Register maps a well-known path (without the "/.well-known/" prefix)
// to a handler. A trailing "*" registers a prefix handler.
func (r *WellKnownRegistry) Register(path string, handler fiber.Handler) {
	path = strings.TrimPrefix(path, "/")
	path = strings.TrimPrefix(path, ".well-known/")

	r.mu.Lock()
	defer r.mu.Unlock()

	if strings.HasSuffix(path, "*") {
		r.prefixes = append(r.prefixes, wkPrefix{
			prefix:  strings.TrimSuffix(path, "*"),
			handler: handler,
		})
		return
	}
	r.exact[path] = handler
}

// Mount registers the /.well-known/* route on the Fiber app. Must be
// called before the public content catch-all.
func (r *WellKnownRegistry) Mount(app *fiber.App) {
	app.Get("/.well-known/*", func(c *fiber.Ctx) error {
		sub := c.Params("*")

		r.mu.RLock()
		handler, ok := r.exact[sub]
		if !ok {
			for _, p := range r.prefixes {
				if strings.HasPrefix(sub, p.prefix) {
					handler = p.handler
					ok = true
					break
				}
			}
		}
		r.mu.RUnlock()

		if !ok {
			return c.SendStatus(fiber.StatusNotFound)
		}
		return handler(c)
	})
}
