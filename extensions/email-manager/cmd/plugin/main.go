package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	goplugin "github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"

	"vibecms/internal/coreapi"
	vibeplugin "vibecms/pkg/plugin"
	coreapipb "vibecms/pkg/plugin/coreapipb"
	pb "vibecms/pkg/plugin/proto"
)

// EmailManagerPlugin implements the ExtensionPlugin interface for email admin management.
type EmailManagerPlugin struct {
	host *coreapi.GRPCHostClient
}

func (p *EmailManagerPlugin) GetSubscriptions() ([]*pb.Subscription, error) {
	return nil, nil
}

func (p *EmailManagerPlugin) HandleEvent(action string, payload []byte) (*pb.EventResponse, error) {
	return &pb.EventResponse{Handled: false}, nil
}

func (p *EmailManagerPlugin) Shutdown() error {
	return nil
}

func (p *EmailManagerPlugin) Initialize(hostConn *grpc.ClientConn) error {
	p.host = coreapi.NewGRPCHostClient(coreapipb.NewVibeCMSHostClient(hostConn))
	return nil
}

func (p *EmailManagerPlugin) HandleHTTPRequest(req *pb.PluginHTTPRequest) (*pb.PluginHTTPResponse, error) {
	path := strings.TrimSuffix(req.GetPath(), "/")
	method := strings.ToUpper(req.GetMethod())
	ctx := context.Background()

	// Determine the resource and sub-path.
	// Paths arrive as: /templates, /templates/123, /rules, /rules/123, /logs, /settings, /settings/test, /logs/123/resend
	resource, subPath := splitResource(path)

	switch resource {
	case "templates", "email-templates":
		return p.routeTemplates(ctx, method, subPath, req)
	case "rules", "email-rules":
		return p.routeRules(ctx, method, subPath, req)
	case "logs", "email-logs":
		return p.routeLogs(ctx, method, subPath, req)
	case "settings", "email-settings":
		return p.routeSettings(ctx, method, subPath, req)
	case "layouts", "email-layouts":
		return p.routeLayouts(ctx, method, subPath, req)
	default:
		return jsonError(404, "NOT_FOUND", "Route not found"), nil
	}
}

// ---------------------------------------------------------------------------
// Routing helpers
// ---------------------------------------------------------------------------

func splitResource(path string) (string, string) {
	path = strings.TrimPrefix(path, "/")
	if path == "" {
		return "", ""
	}
	parts := strings.SplitN(path, "/", 2)
	resource := parts[0]
	sub := ""
	if len(parts) > 1 {
		sub = parts[1]
	}
	return resource, sub
}

func (p *EmailManagerPlugin) routeTemplates(ctx context.Context, method, subPath string, req *pb.PluginHTTPRequest) (*pb.PluginHTTPResponse, error) {
	if subPath == "" {
		switch method {
		case "GET":
			return p.listTemplates(ctx, req)
		case "POST":
			return p.createTemplate(ctx, req.GetBody())
		default:
			return jsonError(405, "METHOD_NOT_ALLOWED", "Method not allowed"), nil
		}
	}

	id := parseID(subPath, req.GetPathParams(), "id")
	if id == 0 {
		return jsonError(400, "INVALID_ID", "Template ID must be a valid integer"), nil
	}

	switch method {
	case "GET":
		return p.getTemplate(ctx, id)
	case "PUT", "PATCH":
		return p.updateTemplate(ctx, id, req.GetBody())
	case "DELETE":
		return p.deleteTemplate(ctx, id)
	default:
		return jsonError(405, "METHOD_NOT_ALLOWED", "Method not allowed"), nil
	}
}

func (p *EmailManagerPlugin) routeRules(ctx context.Context, method, subPath string, req *pb.PluginHTTPRequest) (*pb.PluginHTTPResponse, error) {
	if subPath == "" {
		switch method {
		case "GET":
			return p.listRules(ctx)
		case "POST":
			return p.createRule(ctx, req.GetBody())
		default:
			return jsonError(405, "METHOD_NOT_ALLOWED", "Method not allowed"), nil
		}
	}

	id := parseID(subPath, req.GetPathParams(), "id")
	if id == 0 {
		return jsonError(400, "INVALID_ID", "Rule ID must be a valid integer"), nil
	}

	switch method {
	case "GET":
		return p.getRule(ctx, id)
	case "PUT", "PATCH":
		return p.updateRule(ctx, id, req.GetBody())
	case "DELETE":
		return p.deleteRule(ctx, id)
	default:
		return jsonError(405, "METHOD_NOT_ALLOWED", "Method not allowed"), nil
	}
}

