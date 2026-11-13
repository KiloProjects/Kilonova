package ui

import (
	"testing"

	"github.com/KiloProjects/kilonova/web/ui/uitest"
	"github.com/stretchr/testify/require"
	. "maragu.dev/gomponents"
)

func TestTabs(t *testing.T) {
	got := uitest.Render(t, Tabs("b",
		Tab{Key: "a", Href: "/a", Label: Text("A")},
		Tab{Key: "x", Href: "/x", Label: Text("X"), Hide: true},
		Tab{Key: "b", Href: "/b", Label: Text("B"), Attrs: Attr("hx-get", "/b?m")},
	))
	want := `<a class="p-1" href="/a">A</a> <div class="topbar-separator"></div> <a class="p-1 topbar-selected" href="/b" hx-get="/b?m">B</a>`
	require.Equal(t, want, got)
	require.Empty(t, uitest.Render(t, Tabs("a", Tab{Hide: true})), "all hidden should render nothing")
}
