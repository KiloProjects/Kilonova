package logic

import (
	"errors"

	"github.com/KiloProjects/Kilonova/internal/config"
	"gopkg.in/gomail.v2"
)

type Email struct {
	sender gomail.SendCloser
}

func (e *Email) SendEmail(to, subject, content string) error {
	if to == "" {
		return errors.New("No `to` specified")
	}
	m := gomail.NewMessage()
	m.SetHeader("From", config.C.Email.Username)
	m.SetHeader("To", to)
	if subject != "" {
		m.SetHeader("Subject", subject)
	}
	m.SetBody("text/plain", content)

	return gomail.Send(e.sender, m)
}

func (e *Email) SendComplexEmail(msg *gomail.Message) error {
	return gomail.Send(e.sender, msg)
}

func (e *Email) Close() error {
	return e.sender.Close()
}

func NewEmail() (*Email, error) {
	d := gomail.NewDialer(config.C.Email.Host, config.C.Email.Port, config.C.Email.Username, config.C.Email.Password)
	sender, err := d.Dial()
	if err != nil {
		return nil, err
	}
	return &Email{sender}, nil
}
