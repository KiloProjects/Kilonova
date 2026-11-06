package ui

import (
	"html/template"
	"io"
	"strings"

	. "maragu.dev/gomponents"
)

// Bridges between gomponents and the legacy html/template pages.

// FromGoTmpl embeds an html/template in a gomponents tree. For a named
// block, pass t.Lookup("name"); a nil template (an undefined block) renders nothing.
func FromGoTmpl(t *template.Template, data any) Node {
	if t == nil {
		return nil
	}
	return NodeFunc(func(w io.Writer) error {
		return t.Execute(w, data)
	})
}

// ToGoHTML renders a gomponents node for html/template; exposed there as {{ node . }}.
// A nil node renders nothing.
func ToGoHTML(n Node) (template.HTML, error) {
	if n == nil {
		return "", nil
	}
	var sb strings.Builder
	err := n.Render(&sb)
	return template.HTML(sb.String()), err //nolint:gosec // gomponents output is already escaped
}
