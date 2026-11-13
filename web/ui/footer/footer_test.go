package footer

import (
	"testing"
	"time"

	"github.com/KiloProjects/kilonova/domain/config"
	"github.com/KiloProjects/kilonova/sudoapi/flags"
	"github.com/KiloProjects/kilonova/web/ui/uitest"
	"github.com/stretchr/testify/require"
)

func TestView(t *testing.T) {
	ctx := uitest.Ctx("ro")

	t.Run("selects the session language", func(t *testing.T) {
		got := uitest.Render(t, View(ctx, 0, time.Now()))
		require.Contains(t, got, `<option value="ro" selected>`)
		require.NotContains(t, got, `<option value="en" selected>`)
	})

	t.Run("donate link follows its flag", func(t *testing.T) {
		ctx := config.WithFlagset(ctx, config.NewFlagset())
		require.NoError(t, flags.DonationsEnabled.Update(ctx, false))
		require.NotContains(t, uitest.Render(t, View(ctx, 0, time.Now())), `href="/donate"`, "donate link shown while disabled")
	})

	t.Run("query count is admin-only", func(t *testing.T) {
		got := uitest.Render(t, View(ctx, 0, time.Now()))
		require.Contains(t, got, "<details", "timings missing")
		require.NotContains(t, got, "~", "query count shown to visitor")
	})
}
