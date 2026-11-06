package settingsviews

import (
	"context"
	"strings"
	"testing"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/internal/auth"
	"github.com/KiloProjects/kilonova/internal/util"
	. "maragu.dev/gomponents"
)

var ctx = context.WithValue(context.Background(), util.LangKey, "en")

func TestOAuthClients(t *testing.T) {
	aid, missing := 1, 99
	clients := []*auth.Client{{Name: "<b>App</b>", AuthorID: &aid, DeveloperMode: true}, {Name: "Other", AuthorID: &missing}}

	t.Run("settings page: create form, no author column, names escaped", func(t *testing.T) {
		got := render(t, OAuthClients(ctx, OAuthClientsParams{Clients: clients, ShowCreate: true}))
		must(t, got, `action="/settings/oauth"`, "&lt;b&gt;App", ">dev<")
		mustNot(t, got, "<b>App", `name="back"`, `href="/profile/`)
	})
	t.Run("secret is shown once after creation", func(t *testing.T) {
		must(t, render(t, OAuthClients(ctx, OAuthClientsParams{CreatedID: "id1", CreatedSecret: "s3cr3t"})), "<code>s3cr3t</code>")
	})
	t.Run("admin overview links authors and returns to admin after delete", func(t *testing.T) {
		got := render(t, OAuthClients(ctx, OAuthClientsParams{Clients: clients, Authors: map[int]*kilonova.UserBrief{1: {Name: "alice"}}}))
		must(t, got, `<a href="/profile/alice">alice</a>`, `name="back" value="admin"`, `<td class="kn-table-cell">-</td>`)
		mustNot(t, got, `action="/settings/oauth"`)
	})
	t.Run("empty list shows the placeholder", func(t *testing.T) {
		mustNot(t, render(t, OAuthClients(ctx, OAuthClientsParams{})), "<table")
	})
}

func TestAuthorizedApps(t *testing.T) {
	got := render(t, AuthorizedApps(ctx, []*auth.Client{{Name: "App"}}))
	must(t, got, "/revoke", `onsubmit="return confirm(this.dataset.confirm)"`, `type="submit"`)
	mustNot(t, render(t, AuthorizedApps(ctx, nil)), "<table")
}

func render(t *testing.T, n Node) string {
	t.Helper()
	var b strings.Builder
	if err := n.Render(&b); err != nil {
		t.Fatal(err)
	}
	return b.String()
}

func must(t *testing.T, got string, want ...string) {
	t.Helper()
	for _, w := range want {
		if !strings.Contains(got, w) {
			t.Fatalf("missing %q in %s", w, got)
		}
	}
}

func mustNot(t *testing.T, got string, bad ...string) {
	t.Helper()
	for _, b := range bad {
		if strings.Contains(got, b) {
			t.Fatalf("unexpected %q in %s", b, got)
		}
	}
}
