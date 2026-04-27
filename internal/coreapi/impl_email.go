package coreapi

import (
	"context"
	"strings"
	"vibecms/internal/events"
	"vibecms/internal/models"
)

func (c *coreImpl) SendEmail(ctx context.Context, req EmailRequest) error {
	if len(req.To) == 0 {
		return NewValidation("at least one recipient is required")
	}
	if req.Subject == "" {
		return NewValidation("subject is required")
	}

	// Load provider settings so the provider plugin can actually send.
	var allSettings []models.SiteSetting
	c.db.Find(&allSettings)

	settingsMap := make(map[string]string, len(allSettings))
	for _, s := range allSettings {
		if s.Value != nil {
			settingsMap[s.Key] = *s.Value
		}
	}

	providerName := settingsMap["email_provider"]
	if providerName == "" {
		return NewValidation("no email provider configured — set email_provider in site settings")
	}

	providerSettings := map[string]string{
		"provider":   providerName,
		"from_email": settingsMap["from_email"],
		"from_name":  settingsMap["from_name"],
	}
	extPrefix := "ext." + providerName + "."
	for k, v := range settingsMap {
		if strings.HasPrefix(k, extPrefix) {
			providerSettings[strings.TrimPrefix(k, extPrefix)] = v
		}
	}

	if !c.eventBus.HasHandlers("email.send") {
		return NewInternal("no email provider plugin is handling email.send events")
	}

	c.eventBus.PublishSync("email.send", events.Payload{
		"to":       req.To,
		"subject":  req.Subject,
		"html":     req.HTML,
		"settings": providerSettings,
	})
	return nil
}
