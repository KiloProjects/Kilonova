package authviews

import (
	"context"
	"strings"
	"testing"

	"github.com/KiloProjects/kilonova/internal/auth"
	"github.com/KiloProjects/kilonova/internal/util"
	. "maragu.dev/gomponents"
)

var ctx = context.WithValue(context.Background(), util.LangKey, "en")

func TestLoginPage(t *testing.T) {
	t.Run("hidden fields only when set, error escaped", func(t *testing.T) {
		must(t, render(t, LoginPage(ctx, "oid", "/back", "<b>bad</b>")), `name="oidcID" value="oid"`, `name="back" value="/back"`, "&lt;b&gt;bad")
		mustNot(t, render(t, LoginPage(ctx, "", "", "")), "oidcID", `name="back"`, "text-red-500")
	})
}

func TestOAuthGrant(t *testing.T) {
	t.Run("client name is escaped inside the markup translation", func(t *testing.T) {
		got := render(t, OAuthGrant(ctx, &auth.Request{Scopes: []string{"openid"}}, &auth.Client{Name: "<script>x</script>"}))
		mustNot(t, got, "<script>x")
		must(t, got, "<li>openid</li>")
	})
}

func TestConfirmLogout(t *testing.T) {
	must(t, render(t, ConfirmLogout(ctx, "/p/1")), `action="/logout"`, `name="back" value="/p/1"`)
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
