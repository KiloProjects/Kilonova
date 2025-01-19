package kilonova

import (
	"context"
	"github.com/gosimple/slug"
)

type Mailer interface {
	SendEmail(ctx context.Context, msg *MailerMessage) error
}

type MailerMessage struct {
	To      string
	Subject string
	ReplyTo string

	PlainContent string
	HTMLContent  string
}

type MarkdownRenderContext struct {
	Problem  *Problem
	BlogPost *BlogPost
}

func MakeSlug(org string) string {
	return slug.MakeLang(org, "ro")
}
