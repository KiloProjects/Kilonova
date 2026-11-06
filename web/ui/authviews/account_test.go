package authviews

import (
	"testing"

	"github.com/KiloProjects/kilonova"
)

func TestSignupPage(t *testing.T) {
	t.Run("disabled shows only the notice", func(t *testing.T) {
		mustNot(t, render(t, SignupPage(ctx, SignupParams{})), "signup_form")
	})
	t.Run("no captcha defines the stub functions", func(t *testing.T) {
		got := render(t, SignupPage(ctx, SignupParams{Enabled: true}))
		must(t, got, `id="signup_form"`, "window.location.reload()", `<option value="en" selected>`)
		mustNot(t, got, "captcha_container")
	})
	t.Run("captcha hidden until it must be solved", func(t *testing.T) {
		must(t, render(t, SignupPage(ctx, SignupParams{Enabled: true, CaptchaID: "abc"})), `src="/api/auth/captcha/abc.png"`, `style="display: none;"`)
		mustNot(t, render(t, SignupPage(ctx, SignupParams{Enabled: true, CaptchaID: "abc", ShowCaptcha: true})), "display: none")
	})
}

func TestForgotPasswordPage(t *testing.T) {
	got := render(t, ForgotPasswordPage(ctx, "abc"))
	must(t, got, `id="pwd_forgot_form"`, `value="abc"`, "/auth/forgotPassword")
	mustNot(t, got, "display: none")
}

func TestResetPasswordPage(t *testing.T) {
	u := &kilonova.UserFull{UserBrief: kilonova.UserBrief{Name: "<u>"}, Email: "a@b.c"}
	must(t, render(t, ResetPasswordPage(ctx, u, `r"1`)), "&lt;u&gt;", `value="r&#34;1"`, `value="a@b.c"`)
}

func TestEmailPages(t *testing.T) {
	must(t, render(t, EmailResent(ctx)), `<a href="/">`)
	mustNot(t, render(t, EmailVerified(ctx, "<x>")), "<x>")
}
