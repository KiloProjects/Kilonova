package resources

import (
	"context"
	"strings"
	"testing"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/domain/user"
	"github.com/KiloProjects/kilonova/internal/util"
)

func TestNewForm(t *testing.T) {
	base := context.WithValue(context.Background(), util.LangKey, "en")
	render := func(ctx context.Context) string {
		var b strings.Builder
		if err := NewForm(ctx, 12).Render(&b); err != nil {
			t.Fatal(err)
		}
		return b.String()
	}
	t.Run("posts to the problem and hides pre-approval from non-admins", func(t *testing.T) {
		got := render(base)
		if !strings.Contains(got, `hx-post="/problems/12/externalResources/create"`) || strings.Contains(got, "preApproved") {
			t.Fatal(got)
		}
	})
	t.Run("admins can pre-approve", func(t *testing.T) {
		if got := render(context.WithValue(base, user.AuthedUserKey, &kilonova.UserBrief{ID: 1, Admin: true})); !strings.Contains(got, `name="preApproved" type="checkbox" checked`) {
			t.Fatal(got)
		}
	})
}
