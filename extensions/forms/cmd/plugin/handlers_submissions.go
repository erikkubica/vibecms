package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"vibecms/internal/coreapi"
	pb "vibecms/pkg/plugin/proto"
)

// --- Submission Handlers ---

// handleSubmissionPatch handles PATCH /submissions/{id} — update status.
func (p *FormsPlugin) handleSubmissionPatch(ctx context.Context, id uint, body []byte) (*pb.PluginHTTPResponse, error) {
	var req struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		return jsonError(400, "INVALID_JSON", err.Error()), nil
	}
	if req.Status != "unread" && req.Status != "read" && req.Status != "archived" {
		return jsonError(422, "VALIDATION_FAILED", "status must be unread|read|archived"), nil
	}
	if err := p.host.DataUpdate(ctx, submissionsTable, id, map[string]any{"status": req.Status}); err != nil {
		return jsonError(500, "UPDATE_FAILED", err.Error()), nil
	}
	return jsonResponse(200, map[string]string{"status": "updated"}), nil
}

// handleSubmissionGet handles GET /submissions/{id} — returns enriched submission with form_fields.
func (p *FormsPlugin) handleSubmissionGet(ctx context.Context, id uint) (*pb.PluginHTTPResponse, error) {
	row, err := p.host.DataGet(ctx, submissionsTable, id)
	if err != nil {
		return jsonError(404, "SUBMISSION_NOT_FOUND", "Submission not found"), nil
	}
	row = normalizeSubmission(row)

	// Enrich with form data
	if formIDVal, ok := row["form_id"].(float64); ok {
		formID := uint(formIDVal)
		if f, err := p.host.DataGet(ctx, formsTable, formID); err == nil {
			form := normalizeForm(f)
			if n, ok := form["name"].(string); ok {
				row["form_name"] = n
			}
			row["form_fields"] = getFormFields(form)
		}
	}

	return jsonResponse(200, row), nil
}

// handleSubmissionDelete handles DELETE /submissions/{id}.
func (p *FormsPlugin) handleSubmissionDelete(ctx context.Context, id uint) (*pb.PluginHTTPResponse, error) {
	// Load submission to clean up file fields
	row, err := p.host.DataGet(ctx, submissionsTable, id)
	if err != nil {
		return jsonError(404, "SUBMISSION_NOT_FOUND", "Submission not found"), nil
	}
	data := parseJSONMap(row["data"])
	for _, v := range data {
		deleteFileValueIfPresent(ctx, p.host, v)
	}
	if err := p.host.DataDelete(ctx, submissionsTable, id); err != nil {
		return jsonError(500, "DELETE_FAILED", err.Error()), nil
	}
	return jsonResponse(200, map[string]string{"status": "deleted"}), nil
}

