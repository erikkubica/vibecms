package main

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"strconv"

	"vibecms/internal/coreapi"
	pb "vibecms/pkg/plugin/proto"
)

// --- CSV Export ---

func (p *FormsPlugin) handleCSVExport(ctx context.Context, req *pb.PluginHTTPRequest) (*pb.PluginHTTPResponse, error) {
	params := req.GetQueryParams()
	formIDStr := params["form_id"]
	if formIDStr == "" {
		return jsonError(400, "MISSING_FORM_ID", "form_id query parameter is required"), nil
	}

	formID, err := strconv.Atoi(formIDStr)
	if err != nil {
		return jsonError(400, "INVALID_FORM_ID", "form_id must be a number"), nil
	}

	// Look up the form to get field definitions
	formRes, err := p.host.DataGet(ctx, formsTable, uint(formID))
	if err != nil {
		return jsonError(404, "FORM_NOT_FOUND", "Form not found"), nil
	}
	form := normalizeForm(formRes)
	fields := getFormFields(form)
	formSlug, _ := form["slug"].(string)

	// Query all submissions for this form
	subRes, err := p.host.DataQuery(ctx, submissionsTable, coreapi.DataStoreQuery{
		Where:   map[string]any{"form_id": formID},
		OrderBy: "created_at DESC",
	})
	if err != nil {
		return jsonError(500, "QUERY_FAILED", err.Error()), nil
	}

	// Build CSV
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)

	formSettings := getFormSettings(form)
	includeIP := true
	if b, ok := formSettings["store_ip"].(bool); ok {
		includeIP = b
	}

	// Headers: field labels + "Submitted At" + "Status" [+ "IP Address"]
	headers := make([]string, 0, len(fields)+3)
	for _, f := range fields {
		label, _ := f["label"].(string)
		if label == "" {
			label, _ = f["id"].(string)
		}
		headers = append(headers, label)
	}
	headers = append(headers, "Submitted At", "Status")
	if includeIP {
		headers = append(headers, "IP Address")
	}

	if err := writer.Write(headers); err != nil {
		return jsonError(500, "CSV_ERROR", err.Error()), nil
	}

	// Rows
	for _, row := range subRes.Rows {
		record := make([]string, 0, len(headers))

		// Parse submission data and metadata from JSON strings
		data := parseJSONMap(row["data"])
		metadata := parseJSONMap(row["metadata"])

		// Field values
		for _, f := range fields {
			id, _ := f["id"].(string)
			val := ""
			if data != nil {
				if v, ok := data[id]; ok {
					val = fieldValueToString(v)
				}
			}
			record = append(record, val)
		}

		// Submitted At
		submittedAt := fmt.Sprintf("%v", row["created_at"])
		record = append(record, submittedAt)

		// Status
		status, _ := row["status"].(string)
		if status == "" {
			status = "unread"
		}
		record = append(record, status)

		// IP Address (only if store_ip=true for this form)
		if includeIP {
			ipVal := ""
			if metadata != nil {
				if v, ok := metadata["ip"].(string); ok {
					ipVal = v
				}
			}
			record = append(record, ipVal)
		}

		if err := writer.Write(record); err != nil {
			return jsonError(500, "CSV_ERROR", err.Error()), nil
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return jsonError(500, "CSV_ERROR", err.Error()), nil
	}

	return &pb.PluginHTTPResponse{
		StatusCode: 200,
		Headers: map[string]string{
			"Content-Type":        "text/csv",
			"Content-Disposition": fmt.Sprintf(`attachment; filename="form-%s-submissions.csv"`, formSlug),
		},
		Body: buf.Bytes(),
	}, nil
}
