package email

import (
	"bytes"
	"encoding/json"
	"fmt"
	htmltemplate "html/template"
	"log"
	"strings"

	"gorm.io/gorm"

	"vibecms/internal/events"
	"vibecms/internal/models"
)

// SendRequest contains everything needed to send an email.
type SendRequest struct {
	To        []string          `json:"to"`
	Subject   string            `json:"subject"`
	HTML      string            `json:"html"`
	FromEmail string            `json:"from_email"`
	FromName  string            `json:"from_name"`
	Settings  map[string]string `json:"settings"` // provider-specific settings
}

// SendFunc is a function that sends an email. Returns nil on success.
// This is the hook point — plugins and Tengo scripts implement this.
type SendFunc func(req SendRequest) error

// Dispatcher ties together rules, templates, and logging to handle
// system events and send the appropriate emails via a pluggable SendFunc.
type Dispatcher struct {
	db       *gorm.DB
	ruleSvc  *RuleService
	logSvc   *LogService
	sendFunc SendFunc
}

// NewDispatcher creates a new Dispatcher.
func NewDispatcher(db *gorm.DB, ruleSvc *RuleService, logSvc *LogService) *Dispatcher {
	return &Dispatcher{
		db:      db,
		ruleSvc: ruleSvc,
		logSvc:  logSvc,
	}
}

// SetSendFunc sets the function used to actually send emails.
// If nil, emails will be logged as "no provider configured".
func (d *Dispatcher) SetSendFunc(fn SendFunc) {
	d.sendFunc = fn
}

// HandleEvent is called by the event bus for every event.
// It finds matching rules, resolves recipients, renders templates, sends emails, and logs results.
func (d *Dispatcher) HandleEvent(action string, payload events.Payload) {
	// 1. Extract node_type from payload if present.
	nodeType := ""
	if nt, ok := payload["node_type"].(string); ok {
		nodeType = nt
	}

	// 2. Find matching enabled rules.
	rules, err := d.ruleSvc.FindByAction(action, nodeType)
	if err != nil {
		log.Printf("[email] error finding rules for action %q: %v", action, err)
		return
	}
	if len(rules) == 0 {
		return
	}

	// 3. Load site settings once for all rules.
	settings := loadSiteSettings(d.db)
	providerName := settings["email_provider"]

	// Load provider extension settings if applicable.
	providerSettings := make(map[string]string)
	providerSettings["provider"] = providerName
	if providerName != "" {
		extPrefix := "ext." + providerName + "."
		for k, v := range settings {
			if strings.HasPrefix(k, extPrefix) {
				providerSettings[strings.TrimPrefix(k, extPrefix)] = v
			}
		}
	}
	// Top-level from_email/from_name override extension defaults if set.
	if v := settings["from_email"]; v != "" {
		providerSettings["from_email"] = v
	}
	if v := settings["from_name"]; v != "" {
		providerSettings["from_name"] = v
	}

	// Build site data exposed to templates as `.site.X`. Friendly keys
	// (`name`, `url`) are the canonical accessors; the redundant `site_*`
	// aliases keep older templates rendering after the rename.
	siteData := map[string]string{
		"name":      settings["site_name"],
		"url":       settings["site_url"],
		"site_name": settings["site_name"],
		"site_url":  settings["site_url"],
	}

	for _, rule := range rules {
		d.processRule(action, payload, rule, siteData, providerSettings)
	}
}

// resolveLayoutForLang finds the best email layout for a recipient's language.
// Fallback: language-specific → universal (NULL language_id).
func (d *Dispatcher) resolveLayoutForLang(langID *int) string {
	// 1. Try language-specific layout.
	if langID != nil {
		var layout models.EmailLayout
		if err := d.db.Where("language_id = ?", *langID).First(&layout).Error; err == nil {
			return layout.BodyTemplate
		}
	}
	// 2. Fall back to universal layout (language_id IS NULL).
	var layout models.EmailLayout
	if err := d.db.Where("language_id IS NULL").First(&layout).Error; err == nil {
		return layout.BodyTemplate
	}
	return ""
}

