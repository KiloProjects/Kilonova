package ui

import (
	"html/template"
	"testing"

	. "maragu.dev/gomponents"
)

// Round-trips a button through both adapters; output must survive unescaped.
func TestAdapters(t *testing.T) {
	btn := Button(Text("Go"))
	want := render(t, btn)

	tmpl := template.Must(template.New("t").Funcs(template.FuncMap{"node": ToGoHTML}).Parse(`{{ node . }}`))
	if got := render(t, FromGoTmpl(tmpl, btn)); got != want {
		t.Fatalf("got %s, want %s", got, want)
	}
	if got := render(t, Group{FromGoTmpl(nil, nil)}); got != "" {
		t.Fatalf("nil template rendered %q", got)
	}
	if got, err := ToGoHTML(nil); got != "" || err != nil {
		t.Fatalf("nil node rendered %q, %v", got, err)
	}
}
