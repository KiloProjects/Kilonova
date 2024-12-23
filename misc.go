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

type RenderContext struct {
	Problem  *Problem
	BlogPost *BlogPost
}

type MarkdownRenderer interface {
	Render(src []byte, ctx *RenderContext) ([]byte, error)
}

func MakeSlug(org string) string {
	return slug.MakeLang(org, "ro")
}
