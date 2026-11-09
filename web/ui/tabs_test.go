package ui

import (
	"testing"

	. "maragu.dev/gomponents"
)

func TestTabs(t *testing.T) {
	got := render(t, Tabs("b",
		Tab{Key: "a", Href: "/a", Label: Text("A")},
		Tab{Key: "x", Href: "/x", Label: Text("X"), Hide: true},
		Tab{Key: "b", Href: "/b", Label: Text("B"), Attrs: Attr("hx-get", "/b?m")},
	))
	want := `<a class="p-1" href="/a">A</a> <div class="topbar-separator"></div> <a class="p-1 topbar-selected" href="/b" hx-get="/b?m">B</a>`
	if got != want {
		t.Fatalf("got %s\nwant %s", got, want)
	}
	if got := render(t, Tabs("a", Tab{Hide: true})); got != "" {
		t.Fatalf("all hidden should render nothing, got %q", got)
	}
}
