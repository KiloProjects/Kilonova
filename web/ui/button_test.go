package ui

import (
	"strings"
	"testing"

	. "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

func TestButton(t *testing.T) {
	t.Run("merges caller classes into one attribute", func(t *testing.T) {
		got := render(t, SubmitButton(h.Class("mr-2"), h.ID("x"), Text("Go")))
		if strings.Count(got, "class=") != 1 || !strings.Contains(got, "mr-2") || !strings.Contains(got, "btn btn-blue") {
			t.Fatalf("classes not merged: %s", got)
		}
		if !strings.Contains(got, `type="submit"`) || !strings.Contains(got, `id="x"`) || !strings.Contains(got, ">Go</button>") {
			t.Fatalf("missing type/attrs/text: %s", got)
		}
	})
	t.Run("defaults to type=button", func(t *testing.T) {
		if got := render(t, Button(Text("Go"))); got != `<button type="button" class="btn btn-blue">Go</button>` {
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
