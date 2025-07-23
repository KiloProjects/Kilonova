package web

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/internal/util"
	"github.com/KiloProjects/kilonova/web/views/authviews"
	"github.com/KiloProjects/kilonova/web/views/utilviews"
	"github.com/zitadel/oidc/v3/pkg/op"
)

func (rt *Web) getLogin(w http.ResponseWriter, r *http.Request) {
	back := r.URL.Query().Get("back")
	oidcID := r.URL.Query().Get("authRequestID")
	if util.UserBrief(r) == nil {
		rt.runLayout(w, r, &LayoutParams{
			Title:   kilonova.GetText(util.Language(r), "auth.login"),
			Head:    utilviews.CanonicalURL("/login"),
			Content: authviews.LoginPage(oidcID, back, ""),
		})
		return
	}

	if oidcID == "" {
		// authed, no openid flow, just redirect back
		http.Redirect(w, r, rt.hostURL.JoinPath(back).String(), http.StatusFound)
		return
	}

	request, err := rt.base.GetAuthRequest(r.Context(), oidcID)
	if err != nil {
		rt.statusPage(w, r, http.StatusBadRequest, "Invalid auth request")
		return
	}

	client, err := rt.base.GetOAuthClient(r.Context(), request.ApplicationID)
	if err != nil {
		rt.statusPage(w, r, http.StatusBadRequest, "Invalid auth request")
		return
	}

	rt.runLayout(w, r, &LayoutParams{
		Title:   kilonova.GetText(util.Language(r), "auth.oauth_grant"),
		Head:    utilviews.CanonicalURL("/login"),
		Content: authviews.OAuthGrant(request, client),
	})

}

func (rt *Web) handleLogin(w http.ResponseWriter, r *http.Request) {
	switch r.FormValue("form_type") {
	case "login":
		rt.postLogin(w, r)
	case "oauth_grant":
		rt.postOAuthGrant(w, r)
	}
}

func (rt *Web) postLogin(w http.ResponseWriter, r *http.Request) {
	username := r.FormValue("username")
	password := r.FormValue("password")
	oidcID := r.FormValue("oidcID")
	back := r.FormValue("back")

	user, status := rt.base.Login(r.Context(), username, password)
	if status != nil {
		w.WriteHeader(kilonova.ErrorCode(status))
		rt.runLayout(w, r, &LayoutParams{
			Title:   kilonova.GetText(util.Language(r), "auth.login"),
			Head:    utilviews.CanonicalURL("/login"),
			Content: authviews.LoginPage(oidcID, back, status.Error()),
		})
		return
	}

	if user.LockedLogin && !user.Admin {
		// Lockout but don't lockout admins
		w.WriteHeader(401)
		rt.runLayout(w, r, &LayoutParams{
			Title:   kilonova.GetText(util.Language(r), "auth.login"),
			Head:    utilviews.CanonicalURL("/login"),
			Content: authviews.LoginPage(oidcID, back, "Login for this account has been restricted by an administrator"),
		})
		return
	}

	sid, err := rt.base.CreateSession(r.Context(), user.ID)
	if err != nil {
		w.WriteHeader(kilonova.ErrorCode(err))
		rt.runLayout(w, r, &LayoutParams{
			Title:   kilonova.GetText(util.Language(r), "auth.login"),
			Head:    utilviews.CanonicalURL("/login"),
			Content: authviews.LoginPage(oidcID, back, err.Error()),
		})
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "kn-sessionid",
		Value:    sid,
		Expires:  time.Now().Add(29 * 24 * time.Hour),
		SameSite: http.SameSiteLaxMode,
	})
	checkDate := time.Now().Add(10 * 24 * time.Hour)
	http.SetCookie(w, &http.Cookie{
		Name:     "kn-session-check-date",
		Value:    strconv.FormatInt(checkDate.UnixMilli(), 10),
		Expires:  time.Now().Add(29 * 24 * time.Hour),
		SameSite: http.SameSiteLaxMode,
	})

	r = r.WithContext(context.WithValue(r.Context(), util.AuthedUserKey, user))

	if oidcID == "" {
		// authed, no openid flow, just redirect back
		if back == "" {
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}
		http.Redirect(w, r, rt.hostURL.JoinPath(back).String(), http.StatusFound)
		return
	}

	request, err := rt.base.GetAuthRequest(r.Context(), oidcID)
	if err != nil {
		rt.statusPage(w, r, http.StatusBadRequest, "Invalid auth request")
		return
	}

	client, err := rt.base.GetOAuthClient(r.Context(), request.ApplicationID)
	if err != nil {
		rt.statusPage(w, r, http.StatusBadRequest, "Invalid auth request")
		return
	}

	rt.runLayout(w, r, &LayoutParams{
		Title:   kilonova.GetText(util.Language(r), "auth.oauth_grant"),
		Head:    utilviews.CanonicalURL("/login"),
		Content: authviews.OAuthGrant(request, client),
	})
}

func (rt *Web) postOAuthGrant(w http.ResponseWriter, r *http.Request) {
	id := r.FormValue("authRequestID")

	if err := rt.base.ApproveAuthRequest(r.Context(), id, util.UserBrief(r).ID); err != nil {
		slog.ErrorContext(r.Context(), "Failed to approve auth request", slog.Any("error", err))
		rt.statusPage(w, r, http.StatusInternalServerError, "Invalid auth request")
		return
	}

	http.Redirect(w, r, op.AuthCallbackURL(rt.base.OIDCProvider())(r.Context(), id), http.StatusFound)
}
