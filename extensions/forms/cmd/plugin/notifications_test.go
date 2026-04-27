package main

import (
	"strings"
	"testing"
)

func makeFormWithNotification(to, subject, body string, enabled bool) map[string]any {
	notif := map[string]any{
		"enabled": enabled,
		"to":      to,
		"subject": subject,
		"body":    body,
	}
	return map[string]any{
		"id":            float64(1),
		"name":          "Contact Form",
		"slug":          "contact-form",
		"fields":        []any{},
		"notifications": []any{notif},
	}
}

func TestProcessNotifications_BasicEmail(t *testing.T) {
	h := NewFakeHost()
	p := newPlugin(h)

	form := makeFormWithNotification("admin@example.com", "New submission", "<p>Hello</p>", true)
	p.triggerNotifications(form, map[string]any{"name": "Alice"})

	if len(h.Sent) != 1 {
		t.Fatalf("expected 1 email, got %d", len(h.Sent))
	}
	if h.Sent[0].To[0] != "admin@example.com" {
		t.Errorf("to: got %v", h.Sent[0].To)
	}
	if h.Sent[0].Subject != "New submission" {
		t.Errorf("subject: got %q", h.Sent[0].Subject)
	}
}

func TestProcessNotifications_Disabled(t *testing.T) {
	h := NewFakeHost()
	p := newPlugin(h)

	form := makeFormWithNotification("admin@example.com", "Sub", "Body", false)
	p.triggerNotifications(form, map[string]any{})

	if len(h.Sent) != 0 {
		t.Errorf("disabled notification should not send email, got %d", len(h.Sent))
	}
}

func TestProcessNotifications_EmptyTo(t *testing.T) {
	h := NewFakeHost()
	p := newPlugin(h)

	form := makeFormWithNotification("", "Sub", "Body", true)
	p.triggerNotifications(form, map[string]any{})

	if len(h.Sent) != 0 {
		t.Errorf("empty to should not send email, got %d", len(h.Sent))
	}
}

func TestProcessNotifications_TemplateRendering(t *testing.T) {
	h := NewFakeHost()
	p := newPlugin(h)

	form := map[string]any{
		"id":   float64(1),
		"name": "Contact",
		"slug": "contact",
		"fields": []any{
			map[string]any{"id": "name", "label": "Name", "type": "text"},
		},
		"notifications": []any{
			map[string]any{
				"enabled": true,
				"to":      "admin@example.com",
				"subject": "New submission from {{.Field.name}}",
				"body":    "<p>Name: {{.Field.name}}</p>",
			},
		},
	}

	p.triggerNotifications(form, map[string]any{"name": "Bob"})

	if len(h.Sent) != 1 {
		t.Fatalf("expected 1 email, got %d", len(h.Sent))
	}
	if !strings.Contains(h.Sent[0].Subject, "Bob") {
		t.Errorf("subject should contain 'Bob', got %q", h.Sent[0].Subject)
	}
	if !strings.Contains(h.Sent[0].HTML, "Bob") {
		t.Errorf("body should contain 'Bob', got %q", h.Sent[0].HTML)
	}
}

func TestProcessNotifications_DefaultBodyGenerated(t *testing.T) {
	h := NewFakeHost()
	p := newPlugin(h)

	form := map[string]any{
		"id":   float64(1),
		"name": "Contact",
		"slug": "contact",
		"fields": []any{
			map[string]any{"id": "name", "label": "Name", "type": "text"},
		},
		"notifications": []any{
			map[string]any{
				"enabled": true,
				"to":      "admin@example.com",
				"subject": "Submission",
				// no body → default HTML table
			},
		},
	}

	p.triggerNotifications(form, map[string]any{"name": "Alice"})

	if len(h.Sent) != 1 {
		t.Fatalf("expected 1 email, got %d", len(h.Sent))
	}
	if !strings.Contains(h.Sent[0].HTML, "<table") {
		t.Errorf("default body should be an HTML table, got %q", h.Sent[0].HTML)
	}
}