// processRule handles a single rule: resolve recipients, render, send, and log.
func (d *Dispatcher) processRule(
	action string,
	payload events.Payload,
	rule models.EmailRule,
	siteData map[string]string,
	providerSettings map[string]string,
) {
	// Resolve recipients with their language info.
	recipientInfos := d.resolveRecipientsWithLang(action, rule, payload)
	if len(recipientInfos) == 0 {
		return
	}

	// Base template from the rule (universal fallback).
	baseTmpl := rule.Template

	// Build template data: payload (flat keys) + nested objects so authors
	// can write either `{{.user_full_name}}` (legacy flat) or
	// `{{.user.full_name}}` (preferred, mirrors Twig/Jinja conventions).
	data := make(map[string]interface{})
	for k, v := range payload {
		data[k] = v
	}
	data["site"] = siteData

	// `.user` = subject of the event (the user being created/reset/etc).
	if u := d.lookupUserMap(payload["user_id"], payload["user_email"]); u != nil {
		data["user"] = u
	}
	// `.actor` = who performed the action (often == user, but distinguished
	// for admin-on-behalf actions like "admin reset Bob's password").
	if a := d.lookupUserMap(nil, payload["actor_email"]); a != nil {
		data["actor"] = a
	}

	// Group recipients by language for efficient template resolution.
	for _, ri := range recipientInfos {
		// `.recipient` = the address this email is going to. Useful for
		// role-based rules where the recipient differs from `.user`/`.actor`.
		if r := d.lookupUserMap(nil, ri.email); r != nil {
			data["recipient"] = r
		} else {
			data["recipient"] = map[string]any{"email": ri.email}
		}

		// Resolve best template for this recipient's language.
		tmpl := d.resolveTemplateForLang(baseTmpl.Slug, ri.languageID)
		if tmpl == nil {
			tmpl = &baseTmpl
		}

		subject, err := renderTemplate("subject", tmpl.SubjectTemplate, data)
		if err != nil {
			log.Printf("[email] error rendering subject for rule %d: %v", rule.ID, err)
			continue
		}

		body, err := renderTemplate("body", tmpl.BodyTemplate, data)
		if err != nil {
			log.Printf("[email] error rendering body for rule %d: %v", rule.ID, err)
			continue
		}

		// Resolve base layout for this recipient's language.
		baseLayout := d.resolveLayoutForLang(ri.languageID)
		if baseLayout != "" {
			data["email_body"] = htmltemplate.HTML(body)
			wrapped, wrapErr := renderTemplate("base_layout", baseLayout, data)
			if wrapErr != nil {
				log.Printf("[email] error rendering base layout for rule %d: %v", rule.ID, wrapErr)
			} else {
				body = wrapped
			}
		}

		d.sendAndLog(action, rule, tmpl.Slug, ri.email, subject, body, providerSettings)
	}
}

// recipientInfo holds a recipient email and their preferred language.
type recipientInfo struct {
	email      string
	languageID *int
}

// resolveTemplateForLang finds the best email template for a slug + language.
// Fallback: language-specific → site default language → universal (NULL).
func (d *Dispatcher) resolveTemplateForLang(slug string, langID *int) *models.EmailTemplate {
	// 1. Try language-specific template.
	if langID != nil {
		var tmpl models.EmailTemplate
		if err := d.db.Where("slug = ? AND language_id = ?", slug, *langID).First(&tmpl).Error; err == nil {
			return &tmpl
		}
	}

	// 2. Try site default language.
	var defaultLang models.Language
	if err := d.db.Where("is_default = ?", true).First(&defaultLang).Error; err == nil {
		if langID == nil || *langID != defaultLang.ID {
			var tmpl models.EmailTemplate
			if err := d.db.Where("slug = ? AND language_id = ?", slug, defaultLang.ID).First(&tmpl).Error; err == nil {
				return &tmpl
			}
		}
	}

	// 3. Universal fallback (language_id IS NULL).
	var tmpl models.EmailTemplate
	if err := d.db.Where("slug = ? AND language_id IS NULL", slug).First(&tmpl).Error; err == nil {
		return &tmpl
	}

	return nil
}

