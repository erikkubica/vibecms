package cms

import (
	"log"

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

// RegisterRoutes registers the catch-all proxy route on the given router.
func (ep *ExtensionProxy) RegisterRoutes(router fiber.Router) {
	router.All("/ext/:slug/*", ep.handleRequest)
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

	// Build headers map.
	headers := make(map[string]string)
	c.Request().Header.VisitAll(func(key, value []byte) {
		headers[string(key)] = string(value)
	})

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

	// Get authenticated user ID.
	var userID uint64
	if user := auth.GetCurrentUser(c); user != nil {
		userID = uint64(user.ID)
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

	// Write response headers.
	for k, v := range resp.Headers {
		c.Set(k, v)
	}

	return c.Status(int(resp.StatusCode)).Send(resp.Body)
}
