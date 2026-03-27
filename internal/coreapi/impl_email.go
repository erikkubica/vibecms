package coreapi

import (
	"context"
	"vibecms/internal/events"
)

func (c *coreImpl) SendEmail(_ context.Context, req EmailRequest) error {
	if len(req.To) == 0 {
		return NewValidation("at least one recipient is required")
	}
	if req.Subject == "" {
		return NewValidation("subject is required")
	}
	c.eventBus.Publish("email.send", events.Payload{
		"to":      req.To,
		"subject": req.Subject,
		"html":    req.HTML,
	})
	return nil
}