// handleSubmissions handles GET /submissions — server-side pagination, filtering, and search.
func (p *FormsPlugin) handleSubmissions(ctx context.Context, req *pb.PluginHTTPRequest) (*pb.PluginHTTPResponse, error) {
	params := req.GetQueryParams()

	// Pagination
	page := 1
	perPage := 25
	if v, err := strconv.Atoi(params["page"]); err == nil && v > 0 {
		page = v
	}
	if v, err := strconv.Atoi(params["per_page"]); err == nil && v > 0 && v <= 100 {
		perPage = v
	}
	offset := (page - 1) * perPage

	// Build raw WHERE conditions (non-status; status counts are computed separately)
	var baseConditions []string
	var baseArgs []any

	if formIDStr := params["form_id"]; formIDStr != "" {
		if formID, err := strconv.Atoi(formIDStr); err == nil {
			baseConditions = append(baseConditions, "form_id = ?")
			baseArgs = append(baseArgs, formID)
		}
	}

	if dateFrom := params["date_from"]; dateFrom != "" {
		if t, err := time.Parse("2006-01-02", dateFrom); err == nil {
			baseConditions = append(baseConditions, "created_at >= ?")
			baseArgs = append(baseArgs, t)
		}
	}

	if dateTo := params["date_to"]; dateTo != "" {
		if t, err := time.Parse("2006-01-02", dateTo); err == nil {
			baseConditions = append(baseConditions, "created_at < ?")
			baseArgs = append(baseArgs, t.Add(24*time.Hour))
		}
	}

	if search := params["search"]; search != "" {
		baseConditions = append(baseConditions, "data::text ILIKE ?")
		baseArgs = append(baseArgs, "%"+search+"%")
	}

	statusFilter := params["status"]
	conditions := append([]string{}, baseConditions...)
	args := append([]any{}, baseArgs...)
	if statusFilter != "" {
		conditions = append(conditions, "status = ?")
		args = append(args, statusFilter)
	}

	rawSQL := ""
	if len(conditions) > 0 {
		rawSQL = strings.Join(conditions, " AND ")
	}

	query := coreapi.DataStoreQuery{
		OrderBy: "created_at DESC",
		Limit:   perPage,
		Offset:  offset,
		Raw:     rawSQL,
		Args:    args,
	}

	result, err := p.host.DataQuery(ctx, submissionsTable, query)
	if err != nil {
		return jsonError(500, "QUERY_FAILED", err.Error()), nil
	}

	// Enrich rows with form_name
	formIDs := map[uint]bool{}
	for _, r := range result.Rows {
		if v, ok := r["form_id"].(float64); ok {
			formIDs[uint(v)] = true
		}
	}
	nameByID := map[uint]string{}
	for id := range formIDs {
		if f, err := p.host.DataGet(ctx, formsTable, id); err == nil {
			if n, ok := f["name"].(string); ok {
				nameByID[id] = n
			}
		}
	}
	for i, r := range result.Rows {
		result.Rows[i] = normalizeSubmission(r)
		if v, ok := result.Rows[i]["form_id"].(float64); ok {
			result.Rows[i]["form_name"] = nameByID[uint(v)]
		}
	}

	totalPages := int64(1)
	if result.Total > 0 {
		totalPages = (result.Total + int64(perPage) - 1) / int64(perPage)
	}

	// Per-status counts (independent of current status filter, but respect base filters)
	statusCounts := map[string]int64{"unread": 0, "read": 0, "archived": 0}
	for s := range statusCounts {
		cConds := append([]string{}, baseConditions...)
		cArgs := append([]any{}, baseArgs...)
		cConds = append(cConds, "status = ?")
		cArgs = append(cArgs, s)
		cRaw := strings.Join(cConds, " AND ")
		cRes, err := p.host.DataQuery(ctx, submissionsTable, coreapi.DataStoreQuery{
			Limit: 1,
			Raw:   cRaw,
			Args:  cArgs,
		})
		if err == nil {
			statusCounts[s] = cRes.Total
		}
	}
	statusCounts["all"] = statusCounts["unread"] + statusCounts["read"] + statusCounts["archived"]

	return jsonResponse(200, map[string]any{
		"rows":          result.Rows,
		"total":         result.Total,
		"page":          page,
		"per_page":      perPage,
		"total_pages":   totalPages,
		"status_counts": statusCounts,
	}), nil
}

// handleSubmissionsBulk handles POST /submissions/bulk — batch status update or delete.
func (p *FormsPlugin) handleSubmissionsBulk(ctx context.Context, body []byte) (*pb.PluginHTTPResponse, error) {
	var req struct {
		Action string `json:"action"`
		IDs    []uint `json:"ids"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		return jsonError(400, "INVALID_JSON", err.Error()), nil
	}
	if len(req.IDs) == 0 {
		return jsonResponse(200, map[string]int{"count": 0}), nil
	}
	if len(req.IDs) > 1000 {
		return jsonError(422, "VALIDATION_FAILED", "max 1000 ids per bulk"), nil
	}

	var statusVal string
	switch req.Action {
	case "mark_read":
		statusVal = "read"
	case "mark_unread":
		statusVal = "unread"
	case "archive":
		statusVal = "archived"
	case "delete":
		// handled below
	default:
		return jsonError(422, "VALIDATION_FAILED", fmt.Sprintf("unknown action: %s", req.Action)), nil
	}

	count := 0
	for _, id := range req.IDs {
		if req.Action == "delete" {
			if sub, err := p.host.DataGet(ctx, submissionsTable, id); err == nil {
				data := parseJSONMap(sub["data"])
				for _, v := range data {
					deleteFileValueIfPresent(ctx, p.host, v)
				}
			}
			if err := p.host.DataDelete(ctx, submissionsTable, id); err == nil {
				count++
			}
		} else {
			if err := p.host.DataUpdate(ctx, submissionsTable, id, map[string]any{"status": statusVal}); err == nil {
				count++
			}
		}
	}
	return jsonResponse(200, map[string]int{"count": count}), nil
}
