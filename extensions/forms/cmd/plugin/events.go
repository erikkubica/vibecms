package main

import (
	"context"
)

// emitSubmitted emits the "forms:submitted" event after a successful form submission.
func (p *FormsPlugin) emitSubmitted(ctx context.Context, formID uint, formSlug string, submissionID uint, data map[string]any, metadata map[string]any) {
	payload := map[string]any{
		"form_id":       formID,
		"form_slug":     formSlug,
		"submission_id": submissionID,
		"data":          data,
		"metadata":      metadata,
	}
	if err := p.host.Emit(ctx, "forms:submitted", payload); err != nil {
		p.host.Log(ctx, "warn", "emit forms:submitted failed: "+err.Error(), nil)
	}
}
