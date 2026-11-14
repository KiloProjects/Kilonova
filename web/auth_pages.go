package web

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/Yiling-J/theine-go"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/domain/user"
	"github.com/KiloProjects/kilonova/internal/util"
	"github.com/KiloProjects/kilonova/sudoapi/flags"
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
		Head:    rt.canonicalURL("/signup"),
		Content: authviews.SignupPage(r.Context(), params),
	})
}

func (rt *Web) forgotPasswordPage(w http.ResponseWriter, r *http.Request, status int, errorMessage string, sent bool) {
	captchaID := ""
	if !sent {
		captchaID = rt.captchaID(r.Context())
	}
	w.WriteHeader(status)
	rt.runLayout(w, r, &LayoutParams{
		Title:   kilonova.GetText(util.Language(r), "auth.forgot_pwd"),
		Content: authviews.ForgotPasswordPage(r.Context(), captchaID, r.FormValue("email"), errorMessage, sent),
	})
}

func (rt *Web) forgotPassword(w http.ResponseWriter, r *http.Request) {
	rt.forgotPasswordPage(w, r, http.StatusOK, "", false)
}

func (rt *Web) postForgotPassword() http.HandlerFunc {
	// Limits reset emails to one per address every 2 minutes
	throttler, err := theine.NewBuilder[string, time.Time](5000).Build()
	if err != nil {
		panic(err)
	}
	var mu sync.Mutex
	return func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		lang := util.Language(r)
		email := r.FormValue("email")
		if email == "" {
			rt.forgotPasswordPage(w, r, http.StatusBadRequest, "No email address specified", false)
			return
		}
		if !rt.base.MailerEnabled() {
			rt.forgotPasswordPage(w, r, http.StatusBadRequest, "Mailing subsystem was disabled by administrator", false)
			return
		}
		if flags.CaptchaEnabled.Get(r.Context()) && !rt.base.CheckCaptcha(r.FormValue("captcha_id"), r.FormValue("captcha_response")) {
			rt.forgotPasswordPage(w, r, http.StatusBadRequest, kilonova.GetText(lang, "auth.captcha.invalid"), false)
			return
		}

		if lastSent, ok := throttler.Get(email); ok && time.Since(lastSent) <= 2*time.Minute {
			rt.forgotPasswordPage(w, r, http.StatusTooManyRequests, kilonova.GetText(lang, "auth.forgot_pwd.throttled"), false)
			return
		}
		throttler.SetWithTTL(email, time.Now(), 1, 10*time.Minute)

		userFull, err := rt.base.UserFullByEmail(r.Context(), email)
		if err != nil && !errors.Is(err, kilonova.ErrNotFound) {
			rt.forgotPasswordPage(w, r, kilonova.ErrorCode(err), err.Error(), false)
			return
		}
		if err == nil {
			ctx := context.WithoutCancel(r.Context())
			go func() {
				if err := rt.base.SendPasswordResetEmail(ctx, userFull.ID, userFull.Name, userFull.Email, userFull.PreferredLanguage); err != nil {
					slog.InfoContext(ctx, "Could not send password reset email", slog.Any("err", err), slog.String("email", email))
				}
			}()
		}

		// Same answer whether or not the address has an account
		rt.forgotPasswordPage(w, r, http.StatusOK, "", true)
	}
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

// resetPassword shows the new password form on GET and applies it on POST.
func (rt *Web) resetPassword(w http.ResponseWriter, r *http.Request) {
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

	params := &LayoutParams{Title: kilonova.GetText(util.Language(r), "reset_pwd")}
	errorMessage := ""
	if r.Method == http.MethodPost {
		if r.FormValue("password") != r.FormValue("password_confirm") {
			errorMessage = kilonova.GetText(util.Language(r), "differentPwds")
			w.WriteHeader(http.StatusBadRequest)
		} else if err := rt.base.FinalizePasswordReset(r.Context(), reqid, r.FormValue("password")); err != nil {
			errorMessage = err.Error()
			w.WriteHeader(kilonova.ErrorCode(err))
		} else {
			params.Content = authviews.PasswordResetDone(r.Context())
			rt.runLayout(w, r, params)
			return
		}
	}
	params.Content = authviews.ResetPasswordPage(r.Context(), userFull, reqid, errorMessage)
	rt.runLayout(w, r, params)
}
