package email

import (
	"net"
	"net/smtp"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/internal/config"
	"github.com/jordan-wright/email"
	"go.uber.org/zap"
)

var _ kilonova.Mailer = &emailer{}

type emailer struct {
	host string
	auth smtp.Auth
	from string
}

func (e *emailer) SendEmail(msg *kilonova.MailerMessage) error {
	em := email.NewEmail()

	em.From = "noreply@kilonova.ro"
	em.To = []string{msg.To}
	if msg.ReplyTo != "" {
		em.ReplyTo = []string{msg.ReplyTo}
	}

	em.Subject = msg.Subject
	em.Text = []byte(msg.PlainContent)
	em.HTML = []byte(msg.HTMLContent)
	return em.Send(e.host, e.auth)

}

func NewMailer() (kilonova.Mailer, error) {
	host, _, err := net.SplitHostPort(config.Email.Host)
	if err != nil {
		return nil, err
	}
	return &emailer{config.Email.Host, smtp.PlainAuth("", config.Email.Username, config.Email.Password, host), config.Email.Username}, nil
}

type mockMailer struct{}

func (e *mockMailer) SendEmail(msg *kilonova.MailerMessage) error {
	zap.S().Infof("Mailer: mock send message to %q with subject %q (reply-to: %q)", msg.To, msg.Subject, msg.ReplyTo)
	if len(msg.HTMLContent) > 0 {
		zap.S().Info("HTML content:\n", msg.HTMLContent)
	}
	if len(msg.PlainContent) > 0 {
		zap.S().Info("Plaintext content:\n", msg.PlainContent)
	}
	return nil
}

func NewMockMailer() kilonova.Mailer {
	zap.S().Warn("Initializing mock mailer")
	return &mockMailer{}
}
