package statuscode

import (
	"context"
	"strings"
	"testing"

	. "maragu.dev/gomponents"
)

func TestView(t *testing.T) {
	t.Run("shows the escaped message", func(t *testing.T) {
		got := render(t, View(context.Background(), 404, "<nope>"))
		if !strings.Contains(got, "<h2>404: Not Found</h2>") || !strings.Contains(got, "<h1>&lt;nope&gt;</h1>") || strings.Contains(got, "<p>") {
			t.Fatalf("got %s", got)
		}
	})
	t.Run("falls back to the generic hint", func(t *testing.T) {
		if got := render(t, View(context.Background(), 500, "")); !strings.Contains(got, "<p>") || strings.Contains(got, "<h1>") {
			t.Fatalf("got %s", got)
		}
	})
}

func render(t *testing.T, n Node) string {
	t.Helper()
	var b strings.Builder
	if err := n.Render(&b); err != nil {
		t.Fatal(err)
	}
	return b.String()
}
