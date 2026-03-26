package email

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"vibecms/internal/api"
	"vibecms/internal/auth"
	"vibecms/internal/models"
)

// EmailHandler provides admin API endpoints for email templates, rules, logs, and settings.
type EmailHandler struct {
	db *gorm.DB
}

// NewEmailHandler creates a new EmailHandler with the given database connection.
func NewEmailHandler(db *gorm.DB) *EmailHandler {
	return &EmailHandler{db: db}
}

// RegisterRoutes registers all email-related routes on the provided router group.
func (h *EmailHandler) RegisterRoutes(router fiber.Router) {
	// Email templates — require manage_email.
	tpl := router.Group("/email-templates", auth.CapabilityRequired("manage_email"))
	tpl.Get("/", h.ListTemplates)
	tpl.Get("/:id", h.GetTemplate)
	tpl.Post("/", h.CreateTemplate)
	tpl.Patch("/:id", h.UpdateTemplate)
	tpl.Delete("/:id", h.DeleteTemplate)

	// Email rules — require manage_email.
	rules := router.Group("/email-rules", auth.CapabilityRequired("manage_email"))
	rules.Get("/", h.ListRules)
	rules.Get("/:id", h.GetRule)
	rules.Post("/", h.CreateRule)
	rules.Patch("/:id", h.UpdateRule)
	rules.Delete("/:id", h.DeleteRule)

	// Email logs — require manage_email.
	logs := router.Group("/email-logs", auth.CapabilityRequired("manage_email"))
	logs.Get("/", h.ListLogs)
	logs.Get("/:id", h.GetLog)
	logs.Post("/:id/resend", h.ResendLog)

	// Email settings — require manage_settings.
	settings := router.Group("/email-settings", auth.CapabilityRequired("manage_settings"))
	settings.Get("/", h.GetSettings)
	settings.Post("/", h.SaveSettings)
	settings.Post("/test", h.TestEmail)
}

// ---------------------------------------------------------------------------
// Email Templates
// ---------------------------------------------------------------------------

// ListTemplates handles GET /email-templates.
func (h *EmailHandler) ListTemplates(c *fiber.Ctx) error {
	var templates []models.EmailTemplate
	if err := h.db.Order("id ASC").Find(&templates).Error; err != nil {
		return api.Error(c, fiber.StatusInternalServerError, "LIST_FAILED", "Failed to list email templates")
	}
	return api.Success(c, templates)
}

// GetTemplate handles GET /email-templates/:id.
func (h *EmailHandler) GetTemplate(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return api.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Template ID must be a valid integer")
	}

	var tpl models.EmailTemplate
	if err := h.db.First(&tpl, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return api.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Email template not found")
		}
		return api.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", "Failed to fetch email template")
	}

	return api.Success(c, tpl)
}

// createTemplateRequest represents the JSON body for creating an email template.
type createTemplateRequest struct {
	Slug            string          `json:"slug"`
	Name            string          `json:"name"`
	SubjectTemplate string          `json:"subject_template"`
	BodyTemplate    string          `json:"body_template"`
	TestData        models.JSONB    `json:"test_data"`
}

// CreateTemplate handles POST /email-templates.
func (h *EmailHandler) CreateTemplate(c *fiber.Ctx) error {
	var req createTemplateRequest
	if err := c.BodyParser(&req); err != nil {
		return api.Error(c, fiber.StatusBadRequest, "INVALID_BODY", "Invalid request body")
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
		return api.ValidationError(c, fields)
	}

	testData := models.JSONB("{}")
	if len(req.TestData) > 0 {
		testData = req.TestData
	}

	tpl := models.EmailTemplate{
		Slug:            req.Slug,
		Name:            req.Name,
		SubjectTemplate: req.SubjectTemplate,
		BodyTemplate:    req.BodyTemplate,
		TestData:        testData,
	}

	if err := h.db.Create(&tpl).Error; err != nil {
		return api.Error(c, fiber.StatusInternalServerError, "CREATE_FAILED", "Failed to create email template")
	}

	return api.Created(c, tpl)
}

