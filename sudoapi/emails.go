package sudoapi

import (
	"fmt"
	"strings"

	"github.com/KiloProjects/kilonova"
	g "maragu.dev/gomponents"
	c "maragu.dev/gomponents/components"
	h "maragu.dev/gomponents/html"
)

// emailLayout is the document every HTML email shares: the content, then the signature.
// It is deliberately unstyled, so mail clients show it like a plain message.
func emailLayout(lang, title, branding, host string, content ...g.Node) g.Node {
	return c.HTML5(c.HTML5Props{
		Title:    title,
		Language: lang,
		Body: g.Group{
			g.Group(content),
			h.Hr(),
			h.P(
				g.Text(kilonova.GetText(lang, "mail.signature", branding)), h.Br(),
				h.A(h.Href(host), g.Text(host)),
			),
		},
	})
}

// linkEmail is an email that asks the user to open one link, such as a password reset or an email verification.
// Kind picks the wording: the mail.{subject,body,ignore}.<Kind> translations.
type linkEmail struct {
	Kind     string
	Lang     string
	Name     string
	URL      string
	Host     string
	Branding string
}

func (e linkEmail) text(part string) string {
	return kilonova.GetText(e.Lang, "mail."+part+"."+e.Kind)
}

func (e linkEmail) html() g.Node {
	return emailLayout(e.Lang, e.text("subject"), e.Branding, e.Host,
		h.P(g.Text(kilonova.GetText(e.Lang, "mail.greeting", e.Name))),
		h.P(g.Text(e.text("body")), g.Text(" "), h.A(h.Href(e.URL), g.Text(e.URL))),
		h.P(g.Text(e.text("ignore"))),
	)
}

func (e linkEmail) plain() string {
	return plainEmail(e.Lang, e.Branding, e.Host,
		kilonova.GetText(e.Lang, "mail.greeting", e.Name),
		e.text("body")+"\n"+e.URL,
		e.text("ignore"),
	)
}

func (e linkEmail) message(to string) (*kilonova.MailerMessage, error) {
	return emailMessage(to, e.text("subject"), e.plain(), e.html())
}

// generatedUserEmail gives the login details to a user whose account was created for them.
type generatedUserEmail struct {
	Subject  string
	Lang     string
	Name     string
	Username string
	Password string
	// Contest is the contest the user was registered in on creation, if any.
	Contest  *kilonova.Contest
	Host     string
	Branding string
}

func (e generatedUserEmail) contestURL() string {
	return fmt.Sprintf("%s/contests/%d", e.Host, e.Contest.ID)
}

func (e generatedUserEmail) html() g.Node {
	t := func(key string) g.Node { return g.Text(kilonova.GetText(e.Lang, key)) }
	return emailLayout(e.Lang, e.Subject, e.Branding, e.Host,
		h.P(g.Text(kilonova.GetText(e.Lang, "mail.greeting", e.Name))),
		h.P(t("mail.body.generated")),
		h.P(
			t("username"), g.Text(": "), h.Code(g.Text(e.Username)), h.Br(),
			t("password"), g.Text(": "), h.Code(g.Text(e.Password)),
		),
		g.Iff(e.Contest != nil, func() g.Node {
			return g.Group{
				h.P(t("mail.generated.contest"), g.Text(" "), h.A(h.Href(e.contestURL()), g.Text(e.Contest.Name))),
				h.P(t("mail.generated.contest_link"), g.Text(" "), h.A(h.Href(e.contestURL()), g.Text(e.contestURL()))),
			}
		}),
		h.P(t("mail.generated.outro")),
	)
}

func (e generatedUserEmail) plain() string {
	t := func(key string) string { return kilonova.GetText(e.Lang, key) }
	paragraphs := []string{
		kilonova.GetText(e.Lang, "mail.greeting", e.Name),
		t("mail.body.generated") + "\n" + t("username") + ": " + e.Username + "\n" + t("password") + ": " + e.Password,
	}
	if e.Contest != nil {
		paragraphs = append(paragraphs, t("mail.generated.contest")+" "+e.Contest.Name+"\n"+t("mail.generated.contest_link")+" "+e.contestURL())
	}
	return plainEmail(e.Lang, e.Branding, e.Host, append(paragraphs, t("mail.generated.outro"))...)
}

func (e generatedUserEmail) message(to string) (*kilonova.MailerMessage, error) {
	return emailMessage(to, e.Subject, e.plain(), e.html())
}

// plainEmail joins the paragraphs of a plain text email and signs it like emailLayout does.
func plainEmail(lang, branding, host string, paragraphs ...string) string {
	return strings.Join(append(paragraphs, "------\n"+kilonova.GetText(lang, "mail.signature", branding)+"\n"+host), "\n\n")
}

// emailMessage carries both variants, so mail clients can pick the one they support.
func emailMessage(to, subject, plain string, html g.Node) (*kilonova.MailerMessage, error) {
	var b strings.Builder
	if err := html.Render(&b); err != nil {
		return nil, err
	}
	return &kilonova.MailerMessage{To: to, Subject: subject, PlainContent: plain, HTMLContent: b.String()}, nil
}
