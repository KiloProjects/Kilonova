package kilonova

import "context"

type Mailer interface {
	SendEmail(msg *MailerMessage) error
}

type MailerMessage struct {
	To      string
	Subject string
	ReplyTo string

	PlainContent string
	HTMLContent  string
}

type MarkdownRenderer interface {
	Render(src []byte) ([]byte, error)
}

type Sessioner interface {
	CreateSession(ctx context.Context, uid int) (string, error)
	GetSession(ctx context.Context, sess string) (int, error)
	RemoveSession(ctx context.Context, sess string) error
}

type Verificationer interface {
	CreateVerification(ctx context.Context, id int) (string, error)
	GetVerification(ctx context.Context, verif string) (int, error)
	RemoveVerification(ctx context.Context, verif string) error
}

// TypeServicer is an interface for a provider for UserService, ProblemService, TestService, SubmissionService and SubTestService
type TypeServicer interface {
	UserService() UserService
	ProblemService() ProblemService
	TestService() TestService
	SubmissionService() SubmissionService
	SubTestService() SubTestService
}
