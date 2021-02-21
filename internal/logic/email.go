package logic

import (
	"net"
	"net/smtp"

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

func (e *emailer) SendEmail(msg *kilonova.MailerMessage) error {
	em := email.NewEmail()

	em.From = e.from
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
