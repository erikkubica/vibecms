package main

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"vibecms/internal/coreapi"
	pb "vibecms/pkg/plugin/proto"
)

var slugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

// validateFormFields checks name + slug shape on create/update payloads.
// Returns a validation response (422) or nil on success.
// excludeID is the form being updated (excluded from slug-uniqueness checks); 0 on create.
func validateFormFields(ctx context.Context, host coreapi.CoreAPI, data map[string]any, excludeID uint) *pb.PluginHTTPResponse {
	fields := map[string]string{}

	name, _ := data["name"].(string)
	if strings.TrimSpace(name) == "" {
		fields["name"] = "Name is required"
	} else if len(name) > 200 {
		fields["name"] = "Name must be 200 characters or fewer"
	}

	slug, _ := data["slug"].(string)
	slug = strings.TrimSpace(slug)
	switch {
	case slug == "":
		fields["slug"] = "Slug is required"
	case len(slug) > 100:
		fields["slug"] = "Slug must be 100 characters or fewer"
	case !slugPattern.MatchString(slug):
		fields["slug"] = "Slug must be lowercase letters, numbers, and hyphens (no leading/trailing or repeated hyphens)"
	default:
		res, err := host.DataQuery(ctx, formsTable, coreapi.DataStoreQuery{
			Where: map[string]any{"slug": slug},
			Limit: 1,
		})
		if err == nil && res.Total > 0 {
			taken := true
			if excludeID > 0 {
				if existing, ok := res.Rows[0]["id"].(float64); ok && uint(existing) == excludeID {
					taken = false
				}
			}
			if taken {
				fields["slug"] = "Slug is already in use"
			}
		}
	}

	if len(fields) == 0 {
		return nil
	}
	return jsonResponse(422, map[string]any{
		"error":   "VALIDATION_FAILED",
		"message": "Form validation failed",
		"fields":  fields,
	})
}

// --- Form Handlers ---

func (p *FormsPlugin) handleListForms(ctx context.Context, req *pb.PluginHTTPRequest) (*pb.PluginHTTPResponse, error) {
	result, err := p.host.DataQuery(ctx, formsTable, coreapi.DataStoreQuery{
		OrderBy: "created_at DESC",
	})
	if err != nil {
		return jsonError(500, "QUERY_FAILED", err.Error()), nil
	}
	for i, row := range result.Rows {
		result.Rows[i] = normalizeForm(row)
		result.Rows[i] = enrichFormWithStats(ctx, p, row)
	}
	return jsonResponse(200, result), nil
}

// enrichFormWithStats appends submission_count and last_submission_at to a form row.
// Uses N+1 queries (acceptable for admin-page form lists which are bounded to <50 forms).
func enrichFormWithStats(ctx context.Context, p *FormsPlugin, form map[string]any) map[string]any {
	formID, ok := form["id"].(float64)
	if !ok {
		form["submission_count"] = 0
		form["last_submission_at"] = nil
		return form
	}

	countRes, err := p.host.DataQuery(ctx, submissionsTable, coreapi.DataStoreQuery{
		Where: map[string]any{"form_id": int(formID)},
	})
	if err != nil {
		form["submission_count"] = 0
		form["last_submission_at"] = nil
		return form
	}
	form["submission_count"] = countRes.Total

	// Get the most recent submission timestamp
	latestRes, err := p.host.DataQuery(ctx, submissionsTable, coreapi.DataStoreQuery{
		Where:   map[string]any{"form_id": int(formID)},
		OrderBy: "created_at DESC",
		Limit:   1,
	})
	if err != nil || len(latestRes.Rows) == 0 {
		form["last_submission_at"] = nil
	} else {
		form["last_submission_at"] = latestRes.Rows[0]["created_at"]
	}
	return form
}

func (p *FormsPlugin) handleCreateForm(ctx context.Context, req *pb.PluginHTTPRequest) (*pb.PluginHTTPResponse, error) {
	var data map[string]any
	if err := json.Unmarshal(req.GetBody(), &data); err != nil {
		return jsonError(400, "INVALID_JSON", err.Error()), nil
	}

	if resp := validateFormFields(ctx, p.host, data, 0); resp != nil {
		return resp, nil
	}

	res, err := p.host.DataCreate(ctx, formsTable, data)
	if err != nil {
		return jsonError(500, "CREATE_FAILED", err.Error()), nil
	}
	return jsonResponse(201, normalizeForm(res)), nil
}

func (p *FormsPlugin) handleGetForm(ctx context.Context, id uint) (*pb.PluginHTTPResponse, error) {
	res, err := p.host.DataGet(ctx, formsTable, id)
	if err != nil {
		return jsonError(404, "NOT_FOUND", "Form not found"), nil
	}
	return jsonResponse(200, normalizeForm(res)), nil
}

