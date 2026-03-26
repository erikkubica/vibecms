package email

// Provider defines the interface for sending emails.
type Provider interface {
	Name() string
	Send(to []string, subject string, html string) error
}

// NewProvider creates a provider from site settings map.
// Returns nil if provider name is empty or unknown.
func NewProvider(name string, settings map[string]string) Provider {
	switch name {
	case "smtp":
		return NewSMTPProvider(settings)
	case "resend":
		return NewResendProvider(settings)
	default:
		return nil
	}
}
