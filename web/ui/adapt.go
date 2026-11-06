package ui

import (
	"context"
	"html/template"
	"io"
	"strings"

	"github.com/a-h/templ"
	. "maragu.dev/gomponents"
)

// Bridges between gomponents, templ and html/template while they coexist.
// templ -> html/template already exists as the per-request "renderComponent" func,
// and html/template -> templ is templ.FromGoHTML.

// ToTempl lets a templ component (or the templ layout) render a gomponents node.
func ToTempl(n Node) templ.Component {
	return templ.ComponentFunc(func(_ context.Context, w io.Writer) error {
		return n.Render(w)
	})
}

// FromTempl embeds a templ component in a gomponents tree. ctx is the request context,
// since templ components read the language/user from it. A nil component renders nothing.
func FromTempl(ctx context.Context, c templ.Component) Node {
	if c == nil {
		return nil
	}
	return NodeFunc(func(w io.Writer) error {
		return c.Render(ctx, w)
	})
}

// FromGoTmpl embeds an html/template in a gomponents tree. For a named
// block, pass t.Lookup("name").
func FromGoTmpl(t *template.Template, data any) Node {
	return NodeFunc(func(w io.Writer) error {
		return t.Execute(w, data)
	})
}

// ToGoHTML renders a gomponents node for html/template; exposed there as {{ node . }}.
func ToGoHTML(n Node) (template.HTML, error) {
	var sb strings.Builder
	err := n.Render(&sb)
	return template.HTML(sb.String()), err //nolint:gosec // gomponents output is already escaped
}
