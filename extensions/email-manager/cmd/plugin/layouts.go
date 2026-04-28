package main

import (
	"context"
	"encoding/json"
	"math"
	"strconv"
	"time"

	"vibecms/internal/coreapi"
	pb "vibecms/pkg/plugin/proto"
)

// This file owns the email-layout CRUD handlers. Layouts are HTML
// scaffolds that wrap rendered template bodies — header/footer,
// brand colors, etc. — applied per email at send time.

func (p *EmailManagerPlugin) routeLayouts(ctx context.Context, method, subPath string, req *pb.PluginHTTPRequest) (*pb.PluginHTTPResponse, error) {
	if subPath == "" {
		switch method {
		case "GET":
			return p.listLayouts(ctx, req)
		case "POST":
			return p.createLayout(ctx, req.GetBody())
		default:
			return jsonError(405, "METHOD_NOT_ALLOWED", "Method not allowed"), nil
		}
	}
	id, err := strconv.ParseUint(subPath, 10, 32)
	if err != nil {
		return jsonError(400, "INVALID_ID", "Invalid layout ID"), nil
	}
	switch method {
	case "GET":
		return p.getLayout(ctx, uint(id))
	case "PUT", "PATCH":
		return p.updateLayout(ctx, uint(id), req.GetBody())
	case "DELETE":
		return p.deleteLayout(ctx, uint(id))
	default:
		return jsonError(405, "METHOD_NOT_ALLOWED", "Method not allowed"), nil
	}
}

func (p *EmailManagerPlugin) listLayouts(ctx context.Context, req *pb.PluginHTTPRequest) (*pb.PluginHTTPResponse, error) {
	params := req.GetQueryParams()
	page, perPage := parsePagination(params)

	result, err := p.host.DataQuery(ctx, "email_layouts", coreapi.DataStoreQuery{
		OrderBy: "id ASC",
		Limit:   perPage,
		Offset:  (page - 1) * perPage,
	})
	if err != nil {
		return jsonError(500, "LIST_FAILED", "Failed to list email layouts"), nil
	}

	// Strip heavy fields from list response.
	rows := stripFields(result.Rows, "body_template")

	totalPages := int(math.Ceil(float64(result.Total) / float64(perPage)))
	return jsonResponse(200, map[string]any{
		"data": rows,
		"meta": map[string]any{
			"total":       result.Total,
			"page":        page,
			"per_page":    perPage,
			"total_pages": totalPages,
		},
	}), nil
}

func (p *EmailManagerPlugin) getLayout(ctx context.Context, id uint) (*pb.PluginHTTPResponse, error) {
	row, err := p.host.DataGet(ctx, "email_layouts", id)
	if err != nil {
		if isNotFound(err) {
			return jsonError(404, "NOT_FOUND", "Email layout not found"), nil
		}
		return jsonError(500, "FETCH_FAILED", "Failed to fetch email layout"), nil
	}
	return jsonResponse(200, map[string]any{"data": row}), nil
}

func (p *EmailManagerPlugin) createLayout(ctx context.Context, body []byte) (*pb.PluginHTTPResponse, error) {
	var req struct {
		Name         string `json:"name"`
		LanguageID   *int   `json:"language_id"`
		BodyTemplate string `json:"body_template"`
		IsDefault    bool   `json:"is_default"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		return jsonError(400, "INVALID_BODY", "Invalid request body"), nil
	}

	fields := map[string]string{}
	if req.Name == "" {
		fields["name"] = "Name is required"
	}
	if req.BodyTemplate == "" {
		fields["body_template"] = "Body template is required"
	}
	if len(fields) > 0 {
		return jsonValidationError(fields), nil
	}

	record := map[string]any{
		"name":          req.Name,
		"body_template": req.BodyTemplate,
		"is_default":    req.IsDefault,
	}
	if req.LanguageID != nil {
		record["language_id"] = *req.LanguageID
	}

	created, err := p.host.DataCreate(ctx, "email_layouts", record)
	if err != nil {
		return jsonError(500, "CREATE_FAILED", "Failed to create email layout"), nil
	}

	return jsonResponse(201, map[string]any{"data": created}), nil
}

func (p *EmailManagerPlugin) updateLayout(ctx context.Context, id uint, body []byte) (*pb.PluginHTTPResponse, error) {
	// Verify exists.
	_, err := p.host.DataGet(ctx, "email_layouts", id)
	if err != nil {
		if isNotFound(err) {
			return jsonError(404, "NOT_FOUND", "Email layout not found"), nil
		}
		return jsonError(500, "FETCH_FAILED", "Failed to fetch email layout"), nil
	}

	var updates map[string]any
	if err := json.Unmarshal(body, &updates); err != nil {
		return jsonError(400, "INVALID_BODY", "Invalid request body"), nil
	}

	// Strip read-only fields.
	delete(updates, "id")
	delete(updates, "created_at")
	delete(updates, "updated_at")

	if len(updates) == 0 {
		return jsonError(400, "NO_UPDATES", "No valid fields to update"), nil
	}

	// `language_id: null` means universal fallback. DataUpdate maps Go nil
	// to a SQL NULL parameter, so we pass it through directly — no DataExec
	// workaround (extensions can't call DataExec, it's internal-only).
	updates["updated_at"] = time.Now().Format(time.RFC3339)

	if err := p.host.DataUpdate(ctx, "email_layouts", id, updates); err != nil {
		return jsonError(500, "UPDATE_FAILED", "Failed to update email layout"), nil
	}

	row, err := p.host.DataGet(ctx, "email_layouts", id)
	if err != nil {
		return jsonError(500, "FETCH_FAILED", "Failed to fetch updated email layout"), nil
	}

	return jsonResponse(200, map[string]any{"data": row}), nil
}

func (p *EmailManagerPlugin) deleteLayout(ctx context.Context, id uint) (*pb.PluginHTTPResponse, error) {
	_, err := p.host.DataGet(ctx, "email_layouts", id)
	if err != nil {
		if isNotFound(err) {
			return jsonError(404, "NOT_FOUND", "Email layout not found"), nil
		}
		return jsonError(500, "FETCH_FAILED", "Failed to fetch email layout"), nil
	}

	if err := p.host.DataDelete(ctx, "email_layouts", id); err != nil {
		return jsonError(500, "DELETE_FAILED", "Failed to delete email layout"), nil
	}

	return &pb.PluginHTTPResponse{StatusCode: 204, Headers: map[string]string{"Content-Type": "application/json"}}, nil
}

// ---------------------------------------------------------------------------
// Email Settings
// ---------------------------------------------------------------------------

var maskedKeys = map[string]bool{
	"email_smtp_password":  true,
	"email_resend_api_key": true,
}