func (p *EmailManagerPlugin) routeLogs(ctx context.Context, method, subPath string, req *pb.PluginHTTPRequest) (*pb.PluginHTTPResponse, error) {
	if subPath == "" {
		if method == "GET" {
			return p.listLogs(ctx, req)
		}
		return jsonError(405, "METHOD_NOT_ALLOWED", "Method not allowed"), nil
	}

	// Check for /logs/:id/resend
	parts := strings.SplitN(subPath, "/", 2)
	id := parseIDStr(parts[0])
	if id == 0 {
		return jsonError(400, "INVALID_ID", "Log ID must be a valid integer"), nil
	}

	if len(parts) > 1 && parts[1] == "resend" && method == "POST" {
		return p.resendLog(ctx, id)
	}

	if method == "GET" {
		return p.getLog(ctx, id)
	}

	return jsonError(405, "METHOD_NOT_ALLOWED", "Method not allowed"), nil
}

func (p *EmailManagerPlugin) routeSettings(ctx context.Context, method, subPath string, req *pb.PluginHTTPRequest) (*pb.PluginHTTPResponse, error) {
	if subPath == "test" && method == "POST" {
		return p.testEmail(ctx, req)
	}

	if subPath == "" {
		switch method {
		case "GET":
			return p.getSettings(ctx)
		case "POST", "PUT":
			return p.saveSettings(ctx, req.GetBody())
		default:
			return jsonError(405, "METHOD_NOT_ALLOWED", "Method not allowed"), nil
		}
	}

	return jsonError(404, "NOT_FOUND", "Route not found"), nil
}

// ---------------------------------------------------------------------------
// Email Templates
// ---------------------------------------------------------------------------

