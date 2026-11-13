package web

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/domain/user"
	"github.com/KiloProjects/kilonova/internal/util"
	"github.com/KiloProjects/kilonova/sudoapi/flags"
	"github.com/KiloProjects/kilonova/web/ui"
	"github.com/KiloProjects/kilonova/web/ui/authviews"
)

// captchaID returns a fresh CAPTCHA ID, or "" when CAPTCHAs are disabled.
func (rt *Web) captchaID(ctx context.Context) string {
	if !flags.CaptchaEnabled.Get(ctx) {
		return ""
	}
	return rt.base.NewCaptchaID()
}

func (rt *Web) signup(w http.ResponseWriter, r *http.Request) {
	params := authviews.SignupParams{Enabled: flags.SignupEnabled.Get(r.Context())}
	if params.Enabled {
		params.CaptchaID = rt.captchaID(r.Context())
		if params.CaptchaID != "" {
			ip, _ := rt.base.GetRequestInfo(r)
			params.ShowCaptcha = rt.base.MustSolveCaptcha(r.Context(), ip)
		}
	}
	rt.runLayout(w, r, &LayoutParams{
		Title:   kilonova.GetText(util.Language(r), "auth.signup"),
		Head:    ui.CanonicalURL("/signup"),
		Content: authviews.SignupPage(r.Context(), params),
	})
}

func (rt *Web) forgotPassword(w http.ResponseWriter, r *http.Request) {
	rt.runLayout(w, r, &LayoutParams{
		Title:   kilonova.GetText(util.Language(r), "auth.forgot_pwd"),
		Content: authviews.ForgotPasswordPage(r.Context(), rt.captchaID(r.Context())),
	})
}

func (rt *Web) resendEmail() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u := user.UserFull(r)
		if u.VerifiedEmail {
			rt.statusPage(w, r, 403, "Deja ai verificat emailul!")
			return
		}
		t := time.Since(u.EmailVerifResent)
		if t < 5*time.Minute {
			text := fmt.Sprintf("Trebuie să mai aștepți %s până poți retrimite email de verificare", (5*time.Minute - t).Truncate(time.Millisecond))
			rt.statusPage(w, r, 403, text)
			return
		}
		if err := rt.base.SendVerificationEmail(context.WithoutCancel(r.Context()), u.ID, u.Name, u.Email, u.PreferredLanguage); err != nil {
			slog.WarnContext(r.Context(), "Could not resend verification email", slog.Any("err", err))
			rt.statusPage(w, r, 500, "N-am putut retrimite emailul de verificare")
			return
		}

		rt.runLayout(w, r, &LayoutParams{
			Title:   kilonova.GetText(util.Language(r), "email.resent"),
			Content: authviews.EmailResent(r.Context()),
		})
	}
}

func (rt *Web) verifyEmail() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vid := r.PathValue("vid")
		if !rt.base.CheckVerificationEmail(r.Context(), vid) {
			rt.statusPage(w, r, 404, "")
			return
		}

		uid, err := rt.base.GetVerificationUser(r.Context(), vid)
		if err != nil {
			slog.WarnContext(r.Context(), "Could not get email verification", slog.Any("err", err))
			rt.statusPage(w, r, 404, "")
			return
		}

		userBrief, err := rt.base.UserBrief(r.Context(), uid)
		if err != nil {
			slog.WarnContext(r.Context(), "Couldn't get user", slog.Any("err", err))
			rt.statusPage(w, r, 404, "")
			return
		}

		if err := rt.base.ConfirmVerificationEmail(context.WithoutCancel(r.Context()), vid, userBrief); err != nil {
			slog.WarnContext(r.Context(), "Could not confirm email", slog.Any("err", err))
			rt.statusPage(w, r, 404, "")
			return
		}

		// rebuild session for user to disable popup
		rt.initSession(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rt.runLayout(w, r, &LayoutParams{
				Title:   kilonova.GetText(util.Language(r), "emailConfirm"),
				Content: authviews.EmailVerified(r.Context(), userBrief.Name),
			})
		})).ServeHTTP(w, r)
	}
}

func (rt *Web) resetPassword() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqid := r.PathValue("reqid")
		if !rt.base.CheckPasswordResetRequest(r.Context(), reqid) {
			rt.statusPage(w, r, 404, "")
			return
		}

		uid, err := rt.base.GetPwdResetRequestUser(r.Context(), reqid)
		if err != nil {
			slog.WarnContext(r.Context(), "Couldn't get user from password reset request", slog.Any("err", err))
			rt.statusPage(w, r, 404, "")
			return
		}

		userFull, err := rt.base.UserFull(r.Context(), uid)
		if err != nil {
			slog.WarnContext(r.Context(), "Couldn't get full user", slog.Any("err", err))
			rt.statusPage(w, r, 404, "")
			return
		}

		rt.runLayout(w, r, &LayoutParams{
			Title:   kilonova.GetText(util.Language(r), "reset_pwd"),
			Content: authviews.ResetPasswordPage(r.Context(), userFull, reqid),
		})
	}
}
