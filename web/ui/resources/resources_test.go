package resources

import (
	"context"
	"strings"
	"testing"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/domain/user"
	"github.com/KiloProjects/kilonova/internal/util"
	"github.com/stretchr/testify/require"
)

func TestNewForm(t *testing.T) {
	base := context.WithValue(context.Background(), util.LangKey, "en")
	render := func(ctx context.Context) string {
		var b strings.Builder
		require.NoError(t, NewForm(ctx, 12).Render(&b))
		return b.String()
	}
	t.Run("posts to the problem and hides pre-approval from non-admins", func(t *testing.T) {
		got := render(base)
		require.Contains(t, got, `hx-post="/problems/12/externalResources/create"`)
		require.NotContains(t, got, "preApproved")
	})
	t.Run("admins can pre-approve", func(t *testing.T) {
		require.Contains(t, render(context.WithValue(base, user.AuthedUserKey, &kilonova.UserBrief{ID: 1, Admin: true})), `name="preApproved" type="checkbox" checked`)
	})
}
