package cms

import (
	"log"
	"strings"

	"github.com/gofiber/fiber/v2"

	"vibecms/internal/auth"
	pb "vibecms/pkg/plugin/proto"
)

// ExtensionProxy proxies admin API requests to extension gRPC plugins.
// Route: /admin/api/ext/:slug/*
type ExtensionProxy struct {
	pluginMgr *PluginManager
}

// NewExtensionProxy creates a new ExtensionProxy.
func NewExtensionProxy(pm *PluginManager) *ExtensionProxy {
	return &ExtensionProxy{pluginMgr: pm}
}

// RegisterRoutes registers the catch-all proxy route on the given
// router. Gated by `admin_access` because plugins themselves don't
// enforce per-user RBAC — they trust the kernel-side gate to have
// done so. Without this, any authenticated user (including a
// freshly-registered member) could hit /admin/api/ext/forms/submissions
// and dump every PII-bearing form submission. admin / editor / author
// roles all carry admin_access; member does not.
func (ep *ExtensionProxy) RegisterRoutes(router fiber.Router) {
	log.Println("[extension-proxy] registering routes on /ext/:slug/* (gated: admin_access)")
	guard := auth.CapabilityRequired("admin_access")
	router.All("/ext/:slug/*", guard, ep.handleRequest)
	router.All("/ext/:slug", guard, ep.handleRequest)
}

func (ep *ExtensionProxy) handleRequest(c *fiber.Ctx) error {
	slug := c.Params("slug")
	if slug == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "missing extension slug"})
	}

	// Get plugin client for this extension.
	client := ep.pluginMgr.GetClient(slug)
	if client == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "extension not found or not running"})
	}

	// Build headers map, stripping sensitive headers to prevent token
	// leakage to plugins. Also drop any client-supplied
	// X-Forwarded-For / X-Real-IP — those are spoofable, and plugins
	// rely on them for per-IP rate limiting; the kernel rewrites them
	// below with c.IP() (Fiber's authoritative remote address).
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
	// Insert the trusted remote IP. Plugins read X-Forwarded-For for
	// historical reasons; this overwrite means downstream code keeps
	// working without each plugin having to learn a new header name.
	headers["X-Forwarded-For"] = c.IP()

	// Build query params map.
	queryParams := make(map[string]string)
	c.Request().URI().QueryArgs().VisitAll(func(key, value []byte) {
		queryParams[string(key)] = string(value)
	})

	// Path params (the wildcard part after /ext/:slug/).
	pathParams := make(map[string]string)
	pathParams["slug"] = slug
	// The "*" param contains the rest of the path.
	wildcard := c.Params("*")
	if wildcard != "" {
		pathParams["path"] = wildcard
	}

	// Get authenticated user info.
	var userID uint64
	if user := auth.GetCurrentUser(c); user != nil {
		userID = uint64(user.ID)
		headers["X-User-Email"] = user.Email
		if user.FullName != nil {
			headers["X-User-Name"] = *user.FullName
		}
	}

	// Relative path: the wildcard portion after /ext/:slug/
	relativePath := "/" + c.Params("*")

	req := &pb.PluginHTTPRequest{
		Method:      c.Method(),
		Path:        relativePath,
		Headers:     headers,
		Body:        c.Body(),
		QueryParams: queryParams,
		PathParams:  pathParams,
		UserId:      userID,
	}

	resp, err := client.HandleHTTPRequest(req)
	if err != nil {
		log.Printf("[extension-proxy] gRPC error from %s: %v", slug, err)
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{"error": "plugin request failed"})
	}

	// Debug: log first 500 chars of response body
	bodyPreview := string(resp.Body)
	if len(bodyPreview) > 500 {
		bodyPreview = bodyPreview[:500]
	}
	log.Printf("[extension-proxy] %s %s -> %d (%d bytes): %s", c.Method(), relativePath, resp.StatusCode, len(resp.Body), bodyPreview)

	// Write response headers.
	for k, v := range resp.Headers {
		c.Set(k, v)
	}

	return c.Status(int(resp.StatusCode)).Send(resp.Body)
}
