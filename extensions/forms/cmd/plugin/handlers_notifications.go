package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"vibecms/internal/coreapi"
	pb "vibecms/pkg/plugin/proto"
)

// testEmailBody represents the request body for the Send Test Email endpoint.
type testEmailBody struct {
	SampleData        map[string]any `json:"sample_data"`
	OverrideRecipient string         `json:"override_recipient"`
}

// handleNotificationTest handles POST /{form_id}/notifications/{idx}/test
// It sends a test email for the notification at index idx in the specified form.
func (p *FormsPlugin) handleNotificationTest(ctx context.Context, formID uint, idx int, req *pb.PluginHTTPRequest) (*pb.PluginHTTPResponse, error) {
	// 1. Load form
	row, err := p.host.DataGet(ctx, formsTable, formID)
	if err != nil {
		return jsonError(404, "FORM_NOT_FOUND", "Form not found"), nil
	}
	form := normalizeForm(row)

	// 2. Get notifications list
	notifications, ok := form["notifications"].([]any)
	if !ok || idx < 0 || idx >= len(notifications) {
		return jsonError(400, "INVALID_INDEX", "Notification index out of range"), nil
	}
	config, ok := notifications[idx].(map[string]any)
	if !ok {
		return jsonError(400, "INVALID_NOTIFICATION", "Could not read notification config"), nil
	}

	// 3. Parse request body
	var body testEmailBody
	if raw := req.GetBody(); len(raw) > 0 {
		if err := json.Unmarshal(raw, &body); err != nil {
			return jsonError(400, "INVALID_BODY", "Invalid request body"), nil
		}
	}

	// 4. Resolve recipient: override > notification recipients > current user email
	recipient := strings.TrimSpace(body.OverrideRecipient)
	if recipient == "" {
		// Try notification recipients (skip template variables)
		notifRecipients, _ := config["recipients"].(string)
		if notifRecipients != "" && !strings.Contains(notifRecipients, "{{") {
			recipient = strings.Split(notifRecipients, ",")[0]
			recipient = strings.TrimSpace(recipient)
		}
	}
	if recipient == "" {
		// Fall back to current admin user's email
		userID := req.GetUserId()
		if userID > 0 {
			user, err := p.host.GetUser(ctx, uint(userID))
			if err == nil && user != nil {
				recipient = user.Email
			}
		}
	}
	if recipient == "" {
		return jsonError(400, "NO_RECIPIENT", "Could not determine a recipient — provide override_recipient"), nil
	}

	// Validate recipient email address
	if _, err := mail.ParseAddress(recipient); err != nil {
		return jsonError(400, "INVALID_EMAIL", fmt.Sprintf("Invalid recipient email: %s", recipient)), nil
	}

	// 5. Build sample template data
	fields := getFormFields(form)
	formName, _ := form["name"].(string)
	formSlug, _ := form["slug"].(string)
	formIDStr := fmt.Sprintf("%v", form["id"])

	labelMap := make(map[string]string)
	for _, f := range fields {
		fid, _ := f["id"].(string)
		flabel, _ := f["label"].(string)
		if fid != "" {
			labelMap[fid] = flabel
		}
	}

	// Use provided sample data or auto-generate from field defaults
	sampleData := body.SampleData
	if len(sampleData) == 0 {
		sampleData = make(map[string]any)
		for _, f := range fields {
			fid, _ := f["id"].(string)
			ftype, _ := f["type"].(string)
			if fid == "" {
				continue
			}
			switch ftype {
			case "email":
				sampleData[fid] = "test@example.com"
			case "tel":
				sampleData[fid] = "+1 555 000 0000"
			case "number", "range":
				sampleData[fid] = "42"
			case "date":
				sampleData[fid] = "2025-01-01"
			case "checkbox", "gdpr_consent":
				sampleData[fid] = "true"
			case "textarea":
				sampleData[fid] = "Sample text content for testing."
			default:
				if def, _ := f["default_value"].(string); def != "" {
					sampleData[fid] = def
				} else {
					flabel, _ := f["label"].(string)
					sampleData[fid] = fmt.Sprintf("[%s sample value]", flabel)
				}
			}
		}
	}

	tplData := notificationTemplateData{
		FormName:    formName,
		FormSlug:    formSlug,
		FormID:      formIDStr,
		SubmittedAt: time.Now().Format(time.RFC3339),
		Data:        make([]notificationField, 0, len(sampleData)),
		Field:       make(map[string]string),
	}
	for k, v := range sampleData {
		valStr := fmt.Sprintf("%v", v)
		label := labelMap[k]
		if label == "" {
			label = k
		}
		tplData.Data = append(tplData.Data, notificationField{
			Label: label,
			Value: valStr,
			Key:   k,
		})
		tplData.Field[k] = valStr
	}

	// 6. Render subject and body
	subjectTmpl, _ := config["subject"].(string)
	bodyTmpl, _ := config["body"].(string)

	renderedSubject, err := renderNotificationTemplate("subject", subjectTmpl, tplData)
	if err != nil {
		renderedSubject = subjectTmpl
	}
	if renderedSubject == "" {
		renderedSubject = fmt.Sprintf("[Test] New submission: %s", formName)
	}

	var renderedBody string
	if bodyTmpl != "" {
		renderedBody, err = renderNotificationTemplate("body", bodyTmpl, tplData)
		if err != nil {
			renderedBody = bodyTmpl
		}
	} else {
		renderedBody = defaultNotificationHTML(tplData)
	}

	// 7. Send email
	if err := p.host.SendEmail(ctx, coreapi.EmailRequest{
		To:      []string{recipient},
		Subject: fmt.Sprintf("[Test] %s", renderedSubject),
		HTML:    renderedBody,
	}); err != nil {
		return jsonError(500, "SEND_FAILED", fmt.Sprintf("Failed to send test email: %v", err)), nil
	}

	return jsonResponse(200, map[string]string{
		"status":  "sent",
		"message": fmt.Sprintf("Test email sent to %s", recipient),
	}), nil
}
