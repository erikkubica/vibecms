package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"strings"
	"time"

	"vibecms/internal/coreapi"
)

// notificationTemplateData holds the data available to notification templates.
type notificationTemplateData struct {
	FormName    string
	FormSlug    string
	FormID      string
	SubmittedAt string
	Data        []notificationField
	Field       map[string]string
}

type notificationField struct {
	Label string
	Value string
	Key   string
}

func (p *FormsPlugin) triggerNotifications(form map[string]any, data map[string]any) {
	ctx := context.Background()
	notificationsJSON, ok := form["notifications"].(string)
	if !ok {
		// GORM might return it as []byte or map depending on driver/setup
		// If it's already a slice of maps:
		if n, ok := form["notifications"].([]any); ok {
			p.processNotifications(ctx, n, form, data)
			return
		}
		return
	}

	var notifications []any
	if err := json.Unmarshal([]byte(notificationsJSON), &notifications); err != nil {
		return
	}
	p.processNotifications(ctx, notifications, form, data)
}

// processNotifications renders notification subjects and bodies as Go templates.
func (p *FormsPlugin) processNotifications(ctx context.Context, notifications []any, form map[string]any, submissionData map[string]any) {
	formName, _ := form["name"].(string)
	formSlug, _ := form["slug"].(string)
	formID := fmt.Sprintf("%v", form["id"])

	fields := getFormFields(form)

	// Build a label lookup from form fields
	labelMap := make(map[string]string)
	for _, f := range fields {
		id, _ := f["id"].(string)
		label, _ := f["label"].(string)
		if id != "" && label != "" {
			labelMap[id] = label
		}
	}

	// Build notification template data
	tplData := notificationTemplateData{
		FormName:    formName,
		FormSlug:    formSlug,
		FormID:      formID,
		SubmittedAt: time.Now().Format(time.RFC3339),
		Data:        make([]notificationField, 0, len(submissionData)),
		Field:       make(map[string]string),
	}

	for k, v := range submissionData {
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

	for _, n := range notifications {
		config, ok := n.(map[string]any)
		if !ok {
			continue
		}

		enabled, _ := config["enabled"].(bool)
		if !enabled {
			continue
		}

		// Skip notification if route_when condition group evaluates to false.
		if routeWhen := parseJSONMap(config["route_when"]); routeWhen != nil {
			if !EvaluateGroup(routeWhen, submissionData) {
				continue
			}
		}

		to, _ := config["to"].(string)
		if to == "" {
			continue
		}

		subjectTmpl, _ := config["subject"].(string)
		bodyTmpl, _ := config["body"].(string)

		// Render subject as Go template
		renderedSubject, err := renderNotificationTemplate("subject", subjectTmpl, tplData)
		if err != nil {
			p.host.Log(ctx, "error", fmt.Sprintf("Failed to render notification subject: %v", err), nil)
			renderedSubject = subjectTmpl
		}

		// If no body template provided, generate a default HTML table
		renderedBody := ""
		if bodyTmpl != "" {
			renderedBody, err = renderNotificationTemplate("body", bodyTmpl, tplData)
			if err != nil {
				p.host.Log(ctx, "error", fmt.Sprintf("Failed to render notification body: %v", err), nil)
				renderedBody = bodyTmpl
			}
		} else {
			renderedBody = defaultNotificationHTML(tplData)
		}

		_ = p.host.SendEmail(ctx, coreapi.EmailRequest{
			To:      strings.Split(to, ","),
			Subject: renderedSubject,
			HTML:    renderedBody,
		})
	}
}

// renderNotificationTemplate renders a Go html/template string with notification data.
func renderNotificationTemplate(name, text string, data notificationTemplateData) (string, error) {
	if text == "" {
		return "", nil
	}
	tmpl, err := template.New(name).Parse(text)
	if err != nil {
		return "", fmt.Errorf("template parse error: %w", err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("template execute error: %w", err)
	}
	return buf.String(), nil
}

// defaultNotificationHTML generates a simple HTML table for the submission data.
func defaultNotificationHTML(data notificationTemplateData) string {
	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("<h2>New submission for: %s</h2>", data.FormName))
	buf.WriteString("<table border='1' cellpadding='6' cellspacing='0'>")
	buf.WriteString("<tr><th>Field</th><th>Value</th></tr>")
	for _, f := range data.Data {
		buf.WriteString(fmt.Sprintf("<tr><td><b>%s</b></td><td>%s</td></tr>",
			template.HTMLEscapeString(f.Label),
			template.HTMLEscapeString(f.Value),
		))
	}
	buf.WriteString("</table>")
	return buf.String()
}
