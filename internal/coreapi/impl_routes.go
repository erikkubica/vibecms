package coreapi

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
)

var (
	routeRegistryMu sync.Mutex
	routeRegistry   = map[string]bool{}
)

func routeKey(method, path string) string {
	return strings.ToUpper(method) + " " + path
}

func (c *coreImpl) RegisterRoute(ctx context.Context, method, path string, meta RouteMeta) error {
	method = strings.ToUpper(strings.TrimSpace(method))
	path = strings.TrimSpace(path)

	if method == "" {
		return NewValidation("route method is required")
	}
	validMethods := map[string]bool{
		"GET": true, "POST": true, "PUT": true, "PATCH": true, "DELETE": true, "HEAD": true, "OPTIONS": true,
	}
	if !validMethods[method] {
		return NewValidation(fmt.Sprintf("unsupported HTTP method: %s", method))
	}
	if path == "" || path[0] != '/' {
		return NewValidation("route path must start with /")
	}

	key := routeKey(method, path)

	routeRegistryMu.Lock()
	defer routeRegistryMu.Unlock()

	if routeRegistry[key] {
		return NewValidation(fmt.Sprintf("route already registered: %s %s", method, path))
	}

	routeRegistry[key] = true
	log.Printf("[coreapi] route registered: %s %s", method, path)
	return nil
}

func (c *coreImpl) RemoveRoute(ctx context.Context, method, path string) error {
	method = strings.ToUpper(strings.TrimSpace(method))
	path = strings.TrimSpace(path)
	key := routeKey(method, path)

	routeRegistryMu.Lock()
	defer routeRegistryMu.Unlock()

	if !routeRegistry[key] {
		return NewNotFound("route", key)
	}

	delete(routeRegistry, key)
	log.Printf("[coreapi] route removed: %s %s", method, path)
	return nil
}
