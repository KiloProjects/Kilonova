package ui

import (
	"context"
	"html/template"
	"strings"
	"testing"

	"github.com/a-h/templ"
	. "maragu.dev/gomponents"
)

// Round-trips a button through every adapter; output must survive unescaped.
func TestAdapters(t *testing.T) {
	btn := Button(Text("Go"))
	want := render(t, btn)

	tmpl := template.Must(template.New("t").Funcs(template.FuncMap{"node": ToGoHTML}).Parse(`{{ node . }}`))
	got := render(t, Group{
		FromTempl(context.Background(), ToTempl(btn)),
		FromGoTmpl(tmpl, btn),
		FromTempl(context.Background(), templ.FromGoHTML(tmpl, btn)),
	})
	if got != strings.Repeat(want, 3) {
		t.Fatalf("got %s", got)
	}
}
