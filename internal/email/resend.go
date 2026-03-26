package email

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// ResendProvider sends email via the Resend HTTP API.
type ResendProvider struct {
	apiKey   string
	from     string
	fromName string
}

// NewResendProvider creates a Resend provider from settings.
func NewResendProvider(settings map[string]string) *ResendProvider {
	return &ResendProvider{
		apiKey:   settings["email_resend_api_key"],
		from:     settings["email_resend_from"],
		fromName: settings["email_resend_from_name"],
	}
}

func (r *ResendProvider) Name() string { return "resend" }

func (r *ResendProvider) Send(to []string, subject string, html string) error {
	if len(to) == 0 {
		return fmt.Errorf("resend: no recipients specified")
	}

	from := r.from
	if r.fromName != "" {
		from = fmt.Sprintf("%s <%s>", r.fromName, r.from)
	}

	payload := map[string]interface{}{
		"from":    from,
		"to":      to,
		"subject": subject,
		"html":    html,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("resend: marshal payload: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, "https://api.resend.com/emails", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("resend: create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+r.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("resend: send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("resend: API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}
