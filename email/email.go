package email

import (
	"context"
	"net"
	"net/smtp"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/internal/config"
	"github.com/jordan-wright/email"
)

var _ kilonova.Mailer = &emailer{}

type emailer struct {
	host string
	auth smtp.Auth
	from string
}

func (e *emailer) SendEmail(ctx context.Context, msg *kilonova.MailerMessage) error {
	ctx, span := otel.Tracer("email").Start(ctx, "SendEmail")
	defer span.End()
	span.SetAttributes(attribute.String("email", msg.To), attribute.String("subject", msg.Subject))

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
