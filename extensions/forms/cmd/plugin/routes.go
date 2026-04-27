package main

import (
	"context"
	"strconv"
	"strings"

	pb "vibecms/pkg/plugin/proto"
)

// routeRequest dispatches an incoming HTTP request to the appropriate handler.
// HandleHTTPRequest is a thin wrapper around this function.
func (p *FormsPlugin) routeRequest(ctx context.Context, req *pb.PluginHTTPRequest) (*pb.PluginHTTPResponse, error) {
	path := strings.TrimSuffix(req.GetPath(), "/")
	method := strings.ToUpper(req.GetMethod())

	// Strip any known prefixes so we work with both the admin proxy
	// (which sends relative paths like "/", "/preview", "/123") and
	// the public proxy (which sends full paths like "/forms/submit/contact").
	path = strings.TrimPrefix(path, "/admin/api/ext/forms")
	path = strings.TrimPrefix(path, "/api/ext/forms")
	path = strings.TrimPrefix(path, "/")
	path = strings.TrimSuffix(path, "/")

	// Public submit route: forms/submit/{slug}
	if strings.HasPrefix(path, "forms/submit/") && method == "POST" {
		slug := strings.TrimPrefix(path, "forms/submit/")
		return p.handleSubmit(ctx, slug, req)
	}

	// Public render route: forms/render/{slug}
	if strings.HasPrefix(path, "forms/render/") && method == "GET" {
		slug := strings.TrimPrefix(path, "forms/render/")
		return p.handleRender(ctx, slug)
	}

	// Admin routes (relative paths from proxy)
	if path == "defaults/layout" && method == "GET" {
		style := req.GetQueryParams()["style"]
		return jsonResponse(200, map[string]string{"layout": defaultLayoutForStyle(style)}), nil
	}
	if path == "preview" && method == "POST" {
		return p.handlePreview(ctx, req)
	}
	if path == "submissions/export" && method == "GET" {
		return p.handleCSVExport(ctx, req)
	}
	if path == "submissions/bulk" && method == "POST" {
		return p.handleSubmissionsBulk(ctx, req.GetBody())
	}
	// Submissions by ID: submissions/{id}
	if strings.HasPrefix(path, "submissions/") {
		rest := strings.TrimPrefix(path, "submissions/")
		if subID, err := strconv.ParseUint(rest, 10, 64); err == nil && subID > 0 {
			switch method {
			case "PATCH":
				return p.handleSubmissionPatch(ctx, uint(subID), req.GetBody())
			case "DELETE":
				return p.handleSubmissionDelete(ctx, uint(subID))
			case "GET":
				return p.handleSubmissionGet(ctx, uint(subID))
			}
		}
	}
	if path == "submissions" || strings.HasPrefix(path, "submissions") {
		return p.handleSubmissions(ctx, req)
	}
	if path == "" {
		if method == "GET" {
			return p.handleListForms(ctx, req)
		}
		if method == "POST" {
			return p.handleCreateForm(ctx, req)
		}
	}

	// Import: POST /import
	if path == "import" && method == "POST" {
		return p.handleFormImport(ctx, req.GetBody())
	}

	// Routes with a numeric ID and sub-path: {id}/duplicate, {id}/export, {id}/webhooks
	// Also handles: {id}/notifications/{idx}/test
	parts := strings.SplitN(path, "/", 2)
	if len(parts) == 2 {
		if rid, rerr := strconv.ParseUint(parts[0], 10, 64); rerr == nil && rid > 0 {
			switch {
			case parts[1] == "duplicate" && method == "POST":
				return p.handleFormDuplicate(ctx, uint(rid))
			case parts[1] == "export" && method == "GET":
				return p.handleFormExport(ctx, uint(rid))
			case parts[1] == "webhooks" && method == "GET":
				return p.handleWebhookLogs(ctx, uint(rid))
			}

			// Notification test: {id}/notifications/{idx}/test
			if method == "POST" && strings.HasPrefix(parts[1], "notifications/") && strings.HasSuffix(parts[1], "/test") {
				notifParts := strings.Split(parts[1], "/")
				// notifParts = ["notifications", "{idx}", "test"]
				if len(notifParts) == 3 && notifParts[0] == "notifications" && notifParts[2] == "test" {
					if idx, err := strconv.Atoi(notifParts[1]); err == nil && idx >= 0 {
						return p.handleNotificationTest(ctx, uint(rid), idx, req)
					}
				}
			}
		}
	}

	// Single form CRUD: {id}
	id, err := strconv.ParseUint(path, 10, 64)
	if err == nil && id > 0 {
		switch method {
		case "GET":
			return p.handleGetForm(ctx, uint(id))
		case "PUT":
			return p.handleUpdateForm(ctx, uint(id), req.GetBody())
		case "DELETE":
			return p.handleDeleteForm(ctx, uint(id))
		}
	}

	return jsonError(404, "NOT_FOUND", "Route not found"), nil
}
