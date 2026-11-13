package authviews

import (
	"testing"

	"github.com/KiloProjects/kilonova/internal/auth"
	"github.com/KiloProjects/kilonova/web/ui/uitest"
)

var ctx = uitest.Ctx("en")

func TestLoginPage(t *testing.T) {
	t.Run("hidden fields only when set, error escaped", func(t *testing.T) {
		uitest.Contains(t, uitest.Render(t, LoginPage(ctx, "oid", "/back", "<b>bad</b>")), `name="oidcID" value="oid"`, `name="back" value="/back"`, "&lt;b&gt;bad")
		uitest.NotContains(t, uitest.Render(t, LoginPage(ctx, "", "", "")), "oidcID", `name="back"`, "text-red-500")
	})
}

func TestOAuthGrant(t *testing.T) {
	t.Run("client name is escaped inside the markup translation", func(t *testing.T) {
		got := uitest.Render(t, OAuthGrant(ctx, &auth.Request{Scopes: []string{"openid"}}, &auth.Client{Name: "<script>x</script>"}))
		uitest.NotContains(t, got, "<script>x")
		uitest.Contains(t, got, "<li>openid</li>")
	})
}

func TestConfirmLogout(t *testing.T) {
	uitest.Contains(t, uitest.Render(t, ConfirmLogout(ctx, "/p/1")), `action="/logout"`, `name="back" value="/p/1"`)
}
