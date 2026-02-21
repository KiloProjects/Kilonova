package email

import (
	"context"
	"log/slog"
	"net"
	"net/smtp"
	"path"
	"sync"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/internal/config"
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
		emailLogger = slog.New(slog.NewJSONHandler(&lumberjack.Logger{
			Filename: path.Join(config.Common.LogDir, "email.log"),
			MaxSize:  200, // MB
			Compress: true,
		}, &slog.HandlerOptions{
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

func NewMailer() (kilonova.Mailer, error) {
	host, _, err := net.SplitHostPort(config.Email.Host)
	if err != nil {
		return nil, err
	}
	return &emailer{config.Email.Host, smtp.PlainAuth("", config.Email.Username, config.Email.Password, host), config.Email.Username}, nil
}