// sensitiveActions are events whose rendered body must NOT be persisted —
// e.g. password reset emails contain single-use tokens that an attacker
// with DB read access could otherwise replay. The log row still gets
// recipient/subject/status for auditing; only the body is redacted.
var sensitiveActions = map[string]bool{
	"user.password_reset_requested": true,
	"user.password_reset_completed": true,
}

// sendAndLog sends an email via the configured SendFunc and logs the result.
func (d *Dispatcher) sendAndLog(
	action string,
	rule models.EmailRule,
	templateSlug string,
	recipient string,
	subject string,
	body string,
	providerSettings map[string]string,
) {
	providerName := providerSettings["provider"]
	storedBody := body
	if sensitiveActions[action] {
		storedBody = "[redacted — contains time-sensitive token]"
	}
	logEntry := &models.EmailLog{
		RuleID:         &rule.ID,
		TemplateSlug:   templateSlug,
		Action:         action,
		RecipientEmail: recipient,
		Subject:        subject,
		RenderedBody:   storedBody,
		Provider:       &providerName,
	}

	if d.sendFunc == nil {
		errMsg := "no email provider configured"
		logEntry.Status = "failed"
		logEntry.ErrorMessage = &errMsg
		if err := d.logSvc.Create(logEntry); err != nil {
			log.Printf("[email] error logging failed send: %v", err)
		}
		return
	}

	req := SendRequest{
		To:        []string{recipient},
		Subject:   subject,
		HTML:      body,
		FromEmail: providerSettings["from_email"],
		FromName:  providerSettings["from_name"],
		Settings:  providerSettings,
	}

	if err := d.sendFunc(req); err != nil {
		errMsg := err.Error()
		logEntry.Status = "failed"
		logEntry.ErrorMessage = &errMsg
	} else {
		logEntry.Status = "sent"
	}

	if err := d.logSvc.Create(logEntry); err != nil {
		log.Printf("[email] error logging send attempt: %v", err)
	}
}

// resolveRecipientsWithLang determines recipients with their preferred language.
func (d *Dispatcher) resolveRecipientsWithLang(action string, rule models.EmailRule, payload events.Payload) []recipientInfo {
	switch rule.RecipientType {
	case "actor":
		if email, ok := payload["actor_email"].(string); ok && email != "" {
			langID := d.lookupUserLangByEmail(email)
			return []recipientInfo{{email: email, languageID: langID}}
		}
	case "node_author":
		if email, ok := payload["author_email"].(string); ok && email != "" {
			langID := d.lookupUserLangByEmail(email)
			return []recipientInfo{{email: email, languageID: langID}}
		}
	case "fixed":
		// Fixed emails have no user — use actor's language as best guess, then nil.
		var actorLang *int
		if email, ok := payload["actor_email"].(string); ok && email != "" {
			actorLang = d.lookupUserLangByEmail(email)
		}
		emails := splitEmails(rule.RecipientValue)
		infos := make([]recipientInfo, 0, len(emails))
		for _, e := range emails {
			infos = append(infos, recipientInfo{email: e, languageID: actorLang})
		}
		return infos
	case "role":
		return d.resolveRoleRecipientsWithLang(rule.RecipientValue, action)
	}
	return nil
}

// lookupUserLangByEmail finds a user's preferred language_id by email.
func (d *Dispatcher) lookupUserLangByEmail(email string) *int {
	var user models.User
	if err := d.db.Select("language_id").Where("email = ?", email).First(&user).Error; err == nil {
		return user.LanguageID
	}
	return nil
}

