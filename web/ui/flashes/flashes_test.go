package flashes_test

import (
	"context"
	"testing"

	"github.com/KiloProjects/kilonova/sudoapi/flags"
	"github.com/KiloProjects/kilonova/web/ui/flashes"
	"github.com/KiloProjects/kilonova/web/ui/modals"
	"github.com/KiloProjects/kilonova/web/ui/problems"
	"github.com/KiloProjects/kilonova/web/ui/uitest"
	"github.com/stretchr/testify/require"
	. "maragu.dev/gomponents"
)

func TestRegistry(t *testing.T) {
	seen := map[string]bool{}
	for _, f := range flashes.All() {
		require.NotEmpty(t, f.Name())
		require.NotEmpty(t, f.Description(), f.Name())
		require.False(t, seen[f.Name()], "duplicate flash name %q", f.Name())
		seen[f.Name()] = true
	}
}

func TestPanel(t *testing.T) {
	t.Run("visible by default", func(t *testing.T) {
		got := uitest.Render(t, flashes.Panel(uitest.Ctx("en"), flashes.GPC, Text("hi")))
		uitest.Contains(t, got, `data-flashname="gpc_announcement"`, "hi")
	})

	t.Run("hidden flash renders nothing", func(t *testing.T) {
		ctx := uitest.Ctx("en")
		require.NoError(t, flags.HiddenFlashes.Update(ctx, []string{"gpc_announcement", "no_longer_registered"}))
		require.Empty(t, uitest.Render(t, flashes.Panel(ctx, flashes.GPC, Text("hi"))))
		require.NotEmpty(t, uitest.Render(t, flashes.Panel(ctx, flashes.ContestDisclaimer, Text("hi"))), "other flashes are unaffected")
	})
}

// The real announcements go through the registry, so hiding them works end to end.
func TestAnnouncementsHonourHiddenList(t *testing.T) {
	for _, tc := range []struct {
		flash flashes.Flash
		node  func(ctx context.Context) Node
	}{
		{flashes.GPC, modals.GPCAnnouncement},
		{flashes.ContestDisclaimer, problems.ContestDisclaimer},
	} {
		t.Run(tc.flash.Name(), func(t *testing.T) {
			marker := `data-flashname="` + tc.flash.Name() + `"`
			ctx := uitest.Ctx("en")
			uitest.Contains(t, uitest.Render(t, tc.node(ctx)), marker, `data-flashdiscard="`+tc.flash.Name()+`"`)

			require.NoError(t, flags.HiddenFlashes.Update(ctx, []string{tc.flash.Name()}))
			uitest.NotContains(t, uitest.Render(t, tc.node(ctx)), marker)
		})
	}
}