// UpdateTemplate handles PATCH /email-templates/:id.
func (h *EmailHandler) UpdateTemplate(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return api.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Template ID must be a valid integer")
	}

	var tpl models.EmailTemplate
	if err := h.db.First(&tpl, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return api.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Email template not found")
		}
		return api.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", "Failed to fetch email template")
	}

	var body map[string]interface{}
	if err := c.BodyParser(&body); err != nil {
		return api.Error(c, fiber.StatusBadRequest, "INVALID_BODY", "Invalid request body")
	}

	delete(body, "id")
	delete(body, "created_at")
	delete(body, "updated_at")

	if len(body) == 0 {
		return api.Error(c, fiber.StatusBadRequest, "NO_UPDATES", "No valid fields to update")
	}

	if err := h.db.Model(&tpl).Updates(body).Error; err != nil {
		return api.Error(c, fiber.StatusInternalServerError, "UPDATE_FAILED", "Failed to update email template")
	}

	h.db.First(&tpl, id)
	return api.Success(c, tpl)
}

// DeleteTemplate handles DELETE /email-templates/:id.
func (h *EmailHandler) DeleteTemplate(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return api.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Template ID must be a valid integer")
	}

	var tpl models.EmailTemplate
	if err := h.db.First(&tpl, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return api.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Email template not found")
		}
		return api.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", "Failed to fetch email template")
	}

	if err := h.db.Delete(&tpl).Error; err != nil {
		return api.Error(c, fiber.StatusInternalServerError, "DELETE_FAILED", "Failed to delete email template")
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// ---------------------------------------------------------------------------
// Email Rules
// ---------------------------------------------------------------------------

// ListRules handles GET /email-rules.
func (h *EmailHandler) ListRules(c *fiber.Ctx) error {
	var rules []models.EmailRule
	if err := h.db.Preload("Template").Order("id ASC").Find(&rules).Error; err != nil {
		return api.Error(c, fiber.StatusInternalServerError, "LIST_FAILED", "Failed to list email rules")
	}
	return api.Success(c, rules)
}

// GetRule handles GET /email-rules/:id.
func (h *EmailHandler) GetRule(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return api.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Rule ID must be a valid integer")
	}

	var rule models.EmailRule
	if err := h.db.Preload("Template").First(&rule, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return api.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Email rule not found")
		}
		return api.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", "Failed to fetch email rule")
	}

	return api.Success(c, rule)
}

// createRuleRequest represents the JSON body for creating an email rule.
type createRuleRequest struct {
	Action         string  `json:"action"`
	NodeType       *string `json:"node_type"`
	TemplateID     int     `json:"template_id"`
	RecipientType  string  `json:"recipient_type"`
	RecipientValue string  `json:"recipient_value"`
	Enabled        *bool   `json:"enabled"`
}

// CreateRule handles POST /email-rules.
func (h *EmailHandler) CreateRule(c *fiber.Ctx) error {
	var req createRuleRequest
	if err := c.BodyParser(&req); err != nil {
		return api.Error(c, fiber.StatusBadRequest, "INVALID_BODY", "Invalid request body")
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
	if req.RecipientValue == "" {
		fields["recipient_value"] = "Recipient value is required"
	}
	if len(fields) > 0 {
		return api.ValidationError(c, fields)
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	rule := models.EmailRule{
		Action:         req.Action,
		NodeType:       req.NodeType,
		TemplateID:     req.TemplateID,
		RecipientType:  req.RecipientType,
		RecipientValue: req.RecipientValue,
		Enabled:        enabled,
	}

	if err := h.db.Create(&rule).Error; err != nil {
		return api.Error(c, fiber.StatusInternalServerError, "CREATE_FAILED", "Failed to create email rule")
	}

	// Reload with template.
	h.db.Preload("Template").First(&rule, rule.ID)
	return api.Created(c, rule)
}

// UpdateRule handles PATCH /email-rules/:id.
func (h *EmailHandler) UpdateRule(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return api.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Rule ID must be a valid integer")
	}

	var rule models.EmailRule
	if err := h.db.First(&rule, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return api.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Email rule not found")
		}
		return api.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", "Failed to fetch email rule")
	}

	var body map[string]interface{}
	if err := c.BodyParser(&body); err != nil {
		return api.Error(c, fiber.StatusBadRequest, "INVALID_BODY", "Invalid request body")
	}

	delete(body, "id")
	delete(body, "created_at")
	delete(body, "updated_at")

	if len(body) == 0 {
		return api.Error(c, fiber.StatusBadRequest, "NO_UPDATES", "No valid fields to update")
	}

	if err := h.db.Model(&rule).Updates(body).Error; err != nil {
		return api.Error(c, fiber.StatusInternalServerError, "UPDATE_FAILED", "Failed to update email rule")
	}

	h.db.Preload("Template").First(&rule, id)
	return api.Success(c, rule)
}

