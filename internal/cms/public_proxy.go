package cms

import (
	"encoding/json"
	"log"
	"strings"

	"github.com/gofiber/fiber/v2"

	"vibecms/internal/models"
	pb "vibecms/pkg/plugin/proto"
)

// PublicExtensionProxy registers public (no auth) routes declared by extensions
// and proxies them to the corresponding gRPC plugin's HandleHTTPRequest.
type PublicExtensionProxy struct {
	pluginMgr *PluginManager
}

// NewPublicExtensionProxy creates a new PublicExtensionProxy.
func NewPublicExtensionProxy(pm *PluginManager) *PublicExtensionProxy {
	return &PublicExtensionProxy{pluginMgr: pm}
}

// reservedPublicPathPrefixes are kernel-owned URL spaces that
// extensions MUST NOT register through their public_routes manifest.
// An extension that lists path: "/login" would otherwise mount a
// no-auth handler that shadows the kernel login form — perfect for
// credential phishing once installed. Defense in depth: the kernel
// installer doesn't audit manifests today, so we filter at boot.
//
// Stored without trailing slashes; isReservedPublicPath matches both
// the exact prefix and prefix-followed-by-/ : * (Fiber path
// separators). That keeps "/auth/" reserved while leaving
// "/auth-callback" free for an extension to claim.
var reservedPublicPathPrefixes = []string{
	"/admin",           // entire admin SPA + API
	"/auth",            // login/register/forgot/reset POST handlers
	"/api/v1",          // versioned public API (theme webhook, etc.)
	"/me",              // /me identity endpoint
	"/login",           // login page
	"/logout",          // logout endpoint
	"/register",        // registration page
	"/forgot-password", // password reset request page
	"/reset-password",  // password reset completion page
}

// isReservedPublicPath reports whether path belongs to a kernel-owned
// namespace and so cannot be claimed by an extension public_route.
// The match is exact-or-followed-by-Fiber-separator so /auth-callback
// stays free while /auth/login stays reserved.
func isReservedPublicPath(path string) bool {
	if path == "" || path == "/" {
		// Root catch-alls would shadow every other route Fiber resolves.
		return true
	}
	for _, p := range reservedPublicPathPrefixes {
		if path == p {
			return true
		}
		if len(path) > len(p) && strings.HasPrefix(path, p) {
			next := path[len(p)]
			if next == '/' || next == '*' || next == ':' {
				return true
			}
		}
	}
	return false
}

// RegisterPublicRoutes reads the manifest for each active extension, and for
// every entry in public_routes registers the declared Fiber route that proxies
// to the extension plugin — without any auth middleware. Reserved kernel
// paths (login, admin, auth flows, …) are refused with a warning so a
// hostile or careless manifest can't shadow auth-critical routes.
func (pp *PublicExtensionProxy) RegisterPublicRoutes(app *fiber.App, activeExts []models.Extension) {
	for _, ext := range activeExts {
		var manifest ExtensionManifest
		if err := json.Unmarshal([]byte(ext.Manifest), &manifest); err != nil {
			continue
		}
		if len(manifest.PublicRoutes) == 0 {
			continue
		}

		slug := ext.Slug
		for _, route := range manifest.PublicRoutes {
			method := strings.ToUpper(route.Method)
			path := route.Path

			if isReservedPublicPath(path) {
				log.Printf("[public-proxy] REFUSING %s %s for extension %s — path is kernel-reserved", method, path, slug)
				continue
			}

			log.Printf("[public-proxy] %s %s -> extension %s", method, path, slug)

			handler := pp.makeHandler(slug, path)

			switch method {
			case "GET":
				app.Get(path, handler)
			case "POST":
				app.Post(path, handler)
			case "PUT":
				app.Put(path, handler)
			case "DELETE":
				app.Delete(path, handler)
			case "PATCH":
				app.Patch(path, handler)
			default:
				app.All(path, handler)
			}
		}
	}
}

// makeHandler returns a Fiber handler that proxies a public request to the
// given extension slug's gRPC plugin.
func (pp *PublicExtensionProxy) makeHandler(slug, routePath string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		client := pp.pluginMgr.GetClient(slug)
		if client == nil {
			return c.SendStatus(fiber.StatusServiceUnavailable)
		}

		// Build headers map. Strip sensitive auth headers AND any
		// client-supplied X-Forwarded-For / X-Real-IP — those are
		// trivially spoofable and plugins use them for per-IP rate
		// limiting (forms extension does for spam control). The
		// kernel writes the trusted value (c.IP()) below.
		headers := make(map[string]string)
		c.Request().Header.VisitAll(func(key, value []byte) {
			k := string(key)
			kLower := strings.ToLower(k)
			switch kLower {
			case "cookie", "authorization", "x-forwarded-for", "x-real-ip":
				return
			}
			headers[k] = string(value)
		})
		headers["X-Forwarded-For"] = c.IP()

		// Build query params.
		queryParams := make(map[string]string)
		c.Request().URI().QueryArgs().VisitAll(func(key, value []byte) {
			queryParams[string(key)] = string(value)
		})

		// Path params.
		pathParams := make(map[string]string)
		wildcard := c.Params("*")
		if wildcard != "" {
			pathParams["path"] = wildcard
		}

		// Use the full request path as-is for the plugin.
		req := &pb.PluginHTTPRequest{
			Method:      c.Method(),
			Path:        c.Path(),
			Headers:     headers,
			Body:        c.Body(),
			QueryParams: queryParams,
			PathParams:  pathParams,
			UserId:      0, // no auth on public routes
		}

		resp, err := client.HandleHTTPRequest(req)
		if err != nil {
			log.Printf("[public-proxy] gRPC error from %s: %v", slug, err)
			return c.SendStatus(fiber.StatusBadGateway)
		}

		for k, v := range resp.Headers {
			c.Set(k, v)
		}

		return c.Status(int(resp.StatusCode)).Send(resp.Body)
	}
}
