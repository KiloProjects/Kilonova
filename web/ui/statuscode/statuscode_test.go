package statuscode

import (
	"context"
	"testing"

	"github.com/KiloProjects/kilonova/web/ui/uitest"
	"github.com/stretchr/testify/require"
)

func TestView(t *testing.T) {
	t.Run("shows the escaped message", func(t *testing.T) {
		got := uitest.Render(t, View(context.Background(), 404, "<nope>"))
		require.Contains(t, got, "<h2>404: Not Found</h2>")
		require.Contains(t, got, "<h1>&lt;nope&gt;</h1>")
		require.NotContains(t, got, "<p>")
	})
	t.Run("falls back to the generic hint", func(t *testing.T) {
		got := uitest.Render(t, View(context.Background(), 500, ""))
		require.Contains(t, got, "<p>")
		require.NotContains(t, got, "<h1>")
	})
}