func (p *FormsPlugin) handleUpdateForm(ctx context.Context, id uint, body []byte) (*pb.PluginHTTPResponse, error) {
	var data map[string]any
	if err := json.Unmarshal(body, &data); err != nil {
		return jsonError(400, "INVALID_JSON", err.Error()), nil
	}
	// Remove protected fields
	delete(data, "id")
	delete(data, "created_at")

	if resp := validateFormFields(ctx, p.host, data, id); resp != nil {
		return resp, nil
	}

	data["updated_at"] = time.Now()

	err := p.host.DataUpdate(ctx, formsTable, id, data)
	if err != nil {
		return jsonError(500, "UPDATE_FAILED", err.Error()), nil
	}
	return jsonResponse(200, map[string]string{"status": "updated"}), nil
}

func (p *FormsPlugin) handleDeleteForm(ctx context.Context, id uint) (*pb.PluginHTTPResponse, error) {
	err := p.host.DataDelete(ctx, formsTable, id)
	if err != nil {
		return jsonError(500, "DELETE_FAILED", err.Error()), nil
	}
	return jsonResponse(200, map[string]string{"status": "deleted"}), nil
}

// handleFormDuplicate creates a copy of the form with a unique slug.
// POST /admin/api/ext/forms/{id}/duplicate
func (p *FormsPlugin) handleFormDuplicate(ctx context.Context, id uint) (*pb.PluginHTTPResponse, error) {
	src, err := p.host.DataGet(ctx, formsTable, id)
	if err != nil {
		return jsonError(404, "FORM_NOT_FOUND", "Form not found"), nil
	}
	src = normalizeForm(src)

	name, _ := src["name"].(string)
	slug, _ := src["slug"].(string)
	newName := name + " (Copy)"
	newSlug := uniqueSlug(ctx, p.host, slug+"-copy")

	copyData := map[string]any{
		"name":          newName,
		"slug":          newSlug,
		"fields":        src["fields"],
		"layout":        src["layout"],
		"notifications": src["notifications"],
		"settings":      src["settings"],
	}
	res, err := p.host.DataCreate(ctx, formsTable, copyData)
	if err != nil {
		return jsonError(500, "CREATE_FAILED", err.Error()), nil
	}
	return jsonResponse(201, normalizeForm(res)), nil
}

// handleFormExport returns the form definition as a downloadable JSON file.
// GET /admin/api/ext/forms/{id}/export
func (p *FormsPlugin) handleFormExport(ctx context.Context, id uint) (*pb.PluginHTTPResponse, error) {
	res, err := p.host.DataGet(ctx, formsTable, id)
	if err != nil {
		return jsonError(404, "FORM_NOT_FOUND", "Form not found"), nil
	}
	form := normalizeForm(res)
	delete(form, "id")
	delete(form, "created_at")
	delete(form, "updated_at")

	slug, _ := form["slug"].(string)
	body, _ := json.MarshalIndent(form, "", "  ")
	return &pb.PluginHTTPResponse{
		StatusCode: 200,
		Headers: map[string]string{
			"Content-Type":        "application/json",
			"Content-Disposition": fmt.Sprintf(`attachment; filename="form-%s.json"`, slug),
		},
		Body: body,
	}, nil
}

// handleFormImport creates a new form from a JSON body, auto-suffixing the slug on collision.
// POST /admin/api/ext/forms/import
func (p *FormsPlugin) handleFormImport(ctx context.Context, body []byte) (*pb.PluginHTTPResponse, error) {
	var data map[string]any
	if err := json.Unmarshal(body, &data); err != nil {
		return jsonError(400, "INVALID_JSON", err.Error()), nil
	}

	if resp := validateFormFields(ctx, p.host, data, 0); resp != nil {
		// Slug collision is expected on import — auto-suffix instead of failing.
		// Re-run with bumped slug only when the only issue is slug uniqueness.
		slug, _ := data["slug"].(string)
		data["slug"] = uniqueSlug(ctx, p.host, slug)
		if resp2 := validateFormFields(ctx, p.host, data, 0); resp2 != nil {
			return resp2, nil
		}
	}
	delete(data, "id")
	delete(data, "created_at")
	delete(data, "updated_at")

	res, err := p.host.DataCreate(ctx, formsTable, data)
	if err != nil {
		return jsonError(500, "CREATE_FAILED", err.Error()), nil
	}
	return jsonResponse(201, normalizeForm(res)), nil
}

// uniqueSlug appends -2, -3, etc. until there is no collision in the forms table.
func uniqueSlug(ctx context.Context, host coreapi.CoreAPI, base string) string {
	candidate := base
	n := 2
	for {
		res, err := host.DataQuery(ctx, formsTable, coreapi.DataStoreQuery{
			Where: map[string]any{"slug": candidate},
			Limit: 1,
		})
		if err != nil || res.Total == 0 {
			return candidate
		}
		candidate = fmt.Sprintf("%s-%d", base, n)
		n++
	}
}
