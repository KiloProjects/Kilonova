package settingsviews

import (
	"context"
	"testing"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/internal/auth"
	"github.com/KiloProjects/kilonova/internal/util"
	"github.com/KiloProjects/kilonova/web/ui/uitest"
)

var ctx = context.WithValue(context.Background(), util.LangKey, "en")

func TestOAuthClients(t *testing.T) {
	aid, missing := 1, 99
	clients := []*auth.Client{{Name: "<b>App</b>", AuthorID: &aid, DeveloperMode: true}, {Name: "Other", AuthorID: &missing}}

	t.Run("settings page: create form, no author column, names escaped", func(t *testing.T) {
		got := uitest.Render(t, OAuthClients(ctx, OAuthClientsParams{Clients: clients, ShowCreate: true}))
		uitest.Contains(t, got, `action="/settings/oauth"`, "&lt;b&gt;App", ">dev<")
		uitest.NotContains(t, got, "<b>App", `name="back"`, `href="/profile/`)
	})
	t.Run("secret is shown once after creation", func(t *testing.T) {
		uitest.Contains(t, uitest.Render(t, OAuthClients(ctx, OAuthClientsParams{CreatedID: "id1", CreatedSecret: "s3cr3t"})), "<code>s3cr3t</code>")
	})
	t.Run("admin overview links authors and returns to admin after delete", func(t *testing.T) {
		got := uitest.Render(t, OAuthClients(ctx, OAuthClientsParams{Clients: clients, Authors: map[int]*kilonova.UserBrief{1: {Name: "alice"}}}))
		uitest.Contains(t, got, `<a href="/profile/alice">alice</a>`, `name="back" value="admin"`, `<td class="kn-table-cell">-</td>`)
		uitest.NotContains(t, got, `action="/settings/oauth"`)
	})
	t.Run("empty list shows the placeholder", func(t *testing.T) {
		uitest.NotContains(t, uitest.Render(t, OAuthClients(ctx, OAuthClientsParams{})), "<table")
	})
}

func TestAuthorizedApps(t *testing.T) {
	got := uitest.Render(t, AuthorizedApps(ctx, []*auth.Client{{Name: "App"}}))
	uitest.Contains(t, got, "/revoke", `onsubmit="return confirm(this.dataset.confirm)"`, `type="submit"`)
	uitest.NotContains(t, uitest.Render(t, AuthorizedApps(ctx, nil)), "<table")
}
