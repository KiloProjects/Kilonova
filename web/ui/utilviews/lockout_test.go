package utilviews

import (
	"context"
	"testing"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/domain/user"
	"github.com/KiloProjects/kilonova/internal/util"
	"github.com/KiloProjects/kilonova/web/ui/uitest"
)

func TestUserLockout(t *testing.T) {
	as := func(name string) context.Context {
		ctx := context.WithValue(context.Background(), util.LangKey, "en")
		return context.WithValue(ctx, user.AuthedUserKey, &kilonova.UserBrief{ID: 2, Name: name})
	}
	t.Run("warns users with a valid name", func(t *testing.T) {
		uitest.Contains(t, uitest.Render(t, UserLockout(as("bob"))), "<strong class=\"text-lg\">bob</strong>", "text-sm text-muted mb-2")
	})
	t.Run("doesn't warn legacy invalid names", func(t *testing.T) {
		uitest.NotContains(t, uitest.Render(t, UserLockout(as("a b"))), "text-sm text-muted mb-2")
	})
}