// DeleteRule handles DELETE /email-rules/:id.
func (h *EmailHandler) DeleteRule(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return api.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Rule ID must be a valid integer")
	}

	var rule models.EmailRule
	if err := h.db.First(&rule, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return api.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Email rule not found")
		}
		return api.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", "Failed to fetch email rule")
	}

	if err := h.db.Delete(&rule).Error; err != nil {
		return api.Error(c, fiber.StatusInternalServerError, "DELETE_FAILED", "Failed to delete email rule")
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// ---------------------------------------------------------------------------
// Email Logs
// ---------------------------------------------------------------------------

// ListLogs handles GET /email-logs with query param filters and pagination.
func (h *EmailHandler) ListLogs(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	perPage, _ := strconv.Atoi(c.Query("per_page", "25"))
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 25
	}

	q := h.db.Model(&models.EmailLog{})

	if status := c.Query("status"); status != "" {
		q = q.Where("status = ?", status)
	}
	if action := c.Query("action"); action != "" {
		q = q.Where("action = ?", action)
	}
	if recipient := c.Query("recipient"); recipient != "" {
		q = q.Where("recipient_email ILIKE ?", "%"+recipient+"%")
	}
	if dateFrom := c.Query("date_from"); dateFrom != "" {
		if t, err := time.Parse("2006-01-02", dateFrom); err == nil {
			q = q.Where("created_at >= ?", t)
		}
	}
	if dateTo := c.Query("date_to"); dateTo != "" {
		if t, err := time.Parse("2006-01-02", dateTo); err == nil {
			// Include the entire day.
			q = q.Where("created_at < ?", t.AddDate(0, 0, 1))
		}
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return api.Error(c, fiber.StatusInternalServerError, "LIST_FAILED", "Failed to count email logs")
	}

	var logs []models.EmailLog
	offset := (page - 1) * perPage
	if err := q.Order("created_at DESC").Offset(offset).Limit(perPage).Find(&logs).Error; err != nil {
		return api.Error(c, fiber.StatusInternalServerError, "LIST_FAILED", "Failed to list email logs")
	}

	return api.Paginated(c, logs, total, page, perPage)
}

// GetLog handles GET /email-logs/:id.
func (h *EmailHandler) GetLog(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return api.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Log ID must be a valid integer")
	}

	var log models.EmailLog
	if err := h.db.First(&log, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return api.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Email log not found")
		}
		return api.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", "Failed to fetch email log")
	}

	return api.Success(c, log)
}

// ResendLog handles POST /email-logs/:id/resend.
func (h *EmailHandler) ResendLog(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return api.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Log ID must be a valid integer")
	}

	var original models.EmailLog
	if err := h.db.First(&original, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return api.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Email log not found")
		}
		return api.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", "Failed to fetch email log")
	}

	// Load email provider settings.
	providerSettings := h.loadEmailSettings()
	providerName := providerSettings["email_provider"]
	if providerName == "" {
		return api.Error(c, fiber.StatusBadRequest, "NO_PROVIDER", "No email provider configured")
	}

	provider := NewProvider(providerName, providerSettings)
	if provider == nil {
		return api.Error(c, fiber.StatusBadRequest, "INVALID_PROVIDER", "Unknown email provider: "+providerName)
	}

	// Send the email.
	sendErr := provider.Send([]string{original.RecipientEmail}, original.Subject, original.RenderedBody)

	status := "sent"
	var errMsg *string
	if sendErr != nil {
		status = "failed"
		msg := sendErr.Error()
		errMsg = &msg
	}

	pName := provider.Name()
	newLog := models.EmailLog{
		RuleID:         original.RuleID,
		TemplateSlug:   original.TemplateSlug,
		Action:         "resend",
		RecipientEmail: original.RecipientEmail,
		Subject:        original.Subject,
		RenderedBody:   original.RenderedBody,
		Status:         status,
		ErrorMessage:   errMsg,
		Provider:       &pName,
	}

	h.db.Create(&newLog)

	if sendErr != nil {
		return api.Error(c, fiber.StatusInternalServerError, "SEND_FAILED", "Failed to resend email: "+sendErr.Error())
	}

	return api.Success(c, newLog)
}

