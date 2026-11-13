package ui

import (
	"strings"
	"testing"

	"github.com/KiloProjects/kilonova/web/ui/uitest"
	"github.com/stretchr/testify/require"
	. "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

func TestButton(t *testing.T) {
	t.Run("merges caller classes into one attribute", func(t *testing.T) {
		got := uitest.Render(t, SubmitButton(h.Class("mr-2"), h.ID("x"), Text("Go")))
		require.Equal(t, 1, strings.Count(got, "class="), "classes not merged: %s", got)
		require.Contains(t, got, "mr-2")
		require.Contains(t, got, "btn btn-blue")
		require.Contains(t, got, `type="submit"`)
		require.Contains(t, got, `id="x"`)
		require.Contains(t, got, ">Go</button>")
	})
	t.Run("danger submit is red and submits", func(t *testing.T) {
		require.Equal(t, `<button type="submit" class="btn btn-red">Delete</button>`, uitest.Render(t, DangerSubmitButton(Text("Delete"))))
	})
	t.Run("defaults to type=button", func(t *testing.T) {
		require.Equal(t, `<button type="button" class="btn btn-blue">Go</button>`, uitest.Render(t, Button(Text("Go"))))
	})
}
