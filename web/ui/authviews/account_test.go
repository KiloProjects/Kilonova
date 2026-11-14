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
	t.Run("no captcha defines the stub functions", func(t *testing.T) {
		got := uitest.Render(t, SignupPage(ctx, SignupParams{Enabled: true}))
		uitest.Contains(t, got, `id="signup_form"`, "window.location.reload()", `<option value="en" selected>`)
		uitest.NotContains(t, got, "captcha_container")
	})
	t.Run("captcha hidden until it must be solved", func(t *testing.T) {
		uitest.Contains(t, uitest.Render(t, SignupPage(ctx, SignupParams{Enabled: true, CaptchaID: "abc"})), `src="/api/auth/captcha/abc.png"`, `style="display: none;"`)
		uitest.NotContains(t, uitest.Render(t, SignupPage(ctx, SignupParams{Enabled: true, CaptchaID: "abc", ShowCaptcha: true})), "display: none")
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
