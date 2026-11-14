package authviews

import (
	"testing"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/web/ui/uitest"
)

func TestSignupPage(t *testing.T) {
	t.Run("disabled shows only the notice", func(t *testing.T) {
		uitest.NotContains(t, uitest.Render(t, SignupPage(ctx, SignupParams{})), "signup_form")
	})
	t.Run("posts to the web route and refills the form", func(t *testing.T) {
		got := uitest.Render(t, SignupPage(ctx, SignupParams{Enabled: true, Email: "a@b.c", Username: `u"x`, Language: "ro", Back: "/problems", ErrorMessage: "<bad>"}))
		uitest.Contains(t, got, `method="POST"`, `action="/signup"`, `value="a@b.c"`, `value="u&#34;x"`, `<option value="ro" selected>`, `name="back" value="/problems"`, "&lt;bad&gt;")
		uitest.NotContains(t, got, "postCall", "captcha_container")
	})
	t.Run("captcha only when it must be solved", func(t *testing.T) {
		uitest.Contains(t, uitest.Render(t, SignupPage(ctx, SignupParams{Enabled: true, CaptchaID: "abc"})), `src="/captcha/abc.png"`, `name="captcha_response"`)
	})
}

func TestForgotPasswordPage(t *testing.T) {
	t.Run("posts to the web route without scripts when there is no captcha", func(t *testing.T) {
		got := uitest.Render(t, ForgotPasswordPage(ctx, "", "a@b.c", "<bad>", false))
		uitest.Contains(t, got, `method="POST"`, `action="/forgot_pwd"`, `value="a@b.c"`, "&lt;bad&gt;")
		uitest.NotContains(t, got, "<script", "captcha_container")
	})
	t.Run("captcha fields are submitted with the form", func(t *testing.T) {
		got := uitest.Render(t, ForgotPasswordPage(ctx, "abc", "", "", false))
		uitest.Contains(t, got, `name="captcha_id" value="abc"`, `name="captcha_response"`)
		uitest.NotContains(t, got, "display: none")
	})
	t.Run("sent hides the form", func(t *testing.T) {
		uitest.NotContains(t, uitest.Render(t, ForgotPasswordPage(ctx, "abc", "", "", true)), "<form")
	})
}

func TestResetPasswordPage(t *testing.T) {
	u := &kilonova.UserFull{UserBrief: kilonova.UserBrief{Name: "<u>"}, Email: "a@b.c"}
	got := uitest.Render(t, ResetPasswordPage(ctx, u, `r/1`, "<bad>"))
	uitest.Contains(t, got, "&lt;u&gt;", `action="/resetPassword/r%2F1"`, `value="a@b.c"`, `name="password"`, `name="password_confirm"`, "&lt;bad&gt;")
	uitest.NotContains(t, got, "<script")
}

func TestEmailPages(t *testing.T) {
	uitest.Contains(t, uitest.Render(t, EmailResent(ctx)), `<a href="/">`)
	uitest.NotContains(t, uitest.Render(t, EmailVerified(ctx, "<x>")), "<x>")
}
