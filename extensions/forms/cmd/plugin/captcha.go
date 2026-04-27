package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"vibecms/internal/coreapi"
)

const (
	captchaRecaptchaURL = "https://www.google.com/recaptcha/api/siteverify"
	captchaHCaptchaURL  = "https://hcaptcha.com/siteverify"
	captchaTurnstileURL = "https://challenges.cloudflare.com/turnstile/v0/siteverify"
)

// verifyCAPTCHA returns nil on success, error on failure or misconfiguration.
// Token is the CAPTCHA response token from the form. ip is optional.
func (p *FormsPlugin) verifyCAPTCHA(ctx context.Context, provider, secret, token, ip string) error {
	if provider == "" || provider == "none" {
		return nil
	}
	if secret == "" {
		return fmt.Errorf("captcha secret not configured")
	}
	if token == "" {
		return fmt.Errorf("captcha token missing from submission")
	}

	var endpoint string
	switch provider {
	case "recaptcha":
		endpoint = captchaRecaptchaURL
	case "hcaptcha":
		endpoint = captchaHCaptchaURL
	case "turnstile":
		endpoint = captchaTurnstileURL
	default:
		return fmt.Errorf("unknown captcha provider %q", provider)
	}

	form := url.Values{}
	form.Set("secret", secret)
	form.Set("response", token)
	if ip != "" {
		form.Set("remoteip", ip)
	}

	res, err := p.host.Fetch(ctx, coreapi.FetchRequest{
		Method:  "POST",
		URL:     endpoint,
		Headers: map[string]string{"Content-Type": "application/x-www-form-urlencoded"},
		Body:    form.Encode(),
		Timeout: 10,
	})
	if err != nil {
		return fmt.Errorf("captcha verify request: %w", err)
	}
	if res.StatusCode != 200 {
		return fmt.Errorf("captcha verify HTTP %d", res.StatusCode)
	}

	var out struct {
		Success bool `json:"success"`
	}
	if err := json.Unmarshal([]byte(res.Body), &out); err != nil {
		return fmt.Errorf("captcha verify parse: %w", err)
	}
	if !out.Success {
		return fmt.Errorf("captcha verification failed")
	}
	return nil
}

// captchaScriptTag returns the provider's loader script + a placeholder div.
// Injected before </form> in renderFormHTML when captcha_provider != "none".
func captchaScriptTag(provider, siteKey string) string {
	if provider == "" || provider == "none" || siteKey == "" {
		return ""
	}
	switch provider {
	case "recaptcha":
		return fmt.Sprintf(`<script src="https://www.google.com/recaptcha/api.js?render=%s"></script>
<input type="hidden" name="_captcha_token" data-captcha="recaptcha" data-sitekey="%s">`, siteKey, siteKey)
	case "hcaptcha":
		return fmt.Sprintf(`<script src="https://js.hcaptcha.com/1/api.js" async defer></script>
<div class="h-captcha" data-sitekey="%s"></div>
<input type="hidden" name="_captcha_token" data-captcha="hcaptcha">`, siteKey)
	case "turnstile":
		return fmt.Sprintf(`<script src="https://challenges.cloudflare.com/turnstile/v0/api.js" async defer></script>
<div class="cf-turnstile" data-sitekey="%s" data-callback="vibeFormTurnstileCallback"></div>
<input type="hidden" name="_captcha_token" data-captcha="turnstile">`, siteKey)
	}
	return ""
}

// extractCaptchaToken pulls the CAPTCHA token from submissionData and removes it.
func extractCaptchaToken(submissionData map[string]any) string {
	token, _ := submissionData["_captcha_token"].(string)
	delete(submissionData, "_captcha_token")
	return token
}

// isCaptchaEscapedToken is a helper that checks for the turnstile callback reference.
// Embedded in view.html script template.
func captchaClientScript() string {
	return strings.TrimSpace(`
window.vibeFormTurnstileCallback = function(token) {
  document.querySelectorAll('input[name="_captcha_token"][data-captcha="turnstile"]').forEach(function(i) { i.value = token; });
};`)
}