func TestProcessNotifications_RouteWhenCondition(t *testing.T) {
	h := NewFakeHost()
	p := newPlugin(h)

	form := map[string]any{
		"id":     float64(1),
		"name":   "Contact",
		"slug":   "contact",
		"fields": []any{},
		"notifications": []any{
			map[string]any{
				"enabled": true,
				"to":      "sales@example.com",
				"subject": "Sales lead",
				"body":    "...",
				"route_when": map[string]any{
					"all": []any{
						map[string]any{"field": "department", "operator": "equals", "value": "Sales"},
					},
				},
			},
		},
	}

	// Should NOT send when department != Sales
	p.triggerNotifications(form, map[string]any{"department": "Support"})
	if len(h.Sent) != 0 {
		t.Errorf("route_when false should skip notification, got %d emails", len(h.Sent))
	}

	// Should send when department == Sales
	p.triggerNotifications(form, map[string]any{"department": "Sales"})
	if len(h.Sent) != 1 {
		t.Errorf("route_when true should send notification, got %d emails", len(h.Sent))
	}
}

func TestProcessNotifications_MultipleRecipients(t *testing.T) {
	h := NewFakeHost()
	p := newPlugin(h)

	form := makeFormWithNotification("a@example.com,b@example.com", "Sub", "Body", true)
	p.triggerNotifications(form, map[string]any{})

	if len(h.Sent) != 1 {
		t.Fatalf("expected 1 email send call, got %d", len(h.Sent))
	}
	if len(h.Sent[0].To) != 2 {
		t.Errorf("expected 2 recipients, got %d", len(h.Sent[0].To))
	}
}

func TestProcessNotifications_JSONStringNotifications(t *testing.T) {
	h := NewFakeHost()
	p := newPlugin(h)

	form := map[string]any{
		"id":     float64(1),
		"name":   "Contact",
		"slug":   "contact",
		"fields": []any{},
		// notifications as JSON string (as it comes from JSONB)
		"notifications": `[{"enabled":true,"to":"admin@example.com","subject":"Sub","body":"Body"}]`,
	}

	p.triggerNotifications(form, map[string]any{})
	if len(h.Sent) != 1 {
		t.Errorf("expected 1 email from JSON-string notifications, got %d", len(h.Sent))
	}
}

// ---- renderNotificationTemplate ----

func TestRenderNotificationTemplate_Empty(t *testing.T) {
	result, err := renderNotificationTemplate("test", "", notificationTemplateData{})
	if err != nil {
		t.Errorf("empty template should not error: %v", err)
	}
	if result != "" {
		t.Errorf("empty template should return empty string, got %q", result)
	}
}

func TestRenderNotificationTemplate_WithData(t *testing.T) {
	tplData := notificationTemplateData{
		FormName: "My Form",
		Field:    map[string]string{"email": "test@example.com"},
	}
	result, err := renderNotificationTemplate("test", "Form: {{.FormName}}, Email: {{.Field.email}}", tplData)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "My Form") {
		t.Errorf("result should contain form name, got %q", result)
	}
	if !strings.Contains(result, "test@example.com") {
		t.Errorf("result should contain email, got %q", result)
	}
}

func TestRenderNotificationTemplate_InvalidTemplate(t *testing.T) {
	_, err := renderNotificationTemplate("test", "{{.Unclosed", notificationTemplateData{})
	if err == nil {
		t.Error("invalid template should return error")
	}
}

// ---- defaultNotificationHTML ----

func TestDefaultNotificationHTML(t *testing.T) {
	tplData := notificationTemplateData{
		FormName: "Contact Us",
		Data: []notificationField{
			{Label: "Name", Value: "Alice", Key: "name"},
			{Label: "Email", Value: "alice@example.com", Key: "email"},
		},
	}
	html := defaultNotificationHTML(tplData)
	if !strings.Contains(html, "Contact Us") {
		t.Error("HTML should contain form name")
	}
	if !strings.Contains(html, "Alice") {
		t.Error("HTML should contain field value")
	}
	if !strings.Contains(html, "<table") {
		t.Error("HTML should be a table")
	}
}
