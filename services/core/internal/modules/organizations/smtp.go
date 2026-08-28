package organizations

import (
	"context"
	"fmt"
	"net/smtp"
	"strings"
	"time"
)

type SMTPConfig struct{ Address, Host, Username, Password, From string }
type SMTPNotifier struct{ config SMTPConfig }

func NewSMTPNotifier(config SMTPConfig) (*SMTPNotifier, error) {
	if strings.TrimSpace(config.Address) == "" || strings.TrimSpace(config.Host) == "" || !strings.Contains(config.From, "@") {
		return nil, fmt.Errorf("invalid SMTP invitation configuration")
	}
	return &SMTPNotifier{config: config}, nil
}
func (n *SMTPNotifier) SendInvitation(ctx context.Context, email, url string, expires time.Time) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	subject := "Documi organization invitation"
	body := fmt.Sprintf("You have been invited to a Documi organization.\r\n\r\nAccept invitation: %s\r\n\r\nThis single-use link expires at %s.\r\n", url, expires.UTC().Format(time.RFC3339))
	message := []byte("From: " + n.config.From + "\r\nTo: " + email + "\r\nSubject: " + subject + "\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n" + body)
	var auth smtp.Auth
	if n.config.Username != "" {
		auth = smtp.PlainAuth("", n.config.Username, n.config.Password, n.config.Host)
	}
	result := make(chan error, 1)
	go func() { result <- smtp.SendMail(n.config.Address, auth, n.config.From, []string{email}, message) }()
	select {
	case err := <-result:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}