// ---------------------------------------------------------------------------
// Email Settings
// ---------------------------------------------------------------------------

// emailSettingKeys lists the site_settings keys managed by the email settings endpoint.
var emailSettingKeys = []string{
	"email_provider",
	"email_from_name",
	"email_from_address",
	"email_smtp_host",
	"email_smtp_port",
	"email_smtp_username",
	"email_smtp_password",
	"email_smtp_encryption",
	"email_resend_api_key",
}

// maskedKeys are settings whose values should be masked in GET responses.
var maskedKeys = map[string]bool{
	"email_smtp_password": true,
	"email_resend_api_key": true,
}

// GetSettings handles GET /email-settings.
func (h *EmailHandler) GetSettings(c *fiber.Ctx) error {
	var settings []models.SiteSetting
	h.db.Where("`key` LIKE ?", "email_%").Find(&settings)

	result := make(map[string]string)
	for _, s := range settings {
		val := ""
		if s.Value != nil {
			val = *s.Value
		}
		if maskedKeys[s.Key] && val != "" {
			val = "••••"
		}
		result[s.Key] = val
	}

	return api.Success(c, result)
}

// SaveSettings handles POST /email-settings.
func (h *EmailHandler) SaveSettings(c *fiber.Ctx) error {
	var body map[string]string
	if err := c.BodyParser(&body); err != nil {
		return api.Error(c, fiber.StatusBadRequest, "INVALID_BODY", "Invalid request body")
	}

	for key, value := range body {
		if !strings.HasPrefix(key, "email_") {
			continue
		}

		// Skip masked password/key values that the user did not change.
		if maskedKeys[key] && value == "••••" {
			continue
		}

		val := value
		setting := models.SiteSetting{
			Key:   key,
			Value: &val,
		}

		h.db.Where("`key` = ?", key).Assign(setting).FirstOrCreate(&setting)
	}

	return api.Success(c, fiber.Map{"message": "Email settings saved"})
}

// TestEmail handles POST /email-settings/test.
func (h *EmailHandler) TestEmail(c *fiber.Ctx) error {
	user := auth.GetCurrentUser(c)
	if user == nil {
		return api.Error(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
	}

	providerSettings := h.loadEmailSettings()
	providerName := providerSettings["email_provider"]
	if providerName == "" {
		return api.Error(c, fiber.StatusBadRequest, "NO_PROVIDER", "No email provider configured")
	}

	provider := NewProvider(providerName, providerSettings)
	if provider == nil {
		return api.Error(c, fiber.StatusBadRequest, "INVALID_PROVIDER", "Unknown email provider: "+providerName)
	}

	subject := "VibeCMS Test Email"
	body := fmt.Sprintf(`<html><body>
<h2>VibeCMS Test Email</h2>
<p>This is a test email confirming that your email configuration is working correctly.</p>
<p>Provider: <strong>%s</strong></p>
<p>Sent at: <strong>%s</strong></p>
</body></html>`, provider.Name(), time.Now().Format(time.RFC1123))

	if err := provider.Send([]string{user.Email}, subject, body); err != nil {
		return api.Error(c, fiber.StatusInternalServerError, "SEND_FAILED", "Failed to send test email: "+err.Error())
	}

	return api.Success(c, fiber.Map{"message": "Test email sent to " + user.Email})
}

// loadEmailSettings reads all email_* site settings into a map.
func (h *EmailHandler) loadEmailSettings() map[string]string {
	var settings []models.SiteSetting
	h.db.Where("`key` LIKE ?", "email_%").Find(&settings)

	result := make(map[string]string)
	for _, s := range settings {
		if s.Value != nil {
			result[s.Key] = *s.Value
		}
	}
	return result
}