// lookupUserMap loads a user by id (preferred) or email and returns a map
// suitable for use as `.user` / `.actor` / `.recipient` in templates.
// Both `full_name` and `name` are exposed because templates wrote both
// before `name` was settled on as the canonical key.
func (d *Dispatcher) lookupUserMap(idVal any, emailVal any) map[string]any {
	var user models.User
	q := d.db.Select("id", "email", "full_name", "language_id", "role_id")

	switch v := idVal.(type) {
	case nil:
		// fall through to email lookup
	case int:
		if v > 0 {
			if err := q.Where("id = ?", v).First(&user).Error; err == nil {
				return userToMap(&user)
			}
		}
	case int64:
		if v > 0 {
			if err := q.Where("id = ?", v).First(&user).Error; err == nil {
				return userToMap(&user)
			}
		}
	case uint:
		if v > 0 {
			if err := q.Where("id = ?", v).First(&user).Error; err == nil {
				return userToMap(&user)
			}
		}
	case float64:
		if v > 0 {
			if err := q.Where("id = ?", int(v)).First(&user).Error; err == nil {
				return userToMap(&user)
			}
		}
	}

	if email, ok := emailVal.(string); ok && email != "" {
		if err := q.Where("email = ?", email).First(&user).Error; err == nil {
			return userToMap(&user)
		}
	}
	return nil
}

func userToMap(u *models.User) map[string]any {
	fullName := ""
	if u.FullName != nil {
		fullName = *u.FullName
	}
	return map[string]any{
		"id":          u.ID,
		"email":       u.Email,
		"full_name":   fullName,
		"name":        fullName,
		"language_id": u.LanguageID,
		"role_id":     u.RoleID,
	}
}

// resolveRoleRecipientsWithLang finds users with the given role slug whose role capabilities
// include the current action in the email_subscriptions array.
func (d *Dispatcher) resolveRoleRecipientsWithLang(roleSlug string, action string) []recipientInfo {
	var role models.Role
	if err := d.db.Where("slug = ?", roleSlug).First(&role).Error; err != nil {
		log.Printf("[email] role %q not found: %v", roleSlug, err)
		return nil
	}

	var caps map[string]interface{}
	if err := json.Unmarshal([]byte(role.Capabilities), &caps); err != nil {
		log.Printf("[email] error parsing capabilities for role %q: %v", roleSlug, err)
		return nil
	}

	subs, ok := caps["email_subscriptions"]
	if !ok {
		return nil
	}
	subsSlice, ok := subs.([]interface{})
	if !ok {
		return nil
	}

	found := false
	for _, s := range subsSlice {
		if str, ok := s.(string); ok && str == action {
			found = true
			break
		}
	}
	if !found {
		return nil
	}

	var users []models.User
	if err := d.db.Where("role_id = ?", role.ID).Find(&users).Error; err != nil {
		log.Printf("[email] error finding users for role %q: %v", roleSlug, err)
		return nil
	}

	infos := make([]recipientInfo, 0, len(users))
	for _, u := range users {
		if u.Email != "" {
			infos = append(infos, recipientInfo{email: u.Email, languageID: u.LanguageID})
		}
	}
	return infos
}

// splitEmails splits a comma-separated string of emails and trims whitespace.
func splitEmails(value string) []string {
	parts := strings.Split(value, ",")
	emails := make([]string, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			emails = append(emails, trimmed)
		}
	}
	return emails
}

// renderTemplate parses and executes a Go html/template string with the given data.
func renderTemplate(name string, tmplStr string, data interface{}) (string, error) {
	t, err := htmltemplate.New(name).Parse(tmplStr)
	if err != nil {
		return "", fmt.Errorf("template parse error: %w", err)
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("template execute error: %w", err)
	}
	return buf.String(), nil
}
