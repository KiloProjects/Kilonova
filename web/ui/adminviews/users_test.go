package adminviews

import (
	"testing"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/web/ui/uitest"
)

func TestUsersList(t *testing.T) {
	ctx := uitest.Ctx("en")
	t.Run("rows show roles and display name", func(t *testing.T) {
		got := uitest.Render(t, UsersList(ctx, UsersListParams{Users: []*kilonova.UserBrief{{ID: 1, Name: "alice", DisplayName: "Al", Admin: true}}}))
		uitest.Contains(t, got, `href="/profile/alice"`, "alice (Al)", "✅", "❌")
		uitest.NotContains(t, got, "text-4xl")
	})
	t.Run("empty list shows the placeholder", func(t *testing.T) {
		got := uitest.Render(t, UsersList(ctx, UsersListParams{}))
		uitest.Contains(t, got, "text-4xl")
		uitest.NotContains(t, got, "<table")
	})
}
