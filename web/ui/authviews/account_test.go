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
	got := uitest.Render(t, ForgotPasswordPage(ctx, "abc"))
	uitest.Contains(t, got, `id="pwd_forgot_form"`, `value="abc"`, "/auth/forgotPassword")
	uitest.NotContains(t, got, "display: none")
}

func TestResetPasswordPage(t *testing.T) {
	u := &kilonova.UserFull{UserBrief: kilonova.UserBrief{Name: "<u>"}, Email: "a@b.c"}
	uitest.Contains(t, uitest.Render(t, ResetPasswordPage(ctx, u, `r"1`)), "&lt;u&gt;", `value="r&#34;1"`, `value="a@b.c"`)
}

func TestEmailPages(t *testing.T) {
	uitest.Contains(t, uitest.Render(t, EmailResent(ctx)), `<a href="/">`)
	uitest.NotContains(t, uitest.Render(t, EmailVerified(ctx, "<x>")), "<x>")
}