func (p *EmailManagerPlugin) listTemplates(ctx context.Context, req *pb.PluginHTTPRequest) (*pb.PluginHTTPResponse, error) {
	params := req.GetQueryParams()
	page, perPage := parsePagination(params)

	result, err := p.host.DataQuery(ctx, "email_templates", coreapi.DataStoreQuery{
		OrderBy: "id ASC",
		Limit:   perPage,
		Offset:  (page - 1) * perPage,
	})
	if err != nil {
		return jsonError(500, "LIST_FAILED", "Failed to list email templates"), nil
	}

	// Strip heavy fields from list response.
	rows := stripFields(result.Rows, "body_template", "test_data")

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

func (p *EmailManagerPlugin) getTemplate(ctx context.Context, id uint) (*pb.PluginHTTPResponse, error) {
	row, err := p.host.DataGet(ctx, "email_templates", id)
	if err != nil {
		if isNotFound(err) {
			return jsonError(404, "NOT_FOUND", "Email template not found"), nil
		}
		return jsonError(500, "FETCH_FAILED", "Failed to fetch email template"), nil
	}
	return jsonResponse(200, map[string]any{"data": row}), nil
}

func (p *EmailManagerPlugin) createTemplate(ctx context.Context, body []byte) (*pb.PluginHTTPResponse, error) {
	var req struct {
		Slug            string `json:"slug"`
		Name            string `json:"name"`
		LanguageID      *int   `json:"language_id"`
		SubjectTemplate string `json:"subject_template"`
		BodyTemplate    string `json:"body_template"`
		TestData        any    `json:"test_data"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		return jsonError(400, "INVALID_BODY", "Invalid request body"), nil
	}

	fields := map[string]string{}
	if req.Slug == "" {
		fields["slug"] = "Slug is required"
	}
	if req.Name == "" {
		fields["name"] = "Name is required"
	}
	if req.SubjectTemplate == "" {
		fields["subject_template"] = "Subject template is required"
	}
	if req.BodyTemplate == "" {
		fields["body_template"] = "Body template is required"
	}
	if len(fields) > 0 {
		return jsonValidationError(fields), nil
	}

	testData := "{}"
	if req.TestData != nil {
		if td, err := json.Marshal(req.TestData); err == nil && string(td) != "null" {
			testData = string(td)
		}
	}

	record := map[string]any{
		"slug":             req.Slug,
		"name":             req.Name,
		"subject_template": req.SubjectTemplate,
		"body_template":    req.BodyTemplate,
		"test_data":        testData,
	}
	if req.LanguageID != nil {
		record["language_id"] = *req.LanguageID
	}

	created, err := p.host.DataCreate(ctx, "email_templates", record)
	if err != nil {
		return jsonError(500, "CREATE_FAILED", "Failed to create email template"), nil
	}

	return jsonResponse(201, map[string]any{"data": created}), nil
}

func (p *EmailManagerPlugin) updateTemplate(ctx context.Context, id uint, body []byte) (*pb.PluginHTTPResponse, error) {
	// Verify exists.
	_, err := p.host.DataGet(ctx, "email_templates", id)
	if err != nil {
		if isNotFound(err) {
			return jsonError(404, "NOT_FOUND", "Email template not found"), nil
		}
		return jsonError(500, "FETCH_FAILED", "Failed to fetch email template"), nil
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

	// Marshal test_data if it's a complex type.
	if td, ok := updates["test_data"]; ok && td != nil {
		switch td.(type) {
		case string:
			// already a string, fine
		default:
			if b, err := json.Marshal(td); err == nil {
				updates["test_data"] = string(b)
			}
		}
	}

	// Handle language_id: null means universal fallback.
	// Use DataExec for the SET language_id = NULL case since DataUpdate
	// may not handle untyped nil correctly through gRPC/database/sql.
	langIDNull := false
	if lid, ok := updates["language_id"]; ok && lid == nil {
		langIDNull = true
		delete(updates, "language_id")
	}

	updates["updated_at"] = time.Now().Format(time.RFC3339)

	if err := p.host.DataUpdate(ctx, "email_templates", id, updates); err != nil {
		return jsonError(500, "UPDATE_FAILED", "Failed to update email template"), nil
	}

	if langIDNull {
		_, err := p.host.DataExec(ctx, "UPDATE email_templates SET language_id = NULL WHERE id = ?", id)
		if err != nil {
			return jsonError(500, "UPDATE_FAILED", "Failed to clear language on email template"), nil
		}
	}

	row, err := p.host.DataGet(ctx, "email_templates", id)
	if err != nil {
		return jsonError(500, "FETCH_FAILED", "Failed to fetch updated email template"), nil
	}

	return jsonResponse(200, map[string]any{"data": row}), nil
}

func (p *EmailManagerPlugin) deleteTemplate(ctx context.Context, id uint) (*pb.PluginHTTPResponse, error) {
	_, err := p.host.DataGet(ctx, "email_templates", id)
	if err != nil {
		if isNotFound(err) {
			return jsonError(404, "NOT_FOUND", "Email template not found"), nil
		}
		return jsonError(500, "FETCH_FAILED", "Failed to fetch email template"), nil
	}

	if err := p.host.DataDelete(ctx, "email_templates", id); err != nil {
		return jsonError(500, "DELETE_FAILED", "Failed to delete email template"), nil
	}

	return &pb.PluginHTTPResponse{StatusCode: 204, Headers: map[string]string{"Content-Type": "application/json"}}, nil
}

// ---------------------------------------------------------------------------
// Email Rules
// ---------------------------------------------------------------------------

func (p *EmailManagerPlugin) listRules(ctx context.Context) (*pb.PluginHTTPResponse, error) {
	result, err := p.host.DataQuery(ctx, "email_rules", coreapi.DataStoreQuery{
		OrderBy: "id ASC",
		Limit:   1000,
	})
	if err != nil {
		return jsonError(500, "LIST_FAILED", "Failed to list email rules"), nil
	}

	// Enrich rules with their template data.
	rows := result.Rows
	for i, row := range rows {
		tplID := toUint(row["template_id"])
		if tplID > 0 {
			tpl, err := p.host.DataGet(ctx, "email_templates", tplID)
			if err == nil {
				rows[i]["template"] = tpl
			}
		}
	}

	return jsonResponse(200, map[string]any{"data": rows}), nil
}

func (p *EmailManagerPlugin) getRule(ctx context.Context, id uint) (*pb.PluginHTTPResponse, error) {
	row, err := p.host.DataGet(ctx, "email_rules", id)
	if err != nil {
		if isNotFound(err) {
			return jsonError(404, "NOT_FOUND", "Email rule not found"), nil
		}
		return jsonError(500, "FETCH_FAILED", "Failed to fetch email rule"), nil
	}

	// Enrich with template.
	tplID := toUint(row["template_id"])
	if tplID > 0 {
		tpl, err := p.host.DataGet(ctx, "email_templates", tplID)
		if err == nil {
			row["template"] = tpl
		}
	}

	return jsonResponse(200, map[string]any{"data": row}), nil
}

func (p *EmailManagerPlugin) createRule(ctx context.Context, body []byte) (*pb.PluginHTTPResponse, error) {
	var req struct {
		Action         string  `json:"action"`
		NodeType       *string `json:"node_type"`
		TemplateID     int     `json:"template_id"`
		RecipientType  string  `json:"recipient_type"`
		RecipientValue string  `json:"recipient_value"`
		Enabled        *bool   `json:"enabled"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		return jsonError(400, "INVALID_BODY", "Invalid request body"), nil
	}

	fields := map[string]string{}
	if req.Action == "" {
		fields["action"] = "Action is required"
	}
	if req.TemplateID == 0 {
		fields["template_id"] = "Template ID is required"
	}
	if req.RecipientType == "" {
		fields["recipient_type"] = "Recipient type is required"
	}
	if (req.RecipientType == "role" || req.RecipientType == "fixed") && req.RecipientValue == "" {
		fields["recipient_value"] = "Recipient value is required"
	}
	if len(fields) > 0 {
		return jsonValidationError(fields), nil
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	record := map[string]any{
		"action":          req.Action,
		"template_id":     req.TemplateID,
		"recipient_type":  req.RecipientType,
		"recipient_value": req.RecipientValue,
		"enabled":         enabled,
	}
	if req.NodeType != nil {
		record["node_type"] = *req.NodeType
	}

	created, err := p.host.DataCreate(ctx, "email_rules", record)
	if err != nil {
		return jsonError(500, "CREATE_FAILED", "Failed to create email rule"), nil
	}

	// Enrich with template.
	tplID := toUint(created["template_id"])
	if tplID > 0 {
		tpl, tplErr := p.host.DataGet(ctx, "email_templates", tplID)
		if tplErr == nil {
			created["template"] = tpl
		}
	}

	return jsonResponse(201, map[string]any{"data": created}), nil
}

func (p *EmailManagerPlugin) updateRule(ctx context.Context, id uint, body []byte) (*pb.PluginHTTPResponse, error) {
	_, err := p.host.DataGet(ctx, "email_rules", id)
	if err != nil {
		if isNotFound(err) {
			return jsonError(404, "NOT_FOUND", "Email rule not found"), nil
		}
		return jsonError(500, "FETCH_FAILED", "Failed to fetch email rule"), nil
	}

	var updates map[string]any
	if err := json.Unmarshal(body, &updates); err != nil {
		return jsonError(400, "INVALID_BODY", "Invalid request body"), nil
	}

	delete(updates, "id")
	delete(updates, "created_at")
	delete(updates, "updated_at")

	if len(updates) == 0 {
		return jsonError(400, "NO_UPDATES", "No valid fields to update"), nil
	}

	updates["updated_at"] = time.Now().Format(time.RFC3339)

	if err := p.host.DataUpdate(ctx, "email_rules", id, updates); err != nil {
		return jsonError(500, "UPDATE_FAILED", "Failed to update email rule"), nil
	}

	row, err := p.host.DataGet(ctx, "email_rules", id)
	if err != nil {
		return jsonError(500, "FETCH_FAILED", "Failed to fetch updated email rule"), nil
	}

	// Enrich with template.
	tplID := toUint(row["template_id"])
	if tplID > 0 {
		tpl, tplErr := p.host.DataGet(ctx, "email_templates", tplID)
		if tplErr == nil {
			row["template"] = tpl
		}
	}

	return jsonResponse(200, map[string]any{"data": row}), nil
}

func (p *EmailManagerPlugin) deleteRule(ctx context.Context, id uint) (*pb.PluginHTTPResponse, error) {
	_, err := p.host.DataGet(ctx, "email_rules", id)
	if err != nil {
		if isNotFound(err) {
			return jsonError(404, "NOT_FOUND", "Email rule not found"), nil
		}
		return jsonError(500, "FETCH_FAILED", "Failed to fetch email rule"), nil
	}

	if err := p.host.DataDelete(ctx, "email_rules", id); err != nil {
		return jsonError(500, "DELETE_FAILED", "Failed to delete email rule"), nil
	}

	return &pb.PluginHTTPResponse{StatusCode: 204, Headers: map[string]string{"Content-Type": "application/json"}}, nil
}

// ---------------------------------------------------------------------------
// Email Logs
// ---------------------------------------------------------------------------

func (p *EmailManagerPlugin) listLogs(ctx context.Context, req *pb.PluginHTTPRequest) (*pb.PluginHTTPResponse, error) {
	params := req.GetQueryParams()
	page, perPage := parsePagination(params)

	query := coreapi.DataStoreQuery{
		OrderBy: "created_at DESC",
		Limit:   perPage,
		Offset:  (page - 1) * perPage,
	}

	// Build WHERE conditions via Raw.
	var conditions []string
	var args []any

	if status := params["status"]; status != "" {
		conditions = append(conditions, "status = ?")
		args = append(args, status)
	}
	if action := params["action"]; action != "" {
		conditions = append(conditions, "action = ?")
		args = append(args, action)
	}
	if recipient := params["recipient"]; recipient != "" {
		conditions = append(conditions, "recipient_email ILIKE ?")
		args = append(args, "%"+recipient+"%")
	}
	if dateFrom := params["date_from"]; dateFrom != "" {
		if t, err := time.Parse("2006-01-02", dateFrom); err == nil {
			conditions = append(conditions, "created_at >= ?")
			args = append(args, t.Format(time.RFC3339))
		}
	}
	if dateTo := params["date_to"]; dateTo != "" {
		if t, err := time.Parse("2006-01-02", dateTo); err == nil {
			conditions = append(conditions, "created_at < ?")
			args = append(args, t.AddDate(0, 0, 1).Format(time.RFC3339))
		}
	}

	if len(conditions) > 0 {
		query.Raw = strings.Join(conditions, " AND ")
		query.Args = args
	}

	result, err := p.host.DataQuery(ctx, "email_logs", query)
	if err != nil {
		return jsonError(500, "LIST_FAILED", "Failed to list email logs"), nil
	}

	totalPages := int(math.Ceil(float64(result.Total) / float64(perPage)))
	resp := map[string]any{
		"data": result.Rows,
		"meta": map[string]any{
			"total":       result.Total,
			"page":        page,
			"per_page":    perPage,
			"total_pages": totalPages,
		},
	}

	return jsonResponse(200, resp), nil
}

func (p *EmailManagerPlugin) getLog(ctx context.Context, id uint) (*pb.PluginHTTPResponse, error) {
	row, err := p.host.DataGet(ctx, "email_logs", id)
	if err != nil {
		if isNotFound(err) {
			return jsonError(404, "NOT_FOUND", "Email log not found"), nil
		}
		return jsonError(500, "FETCH_FAILED", "Failed to fetch email log"), nil
	}
	return jsonResponse(200, map[string]any{"data": row}), nil
}

func (p *EmailManagerPlugin) resendLog(ctx context.Context, id uint) (*pb.PluginHTTPResponse, error) {
	row, err := p.host.DataGet(ctx, "email_logs", id)
	if err != nil {
		if isNotFound(err) {
			return jsonError(404, "NOT_FOUND", "Email log not found"), nil
		}
		return jsonError(500, "FETCH_FAILED", "Failed to fetch email log"), nil
	}

	recipientEmail, _ := row["recipient_email"].(string)
	subject, _ := row["subject"].(string)
	renderedBody, _ := row["rendered_body"].(string)
	templateSlug, _ := row["template_slug"].(string)

	if recipientEmail == "" {
		return jsonError(400, "NO_RECIPIENT", "Log has no recipient email"), nil
	}

	// Use the CoreAPI SendEmail to dispatch via the configured provider.
	// SendEmail waits for the provider to complete (synchronous event delivery).
	sendErr := p.host.SendEmail(ctx, coreapi.EmailRequest{
		To:      []string{recipientEmail},
		Subject: subject,
		HTML:    renderedBody,
	})

	status := "sent"
	var errMsg interface{}
	if sendErr != nil {
		status = "failed"
		errMsg = sendErr.Error()
	}

	// Log the resend attempt.
	var ruleID interface{}
	if rid, ok := row["rule_id"]; ok {
		ruleID = rid
	}

	newLog := map[string]any{
		"rule_id":         ruleID,
		"template_slug":   templateSlug,
		"action":          "resend",
		"recipient_email": recipientEmail,
		"subject":         subject,
		"rendered_body":   renderedBody,
		"status":          status,
		"error_message":   errMsg,
	}

	created, createErr := p.host.DataCreate(ctx, "email_logs", newLog)
	if createErr != nil {
		_ = created // ignore
	}

	if sendErr != nil {
		return jsonError(500, "SEND_FAILED", "Failed to resend email: "+sendErr.Error()), nil
	}

	if created != nil {
		return jsonResponse(200, map[string]any{"data": created}), nil
	}
	return jsonResponse(200, map[string]any{"data": newLog}), nil
}

// ---------------------------------------------------------------------------
// Email Layouts
// ---------------------------------------------------------------------------

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

	// Handle language_id: null means universal fallback.
	langIDNull := false
	if lid, ok := updates["language_id"]; ok && lid == nil {
		langIDNull = true
		delete(updates, "language_id")
	}

	updates["updated_at"] = time.Now().Format(time.RFC3339)

	if err := p.host.DataUpdate(ctx, "email_layouts", id, updates); err != nil {
		return jsonError(500, "UPDATE_FAILED", "Failed to update email layout"), nil
	}

	if langIDNull {
		_, err := p.host.DataExec(ctx, "UPDATE email_layouts SET language_id = NULL WHERE id = ?", id)
		if err != nil {
			return jsonError(500, "UPDATE_FAILED", "Failed to clear language on email layout"), nil
		}
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

func (p *EmailManagerPlugin) getSettings(ctx context.Context) (*pb.PluginHTTPResponse, error) {
	settings, err := p.host.GetSettings(ctx, "email_")
	if err != nil {
		return jsonError(500, "FETCH_FAILED", "Failed to fetch email settings"), nil
	}

	result := make(map[string]string)
	for k, v := range settings {
		if maskedKeys[k] && v != "" {
			result[k] = "••••"
		} else {
			result[k] = v
		}
	}

	return jsonResponse(200, map[string]any{"data": result}), nil
}

func (p *EmailManagerPlugin) saveSettings(ctx context.Context, body []byte) (*pb.PluginHTTPResponse, error) {
	var settings map[string]string
	if err := json.Unmarshal(body, &settings); err != nil {
		return jsonError(400, "INVALID_BODY", "Invalid request body"), nil
	}

	for key, value := range settings {
		if !strings.HasPrefix(key, "email_") {
			continue
		}
		// Skip masked values that were not changed.
		if maskedKeys[key] && value == "••••" {
			continue
		}
		if err := p.host.SetSetting(ctx, key, value); err != nil {
			return jsonError(500, "SAVE_FAILED", fmt.Sprintf("Failed to save setting %s", key)), nil
		}
	}

	return jsonResponse(200, map[string]any{"data": map[string]any{"message": "Email settings saved"}}), nil
}

func (p *EmailManagerPlugin) testEmail(ctx context.Context, req *pb.PluginHTTPRequest) (*pb.PluginHTTPResponse, error) {
	// Get the current user email from request headers (set by auth middleware).
	userEmail := ""
	for k, v := range req.GetHeaders() {
		if strings.EqualFold(k, "x-user-email") {
			userEmail = v
			break
		}
	}
	if userEmail == "" {
		return jsonError(400, "NO_EMAIL", "Cannot determine user email for test"), nil
	}

	subject := "VibeCMS Test Email"
	body := fmt.Sprintf(`<html><body>
<h2>VibeCMS Test Email</h2>
<p>This is a test email confirming that your email configuration is working correctly.</p>
<p>Sent at: <strong>%s</strong></p>
</body></html>`, time.Now().Format(time.RFC1123))

	if err := p.host.SendEmail(ctx, coreapi.EmailRequest{
		To:      []string{userEmail},
		Subject: subject,
		HTML:    body,
	}); err != nil {
		return jsonError(500, "SEND_FAILED", "Failed to send test email: "+err.Error()), nil
	}

	return jsonResponse(200, map[string]any{"data": map[string]any{"message": "Test email sent to " + userEmail}}), nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func parseID(subPath string, pathParams map[string]string, key string) uint {
	if idStr, ok := pathParams[key]; ok {
		id, _ := strconv.ParseUint(idStr, 10, 64)
		if id > 0 {
			return uint(id)
		}
	}
	return parseIDStr(strings.SplitN(subPath, "/", 2)[0])
}

func parseIDStr(s string) uint {
	id, _ := strconv.ParseUint(s, 10, 64)
	return uint(id)
}

func toUint(v any) uint {
	switch n := v.(type) {
	case float64:
		return uint(n)
	case int:
		return uint(n)
	case int64:
		return uint(n)
	case json.Number:
		i, _ := n.Int64()
		return uint(i)
	default:
		return 0
	}
}

func parsePagination(params map[string]string) (page, perPage int) {
	page, _ = strconv.Atoi(paramOr(params, "page", "1"))
	perPage, _ = strconv.Atoi(paramOr(params, "per_page", "25"))
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 25
	}
	return
}

func stripFields(rows []map[string]any, fields ...string) []map[string]any {
	out := make([]map[string]any, len(rows))
	for i, row := range rows {
		cp := make(map[string]any, len(row))
		for k, v := range row {
			cp[k] = v
		}
		for _, f := range fields {
			delete(cp, f)
		}
		out[i] = cp
	}
	return out
}

func paramOr(params map[string]string, key, def string) string {
	if v, ok := params[key]; ok && v != "" {
		return v
	}
	return def
}

func isNotFound(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "NotFound")
}

func jsonResponse(status int, data any) *pb.PluginHTTPResponse {
	body, _ := json.Marshal(data)
	return &pb.PluginHTTPResponse{
		StatusCode: int32(status),
		Headers:    map[string]string{"Content-Type": "application/json"},
		Body:       body,
	}
}

func jsonError(status int, code, message string) *pb.PluginHTTPResponse {
	body, _ := json.Marshal(map[string]any{
		"error": map[string]any{
			"code":    code,
			"message": message,
		},
	})
	return &pb.PluginHTTPResponse{
		StatusCode: int32(status),
		Headers:    map[string]string{"Content-Type": "application/json"},
		Body:       body,
	}
}

func jsonValidationError(fields map[string]string) *pb.PluginHTTPResponse {
	body, _ := json.Marshal(map[string]any{
		"error": map[string]any{
			"code":    "VALIDATION_ERROR",
			"message": "One or more fields failed validation",
			"fields":  fields,
		},
	})
	return &pb.PluginHTTPResponse{
		StatusCode: 422,
		Headers:    map[string]string{"Content-Type": "application/json"},
		Body:       body,
	}
}

func main() {
	goplugin.Serve(&goplugin.ServeConfig{
		HandshakeConfig: vibeplugin.Handshake,
		VersionedPlugins: map[int]goplugin.PluginSet{
			2: {"extension": &vibeplugin.ExtensionGRPCPlugin{Impl: &EmailManagerPlugin{}}},
		},
		GRPCServer: goplugin.DefaultGRPCServer,
	})
}
