package email

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strings"
)

// SMTPProvider sends email via SMTP with STARTTLS or direct TLS support.
type SMTPProvider struct {
	host     string
	port     string
	user     string
	password string
	from     string
	fromName string
}

// NewSMTPProvider creates an SMTP provider from settings.
func NewSMTPProvider(settings map[string]string) *SMTPProvider {
	return &SMTPProvider{
		host:     settings["email_smtp_host"],
		port:     settings["email_smtp_port"],
		user:     settings["email_smtp_user"],
		password: settings["email_smtp_password"],
		from:     settings["email_smtp_from"],
		fromName: settings["email_smtp_from_name"],
	}
}

func (s *SMTPProvider) Name() string { return "smtp" }

func (s *SMTPProvider) Send(to []string, subject string, html string) error {
	if len(to) == 0 {
		return fmt.Errorf("smtp: no recipients specified")
	}

	from := s.from
	if s.fromName != "" {
		from = fmt.Sprintf("%s <%s>", s.fromName, s.from)
	}

	msg := buildMIME(from, to, subject, html)
	addr := net.JoinHostPort(s.host, s.port)

	var auth smtp.Auth
	if s.user != "" && s.password != "" {
		auth = smtp.PlainAuth("", s.user, s.password, s.host)
	}

	// Port 465 uses implicit TLS (smtps); all others use STARTTLS.
	if s.port == "465" {
		return s.sendDirectTLS(addr, auth, to, msg)
	}
	return s.sendSTARTTLS(addr, auth, to, msg)
}

// sendDirectTLS connects via crypto/tls then creates an smtp.Client.
func (s *SMTPProvider) sendDirectTLS(addr string, auth smtp.Auth, to []string, msg []byte) error {
	tlsConn, err := tls.Dial("tcp", addr, &tls.Config{ServerName: s.host})
	if err != nil {
		return fmt.Errorf("smtp: tls dial %s: %w", addr, err)
	}
	defer tlsConn.Close()

	client, err := smtp.NewClient(tlsConn, s.host)
	if err != nil {
		return fmt.Errorf("smtp: new client: %w", err)
	}
	defer client.Close()

	return s.deliver(client, auth, to, msg)
}

// sendSTARTTLS connects plaintext then upgrades with STARTTLS.
func (s *SMTPProvider) sendSTARTTLS(addr string, auth smtp.Auth, to []string, msg []byte) error {
	client, err := smtp.Dial(addr)
	if err != nil {
		return fmt.Errorf("smtp: dial %s: %w", addr, err)
	}
	defer client.Close()

	if ok, _ := client.Extension("STARTTLS"); ok {
		if err := client.StartTLS(&tls.Config{ServerName: s.host}); err != nil {
			return fmt.Errorf("smtp: starttls: %w", err)
		}
	}

	return s.deliver(client, auth, to, msg)
}

// deliver performs AUTH, MAIL FROM, RCPT TO, DATA, and QUIT on the client.
func (s *SMTPProvider) deliver(client *smtp.Client, auth smtp.Auth, to []string, msg []byte) error {
	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("smtp: auth: %w", err)
		}
	}

	if err := client.Mail(s.from); err != nil {
		return fmt.Errorf("smtp: mail from: %w", err)
	}

	for _, rcpt := range to {
		if err := client.Rcpt(rcpt); err != nil {
			return fmt.Errorf("smtp: rcpt %s: %w", rcpt, err)
		}
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp: data: %w", err)
	}
	if _, err := w.Write(msg); err != nil {
		return fmt.Errorf("smtp: write: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("smtp: close data: %w", err)
	}

	return client.Quit()
}

func buildMIME(from string, to []string, subject string, html string) []byte {
	var buf strings.Builder
	buf.WriteString("From: " + from + "\r\n")
	buf.WriteString("To: " + strings.Join(to, ", ") + "\r\n")
	buf.WriteString("Subject: " + subject + "\r\n")
	buf.WriteString("MIME-Version: 1.0\r\n")
	buf.WriteString("Content-Type: text/html; charset=\"UTF-8\"\r\n")
	buf.WriteString("\r\n")
	buf.WriteString(html)
	return []byte(buf.String())
}
