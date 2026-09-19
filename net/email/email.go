package email

import (
	"cmp"
	"context"
	"log/slog"
	"net"
	"net/smtp"
	"sync"

	"github.com/KiloProjects/kilonova/domain/config"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"

	"github.com/KiloProjects/kilonova"
	"github.com/jordan-wright/email"
)

var (
	loggerOnce  sync.Once
	emailLogger *slog.Logger
)

var _ kilonova.Mailer = &emailer{}

type emailer struct {
	host string
	auth smtp.Auth
	from string
}

func (e *emailer) SendEmail(ctx context.Context, msg *kilonova.MailerMessage) error {
	loggerOnce.Do(func() {
		emailLogger = slog.New(slog.NewJSONHandler(config.LogWriter("email.log", 200), &slog.HandlerOptions{
			Level: slog.LevelInfo,
		}))
	})

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
	err := em.Send(e.host, e.auth)
	if err != nil {
		emailLogger.ErrorContext(ctx, "Error sending email", slog.Any("err", err))
	} else {
		emailLogger.InfoContext(ctx, "Sent email", slog.Any("email", msg.To), slog.String("subject", msg.Subject))
	}
	return err
}

// NewMailer builds an SMTP mailer. hostPort is "host:port"; sendAs defaults to username.
func NewMailer(hostPort, username, password, sendAs string) (kilonova.Mailer, error) {
	host, _, err := net.SplitHostPort(hostPort)
	if err != nil {
		return nil, err
	}
	return &emailer{hostPort, smtp.PlainAuth("", username, password, host), cmp.Or(sendAs, username)}, nil
}
