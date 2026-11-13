package navbar

import (
	"context"
	"testing"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/domain/user"
	"github.com/KiloProjects/kilonova/web/ui/uitest"
	"github.com/stretchr/testify/require"
)

func TestView(t *testing.T) {
	base := uitest.Ctx("en")
	as := func(u *kilonova.UserBrief) context.Context { return context.WithValue(base, user.AuthedUserKey, u) }

	t.Run("visitor gets login with back link and no dropdown", func(t *testing.T) {
		got := uitest.Render(t, View(base, false, "/problems/1"))
		require.Contains(t, got, `href="/login?back=/problems/1"`)
		require.NotContains(t, got, "profile-dropdown")
	})

	t.Run("user gets escaped name, no admin links", func(t *testing.T) {
		got := uitest.Render(t, View(as(&kilonova.UserBrief{ID: 2, Name: "<b>"}), false, "/"))
		require.Contains(t, got, "&lt;b&gt;")
		require.NotContains(t, got, `href="/admin"`)
		require.NotContains(t, got, "/login")
	})

	t.Run("admin gets admin and proposer links", func(t *testing.T) {
		got := uitest.Render(t, View(as(&kilonova.UserBrief{ID: 1, Name: "a", Admin: true}), false, "/"))
		require.Contains(t, got, `href="/admin"`)
		require.Contains(t, got, `href="/proposer"`)
	})
}
